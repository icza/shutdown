package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/icza/shutdown"
)

func main() {
	// Worker goroutine requiring a context (we'll wait for its completion).
	shutdown.Wg.Add(1)
	go func() {
		defer shutdown.Wg.Done()
		ctx := shutdown.Context
		for !shutdown.Initiated() {
			result, err := dbAdapter.RunQuery(ctx, "some-query")
			if err != nil {
				log.Printf("Query error: %v", err)
			} else {
				log.Printf("Query result: %v", result)
			}
		}
	}()

	// Wait for a shutdown event (either signal or manual)
	<-shutdown.C

	// Wait for the worker to finish
	shutdown.Wg.Wait()
}

type mockDB struct {
	count int
}

func (db *mockDB) RunQuery(ctx context.Context, query string) (result any, err error) {
	// Simulate work
	select {
	case <-time.After(time.Second):
	case <-ctx.Done():
		return nil, fmt.Errorf("query aborted: %w", ctx.Err())
	}

	db.count++
	return fmt.Sprint("count:", db.count), nil
}

var dbAdapter = &mockDB{}
