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
  
  for _, mode := range flag.Args() {
    log.Printf("* starting %s", mode)
  }
  
  os.Exit(0)

  sigs := make(chan os.Signal, 1)
  done := make(chan bool, 1)

  signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

  go lib.X(sigs, done)

  fmt.Println("awaiting signal")
  for c := 0; c < 1; c++ {
    <- done
  }
  fmt.Println("exiting")
}
