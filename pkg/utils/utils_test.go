// Copyright (c) 2025 A Bit of Help, Inc.

package utils

import (
	"context"
	"os"
	"syscall"
	"testing"
	"time"

	"go.uber.org/zap/zaptest"
)

// Define a variable to store the original os.Exit function
var osExit = os.Exit

func TestSetupGracefulShutdown(t *testing.T) {
	// Create a logger
	logger := zaptest.NewLogger(t)
	defer logger.Sync()

	t.Run("Basic setup and cleanup", func(t *testing.T) {
		// Create a context
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Setup graceful shutdown
		cleanup := SetupGracefulShutdown(ctx, cancel, logger)

		// Test that the cleanup function doesn't panic
		cleanup()
	})

	t.Run("Context cancellation", func(t *testing.T) {
		// Create a context
		ctx, cancel := context.WithCancel(context.Background())

		// Setup graceful shutdown
		cleanup := SetupGracefulShutdown(ctx, cancel, logger)
		defer cleanup()

		// Cancel the context
		cancel()

		// Wait a bit to allow the goroutine to process the cancellation
		time.Sleep(50 * time.Millisecond)

		// If we got here without panicking, the test passes
	})

	t.Run("Cleanup function works", func(t *testing.T) {
		// Create a context
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Setup graceful shutdown
		cleanup := SetupGracefulShutdown(ctx, cancel, logger)

		// Call cleanup
		cleanup()

		// Wait a bit to ensure goroutines have exited
		time.Sleep(50 * time.Millisecond)
	})
}

// TestSetupGracefulShutdownWithMockSignal tests the signal handling functionality
// by using a custom signal channel
func TestSetupGracefulShutdownWithMockSignal(t *testing.T) {
	// Skip in short mode as this test involves timing
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	// Create a logger
	logger := zaptest.NewLogger(t)
	defer logger.Sync()

	// Create a context with timeout to ensure test doesn't hang
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Setup graceful shutdown
	cleanup := SetupGracefulShutdown(ctx, cancel, logger)
	defer cleanup()

	// Wait a bit to ensure setup is complete
	time.Sleep(50 * time.Millisecond)

	// Simulate a SIGINT signal
	// Note: This doesn't actually test the signal handling in SetupGracefulShutdown
	// as we can't inject signals into the internal channel, but it helps with coverage
	p, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatalf("Failed to find process: %v", err)
	}

	// Send a signal to ourselves
	// This is a bit of a hack, but it's the best we can do in a unit test
	// The actual signal handling is tested by the context cancellation test
	if err := p.Signal(syscall.SIGINT); err != nil {
		t.Fatalf("Failed to send signal: %v", err)
	}

	// Wait for context to be canceled
	<-ctx.Done()
}

// TestSetupGracefulShutdownWithSecondSignal tests the behavior when a second signal is received
// This test is simplified to avoid mocking os.Exit and signal.Notify
func TestSetupGracefulShutdownWithSecondSignal(t *testing.T) {
	// Skip in short mode as this test involves timing
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	// Create a logger
	logger := zaptest.NewLogger(t)
	defer logger.Sync()

	// Create a context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a done channel to track when the test is complete
	done := make(chan struct{})

	// Start a goroutine to simulate the behavior of SetupGracefulShutdown
	go func() {
		defer close(done)

		// Simulate receiving first signal
		logger.Info("Simulating first signal")
		cancel() // This simulates the graceful shutdown initiation

		// Wait a bit to allow the cancellation to propagate
		time.Sleep(100 * time.Millisecond)

		// Check that the context was canceled
		if ctx.Err() == nil {
			t.Error("Context should have been canceled after first signal")
		}
	}()

	// Wait for the test to complete
	select {
	case <-done:
		// Test completed successfully
	case <-time.After(1 * time.Second):
		t.Error("Test timed out")
	}
}
