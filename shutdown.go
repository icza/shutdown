/*

Package shutdown helps controlling app shutdown and graceful termination of goroutines.

It listens for SIGTERM (e.g. kill command) and SIGINT (e.g. CTRL+C) signals,
and also provides a manual way to trigger shutdown.

It publishes a single, shared shutdown channel which is closed when shutdown
is about to happen. Modules (goroutines) should monitor this channel
using a select statement, and terminate ASAP if it is (gets) closed. Additionally,
there is an Initiated() function which tells if a shutdown has been initiated, which
basically checks the shared channel in a non-blocking way.

It also publishes a WaitGroup goroutines may use to "register" themselves
should they wish to be patiently waited for and not get terminated abruptly.
For this to "work", this shared WaitGroup must be "waited for"
in the main() function before returning.

Simple example

If you just want to do something before shutting down:

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

Note that monitoring the shutdown channel must be on the main goroutine and your
task in another one (and not the other way), because the app terminates when the
main() function returns.

Advanced example

A more advanced example where a worker goroutine is to be waited for.
This app also self-terminates after 10 seconds:

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

Advanced example variant

Note that the above worker goroutine does not guarantee that it won't start execution
of a new job after a shutdown has been initiated (because select chooses a "ready" case
pseudo-randomly). If you need guarantee that no new jobs are taken after a shutdown initiation,
you may check the shutdown channel first, in a separate select in a non-blocking way,
or you may simply add the check as the loop condition like this:

	// Example worker goroutine whose completion we will wait for.
	shutdown.Wg.Add(1)
	go func() {
		defer shutdown.Wg.Done()
		defer func() {
			log.Println("[worker] Aborting. Saving progress...")
			time.Sleep(time.Second) // Simulate work...
			log.Println("[worker] Save complete.")
		}()
		for !shutdown.Initiated() {
			// Receive jobs, listen for shutdown:
			select {
			case jobID := <-jobCh:
				log.Printf("[worker] Doing job #%d...", jobID)
				time.Sleep(time.Second) // Simulate work...
			case <-shutdown.C:
				return
			}
		}
	}()

Web server example

The following example starts a web server and provides graceful shutdown for it.
It also handles abnormal (and slient) termination, in which case it triggers a
manual shutdown, making sure the whole app gets terminated (not just its web server):

	func main() {
		helloFunc := func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("hello"))
		}
		srv := &http.Server{
			Addr:    ":8080",
			Handler: http.HandlerFunc(helloFunc),
		}

		go func() {
			if err := srv.ListenAndServe(); err != nil {
				if err == http.ErrServerClosed {
					log.Println("HTTP Server gracefully shut down.")
					return
				}
				log.Printf("Abnormal HTTP Server shut down with error: %v", err)
			} else {
				log.Println("HTTP Server SILENTLY shut down.")
			}

			// If we got to this point, that's not normal:
			log.Println("Initiating manual system shutdown:")
			shutdown.InitiateManual()
		}()

		// Wait for a shutdown event (either signal or manual)
		<-shutdown.C

		log.Println("Stopping HTTP server (system shutdown)...")

		// Shutdown gracefully, but wait no longer than 20 seconds:
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
		err := srv.Shutdown(ctx)
		cancel() // Call cancel to release resources of the context

		if err != nil {
			log.Printf("Failed to shut down HTTP server gracefully: %v", err)
			// Try forceful shutdown:
			if err := srv.Close(); err != nil {
				log.Printf("HTTP server forceful shutdown error: %v", err)
			}
		}
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

// Initiated tells if a shutdown has been initiated, either by a signal or manually.
func Initiated() bool {
	select {
	case <-c:
		return true
	default:
	}
	return false
}
