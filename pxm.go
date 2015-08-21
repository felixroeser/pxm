package main

import (
  "fmt"
  "log"
  "flag"
  "strings"
  "os"
  "os/signal"
  "syscall"

  "github.com/felixroeser/pxm/lib"
  "github.com/felixroeser/pxm/lib/storage"
  "github.com/felixroeser/pxm/lib/roles"
)

func main() {

  log.Println("-> pxm 0.0")

  var configFilename string
  flag.StringVar(&configFilename, "config", "/etc/pxm.ini", "relative or absolute path to a pxm config file")

  cmd := flag.String("cmd", "blank", "internal commands to be execute before start: drop,create,migrate,flush" )

  flag.Parse()
  
  if err := lib.LoadConfig(configFilename, &lib.Cfg); err != nil {
    log.Fatal("Failed to load/parse config", err)
  }
  
  var err error  
  if lib.Session, err = storage.Connect(); err != nil {
    log.Fatal(err)
  }
  context, _ := lib.GetContext()
  defer context.Close()
  
  for _, c := range strings.Split(*cmd, ",") {
    switch c {
      case "drop":
        storage.Drop()        
      case "migrate":
        storage.Migrate()
      case "flush":
        storage.Flush() 
      default:
        log.Printf("unsupported command: %s", c)         
    }
  }
  
  started := 0 
  var stoppers []chan bool
  sigs := make(chan os.Signal)
  done := make(chan bool, 1)
  signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

  for _, m := range flag.Args() {
    switch m {
      case "dummy":
        log.Println("* Starting endless dummy")
        c := make(chan bool)
        go lib.X(c, done)
        stoppers = append(stoppers, c)
        started = started + 1        
      case "mxpsink":
        ms := roles.MxpSink{context.Cfg.Mxpsink.IPort, 0}
        c := make(chan bool)
        go ms.Start(c, done)
        stoppers = append(stoppers, c)        
        started = started + 1
      case "mxpstream":
        log.Println("* mxp stream to be implemented")
      case "debug":
       log.Println("* debug mode to be implemented")
      default:
        log.Println("* unknown mode", m)
    }
    
  }
  
  fmt.Println("Waiting until the sun goes down...")

  _ = <-sigs
  
  for i := range stoppers {
    stoppers[i] <- true
  }
  
  for c := 0; c < started; c++ {
    <- done
  }
  
  os.Exit(0)
}
