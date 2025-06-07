// Copyright (c) 2025 A Bit of Help, Inc.

package reader

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

	// Create a temporary file with test data
	testData := []byte("This is test data for the reader stage. It will be read in chunks and processed.")
	tempFile, err := os.CreateTemp("", "reader_test")
	if err != nil {
		t.Fatalf("Failed to create temporary file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	_, err = tempFile.Write(testData)
	if err != nil {
		t.Fatalf("Failed to write to temporary file: %v", err)
	}

	// Seek to the beginning of the file
	_, err = tempFile.Seek(0, 0)
	if err != nil {
		t.Fatalf("Failed to seek to beginning of file: %v", err)
	}

	// Create a context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create channels
	readCh := make(chan []byte, 10)
	errCh := make(chan error, 10)

	// Create stats
	pipelineStats := stats.NewStats()

	// Create a hasher
	inputHasher := sha256.New()

	// Define chunk size smaller than the test data to ensure multiple chunks
	chunkSize := 16

	// Start the reader stage in a goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		Stage(ctx, logger, tempFile, readCh, errCh, pipelineStats, inputHasher, chunkSize)
	}()

	// Collect all chunks from the read channel
	var chunks [][]byte
	for chunk := range readCh {
		chunks = append(chunks, chunk)
	}

	// Wait for the reader to finish
	wg.Wait()

	// Check that there are no errors
	select {
	case err := <-errCh:
		t.Errorf("Expected no errors, got: %v", err)
	default:
		// No errors, as expected
	}

	// Check that we got the expected number of chunks
	expectedChunks := (len(testData) + chunkSize - 1) / chunkSize
	if len(chunks) != expectedChunks {
		t.Errorf("Expected %d chunks, got %d", expectedChunks, len(chunks))
	}

	// Reconstruct the original data from chunks
	reconstructed := make([]byte, 0, len(testData))
	for _, chunk := range chunks {
		reconstructed = append(reconstructed, chunk...)
	}

	// Check that the reconstructed data matches the original
	if len(reconstructed) != len(testData) {
		t.Errorf("Expected reconstructed data length %d, got %d", len(testData), len(reconstructed))
	}
	for i := range testData {
		if reconstructed[i] != testData[i] {
			t.Errorf("Expected reconstructed[%d] = %d, got %d", i, testData[i], reconstructed[i])
		}
	}

	// Check that the stats were updated correctly
	if pipelineStats.InputBytes.Load() != uint64(len(testData)) {
		t.Errorf("Expected InputBytes to be %d, got %d", len(testData), pipelineStats.InputBytes.Load())
	}

	if pipelineStats.ChunksProcessed.Load() != uint64(len(chunks)) {
		t.Errorf("Expected ChunksProcessed to be %d, got %d", len(chunks), pipelineStats.ChunksProcessed.Load())
	}

	// Check that the final chunk size is correct
	lastChunkSize := len(testData) % chunkSize
	if lastChunkSize == 0 {
		lastChunkSize = chunkSize
	}
	if pipelineStats.FinalChunkSize.Load() != uint64(lastChunkSize) {
		t.Errorf("Expected FinalChunkSize to be %d, got %d", lastChunkSize, pipelineStats.FinalChunkSize.Load())
	}

	// Check that the input hash was calculated
	if len(inputHasher.Sum(nil)) == 0 {
		t.Error("Expected input hash to be calculated")
	}
}

func TestStage_ContextCanceled(t *testing.T) {
	// Create a logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()

	// Create a temporary file with test data
	testData := []byte("This is test data for the reader stage. It will be read in chunks and processed.")
	tempFile, err := os.CreateTemp("", "reader_test")
	if err != nil {
		t.Fatalf("Failed to create temporary file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	_, err = tempFile.Write(testData)
	if err != nil {
		t.Fatalf("Failed to write to temporary file: %v", err)
	}

	// Seek to the beginning of the file
	_, err = tempFile.Seek(0, 0)
	if err != nil {
		t.Fatalf("Failed to seek to beginning of file: %v", err)
	}

	// Create a context that's already canceled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Create channels
	readCh := make(chan []byte, 10)
	errCh := make(chan error, 10)

	// Create stats
	pipelineStats := stats.NewStats()

	// Create a hasher
	inputHasher := sha256.New()

	// Define chunk size
	chunkSize := 16

	// Start the reader stage in a goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		Stage(ctx, logger, tempFile, readCh, errCh, pipelineStats, inputHasher, chunkSize)
	}()

	// Wait for the reader to finish
	wg.Wait()

	// Check that the read channel is closed
	_, ok := <-readCh
	if ok {
		t.Error("Expected read channel to be closed")
	}

	// Check that there are no errors
	select {
	case err := <-errCh:
		t.Errorf("Expected no errors, got: %v", err)
	default:
		// No errors, as expected
	}

	// Check that no data was processed
	if pipelineStats.InputBytes.Load() != 0 {
		t.Errorf("Expected InputBytes to be 0, got %d", pipelineStats.InputBytes.Load())
	}

	if pipelineStats.ChunksProcessed.Load() != 0 {
		t.Errorf("Expected ChunksProcessed to be 0, got %d", pipelineStats.ChunksProcessed.Load())
	}
}

func TestStage_ReadError(t *testing.T) {
	// Create a logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()

	// Create a temporary file
	tempFile, err := os.CreateTemp("", "reader_test")
	if err != nil {
		t.Fatalf("Failed to create temporary file: %v", err)
	}

	// Close the file to cause a read error
	tempFile.Close()
	defer os.Remove(tempFile.Name())

	// Create a context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create channels
	readCh := make(chan []byte, 10)
	errCh := make(chan error, 10)

	// Create stats
	pipelineStats := stats.NewStats()

	// Create a hasher
	inputHasher := sha256.New()

	// Define chunk size
	chunkSize := 16

	// Start the reader stage in a goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		Stage(ctx, logger, tempFile, readCh, errCh, pipelineStats, inputHasher, chunkSize)
	}()

	// Wait for the reader to finish
	wg.Wait()

	// Check that the read channel is closed
	_, ok := <-readCh
	if ok {
		t.Error("Expected read channel to be closed")
	}

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

	// Check that no data was processed
	if pipelineStats.InputBytes.Load() != 0 {
		t.Errorf("Expected InputBytes to be 0, got %d", pipelineStats.InputBytes.Load())
	}

	if pipelineStats.ChunksProcessed.Load() != 0 {
		t.Errorf("Expected ChunksProcessed to be 0, got %d", pipelineStats.ChunksProcessed.Load())
	}
}

// TestStage_BlockedPipeline tests what happens when the next stage is blocked
func TestStage_BlockedPipeline(t *testing.T) {
	// Skip in short mode as this test involves timing
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	// Create a logger
	logger := zaptest.NewLogger(t)
	defer logger.Sync()

	// Create a temporary file with test data
	testData := []byte("This is test data for the reader stage.")
	tempFile, err := os.CreateTemp("", "reader_test")
	if err != nil {
		t.Fatalf("Failed to create temporary file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	_, err = tempFile.Write(testData)
	if err != nil {
		t.Fatalf("Failed to write to temporary file: %v", err)
	}

	// Seek to the beginning of the file
	_, err = tempFile.Seek(0, 0)
	if err != nil {
		t.Fatalf("Failed to seek to beginning of file: %v", err)
	}

	// Create a context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create channels - use a channel with no buffer to block the pipeline
	readCh := make(chan []byte, 0) // No buffer, will block
	errCh := make(chan error, 10)

	// Create stats
	pipelineStats := stats.NewStats()

	// Create a hasher
	inputHasher := sha256.New()

	// Define chunk size
	chunkSize := 16

	// Start the reader stage in a goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		Stage(ctx, logger, tempFile, readCh, errCh, pipelineStats, inputHasher, chunkSize)
	}()

	// Wait for the reader to finish
	wg.Wait()

	// Check that the read channel is closed
	_, ok := <-readCh
	if ok {
		t.Error("Expected read channel to be closed")
	}

	// Check that there is a blocked pipeline error
	select {
	case err := <-errCh:
		// Error expected
		if err == nil {
			t.Error("Expected non-nil error")
		}
		if !errors.Is(err, customErrors.ErrPipelineBlocked) {
			t.Errorf("Expected pipeline blocked error, got: %v", err)
		}
	default:
		t.Error("Expected a pipeline blocked error, got none")
	}
}

// mockSlowReader is a mock implementation of io.Reader that simulates a slow read operation
type mockSlowReader struct {
	readCalled chan struct{}
}

// newMockSlowReader creates a new mockSlowReader
func newMockSlowReader() *mockSlowReader {
	return &mockSlowReader{
		readCalled: make(chan struct{}),
	}
}

// Read implements io.Reader and blocks indefinitely until the channel is closed
func (m *mockSlowReader) Read(p []byte) (n int, err error) {
	// Signal that Read was called
	close(m.readCalled)

	// Block indefinitely
	select {}
}

// mockSlowFile wraps a file with a mockSlowReader
type mockSlowFile struct {
	*os.File
	reader *mockSlowReader
}

// newMockSlowFile creates a new mockSlowFile
func newMockSlowFile() (*mockSlowFile, *mockSlowReader, error) {
	// Create a temporary file
	tempFile, err := os.CreateTemp("", "mock_slow_file")
	if err != nil {
		return nil, nil, err
	}

	reader := newMockSlowReader()
	return &mockSlowFile{
		File:   tempFile,
		reader: reader,
	}, reader, nil
}

// Read overrides the Read method to use the mockSlowReader
func (m *mockSlowFile) Read(p []byte) (n int, err error) {
	return m.reader.Read(p)
}

// TestStage_ReadTimeout tests what happens when a read operation times out
func TestStage_ReadTimeout(t *testing.T) {
	// Skip in short mode as this test involves timing
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	// Create a logger
	logger := zaptest.NewLogger(t)
	defer logger.Sync()

	// Create a mock slow file
	mockFile, reader, err := newMockSlowFile()
	if err != nil {
		t.Fatalf("Failed to create mock file: %v", err)
	}
	defer os.Remove(mockFile.Name())
	defer mockFile.Close()

	// Create a context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create channels
	readCh := make(chan []byte, 10)
	errCh := make(chan error, 10)

	// Create stats
	pipelineStats := stats.NewStats()

	// Create a hasher
	inputHasher := sha256.New()

	// Define chunk size
	chunkSize := 16

	// Start the reader stage in a goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		Stage(ctx, logger, mockFile.File, readCh, errCh, pipelineStats, inputHasher, chunkSize)
	}()

	// Wait for the read to be called
	<-reader.readCalled

	// Wait a bit to ensure the read timeout is triggered
	// The read timeout in the Stage function is 2 seconds, so we need to wait at least that long
	// Adding a buffer to ensure the timeout is triggered and processed
	time.Sleep(5 * time.Second)

	// Wait for the reader to finish
	wg.Wait()

	// Check that the read channel is closed
	_, ok := <-readCh
	if ok {
		t.Error("Expected read channel to be closed")
	}

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

	// Check that no data was processed
	if pipelineStats.InputBytes.Load() != 0 {
		t.Errorf("Expected InputBytes to be 0, got %d", pipelineStats.InputBytes.Load())
	}

	if pipelineStats.ChunksProcessed.Load() != 0 {
		t.Errorf("Expected ChunksProcessed to be 0, got %d", pipelineStats.ChunksProcessed.Load())
	}
}

// TestStage_ContextCanceledDuringRead tests what happens when the context is canceled during a read operation
func TestStage_ContextCanceledDuringRead(t *testing.T) {
	// Skip in short mode as this test involves timing
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	// Create a logger
	logger := zaptest.NewLogger(t)
	defer logger.Sync()

	// Create a mock slow file
	mockFile, reader, err := newMockSlowFile()
	if err != nil {
		t.Fatalf("Failed to create mock file: %v", err)
	}
	defer os.Remove(mockFile.Name())
	defer mockFile.Close()

	// Create a context that will be canceled during the read
	ctx, cancel := context.WithCancel(context.Background())

	// We'll use reader later to wait for the read operation to start

	// Create channels
	readCh := make(chan []byte, 10)
	errCh := make(chan error, 10)

	// Create stats
	pipelineStats := stats.NewStats()

	// Create a hasher
	inputHasher := sha256.New()

	// Define chunk size
	chunkSize := 16

	// Start the reader stage in a goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		Stage(ctx, logger, mockFile.File, readCh, errCh, pipelineStats, inputHasher, chunkSize)
	}()

	// Wait for the read to be called
	<-reader.readCalled

	// Cancel the context
	cancel()

	// Wait for the reader to finish
	wg.Wait()

	// Check that the read channel is closed
	_, ok := <-readCh
	if ok {
		t.Error("Expected read channel to be closed")
	}

	// Check that there are no errors (context cancellation is not reported as an error)
	select {
	case err := <-errCh:
		t.Errorf("Expected no errors in error channel, got: %v", err)
	default:
		// No errors, as expected
	}

	// Check that no data was processed
	if pipelineStats.InputBytes.Load() != 0 {
		t.Errorf("Expected InputBytes to be 0, got %d", pipelineStats.InputBytes.Load())
	}

	if pipelineStats.ChunksProcessed.Load() != 0 {
		t.Errorf("Expected ChunksProcessed to be 0, got %d", pipelineStats.ChunksProcessed.Load())
	}
}

// TestStage_ContextCanceledDuringSend tests what happens when the context is canceled during sending to the compression stage
func TestStage_ContextCanceledDuringSend(t *testing.T) {
	// Skip in short mode as this test involves timing
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	// Create a logger
	logger := zaptest.NewLogger(t)
	defer logger.Sync()

	// Create a temporary file with test data
	testData := []byte("This is test data for the reader stage.")
	tempFile, err := os.CreateTemp("", "reader_test")
	if err != nil {
		t.Fatalf("Failed to create temporary file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	_, err = tempFile.Write(testData)
	if err != nil {
		t.Fatalf("Failed to write to temporary file: %v", err)
	}

	// Seek to the beginning of the file
	_, err = tempFile.Seek(0, 0)
	if err != nil {
		t.Fatalf("Failed to seek to beginning of file: %v", err)
	}

	// Create a context that will be canceled during the send
	ctx, cancel := context.WithCancel(context.Background())

	// Create a channel with no buffer to block the send operation
	readCh := make(chan []byte, 0) // No buffer, will block
	errCh := make(chan error, 10)

	// Create stats
	pipelineStats := stats.NewStats()

	// Create a hasher
	inputHasher := sha256.New()

	// Define chunk size
	chunkSize := 16

	// Create a channel to signal when the read operation is complete
	readComplete := make(chan struct{})

	// Start a goroutine to monitor the read operation
	go func() {
		// Wait a bit to ensure the read operation has started
		time.Sleep(100 * time.Millisecond)

		// Signal that the read is complete
		close(readComplete)
	}()

	// Start the reader stage in a goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		Stage(ctx, logger, tempFile, readCh, errCh, pipelineStats, inputHasher, chunkSize)
	}()

	// Wait for the read operation to complete
	<-readComplete

	// Wait a bit more to ensure the send operation has started
	time.Sleep(100 * time.Millisecond)

	// Cancel the context
	cancel()

	// Wait for the reader to finish
	wg.Wait()

	// Check that the read channel is closed
	_, ok := <-readCh
	if ok {
		t.Error("Expected read channel to be closed")
	}

	// Check that there are no errors (context cancellation is not reported as an error)
	select {
	case err := <-errCh:
		t.Errorf("Expected no errors in error channel, got: %v", err)
	default:
		// No errors, as expected
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

	// Create a temporary file
	tempFile, err := os.CreateTemp("", "reader_test")
	if err != nil {
		t.Fatalf("Failed to create temporary file: %v", err)
	}

	// Close the file to cause a read error
	tempFile.Close()
	defer os.Remove(tempFile.Name())

	// Create a context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create channels - use a full error channel to test error channel full scenario
	readCh := make(chan []byte, 10)
	errCh := make(chan error, 1)
	// Fill the error channel
	errCh <- errors.New("dummy error to fill channel")

	// Create stats
	pipelineStats := stats.NewStats()

	// Create a hasher
	inputHasher := sha256.New()

	// Define chunk size
	chunkSize := 16

	// Start the reader stage in a goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		Stage(ctx, logger, tempFile, readCh, errCh, pipelineStats, inputHasher, chunkSize)
	}()

	// Wait for the reader to finish
	wg.Wait()

	// Check that the read channel is closed
	_, ok := <-readCh
	if ok {
		t.Error("Expected read channel to be closed")
	}

	// Check that the error channel still has only one error (the one we put in)
	if len(errCh) != 1 {
		t.Errorf("Expected error channel to have 1 error, got %d", len(errCh))
	}

	// Check that no data was processed
	if pipelineStats.InputBytes.Load() != 0 {
		t.Errorf("Expected InputBytes to be 0, got %d", pipelineStats.InputBytes.Load())
	}

	if pipelineStats.ChunksProcessed.Load() != 0 {
		t.Errorf("Expected ChunksProcessed to be 0, got %d", pipelineStats.ChunksProcessed.Load())
	}
}
