# shutdown

[![GoDoc](https://godoc.org/github.com/icza/shutdown?status.svg)](https://godoc.org/github.com/icza/shutdown)

Package shutdown aids graceful termination of goroutines on app shutdown.

It listens for SIGTERM and SIGINT signals, and also provides a manual
way to trigger shutdown.

It publishes a single, shared shutdown channel which is closed when shutdown
is about to happen. Modules (goroutines) should monitor this channel
using a `select` statement, and terminate ASAP if it is (gets) closed.

It also publishes a `WaitGroup` goroutines may use to "register" themselves
should they wish to be patiently waited for and not get terminated abruptly.
For this to "work", this shared `WaitGroup` must be "waited for"
in the `main()` function before returning.

Example app using it

	func main() {
		// Initiate a manual shutdown if we're still running after 10 sec
		time.AfterFunc(10*time.Second, shutdown.InitiateManual)

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
