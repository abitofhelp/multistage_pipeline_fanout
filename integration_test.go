// Copyright (c) 2025 A Bit of Help, Inc.

package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"os"
	"testing"
	"time"

	"github.com/abitofhelp/multistage_pipeline_fanout/pkg/compression"
	"github.com/abitofhelp/multistage_pipeline_fanout/pkg/encryption"
	"github.com/abitofhelp/multistage_pipeline_fanout/pkg/logger"
	"github.com/abitofhelp/multistage_pipeline_fanout/pkg/pipeline"
	"github.com/abitofhelp/multistage_pipeline_fanout/pkg/pipeline/encryptor"
	"github.com/abitofhelp/multistage_pipeline_fanout/pkg/pipeline/options"
	"github.com/abitofhelp/multistage_pipeline_fanout/pkg/pipeline/processor"
	"github.com/abitofhelp/multistage_pipeline_fanout/pkg/pipeline/reader"
	"github.com/abitofhelp/multistage_pipeline_fanout/pkg/pipeline/writer"
	"github.com/abitofhelp/multistage_pipeline_fanout/pkg/stats"
	"github.com/andybalholm/brotli"
	"github.com/stretchr/testify/assert"
)

// TestIntegration_FullPipeline tests the entire pipeline from end to end
func TestIntegration_FullPipeline(t *testing.T) {
	// Create a logger
	log := logger.InitLogger()
	defer func() { logger.SafeSync(log) }()

	// Create a context
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create temporary input file with test data
	inputData := []byte("This is test data for the pipeline integration test. It will be processed through all stages.")
	inputFile, err := os.CreateTemp("", "integration_test_input")
	if err != nil {
		t.Fatalf("Failed to create temporary input file: %v", err)
	}
	defer os.Remove(inputFile.Name())

	_, err = inputFile.Write(inputData)
	if err != nil {
		t.Fatalf("Failed to write to temporary input file: %v", err)
	}
	inputFile.Close()

	// Create temporary output file
	outputFile, err := os.CreateTemp("", "integration_test_output")
	if err != nil {
		t.Fatalf("Failed to create temporary output file: %v", err)
	}
	outputFile.Close()
	defer os.Remove(outputFile.Name())

	// Process the file
	stats, err := pipeline.ProcessFile(ctx, log, inputFile.Name(), outputFile.Name())

	// Check for errors
	assert.NoError(t, err, "Expected no error from ProcessFile")
	assert.NotNil(t, stats, "Expected non-nil stats")

	// Verify stats
	assert.Equal(t, uint64(len(inputData)), stats.InputBytes.Load(), "Input bytes should match original data length")
	assert.Greater(t, stats.OutputBytes.Load(), uint64(0), "Output bytes should be non-zero")
	assert.Greater(t, stats.CompressedBytes.Load(), uint64(0), "Compressed bytes should be non-zero")
	assert.Greater(t, stats.EncryptedBytes.Load(), uint64(0), "Encrypted bytes should be non-zero")
	assert.Greater(t, stats.ChunksProcessed.Load(), uint64(0), "Chunks processed should be non-zero")
	assert.NotZero(t, stats.ProcessingTime, "Processing time should be non-zero")
	assert.NotNil(t, stats.InputHash, "Input hash should not be nil")
	assert.NotNil(t, stats.OutputHash, "Output hash should not be nil")

	// Verify output file exists and has content
	outputData, err := os.ReadFile(outputFile.Name())
	assert.NoError(t, err, "Should be able to read output file")
	assert.Greater(t, len(outputData), 0, "Output file should not be empty")
}

// TestIntegration_ComponentInteractions tests the interactions between specific components
func TestIntegration_ComponentInteractions(t *testing.T) {
	// Create a logger
	log := logger.InitLogger()
	defer func() { logger.SafeSync(log) }()

	// Test data - use a repeating pattern to ensure good compression
	testData := bytes.Repeat([]byte("Test data for component interaction testing "), 100)

	// Test compression and decompression
	t.Run("Compression", func(t *testing.T) {
		// Create a context
		ctx := context.Background()

		compressed, err := compression.CompressDataWithContext(ctx, testData)
		assert.NoError(t, err, "Compression should not error")
		assert.NotNil(t, compressed, "Compressed data should not be nil")
		assert.Less(t, len(compressed), len(testData), "Compressed data should be smaller than original")

		// Decompress using brotli
		reader := brotli.NewReader(bytes.NewReader(compressed))
		decompressed := new(bytes.Buffer)
		_, err = decompressed.ReadFrom(reader)
		assert.NoError(t, err, "Decompression should not error")
		assert.Equal(t, testData, decompressed.Bytes(), "Decompressed data should match original")
	})

	// Test encryption and decryption
	t.Run("Encryption", func(t *testing.T) {
		// Create a context
		ctx := context.Background()

		aead, err := encryption.InitEncryption(log)
		assert.NoError(t, err, "Encryption initialization should not error")
		assert.NotNil(t, aead, "AEAD should not be nil")

		encrypted, err := encryption.EncryptDataWithContext(ctx, aead, testData)
		assert.NoError(t, err, "Encryption should not error")
		assert.NotNil(t, encrypted, "Encrypted data should not be nil")
		assert.NotEqual(t, testData, encrypted, "Encrypted data should differ from original")

		decrypted, err := aead.Decrypt(encrypted, []byte{})
		assert.NoError(t, err, "Decryption should not error")
		assert.Equal(t, testData, decrypted, "Decrypted data should match original")
	})
}

// TestIntegration_PipelineStages tests the individual pipeline stages working together
func TestIntegration_PipelineStages(t *testing.T) {
	// Create a logger
	log := logger.InitLogger()
	defer func() { logger.SafeSync(log) }()

	// Create a context
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create channels
	readCh := make(chan []byte, 10)
	compressCh := make(chan []byte, 10)
	encryptCh := make(chan []byte, 10)
	errCh := make(chan error, 10)

	// Create stats
	pipelineStats := stats.NewStats()

	// Create hashers for input and output
	inputHasher := sha256.New()
	outputHasher := sha256.New()

	// Get default options
	opts := options.DefaultPipelineOptions()

	// Create test data
	testData := []byte("Test data for pipeline stages integration test")
	inputFile, err := os.CreateTemp("", "pipeline_stages_test_input")
	if err != nil {
		t.Fatalf("Failed to create temporary input file: %v", err)
	}
	defer os.Remove(inputFile.Name())

	_, err = inputFile.Write(testData)
	if err != nil {
		t.Fatalf("Failed to write to temporary input file: %v", err)
	}
	inputFile.Close()

	// Reopen the file for reading
	inputFile, err = os.Open(inputFile.Name())
	if err != nil {
		t.Fatalf("Failed to open input file: %v", err)
	}
	defer inputFile.Close()

	// Create output file
	outputFile, err := os.CreateTemp("", "pipeline_stages_test_output")
	if err != nil {
		t.Fatalf("Failed to create temporary output file: %v", err)
	}
	defer os.Remove(outputFile.Name())
	defer outputFile.Close()

	// Initialize encryption
	aead, err := encryption.InitEncryption(log)
	if err != nil {
		t.Fatalf("Failed to initialize encryption: %v", err)
	}

	// Start reader stage in a goroutine
	go reader.Stage(ctx, log, inputFile, readCh, errCh, pipelineStats, inputHasher, opts.ChunkSize)

	// Start processor stage (compressor) in a goroutine
	processorFunc := func(ctx context.Context, data []byte) ([]byte, error) {
		return compression.CompressDataWithContext(ctx, data)
	}
	go processor.Stage(
		ctx,
		0,
		"compressor",
		log,
		readCh,
		compressCh,
		errCh,
		pipelineStats,
		cancel,
		processorFunc,
		pipelineStats.UpdateCompressedBytes,
	)

	// Start encryptor stage in a goroutine
	go encryptor.Stage(ctx, 0, aead, compressCh, encryptCh, errCh, pipelineStats, cancel, log)

	// Start writer stage in a goroutine
	go writer.Stage(ctx, log, outputFile, encryptCh, errCh, pipelineStats, outputHasher, cancel)

	// Wait for processing to complete
	time.Sleep(1 * time.Second)

	// Check for errors
	select {
	case err := <-errCh:
		t.Fatalf("Unexpected error: %v", err)
	default:
		// No errors, continue
	}

	// Verify stats
	assert.Greater(t, pipelineStats.InputBytes.Load(), uint64(0), "Input bytes should be non-zero")
	assert.Greater(t, pipelineStats.CompressedBytes.Load(), uint64(0), "Compressed bytes should be non-zero")
	assert.Greater(t, pipelineStats.EncryptedBytes.Load(), uint64(0), "Encrypted bytes should be non-zero")
	assert.Greater(t, pipelineStats.OutputBytes.Load(), uint64(0), "Output bytes should be non-zero")
}

// TestIntegration_ErrorHandling tests how the pipeline handles errors
func TestIntegration_ErrorHandling(t *testing.T) {
	// Create a logger
	log := logger.InitLogger()
	defer func() { logger.SafeSync(log) }()

	// Create a context
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test with non-existent input file
	_, err := pipeline.ProcessFile(ctx, log, "nonexistent_file.txt", "output.txt")
	assert.Error(t, err, "Should error with non-existent input file")

	// Test with invalid output path (directory)
	tempDir, err := os.MkdirTemp("", "pipeline_error_test")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a valid input file
	inputFile, err := os.CreateTemp("", "pipeline_error_test_input")
	if err != nil {
		t.Fatalf("Failed to create temporary input file: %v", err)
	}
	defer os.Remove(inputFile.Name())
	inputFile.Write([]byte("Test data"))
	inputFile.Close()

	// Try to use a directory as output file
	_, err = pipeline.ProcessFile(ctx, log, inputFile.Name(), tempDir)
	assert.Error(t, err, "Should error when output path is a directory")

	// Test with canceled context
	canceledCtx, cancelFunc := context.WithCancel(context.Background())
	cancelFunc() // Cancel immediately
	_, err = pipeline.ProcessFile(canceledCtx, log, inputFile.Name(), "output.txt")
	assert.Error(t, err, "Should error with canceled context")
}
