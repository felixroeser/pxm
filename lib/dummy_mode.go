package lib

import (
  "log"
)

func X(sigs<-chan bool, done chan <- bool ) {
  sig := <-sigs
  log.Println("Stopping x", sig)
  done <- true
}