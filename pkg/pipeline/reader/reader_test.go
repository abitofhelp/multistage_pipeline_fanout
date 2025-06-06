// Copyright (c) 2025 A Bit of Help, Inc.

package reader

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
