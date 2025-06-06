// Copyright (c) 2025 A Bit of Help, Inc.

// Package dataprocessor provides generic data processing functionality with context awareness.
//
// This package serves as a foundation for both the core functionality packages in /pkg
// (like compression, encryption) and indirectly for the pipeline packages in /pkg/pipeline.
//
// It provides context-aware processing capabilities that allow operations to be
// cancellable and respect deadlines, which is essential for both standalone usage
// and integration into the pipeline architecture.
package dataprocessor

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"
	"sync"
	"time"
)

// ProcessWithContext executes a data processing function with context awareness
func ProcessWithContext(ctx context.Context, processFunc func([]byte) ([]byte, error), data []byte) ([]byte, error) {
	// Check if context is already canceled
	if err := ctx.Err(); err != nil {
		if err == context.Canceled {
			return nil, fmt.Errorf("operation canceled: %w", err)
		}
		if err == context.DeadlineExceeded {
			return nil, fmt.Errorf("operation timed out: %w", err)
		}
		return nil, fmt.Errorf("context error: %w", err)
	}

	// Create a channel to signal completion
	done := make(chan struct{})
	var result []byte
	var processErr error

	// Use a WaitGroup to ensure the goroutine completes even if context is canceled
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		defer close(done)
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()
				processErr = fmt.Errorf("panic in data processing: %v\nstack: %s", r, stack)
			}
		}()

		result, processErr = processFunc(data)
	}()

	// Wait for either completion or context cancellation
	select {
	case <-done:
		return result, processErr
	case <-ctx.Done():
		// Wait for the goroutine to finish to prevent leaks
		// Using a separate channel to signal when wait is complete
		waitDone := make(chan struct{})
		go func() {
			wg.Wait()
			close(waitDone)
		}()

		// Wait with a timeout to prevent indefinite blocking
		select {
		case <-waitDone:
			// Goroutine completed
		case <-time.After(5 * time.Second):
			// Timeout waiting for goroutine to complete
			// Log this situation but continue with error return
			// We can't use the logger here as it might not be available
			fmt.Fprintf(os.Stderr, "Warning: Timed out waiting for processing goroutine to complete\n")
		}

		err := ctx.Err()
		if err == context.Canceled {
			return nil, fmt.Errorf("operation canceled while processing data: %w", err)
		}
		if err == context.DeadlineExceeded {
			return nil, fmt.Errorf("operation timed out while processing data: %w", err)
		}
		return nil, fmt.Errorf("context error while processing data: %w", err)
	}
}
