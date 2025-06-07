// Copyright (c) 2025 A Bit of Help, Inc.

package errors

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
)

func TestPipelineError(t *testing.T) {
	// Test NewPipelineError
	baseErr := errors.New("test error")
	pe := NewPipelineError(baseErr, "test-stage", 1, "test-operation", 100, "test-file.txt")

	// Verify fields
	if pe.Err != baseErr {
		t.Errorf("Expected Err to be %v, got %v", baseErr, pe.Err)
	}
	if pe.Stage != "test-stage" {
		t.Errorf("Expected Stage to be %s, got %s", "test-stage", pe.Stage)
	}
	if pe.StageID != 1 {
		t.Errorf("Expected StageID to be %d, got %d", 1, pe.StageID)
	}
	if pe.Operation != "test-operation" {
		t.Errorf("Expected Operation to be %s, got %s", "test-operation", pe.Operation)
	}
	if pe.DataSize != 100 {
		t.Errorf("Expected DataSize to be %d, got %d", 100, pe.DataSize)
	}
	if pe.FilePath != "test-file.txt" {
		t.Errorf("Expected FilePath to be %s, got %s", "test-file.txt", pe.FilePath)
	}

	// Test Error method
	errorStr := pe.Error()
	if errorStr == "" {
		t.Error("Error() returned empty string")
	}

	// Test Unwrap method
	unwrappedErr := pe.Unwrap()
	if unwrappedErr != baseErr {
		t.Errorf("Expected Unwrap() to return %v, got %v", baseErr, unwrappedErr)
	}
}

func TestIsIOError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"ErrIOFailure", ErrIOFailure, true},
		{"PathError", &os.PathError{Op: "open", Path: "nonexistent", Err: os.ErrNotExist}, true},
		{"Other error", errors.New("some other error"), false},
		{"Wrapped IO error", fmt.Errorf("wrapped: %w", ErrIOFailure), true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := IsIOError(tc.err)
			if result != tc.expected {
				t.Errorf("IsIOError(%v) = %v, expected %v", tc.err, result, tc.expected)
			}
		})
	}
}

func TestIsTimeoutError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"ErrTimeout", ErrTimeout, true},
		{"DeadlineExceeded", context.DeadlineExceeded, true},
		{"Other error", errors.New("some other error"), false},
		{"Wrapped timeout error", fmt.Errorf("wrapped: %w", ErrTimeout), true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := IsTimeoutError(tc.err)
			if result != tc.expected {
				t.Errorf("IsTimeoutError(%v) = %v, expected %v", tc.err, result, tc.expected)
			}
		})
	}
}

func TestIsCancellationError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"ErrCanceled", ErrCanceled, true},
		{"Canceled", context.Canceled, true},
		{"Other error", errors.New("some other error"), false},
		{"Wrapped cancel error", fmt.Errorf("wrapped: %w", ErrCanceled), true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := IsCancellationError(tc.err)
			if result != tc.expected {
				t.Errorf("IsCancellationError(%v) = %v, expected %v", tc.err, result, tc.expected)
			}
		})
	}
}

func TestErrorCollector(t *testing.T) {
	// Test NewErrorCollector
	ec := NewErrorCollector()
	if ec == nil {
		t.Fatal("NewErrorCollector() returned nil")
	}

	// Test HasErrors when empty
	if ec.HasErrors() {
		t.Error("New ErrorCollector should not have errors")
	}

	// Test Error when empty
	if ec.Error() != "no errors" {
		t.Errorf("Expected 'no errors', got '%s'", ec.Error())
	}

	// Test Add with nil
	ec.Add(nil)
	if ec.HasErrors() {
		t.Error("ErrorCollector should not have errors after adding nil")
	}

	// Test Add with error
	err1 := errors.New("error 1")
	ec.Add(err1)

	// Test HasErrors after adding
	if !ec.HasErrors() {
		t.Error("ErrorCollector should have errors after Add")
	}

	// Test Error with one error
	if ec.Error() != err1.Error() {
		t.Errorf("Expected '%s', got '%s'", err1.Error(), ec.Error())
	}

	// Test Errors
	errs := ec.Errors()
	if len(errs) != 1 || errs[0] != err1 {
		t.Errorf("Expected [%v], got %v", []error{err1}, errs)
	}

	// Add another error
	err2 := errors.New("error 2")
	ec.Add(err2)

	// Test Error with multiple errors
	errorMsg := ec.Error()
	if errorMsg == "" {
		t.Error("Error() returned empty string")
	}
	if len(ec.Errors()) != 2 {
		t.Errorf("Expected 2 errors, got %d", len(ec.Errors()))
	}
}
