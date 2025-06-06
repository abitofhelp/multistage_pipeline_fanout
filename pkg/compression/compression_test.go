// Copyright (c) 2025 A Bit of Help, Inc.

package compression

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/abitofhelp/multistage_pipeline_fanout/pkg/dataprocessor"
	"github.com/andybalholm/brotli"
)

func TestCompressDataWithContext_Success(t *testing.T) {
	// Create a context
	ctx := context.Background()

	// Test data - use a repeating pattern to ensure good compression
	data := bytes.Repeat([]byte("test data for compression "), 100)

	// Compress the data
	compressed, err := CompressDataWithContext(ctx, data)

	// Check for errors
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Check that compressed data is not nil
	if compressed == nil {
		t.Error("Expected non-nil compressed data, got nil")
	}

	// Check that compression actually reduced the size
	if len(compressed) >= len(data) {
		t.Errorf("Expected compressed data to be smaller than input, got %d >= %d", len(compressed), len(data))
	}

	// Verify that the compressed data can be decompressed
	reader := brotli.NewReader(bytes.NewReader(compressed))
	decompressed := new(bytes.Buffer)
	_, err = decompressed.ReadFrom(reader)
	if err != nil {
		t.Fatalf("Failed to decompress data: %v", err)
	}

	// Check that decompressed data matches original
	if !bytes.Equal(decompressed.Bytes(), data) {
		t.Error("Decompressed data does not match original data")
	}
}

func TestCompressDataWithContext_EmptyData(t *testing.T) {
	// Create a context
	ctx := context.Background()

	// Test with empty data
	data := []byte{}

	// Compress the data
	compressed, err := CompressDataWithContext(ctx, data)

	// Check for errors
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Check that compressed data is not nil
	if compressed == nil {
		t.Error("Expected non-nil compressed data, got nil")
	}

	// Verify that the compressed data can be decompressed
	reader := brotli.NewReader(bytes.NewReader(compressed))
	decompressed := new(bytes.Buffer)
	_, err = decompressed.ReadFrom(reader)
	if err != nil {
		t.Fatalf("Failed to decompress data: %v", err)
	}

	// Check that decompressed data matches original
	if !bytes.Equal(decompressed.Bytes(), data) {
		t.Error("Decompressed data does not match original empty data")
	}
}

func TestCompressDataWithContext_CanceledContext(t *testing.T) {
	// Create a canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Test data
	data := []byte("test data")

	// Compress the data
	compressed, err := CompressDataWithContext(ctx, data)

	// Check for errors
	if err == nil {
		t.Error("Expected an error, got nil")
	}
	if compressed != nil {
		t.Errorf("Expected nil compressed data, got %v", compressed)
	}
}

func TestCompressDataWithContext_Timeout(t *testing.T) {
	// Skip this test in short mode as it involves waiting
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	// Create a context with a very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Wait to ensure the timeout has occurred
	time.Sleep(10 * time.Millisecond)

	// Test data - make it large enough to potentially cause a timeout
	data := make([]byte, 10*1024*1024) // 10MB

	// Compress the data
	compressed, err := CompressDataWithContext(ctx, data)

	// Check for errors
	if err == nil {
		t.Error("Expected an error, got nil")
	}
	if compressed != nil {
		t.Errorf("Expected nil compressed data, got data of length %d", len(compressed))
	}
}

func TestCompressDataWithContext_LargeData(t *testing.T) {
	// Skip this test in short mode as it involves large data
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	// Create a context
	ctx := context.Background()

	// Test data - 1MB of repeating pattern
	data := bytes.Repeat([]byte("large test data for compression "), 25000)

	// Compress the data
	compressed, err := CompressDataWithContext(ctx, data)

	// Check for errors
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Check that compressed data is not nil
	if compressed == nil {
		t.Error("Expected non-nil compressed data, got nil")
	}

	// Check that compression actually reduced the size
	if len(compressed) >= len(data) {
		t.Errorf("Expected compressed data to be smaller than input, got %d >= %d", len(compressed), len(data))
	}

	// Verify that the compressed data can be decompressed
	reader := brotli.NewReader(bytes.NewReader(compressed))
	decompressed := new(bytes.Buffer)
	_, err = decompressed.ReadFrom(reader)
	if err != nil {
		t.Fatalf("Failed to decompress data: %v", err)
	}

	// Check that decompressed data matches original
	if !bytes.Equal(decompressed.Bytes(), data) {
		t.Error("Decompressed data does not match original data")
	}
}

func TestCompressDataWithContext_PanicRecovery(t *testing.T) {
	// Create a context
	ctx := context.Background()

	// Test data
	data := []byte("test data")

	// Create a function that will panic
	panicFunc := func([]byte) ([]byte, error) {
		panic("test panic")
	}

	// Use ProcessWithContext directly to test panic recovery
	result, err := dataprocessor.ProcessWithContext(ctx, panicFunc, data)

	// Check that we got an error
	if err == nil {
		t.Error("Expected an error from panic, got nil")
	}

	// Check that the error message contains information about the panic
	if err != nil && !bytes.Contains([]byte(err.Error()), []byte("panic")) {
		t.Errorf("Expected error message to contain 'panic', got: %v", err)
	}

	// Check that result is nil
	if result != nil {
		t.Errorf("Expected nil result, got: %v", result)
	}
}
