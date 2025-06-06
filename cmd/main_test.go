// Copyright (c) 2025 A Bit of Help, Inc.

package main

import (
	"context"
	"errors"
	"testing"

	"github.com/abitofhelp/multistage_pipeline_fanout/pkg/stats"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

func TestRun_InvalidArgs(t *testing.T) {
	// Create a logger for testing
	logger := zaptest.NewLogger(t)

	// Create a mock exit function
	exitCode := 0
	mockExit := func(code int) {
		exitCode = code
	}

	// Create a mock process file function that should not be called
	mockProcessFile := func(ctx context.Context, log *zap.Logger, inputPath, outputPath string) (*stats.Stats, error) {
		t.Error("ProcessFile should not be called with invalid arguments")
		return nil, nil
	}

	// Test with no arguments
	run([]string{}, logger, mockExit, mockProcessFile)
	if exitCode != 1 {
		t.Errorf("Expected exit code 1, got %d", exitCode)
	}

	// Test with one argument
	exitCode = 0
	run([]string{"input.txt"}, logger, mockExit, mockProcessFile)
	if exitCode != 1 {
		t.Errorf("Expected exit code 1, got %d", exitCode)
	}

	// Test with three arguments
	exitCode = 0
	run([]string{"input.txt", "output.txt", "extra.txt"}, logger, mockExit, mockProcessFile)
	if exitCode != 1 {
		t.Errorf("Expected exit code 1, got %d", exitCode)
	}
}

func TestRun_ProcessFileError(t *testing.T) {
	// Create a logger for testing
	logger := zaptest.NewLogger(t)

	// Create a mock exit function
	exitCode := 0
	mockExit := func(code int) {
		exitCode = code
	}

	// Set up the mock to return an error
	mockProcessFile := func(ctx context.Context, log *zap.Logger, inputPath, outputPath string) (*stats.Stats, error) {
		return nil, errors.New("test error")
	}

	// Run with valid arguments
	run([]string{"input.txt", "output.txt"}, logger, mockExit, mockProcessFile)
	if exitCode != 1 {
		t.Errorf("Expected exit code 1, got %d", exitCode)
	}
}

func TestRun_ProcessFileCanceled(t *testing.T) {
	// Create a logger for testing
	logger := zaptest.NewLogger(t)

	// Create a mock exit function
	exitCode := 0
	mockExit := func(code int) {
		exitCode = code
	}

	// Set up the mock to return a context.Canceled error
	mockProcessFile := func(ctx context.Context, log *zap.Logger, inputPath, outputPath string) (*stats.Stats, error) {
		return nil, errors.New("context canceled")
	}

	// Run with valid arguments
	run([]string{"input.txt", "output.txt"}, logger, mockExit, mockProcessFile)
	if exitCode != 1 {
		t.Errorf("Expected exit code 1, got %d", exitCode)
	}
}

func TestRun_Success(t *testing.T) {
	// Create a logger for testing
	logger := zaptest.NewLogger(t)

	// Create a mock exit function
	exitCode := 0
	mockExit := func(code int) {
		exitCode = code
	}

	// Set up the mock to return success
	mockProcessFile := func(ctx context.Context, log *zap.Logger, inputPath, outputPath string) (*stats.Stats, error) {
		return stats.NewStats(), nil
	}

	// Run with valid arguments
	run([]string{"input.txt", "output.txt"}, logger, mockExit, mockProcessFile)
	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", exitCode)
	}
}
