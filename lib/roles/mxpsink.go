package roles

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/braintree/manners"
	"github.com/gocql/gocql"
	"github.com/gorilla/mux"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/felixroeser/pxm/lib"
	"github.com/felixroeser/pxm/lib/storage"
)

func MxpSinkFactory(context *lib.Context) MxpSink {
	return MxpSink{context.Cfg.Mxpsink.IPort, 0, make(chan Beacon, 500)}
}

type MxpSink struct {
	Port               int
	State              int
	IncomingBeaconChan chan Beacon
}

type trackRequest struct {
	// r *http.Request
	q  url.Values
	i  bool
	ip string
}

type VerboseTrackResponse struct {
	Status     int      `json:"status"`
	RequestIDs []string `json:"request_ids"`
	Error      string
}

type IncomingBeacon struct {
	Event      string      `json:"event"`
	Properties interface{} `json:"properties"`
}

type Beacon struct {
	Event      string            `json:"event"`
	RequestID  gocql.UUID        `json:"request_id"`
	SessionID  string            `json:"session_id"`
	DistinctID string            `json:"distinct_id"`
	Properties interface{}       `json:"properties"`
	Meta       map[string]string `json:"-"`
	Time       time.Time         `json:time`
	IP         string            `json:"ip"`
}

type AliasRequest struct {
	DistinctID string
	AliasID    string
}

type ForSimpleCounter struct {
	Event     string
	Timeframe string
}

func (b *Beacon) ForInsert() []interface{} {
	return []interface{}{b.Event, b.RequestID, b.DistinctID, b.SessionID, b.Properties, b.IP}
}

func parseQueryPayload(q url.Values) (incomingBeacons []IncomingBeacon, verbose bool, returnErr error) {

	// log.Println(q)

	for _, a := range q["verbose"] {
		if a == "1" || a == "true" {
			verbose = true
		}
	}

	for _, encoded := range q["data"] {
		data, _ := base64.StdEncoding.DecodeString(encoded)
		// log.Println(string(data))

		// single or array? https://groups.google.com/forum/#!topic/golang-nuts/rKVn8coJMlQ
		err := json.Unmarshal(data, &incomingBeacons)
		if err != nil {
			var incomingBeacon IncomingBeacon
			err2 := json.Unmarshal(data, &incomingBeacon)
			if err2 != nil {
				returnErr = errors.New("Failed to parsed")
				return
			}

			incomingBeacons = append(incomingBeacons, incomingBeacon)
		}
	}

	return
}

func beaconsFromIncomingBeacons(incomingBeacons *[]IncomingBeacon, tr trackRequest) (beacons []Beacon, aliases []AliasRequest) {
	for _, ib := range *incomingBeacons {
		p := ib.Properties.(map[string]interface{})

		// TODO check token
		delete(p, "token")

		// special handling for the $create_alias event
		if ib.Event == "$create_alias" {
			a := AliasRequest{DistinctID: p["distinct_id"].(string), AliasID: p["alias"].(string)}
			aliases = append(aliases, a)
			continue
		}

		var t time.Time
		if f, ok := p["time"]; ok {
			t = time.Unix(int64(f.(float64)), 0).UTC()

			if tr.i != true && t.Before(time.Now().Add(time.Duration(-5)*time.Second)) {
				t = time.Now().UTC()
			}

			delete(p, "time")
		} else {
			t = time.Now()
		}
		rid := gocql.UUIDFromTime(t)

		var ip string
		if s, ok := p["ip"]; ok {
			ip = s.(string)
			delete(p, "ip")
		} else {
			ip = tr.ip
		}

		var did string
		if d, ok := p["distinct_id"]; ok && d != nil {
			did = d.(string)
		} else {
			// otherwise use the requestid
			did = rid.String()
		}
		delete(p, "distinct_id")

		var sid string
		if d, ok := p["session_id"]; ok && d != nil {
			sid = d.(string)
		} else {
			// otherwise use the requestid
			sid = rid.String()
		}
		delete(p, "session_id")

		properties := make(map[string]interface{})
		meta := make(map[string]string)

		ignoreList := map[string]bool{
			"mp_lib":       true,
			"$lib_version": true,
		}

		for k, v := range p {

			if listed, found := ignoreList[k]; !found {
				properties[k] = v
			} else if listed {
				meta[k] = v.(string)
			}
		}

		b := Beacon{Event: ib.Event, RequestID: rid, DistinctID: did, SessionID: sid, Time: t, Properties: properties, IP: ip, Meta: meta}
		beacons = append(beacons, b)
	}
	return
}

func getRequestIDs(beacons *[]Beacon) (requestIDs []string) {
	for _, b := range *beacons {
		requestIDs = append(requestIDs, b.RequestID.String())
	}
	return
}

func (m *MxpSink) Start(sigs <-chan bool, done chan<- bool) {
	log.Printf("* Starting MxpSink on port %d", m.Port)

	r := mux.NewRouter().StrictSlash(false)

	r.HandleFunc("/", m.rootHandler).Methods("GET")
	r.HandleFunc("/track", m.trackGetHandler).Methods("GET")
	r.HandleFunc("/track", m.trackPostHandler).Methods("POST")

	incomingBeaconKiller := make(chan bool)
	incomingBeaconStopped := make(chan bool)

	go m.incomingBeaconConsumer(incomingBeaconKiller, incomingBeaconStopped)

	go func() {
		sig := <-sigs
		log.Println("Stopping MxpSink", sig)
		incomingBeaconKiller <- true
		manners.Close()
	}()

	if err := manners.ListenAndServe(fmt.Sprintf(":%d", m.Port), r); err != nil {
		log.Fatal(err)
	}

	<-incomingBeaconStopped

	done <- true
}

func (m *MxpSink) incomingBeaconConsumer(killer, done chan bool) {
	c := true

	for c {
		select {
		case b := <-m.IncomingBeaconChan:
			tables := [2]string{"beacons", "beacons_by_did"}
			for _, t := range tables {
				stmt := fmt.Sprintf("INSERT INTO %s (event, request_id, distinct_id, session_id, properties, ip) VALUES (?, ?, ?, ?, ?, ?)", t)				
				if err := storage.ExecWriteQuery(stmt, b.ForInsert()...); err != nil {
					log.Println("Failed to store beacon", err)
				}
			}
		case d := <-killer:
			log.Println("incomingBeaconConsumer going to stop! but still have to flush some beacons", d)
			// FIXME flush channel
			c = false
		}
	}

	done <- true
}

func (m *MxpSink) rootHandler(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(rw, "{}")
}

func (m *MxpSink) trackGetHandler(rw http.ResponseWriter, r *http.Request) {
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	tr := trackRequest{r.URL.Query(), false, ip}
	m.trackRequestHandler(&rw, tr)
}

func (m *MxpSink) trackPostHandler(rw http.ResponseWriter, r *http.Request) {
	buf := new(bytes.Buffer)
	buf.ReadFrom(r.Body)
	q, _ := url.ParseQuery(buf.String())

	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	tr := trackRequest{q, false, ip}
	m.trackRequestHandler(&rw, tr)
}

func (m *MxpSink) trackRequestHandler(rw *http.ResponseWriter, tr trackRequest) {
	var beacons []Beacon
	// var aliases []AliasRequest

	incomingBeacons, verbose, err := parseQueryPayload(tr.q)

	if err == nil {
		beacons, _ = beaconsFromIncomingBeacons(&incomingBeacons, tr)
	}

	for _, b := range beacons {
		m.IncomingBeaconChan <- b
	}

	(*rw).Header().Set("Content-Type", "application/json")

	if verbose {
		response := VerboseTrackResponse{1, getRequestIDs(&beacons), ""}
		res, _ := json.Marshal(response)
		fmt.Fprintf(*rw, string(res))
	} else {
		fmt.Fprintf(*rw, "{\"status\": 1}")
	}
}
