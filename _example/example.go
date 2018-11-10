package main

import (
	"log"
	"time"

	"github.com/icza/shutdown"
)

func main() {
	// Example goroutine that initiates manual shutdown (after 10 seconds)
	go func() {
		time.Sleep(10 * time.Second)
		shutdown.InitiateManual()
	}()

	// Example worker goroutine whose completion we will wait for.
	shutdown.Wg.Add(1)
	go func() {
		defer shutdown.Wg.Done()
		for i := 0; ; i++ {
			log.Printf("[worker] Doing task #%d...", i)
			time.Sleep(time.Second) // Simulate work...
			// Check for shutdown event
			select {
			case <-shutdown.C:
				log.Println("[worker] Aborting; first saving progress (1 sec)...")
				time.Sleep(time.Second)
				log.Println("[worker] Save complete.")
				return
			default:
			}
		}
	}()

	// Wait for a shutdown event (either signal or manual)
	<-shutdown.C

	// Wait for "important" goroutines
	shutdown.Wg.Wait()
}
