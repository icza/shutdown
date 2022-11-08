package shutdown

import (
	"context"
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
)

var (
	// Context's channel is cancelled on shutdown
	Context, cancel = context.WithCancel(context.Background())

	// C is the shutdown channel.
	C <-chan struct{} = Context.Done()

	// Wg is the shared WaitGroup goroutines may use to "register" themselves
	// if they wish to be waited for on app shutdown.
	Wg = &sync.WaitGroup{}
)

func init() {
	// Register sigch for SIGTERM and SIGINT.
	signal.Notify(sigch, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		defer signal.Stop(sigch)

		s := <-sigch
		// We only subscribed to signals to which we have to shutdown
		log.Printf("Received '%v' signal, broadcasting shutdown...", s)

		cancel()
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
	case <-C:
		return true
	default:
	}
	return false
}
