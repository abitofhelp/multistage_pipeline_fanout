// Copyright (c) 2025 A Bit of Help, Inc.

package compressor

import (
	"context"
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

	// Create a context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create channels
	inputCh := make(chan []byte, 10)
	outputCh := make(chan []byte, 10)
	errCh := make(chan error, 10)

	// Create stats
	pipelineStats := stats.NewStats()

	// Start the compressor stage in a goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		Stage(ctx, 0, logger, inputCh, outputCh, errCh, pipelineStats, cancel)
	}()

	// Send test data
	testData := []byte("This is a test string that will be compressed")
	inputCh <- testData

	// Close the input channel to signal no more data
	close(inputCh)

	// Wait for the compressor to finish
	wg.Wait()

	// Check that there are no errors
	select {
	case err := <-errCh:
		t.Errorf("Expected no errors, got: %v", err)
	default:
		// No errors, as expected
	}

	// Check that the output channel has data
	var compressedData []byte
	select {
	case compressedData = <-outputCh:
		// Data received, as expected
	default:
		t.Error("Expected compressed data, got none")
	}

	// Verify that the compressed data is not empty and is different from the input
	if len(compressedData) == 0 {
		t.Error("Compressed data is empty")
	}

	if len(compressedData) >= len(testData) {
		t.Logf("Note: Compressed data (%d bytes) is not smaller than input data (%d bytes). This can happen with small or already compressed data.", len(compressedData), len(testData))
	}

	// Check that the stats were updated
	if pipelineStats.CompressedBytes.Load() == 0 {
		t.Error("Expected CompressedBytes to be non-zero")
	}

	if pipelineStats.CompressedBytes.Load() != uint64(len(compressedData)) {
		t.Errorf("Expected CompressedBytes to be %d, got %d", len(compressedData), pipelineStats.CompressedBytes.Load())
	}
}

func TestStage_MultipleChunks(t *testing.T) {
	// Create a logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()

	// Create a context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create channels
	inputCh := make(chan []byte, 10)
	outputCh := make(chan []byte, 10)
	errCh := make(chan error, 10)

	// Create stats
	pipelineStats := stats.NewStats()

	// Start the compressor stage in a goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		Stage(ctx, 0, logger, inputCh, outputCh, errCh, pipelineStats, cancel)
	}()

	// Send multiple chunks of test data
	numChunks := 5
	for i := 0; i < numChunks; i++ {
		inputData := []byte("Chunk " + string(byte('0'+i)) + " of test data for compression")
		inputCh <- inputData
	}

	// Close the input channel to signal no more data
	close(inputCh)

	// Wait for the compressor to finish
	wg.Wait()

	// Check that there are no errors
	select {
	case err := <-errCh:
		t.Errorf("Expected no errors, got: %v", err)
	default:
		// No errors, as expected
	}

	// Check that the output channel has the expected number of chunks
	var totalCompressedBytes uint64
	for i := 0; i < numChunks; i++ {
		select {
		case compressedData := <-outputCh:
			totalCompressedBytes += uint64(len(compressedData))
		default:
			t.Errorf("Expected %d chunks, got %d", numChunks, i)
			break
		}
	}

	// Check that the stats were updated correctly
	if pipelineStats.CompressedBytes.Load() != totalCompressedBytes {
		t.Errorf("Expected CompressedBytes to be %d, got %d", totalCompressedBytes, pipelineStats.CompressedBytes.Load())
	}
}

func TestStage_ContextCanceled(t *testing.T) {
	// Create a logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()

	// Create a context that's already canceled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Create channels
	inputCh := make(chan []byte, 10)
	outputCh := make(chan []byte, 10)
	errCh := make(chan error, 10)

	// Create stats
	pipelineStats := stats.NewStats()

	// Start the compressor stage in a goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		Stage(ctx, 0, logger, inputCh, outputCh, errCh, pipelineStats, cancel)
	}()

	// Send test data
	inputCh <- []byte("This data should not be processed due to canceled context")

	// Wait for the compressor to finish
	wg.Wait()

	// Check that there are no errors in the error channel (context cancellation is not reported as an error)
	select {
	case err := <-errCh:
		t.Errorf("Expected no errors in error channel, got: %v", err)
	default:
		// No errors, as expected
	}

	// Check that the output channel is empty
	select {
	case outputData := <-outputCh:
		t.Errorf("Expected no output data, got: %v", outputData)
	default:
		// No output data, as expected
	}
}
