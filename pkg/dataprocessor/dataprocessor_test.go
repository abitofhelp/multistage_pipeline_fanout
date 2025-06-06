// Copyright (c) 2025 A Bit of Help, Inc.

package dataprocessor

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestProcessWithContext_Success(t *testing.T) {
	// Create a context
	ctx := context.Background()

	// Create a simple processing function that doubles the input
	processFunc := func(data []byte) ([]byte, error) {
		result := make([]byte, len(data))
		for i, b := range data {
			result[i] = b * 2
		}
		return result, nil
	}

	// Test data
	data := []byte{1, 2, 3, 4, 5}
	expected := []byte{2, 4, 6, 8, 10}

	// Process the data
	result, err := ProcessWithContext(ctx, processFunc, data)

	// Check for errors
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Check the result
	if len(result) != len(expected) {
		t.Errorf("Expected result length %d, got %d", len(expected), len(result))
	}
	for i := range expected {
		if result[i] != expected[i] {
			t.Errorf("Expected result[%d] = %d, got %d", i, expected[i], result[i])
		}
	}
}

func TestProcessWithContext_ProcessFuncError(t *testing.T) {
	// Create a context
	ctx := context.Background()

	// Create a processing function that returns an error
	expectedErr := errors.New("process error")
	processFunc := func(data []byte) ([]byte, error) {
		return nil, expectedErr
	}

	// Test data
	data := []byte{1, 2, 3, 4, 5}

	// Process the data
	result, err := ProcessWithContext(ctx, processFunc, data)

	// Check for errors
	if err == nil {
		t.Error("Expected an error, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}

	// Check the result
	if result != nil {
		t.Errorf("Expected nil result, got %v", result)
	}
}

func TestProcessWithContext_CanceledContext(t *testing.T) {
	// Create a canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Create a simple processing function
	processFunc := func(data []byte) ([]byte, error) {
		return data, nil
	}

	// Test data
	data := []byte{1, 2, 3, 4, 5}

	// Process the data
	result, err := ProcessWithContext(ctx, processFunc, data)

	// Check for errors
	if err == nil {
		t.Error("Expected an error, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}

	// Check the result
	if result != nil {
		t.Errorf("Expected nil result, got %v", result)
	}
}

func TestProcessWithContext_Timeout(t *testing.T) {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Create a processing function that takes longer than the timeout
	processFunc := func(data []byte) ([]byte, error) {
		time.Sleep(100 * time.Millisecond)
		return data, nil
	}

	// Test data
	data := []byte{1, 2, 3, 4, 5}

	// Process the data
	result, err := ProcessWithContext(ctx, processFunc, data)

	// Check for errors
	if err == nil {
		t.Error("Expected an error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Expected context.DeadlineExceeded error, got %v", err)
	}

	// Check the result
	if result != nil {
		t.Errorf("Expected nil result, got %v", result)
	}
}

func TestProcessWithContext_Panic(t *testing.T) {
	// Create a context
	ctx := context.Background()

	// Create a processing function that panics
	processFunc := func(data []byte) ([]byte, error) {
		panic("test panic")
	}

	// Test data
	data := []byte{1, 2, 3, 4, 5}

	// Process the data
	result, err := ProcessWithContext(ctx, processFunc, data)

	// Check for errors
	if err == nil {
		t.Error("Expected an error, got nil")
	}
	if err.Error() == "" {
		t.Error("Expected non-empty error message")
	}

	// Check the result
	if result != nil {
		t.Errorf("Expected nil result, got %v", result)
	}
}
