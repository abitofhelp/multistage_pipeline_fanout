// Copyright (c) 2025 A Bit of Help, Inc.

// Package utils provides utility functions for the application
package utils

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
)

// SetupGracefulShutdown configures signal handling for graceful shutdown
// It returns a function that should be deferred to clean up signal handling
func SetupGracefulShutdown(ctx context.Context, cancel context.CancelFunc, logger *zap.Logger) func() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)

	// Create a channel to track if we've already received a signal
	signalReceived := make(chan struct{}, 1)

	// Create a channel to signal when the goroutine should exit
	done := make(chan struct{})

	// Start goroutine for signal handling
	go func() {
		defer logger.Debug("Signal handling goroutine exited")

		for {
			select {
			case sig, ok := <-sigChan:
				if !ok {
					// sigChan was closed, exit goroutine
					return
				}

				select {
				case <-signalReceived:
					// Second signal received, force immediate exit
					logger.Warn("Received second signal, forcing immediate shutdown",
						zap.String("signal", sig.String()))
					os.Exit(1)
				default:
					// First signal, try graceful shutdown
					logger.Info("Received signal, initiating graceful shutdown",
						zap.String("signal", sig.String()))

					// Mark that we've received a signal
					close(signalReceived)

					// Start a timer for forced shutdown
					shutdownTimerDone := make(chan struct{})
					go func() {
						defer close(shutdownTimerDone)

						shutdownTimer := time.NewTimer(30 * time.Second)
						defer shutdownTimer.Stop()

						select {
						case <-shutdownTimer.C:
							logger.Warn("Graceful shutdown timed out after 30 seconds, forcing exit")
							os.Exit(1)
						case <-ctx.Done():
							// Context was canceled, normal shutdown proceeding
							return
						case <-done:
							// Parent goroutine is exiting
							return
						}
					}()

					// Trigger graceful shutdown
					cancel()
				}
			case <-ctx.Done():
				// Context was canceled elsewhere
				return
			case <-done:
				// Signal to exit
				return
			}
		}
	}()

	// Return a cleanup function
	return func() {
		// Signal the goroutine to exit
		close(done)

		// Stop signal notifications
		signal.Stop(sigChan)
		close(sigChan)

		logger.Debug("Signal handling cleaned up")
	}
}
