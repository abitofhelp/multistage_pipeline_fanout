// Copyright (c) 2025 A Bit of Help, Inc.

package writer

import (
	"context"
	"crypto/sha256"
	"os"
	"sync"
	"testing"

	"github.com/abitofhelp/multistage_pipeline_fanout/pkg/stats"
	"go.uber.org/zap"
)

func TestStage(t *testing.T) {
	// Create a logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()

	// Create a temporary file for output
	tempFile, err := os.CreateTemp("", "writer_test")
	if err != nil {
		t.Fatalf("Failed to create temporary file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	// Create a context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create channels
	encryptCh := make(chan []byte, 10)
	errCh := make(chan error, 10)

	// Create stats
	pipelineStats := stats.NewStats()

	// Create a hasher
	outputHasher := sha256.New()

	// Start the writer stage in a goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		Stage(ctx, logger, tempFile, encryptCh, errCh, pipelineStats, outputHasher, cancel)
	}()

	// Send test data
	testData := [][]byte{
		[]byte("This is the first chunk of test data."),
		[]byte("This is the second chunk of test data."),
		[]byte("This is the third chunk of test data."),
	}

	var totalBytes uint64
	for _, data := range testData {
		encryptCh <- data
		totalBytes += uint64(len(data))
	}

	// Close the encrypt channel to signal no more data
	close(encryptCh)

	// Wait for the writer to finish
	wg.Wait()

	// Check that there are no errors
	select {
	case err := <-errCh:
		t.Errorf("Expected no errors, got: %v", err)
	default:
		// No errors, as expected
	}

	// Check that the stats were updated correctly
	if pipelineStats.OutputBytes.Load() != totalBytes {
		t.Errorf("Expected OutputBytes to be %d, got %d", totalBytes, pipelineStats.OutputBytes.Load())
	}

	// Check that the output hash was calculated
	if len(outputHasher.Sum(nil)) == 0 {
		t.Error("Expected output hash to be calculated")
	}

	// Read the file and verify its contents
	tempFile.Close() // Close for reading
	fileData, err := os.ReadFile(tempFile.Name())
	if err != nil {
		t.Fatalf("Failed to read temporary file: %v", err)
	}

	// Check that the file contains all the test data
	expectedData := make([]byte, 0, totalBytes)
	for _, data := range testData {
		expectedData = append(expectedData, data...)
	}

	if len(fileData) != len(expectedData) {
		t.Errorf("Expected file data length %d, got %d", len(expectedData), len(fileData))
	}

	for i := range expectedData {
		if fileData[i] != expectedData[i] {
			t.Errorf("Expected file data[%d] = %d, got %d", i, expectedData[i], fileData[i])
		}
	}
}

func TestStage_ContextCanceled(t *testing.T) {
	// Create a logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()

	// Create a temporary file for output
	tempFile, err := os.CreateTemp("", "writer_test")
	if err != nil {
		t.Fatalf("Failed to create temporary file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	// Create a context that's already canceled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Create channels
	encryptCh := make(chan []byte, 10)
	errCh := make(chan error, 10)

	// Create stats
	pipelineStats := stats.NewStats()

	// Create a hasher
	outputHasher := sha256.New()

	// Start the writer stage in a goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		Stage(ctx, logger, tempFile, encryptCh, errCh, pipelineStats, outputHasher, cancel)
	}()

	// Send test data
	encryptCh <- []byte("This data should not be processed due to canceled context")

	// Wait for the writer to finish
	wg.Wait()

	// Check that there are no errors in the error channel (context cancellation is not reported as an error)
	select {
	case err := <-errCh:
		t.Errorf("Expected no errors in error channel, got: %v", err)
	default:
		// No errors, as expected
	}

	// Check that the stats were not updated
	if pipelineStats.OutputBytes.Load() != 0 {
		t.Errorf("Expected OutputBytes to be 0, got %d", pipelineStats.OutputBytes.Load())
	}

	// Read the file and verify it's empty
	tempFile.Close() // Close for reading
	fileData, err := os.ReadFile(tempFile.Name())
	if err != nil {
		t.Fatalf("Failed to read temporary file: %v", err)
	}

	if len(fileData) != 0 {
		t.Errorf("Expected empty file, got %d bytes", len(fileData))
	}
}

func TestStage_WriteError(t *testing.T) {
	// Create a logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()

	// Create a temporary file for output
	tempFile, err := os.CreateTemp("", "writer_test")
	if err != nil {
		t.Fatalf("Failed to create temporary file: %v", err)
	}

	// Close the file to cause a write error
	tempFile.Close()
	defer os.Remove(tempFile.Name())

	// Create a context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create channels
	encryptCh := make(chan []byte, 10)
	errCh := make(chan error, 10)

	// Create stats
	pipelineStats := stats.NewStats()

	// Create a hasher
	outputHasher := sha256.New()

	// Start the writer stage in a goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		Stage(ctx, logger, tempFile, encryptCh, errCh, pipelineStats, outputHasher, cancel)
	}()

	// Send test data
	encryptCh <- []byte("This will cause a write error")

	// Wait for the writer to finish
	wg.Wait()

	// Check that there is an error
	select {
	case err := <-errCh:
		// Error expected
		if err == nil {
			t.Error("Expected non-nil error")
		}
	default:
		t.Error("Expected an error, got none")
	}

	// Check that the stats were not updated
	if pipelineStats.OutputBytes.Load() != 0 {
		t.Errorf("Expected OutputBytes to be 0, got %d", pipelineStats.OutputBytes.Load())
	}
}
