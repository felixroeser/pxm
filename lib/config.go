package lib

import (
  "strings"
  "os"
  "net/url"
  "gopkg.in/gcfg.v1"
)

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
}

func pimpCassandra(cfg *ConfigStruct) (err error) {
  var uri string
  for _, s := range cfg.Cassandra.Uri {
    if strings.HasPrefix(s, "$") {
      s = strings.Replace(s, "$", "", 1)
      if v := os.Getenv(s); len(v) > 0 {
        uri = v
        break
      }
    } else {
      uri = s
    }
  }

  url, _ := url.Parse(uri)

  hosts := []string{url.Host}
  if v, ok := url.Query()["host"]; ok {
    hosts = append(hosts, v...)
  }

  cfg.Cassandra.Hosts = hosts
  cfg.Cassandra.Keyspace = strings.Replace(url.Path, "/", "", 1)

  return
}

func LoadConfig(filename string, cfg *ConfigStruct) (err error) {
  if err = gcfg.ReadFileInto(cfg, filename); err == nil {
    err = pimpCassandra(cfg)
  }

  return 
}
