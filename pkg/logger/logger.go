// Copyright (c) 2025 A Bit of Help, Inc.

// Package logger provides logging functionality for the application
package logger

import (
	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ExitFunc is a function that exits the program with a given status code
type ExitFunc func(int)

// DefaultExitFunc is the default implementation of ExitFunc
var DefaultExitFunc = os.Exit

// InitLoggerWithExit initializes and returns a configured zap logger
// It takes an exit function to allow for testing
func InitLoggerWithExit(exit ExitFunc) *zap.Logger {
	config := zap.NewProductionConfig()
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	logger, err := config.Build()
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		exit(1)
	}
	return logger
}

// InitLogger initializes and returns a configured zap logger
// This is a wrapper around InitLoggerWithExit that uses the default exit function
func InitLogger() *zap.Logger {
	return InitLoggerWithExit(DefaultExitFunc)
}

// SafeSync syncs the logger and ignores "bad file descriptor" errors
// which can occur during shutdown when stderr is already closed
func SafeSync(logger *zap.Logger) {
	if logger == nil {
		return
	}

	// Ignore "bad file descriptor" errors which can happen during shutdown
	if err := logger.Sync(); err != nil && err.Error() != "sync /dev/stderr: bad file descriptor" {
		// Can't use logger here as we're syncing it
		fmt.Fprintf(os.Stderr, "Failed to sync logger: %v\n", err)
	}
}
