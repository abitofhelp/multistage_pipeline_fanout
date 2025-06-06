// Copyright (c) 2025 A Bit of Help, Inc.

package pipeline

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/abitofhelp/multistage_pipeline_fanout/pkg/logger"
)

func TestProcessFile_Success(t *testing.T) {
	// Create a logger
	log := logger.InitLogger()
	defer func() { logger.SafeSync(log) }()

	// Create a context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create temporary input file with test data
	inputData := []byte("This is test data for the pipeline. It will be compressed and encrypted.")
	inputFile, err := os.CreateTemp("", "pipeline_test_input")
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
	outputFile, err := os.CreateTemp("", "pipeline_test_output")
	if err != nil {
		t.Fatalf("Failed to create temporary output file: %v", err)
	}
	outputFile.Close()
	defer os.Remove(outputFile.Name())

	// Process the file
	stats, err := ProcessFile(ctx, log, inputFile.Name(), outputFile.Name())

	// Check for errors
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Check that stats are not nil
	if stats == nil {
		t.Fatal("Expected non-nil stats")
	}

	// Check that stats have been updated
	if stats.InputBytes.Load() != uint64(len(inputData)) {
		t.Errorf("Expected InputBytes to be %d, got %d", len(inputData), stats.InputBytes.Load())
	}

	if stats.OutputBytes.Load() == 0 {
		t.Error("Expected OutputBytes to be non-zero")
	}

	if stats.CompressedBytes.Load() == 0 {
		t.Error("Expected CompressedBytes to be non-zero")
	}

	if stats.EncryptedBytes.Load() == 0 {
		t.Error("Expected EncryptedBytes to be non-zero")
	}

	if stats.ChunksProcessed.Load() == 0 {
		t.Error("Expected ChunksProcessed to be non-zero")
	}

	if stats.ProcessingTime == 0 {
		t.Error("Expected ProcessingTime to be non-zero")
	}

	// Check that the output file exists and has content
	outputData, err := os.ReadFile(outputFile.Name())
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	if len(outputData) == 0 {
		t.Error("Expected non-empty output file")
	}
}

func TestProcessFile_InvalidParameters(t *testing.T) {
	// Create a logger
	log := logger.InitLogger()
	defer func() { logger.SafeSync(log) }()

	// Create a context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Test nil context
	_, err := ProcessFile(nil, log, "input.txt", "output.txt")
	if err == nil {
		t.Error("Expected error for nil context, got nil")
	}

	// Test nil logger
	_, err = ProcessFile(ctx, nil, "input.txt", "output.txt")
	if err == nil {
		t.Error("Expected error for nil logger, got nil")
	}

	// Test empty input path
	_, err = ProcessFile(ctx, log, "", "output.txt")
	if err == nil {
		t.Error("Expected error for empty input path, got nil")
	}

	// Test empty output path
	_, err = ProcessFile(ctx, log, "input.txt", "")
	if err == nil {
		t.Error("Expected error for empty output path, got nil")
	}
}

func TestProcessFile_InputFileNotFound(t *testing.T) {
	// Create a logger
	log := logger.InitLogger()
	defer func() { logger.SafeSync(log) }()

	// Create a context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create temporary output file
	outputFile, err := os.CreateTemp("", "pipeline_test_output")
	if err != nil {
		t.Fatalf("Failed to create temporary output file: %v", err)
	}
	outputFile.Close()
	defer os.Remove(outputFile.Name())

	// Process with non-existent input file
	_, err = ProcessFile(ctx, log, "nonexistent_file.txt", outputFile.Name())

	// Check for error
	if err == nil {
		t.Error("Expected error for non-existent input file, got nil")
	}
}

func TestProcessFile_OutputFileError(t *testing.T) {
	// Create a logger
	log := logger.InitLogger()
	defer func() { logger.SafeSync(log) }()

	// Create a context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create temporary input file with test data
	inputData := []byte("This is test data for the pipeline.")
	inputFile, err := os.CreateTemp("", "pipeline_test_input")
	if err != nil {
		t.Fatalf("Failed to create temporary input file: %v", err)
	}
	defer os.Remove(inputFile.Name())

	_, err = inputFile.Write(inputData)
	if err != nil {
		t.Fatalf("Failed to write to temporary input file: %v", err)
	}
	inputFile.Close()

	// Use a directory as output path to cause an error
	tempDir, err := os.MkdirTemp("", "pipeline_test_dir")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Process with directory as output file
	_, err = ProcessFile(ctx, log, inputFile.Name(), tempDir)

	// Check for error
	if err == nil {
		t.Error("Expected error for directory as output file, got nil")
	}
}

func TestProcessFile_ContextCanceled(t *testing.T) {
	// Create a logger
	log := logger.InitLogger()
	defer func() { logger.SafeSync(log) }()

	// Create a context that's already canceled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Create temporary input file with test data
	inputData := []byte("This is test data for the pipeline.")
	inputFile, err := os.CreateTemp("", "pipeline_test_input")
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
	outputFile, err := os.CreateTemp("", "pipeline_test_output")
	if err != nil {
		t.Fatalf("Failed to create temporary output file: %v", err)
	}
	outputFile.Close()
	defer os.Remove(outputFile.Name())

	// Process with canceled context
	_, err = ProcessFile(ctx, log, inputFile.Name(), outputFile.Name())

	// Check for error
	if err == nil {
		t.Error("Expected error for canceled context, got nil")
	}
}

func TestProcessFile_Timeout(t *testing.T) {
	// Skip this test in short mode as it involves waiting
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	// Create a logger
	log := logger.InitLogger()
	defer func() { logger.SafeSync(log) }()

	// Create a context with a very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// Create temporary input file with large test data to ensure timeout
	inputData := make([]byte, 1024*1024) // 1MB
	for i := range inputData {
		inputData[i] = byte(i % 256)
	}

	inputFile, err := os.CreateTemp("", "pipeline_test_input")
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
	outputFile, err := os.CreateTemp("", "pipeline_test_output")
	if err != nil {
		t.Fatalf("Failed to create temporary output file: %v", err)
	}
	outputFile.Close()
	defer os.Remove(outputFile.Name())

	// Wait to ensure the timeout has occurred
	time.Sleep(10 * time.Millisecond)

	// Process with timeout context
	_, err = ProcessFile(ctx, log, inputFile.Name(), outputFile.Name())

	// Check for error
	if err == nil {
		t.Error("Expected error for timeout, got nil")
	}
}
