package lib

import (
  "os"
  // "os/signal"
  "log"
)

func X(sigs<-chan os.Signal, done chan <- bool ) {
  sig := <-sigs
  log.Println("Stopping x", sig)
  done <- true
}