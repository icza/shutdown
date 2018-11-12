package main

import (
	"log"
	"time"

	"github.com/icza/shutdown"
)

func main() {
	// Initiate a manual shutdown if we're still running after 10 sec
	time.AfterFunc(10*time.Second, shutdown.InitiateManual)

	// Example generator (job producer)
	jobCh := make(chan int)
	go func() {
		for i := 0; ; i++ {
			jobCh <- i
		}
	}()

	// Example worker goroutine whose completion we will wait for.
	shutdown.Wg.Add(1)
	go func() {
		defer shutdown.Wg.Done()
		for {
			// Receive jobs, listen for shutdown:
			select {
			case jobID := <-jobCh:
				log.Printf("[worker] Doing job #%d...", jobID)
				time.Sleep(time.Second) // Simulate work...
			case <-shutdown.C:
				log.Println("[worker] Aborting. Saving progress...")
				time.Sleep(time.Second) // Simulate work...
				log.Println("[worker] Save complete.")
				return
			}
		}
	}()

	// Wait for a shutdown event (either signal or manual)
	<-shutdown.C

	// Wait for "important" goroutines
	shutdown.Wg.Wait()
}
