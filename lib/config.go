package lib

import (
  "strings"
  "strconv"
  "os"
  "net/url"
  "gopkg.in/gcfg.v1"
)

// https://github.com/go-gcfg/gcfg/blob/v1/example_test.go
type ConfigStruct struct {
  Cassandra struct {
  	Uri []string
  	Hosts []string
  	Keyspace string
  }
  Policy struct {
  	Default string
  	People string
  	Transactions string
  	Beacons string
  }
  Tokens struct {
    Admin []string
  	Trusted []string
  	Untrusted []string
  }
  Beacons map[string]*struct {
  	Policy string
    Ttl int
  }
  Mxpsink struct {
    RawPort []string `gcfg:"port"`
    IPort int
  }
}

func pimpCassandra(cfg *ConfigStruct) (err error) {
  uri := grabFirstMatch(cfg.Cassandra.Uri)
  url, _ := url.Parse(uri)

  hosts := []string{url.Host}
  if v, ok := url.Query()["host"]; ok {
    hosts = append(hosts, v...)
  }

  cfg.Cassandra.Hosts = hosts
  cfg.Cassandra.Keyspace = strings.Replace(url.Path, "/", "", 1)

  return
}

func pimpMxpSink(cfg *ConfigStruct) (err error) {
  i, _ := strconv.ParseInt(grabFirstMatch(cfg.Mxpsink.RawPort), 0, 64)
  cfg.Mxpsink.IPort = int(i)
  return
}

func LoadConfig(filename string, cfg *ConfigStruct) (err error) {
  // FIXME pretty ugly
  if err = gcfg.ReadFileInto(cfg, filename); err == nil {
    if err = pimpCassandra(cfg); err == nil { 
      err = pimpMxpSink(cfg)
    }
  }
  return 
}

func grabFirstMatch(a []string) (out string) {
    for _, s := range a {
      if strings.HasPrefix(s, "$") {
        s = strings.Replace(s, "$", "", 1)
        if v := os.Getenv(s); len(v) > 0 {
          out = v
          break
        }
      } else {
        out = s
        break
      }  
    }
    return
}