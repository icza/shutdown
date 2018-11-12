package main

import (
	"log"
	"time"

	"github.com/icza/shutdown"
)

func main() {
	go func() {
		// This is your app:
		for {
			log.Println("Tick...")
			time.Sleep(time.Second)
		}
	}()

	<-shutdown.C

	log.Println("Doing this before shutting down.")
}
