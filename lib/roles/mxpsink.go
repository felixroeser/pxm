package roles

import (
	"log"
	"fmt"
	"net/http"
	"github.com/gorilla/mux"
	"github.com/braintree/manners"
)

type MxpSink struct {
	Port int
	State int
}

func (m *MxpSink) Start(sigs<-chan bool, done chan <- bool) {
	log.Printf("* Starting MxpSink on port %d", m.Port)
	
	r := mux.NewRouter().StrictSlash(false)
	r.HandleFunc("/", m.rootHandler).Methods("GET")
	
	go func(){
		sig := <-sigs
		log.Println("Stopping MxpSink", sig)
		manners.Close()
	}()
	
	if err := manners.ListenAndServe(fmt.Sprintf(":%d", m.Port), r); err != nil {
		log.Fatal(err)
	}
	
	done <- true
}

func (m *MxpSink) rootHandler(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(rw, "{}")
}

