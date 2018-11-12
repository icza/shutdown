package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/icza/shutdown"
)

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
