// Copyright (c) 2025 A Bit of Help, Inc.

package encryptor

import (
	"context"
	"sync"
	"testing"

	"github.com/abitofhelp/multistage_pipeline_fanout/pkg/encryption"
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

	// Initialize encryption
	aead, err := encryption.InitEncryption(logger)
	if err != nil {
		t.Fatalf("Failed to initialize encryption: %v", err)
	}

	// Create a context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create channels
	inputCh := make(chan []byte, 10)
	outputCh := make(chan []byte, 10)
	errCh := make(chan error, 10)

	// Create stats
	pipelineStats := stats.NewStats()

	// Start the encryptor stage in a goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		Stage(ctx, 0, aead, inputCh, outputCh, errCh, pipelineStats, cancel, logger)
	}()

	// Send test data
	testData := []byte("This is a test string that will be encrypted")
	inputCh <- testData

	// Close the input channel to signal no more data
	close(inputCh)

	// Wait for the encryptor to finish
	wg.Wait()

	// Check that there are no errors
	select {
	case err := <-errCh:
		t.Errorf("Expected no errors, got: %v", err)
	default:
		// No errors, as expected
	}

	// Check that the output channel has data
	var encryptedData []byte
	select {
	case encryptedData = <-outputCh:
		// Data received, as expected
	default:
		t.Error("Expected encrypted data, got none")
	}

	// Verify that the encrypted data is not empty and is different from the input
	if len(encryptedData) == 0 {
		t.Error("Encrypted data is empty")
	}

	// Encrypted data should be longer than input due to added metadata
	if len(encryptedData) <= len(testData) {
		t.Errorf("Expected encrypted data (%d bytes) to be longer than input data (%d bytes)", len(encryptedData), len(testData))
	}

	// Check that the stats were updated
	if pipelineStats.EncryptedBytes.Load() == 0 {
		t.Error("Expected EncryptedBytes to be non-zero")
	}

	if pipelineStats.EncryptedBytes.Load() != uint64(len(encryptedData)) {
		t.Errorf("Expected EncryptedBytes to be %d, got %d", len(encryptedData), pipelineStats.EncryptedBytes.Load())
	}

	// Verify that the data can be decrypted
	decryptedData, err := aead.Decrypt(encryptedData, []byte{})
	if err != nil {
		t.Errorf("Failed to decrypt data: %v", err)
	}

	// Check that decrypted data matches original
	if len(decryptedData) != len(testData) {
		t.Errorf("Expected decrypted data length %d, got %d", len(testData), len(decryptedData))
	}
	for i := range testData {
		if decryptedData[i] != testData[i] {
			t.Errorf("Expected decrypted[%d] = %d, got %d", i, testData[i], decryptedData[i])
		}
	}
}

func TestStage_MultipleChunks(t *testing.T) {
	// Create a logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()

	// Initialize encryption
	aead, err := encryption.InitEncryption(logger)
	if err != nil {
		t.Fatalf("Failed to initialize encryption: %v", err)
	}

	// Create a context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create channels
	inputCh := make(chan []byte, 10)
	outputCh := make(chan []byte, 10)
	errCh := make(chan error, 10)

	// Create stats
	pipelineStats := stats.NewStats()

	// Start the encryptor stage in a goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		Stage(ctx, 0, aead, inputCh, outputCh, errCh, pipelineStats, cancel, logger)
	}()

	// Send multiple chunks of test data
	numChunks := 5
	for i := 0; i < numChunks; i++ {
		inputData := []byte("Chunk " + string(byte('0'+i)) + " of test data for encryption")
		inputCh <- inputData
	}

	// Close the input channel to signal no more data
	close(inputCh)

	// Wait for the encryptor to finish
	wg.Wait()

	// Check that there are no errors
	select {
	case err := <-errCh:
		t.Errorf("Expected no errors, got: %v", err)
	default:
		// No errors, as expected
	}

	// Check that the output channel has the expected number of chunks
	var totalEncryptedBytes uint64
	for i := 0; i < numChunks; i++ {
		select {
		case encryptedData := <-outputCh:
			totalEncryptedBytes += uint64(len(encryptedData))
		default:
			t.Errorf("Expected %d chunks, got %d", numChunks, i)
			break
		}
	}

	// Check that the stats were updated correctly
	if pipelineStats.EncryptedBytes.Load() != totalEncryptedBytes {
		t.Errorf("Expected EncryptedBytes to be %d, got %d", totalEncryptedBytes, pipelineStats.EncryptedBytes.Load())
	}
}

func TestStage_ContextCanceled(t *testing.T) {
	// Create a logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()

	// Initialize encryption
	aead, err := encryption.InitEncryption(logger)
	if err != nil {
		t.Fatalf("Failed to initialize encryption: %v", err)
	}

	// Create a context that's already canceled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Create channels
	inputCh := make(chan []byte, 10)
	outputCh := make(chan []byte, 10)
	errCh := make(chan error, 10)

	// Create stats
	pipelineStats := stats.NewStats()

	// Start the encryptor stage in a goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		Stage(ctx, 0, aead, inputCh, outputCh, errCh, pipelineStats, cancel, logger)
	}()

	// Send test data
	inputCh <- []byte("This data should not be processed due to canceled context")

	// Wait for the encryptor to finish
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
