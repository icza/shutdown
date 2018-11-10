/*

Package shutdown aids graceful termination of goroutines on app shutdown.

It listens for SIGTERM and SIGINT signals, and also provides a manual
way to trigger shutdown.

It publishes a single, shared shutdown channel which is closed when shutdown
is about to happen. Modules (goroutines) should monitor this channel
using a select statement, and terminate ASAP if it is (gets) closed.

It also publishes a WaitGroup goroutines may use to "register" themselves
should they wish to be patiently waited for and not get terminated abruptly.
For this to "work", this shared WaitGroup must be "waited for"
in the main() function before returning.

Example app using it

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

*/
package shutdown

import (
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

var (
	// sigch is a signal channel used to receive SIGTERM and SIGINT (CTRL+C).
	// Buffered to make sure we don't miss it (send on it is non-blocking).
	sigch = make(chan os.Signal, 1)

	// c is the internal, bidirectional channel
	c = make(chan struct{})
)

// C is the shutdown channel.
var C <-chan struct{} = c

// Wg is the shared WaitGroup goroutines may use to "register" themselves
// if they wish to be waited for on app shutdown.
var Wg = &sync.WaitGroup{}

func init() {
	// Register sigch for SIGTERM and SIGINT.
	signal.Notify(sigch, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		defer signal.Stop(sigch)

		s := <-sigch
		// We only subscribed to signals to which we have to shutdown
		log.Printf("Received '%v' signal, broadcasting shutdown...", s)

		close(c)
	}()
}

// InitiateManual initiates a manual shutdown.
func InitiateManual() {
	log.Println("Manual shutdown initiated...")

	// Imit a SIGTERM signal. Do non-blocking send!
	select {
	case sigch <- syscall.SIGTERM:
	default:
	}
}
