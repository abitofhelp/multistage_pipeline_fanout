// Copyright (c) 2025 A Bit of Help, Inc.

package writer

import (
	"context"
	"crypto/sha256"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	customErrors "github.com/abitofhelp/multistage_pipeline_fanout/pkg/errors"
	"github.com/abitofhelp/multistage_pipeline_fanout/pkg/stats"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
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

// TestStage_ErrorChannelFull tests what happens when the error channel is full
func TestStage_ErrorChannelFull(t *testing.T) {
	// Skip in short mode as this test involves timing
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	// Create a logger
	logger := zaptest.NewLogger(t)
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

	// Create channels - use a full error channel to test error channel full scenario
	encryptCh := make(chan []byte, 10)
	errCh := make(chan error, 1)
	// Fill the error channel
	errCh <- errors.New("dummy error to fill channel")

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

	// Check that the error channel still has only one error (the one we put in)
	if len(errCh) != 1 {
		t.Errorf("Expected error channel to have 1 error, got %d", len(errCh))
	}

	// Check that the stats were not updated
	if pipelineStats.OutputBytes.Load() != 0 {
		t.Errorf("Expected OutputBytes to be 0, got %d", pipelineStats.OutputBytes.Load())
	}
}

// mockSlowFile is a mock implementation of *os.File that simulates a slow write operation
type mockSlowFile struct {
	*os.File
	writeCalled bool
}

// newMockSlowFile creates a new mockSlowFile
func newMockSlowFile() (*mockSlowFile, error) {
	// Create a temporary file
	tempFile, err := os.CreateTemp("", "mock_slow_file")
	if err != nil {
		return nil, err
	}

	return &mockSlowFile{File: tempFile}, nil
}

// Write overrides the Write method to simulate a slow write
func (m *mockSlowFile) Write(p []byte) (n int, err error) {
	if !m.writeCalled {
		m.writeCalled = true
		// Sleep longer than the write timeout to trigger it
		time.Sleep(6 * time.Second)
		return 0, nil
	}
	return m.File.Write(p)
}

// TestStage_ContextCanceledDuringWrite tests what happens when the context is canceled during a write operation
func TestStage_ContextCanceledDuringWrite(t *testing.T) {
	// Skip in short mode as this test involves timing
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	// Create a logger
	logger := zaptest.NewLogger(t)
	defer logger.Sync()

	// Create a mock slow file
	mockFile, err := newMockSlowFile()
	if err != nil {
		t.Fatalf("Failed to create mock file: %v", err)
	}
	defer os.Remove(mockFile.Name())
	defer mockFile.Close()

	// Create a context that will be canceled during the write
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
		Stage(ctx, logger, mockFile.File, encryptCh, errCh, pipelineStats, outputHasher, cancel)
	}()

	// Send test data
	encryptCh <- []byte("This should trigger a slow write")

	// Wait a bit to ensure the write operation has started
	time.Sleep(100 * time.Millisecond)

	// Cancel the context
	cancel()

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
}

// TestStage_WriteTimeout tests what happens when a write operation times out
func TestStage_WriteTimeout(t *testing.T) {
	// Skip in short mode as this test involves timing
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	// Create a logger
	logger := zaptest.NewLogger(t)
	defer logger.Sync()

	// Create a mock slow file
	mockFile, err := newMockSlowFile()
	if err != nil {
		t.Fatalf("Failed to create mock file: %v", err)
	}
	defer os.Remove(mockFile.Name())
	defer mockFile.Close()

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
		Stage(ctx, logger, mockFile.File, encryptCh, errCh, pipelineStats, outputHasher, cancel)
	}()

	// Send test data
	encryptCh <- []byte("This should trigger a write timeout")

	// Wait for the writer to finish
	wg.Wait()

	// Check that there is a timeout error
	select {
	case err := <-errCh:
		// Error expected
		if err == nil {
			t.Error("Expected non-nil error")
		}
		if !errors.Is(err, customErrors.ErrTimeout) {
			t.Errorf("Expected timeout error, got: %v", err)
		}
	default:
		t.Error("Expected a timeout error, got none")
	}

	// Check that the stats were not updated
	if pipelineStats.OutputBytes.Load() != 0 {
		t.Errorf("Expected OutputBytes to be 0, got %d", pipelineStats.OutputBytes.Load())
	}
}
