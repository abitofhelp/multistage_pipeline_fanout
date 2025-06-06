// Copyright (c) 2025 A Bit of Help, Inc.

package utils

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestSetupGracefulShutdown(t *testing.T) {
	// Create a logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()

	// Create a context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup graceful shutdown
	SetupGracefulShutdown(ctx, cancel, logger)

	// Test that the function doesn't panic
	// Note: We can't easily test the signal handling functionality in a unit test

	// Test that the function handles context cancellation
	cancel()

	// Wait a bit to allow the goroutine to process the cancellation
	time.Sleep(10 * time.Millisecond)

	// If we got here without panicking, the test passes
}
