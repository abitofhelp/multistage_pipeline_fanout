// Copyright (c) 2025 A Bit of Help, Inc.

package logger

import (
	"testing"
)

func TestInitLogger(t *testing.T) {
	// This is a simple test to ensure InitLogger doesn't panic
	logger := InitLogger()
	if logger == nil {
		t.Fatal("Expected logger to be non-nil")
	}
}

func TestInitLoggerWithExit(t *testing.T) {
	// Test successful logger initialization
	exitCalled := false
	mockExit := func(code int) {
		exitCalled = true
		if code != 1 {
			t.Errorf("Expected exit code 1, got %d", code)
		}
	}

	logger := InitLoggerWithExit(mockExit)
	if logger == nil {
		t.Fatal("Expected logger to be non-nil")
	}
	if exitCalled {
		t.Fatal("Exit function should not have been called")
	}

	// We can't easily test the error case because zap.Config.Build() is difficult to make fail
	// in a test environment. In a real-world scenario, we might use a mock or a wrapper around
	// zap.Config to simulate failure.
}

// TestLoggerConfiguration tests that the logger is configured correctly
func TestLoggerConfiguration(t *testing.T) {
	logger := InitLogger()

	// Verify the logger is not nil
	if logger == nil {
		t.Fatal("Expected logger to be non-nil")
	}

	// Test that we can use the logger without panicking
	logger.Info("Test log message")
}
