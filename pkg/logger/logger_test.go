// Copyright (c) 2025 A Bit of Help, Inc.

package logger

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
)

func TestInitLogger(t *testing.T) {
	// This is a simple test to ensure InitLogger doesn't panic
	// Save the original DefaultExitFunc and restore it after the test
	originalExitFunc := DefaultExitFunc
	defer func() { DefaultExitFunc = originalExitFunc }()

	// Replace DefaultExitFunc with a mock that doesn't exit
	DefaultExitFunc = func(code int) {
		t.Logf("Mock exit called with code %d", code)
	}

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

// TestSafeSync tests the SafeSync function
func TestSafeSync(t *testing.T) {
	// Test with nil logger
	SafeSync(nil)
	// No panic means the test passed

	// Test with a real logger
	logger := zaptest.NewLogger(t)
	SafeSync(logger)
	// No panic means the test passed

	// Create a custom logger that will return an error on Sync
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(io.Discard),
		zapcore.InfoLevel,
	)
	customLogger := zap.New(core)
	SafeSync(customLogger)
	// No panic means the test passed
}

// TestSafeSyncWithError tests SafeSync with a logger that returns an error on Sync
func TestSafeSyncWithError(t *testing.T) {
	// Create a buffer to capture stderr output
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Create a logger with a custom writer that returns an error on Sync
	errorWriter := &errorWriter{err: errors.New("custom error")}
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(errorWriter),
		zapcore.InfoLevel,
	)
	logger := zap.New(core)

	// Call SafeSync
	SafeSync(logger)

	// Restore stderr and read the output
	w.Close()
	os.Stderr = oldStderr
	var buf bytes.Buffer
	io.Copy(&buf, r)

	// Check that the error message was printed
	if !strings.Contains(buf.String(), "Failed to sync logger") {
		t.Errorf("Expected error message to be printed, got: %s", buf.String())
	}
}

// TestSafeSyncWithBadFileDescriptorError tests SafeSync with a "bad file descriptor" error
func TestSafeSyncWithBadFileDescriptorError(t *testing.T) {
	// Create a buffer to capture stderr output
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Create a logger with a custom writer that returns a "bad file descriptor" error on Sync
	errorWriter := &errorWriter{err: errors.New("sync /dev/stderr: bad file descriptor")}
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(errorWriter),
		zapcore.InfoLevel,
	)
	logger := zap.New(core)

	// Call SafeSync
	SafeSync(logger)

	// Restore stderr and read the output
	w.Close()
	os.Stderr = oldStderr
	var buf bytes.Buffer
	io.Copy(&buf, r)

	// Check that no error message was printed (since we ignore "bad file descriptor" errors)
	if buf.String() != "" {
		t.Errorf("Expected no error message, got: %s", buf.String())
	}
}

// errorWriter is a custom io.Writer that returns an error on Sync
type errorWriter struct {
	err error
}

func (w *errorWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

func (w *errorWriter) Sync() error {
	return w.err
}
