// Copyright (c) 2025 A Bit of Help, Inc.

package processor

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/abitofhelp/multistage_pipeline_fanout/pkg/stats"
	"go.uber.org/zap"
)

func TestStage_Success(t *testing.T) {
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

	// Create a simple processor function that doubles the input
	processorFunc := func(ctx context.Context, data []byte) ([]byte, error) {
		result := make([]byte, len(data))
		for i, b := range data {
			result[i] = b * 2
		}
		return result, nil
	}

	// Create a simple stats update function
	updateStats := func(n uint64) {
		pipelineStats.UpdateOutputBytes(n)
	}

	// Start the processor stage in a goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		Stage(ctx, 0, "test", logger, inputCh, outputCh, errCh, pipelineStats, cancel, processorFunc, updateStats)
	}()

	// Send test data
	inputData := []byte{1, 2, 3, 4, 5}
	expectedOutput := []byte{2, 4, 6, 8, 10}
	inputCh <- inputData

	// Close the input channel to signal no more data
	close(inputCh)

	// Wait for the processor to finish
	wg.Wait()

	// Check that there are no errors
	select {
	case err := <-errCh:
		t.Errorf("Expected no errors, got: %v", err)
	default:
		// No errors, as expected
	}

	// Check that the output channel has the expected data
	select {
	case outputData := <-outputCh:
		// Check that the output data matches the expected output
		if len(outputData) != len(expectedOutput) {
			t.Errorf("Expected output length %d, got %d", len(expectedOutput), len(outputData))
		}
		for i := range expectedOutput {
			if outputData[i] != expectedOutput[i] {
				t.Errorf("Expected output[%d] = %d, got %d", i, expectedOutput[i], outputData[i])
			}
		}
	default:
		t.Error("Expected output data, got none")
	}

	// Check that the stats were updated
	if pipelineStats.OutputBytes.Load() != uint64(len(expectedOutput)) {
		t.Errorf("Expected OutputBytes to be %d, got %d", len(expectedOutput), pipelineStats.OutputBytes.Load())
	}
}

func TestStage_ProcessorError(t *testing.T) {
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

	// Create a processor function that returns an error
	expectedErr := errors.New("processor error")
	processorFunc := func(ctx context.Context, data []byte) ([]byte, error) {
		return nil, expectedErr
	}

	// Create a simple stats update function
	updateStats := func(n uint64) {
		pipelineStats.UpdateOutputBytes(n)
	}

	// Start the processor stage in a goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		Stage(ctx, 0, "test", logger, inputCh, outputCh, errCh, pipelineStats, cancel, processorFunc, updateStats)
	}()

	// Send test data
	inputCh <- []byte{1, 2, 3, 4, 5}

	// Wait for the processor to finish (it should exit due to the error)
	wg.Wait()

	// Check that there is an error
	select {
	case err := <-errCh:
		// Check that the error contains the expected error
		if !errors.Is(err, expectedErr) {
			t.Errorf("Expected error to contain %v, got: %v", expectedErr, err)
		}
	default:
		t.Error("Expected an error, got none")
	}

	// Check that the output channel is empty
	select {
	case outputData := <-outputCh:
		t.Errorf("Expected no output data, got: %v", outputData)
	default:
		// No output data, as expected
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

	// Create a simple processor function
	processorFunc := func(ctx context.Context, data []byte) ([]byte, error) {
		return data, nil
	}

	// Create a simple stats update function
	updateStats := func(n uint64) {
		pipelineStats.UpdateOutputBytes(n)
	}

	// Start the processor stage in a goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		Stage(ctx, 0, "test", logger, inputCh, outputCh, errCh, pipelineStats, cancel, processorFunc, updateStats)
	}()

	// Send test data
	inputCh <- []byte{1, 2, 3, 4, 5}

	// Wait for the processor to finish (it should exit due to the canceled context)
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

func TestStage_ProcessorPanic(t *testing.T) {
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

	// Create a processor function that panics
	processorFunc := func(ctx context.Context, data []byte) ([]byte, error) {
		panic("processor panic")
	}

	// Create a simple stats update function
	updateStats := func(n uint64) {
		pipelineStats.UpdateOutputBytes(n)
	}

	// Start the processor stage in a goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		Stage(ctx, 0, "test", logger, inputCh, outputCh, errCh, pipelineStats, cancel, processorFunc, updateStats)
	}()

	// Send test data
	inputCh <- []byte{1, 2, 3, 4, 5}

	// Wait for the processor to finish (it should exit due to the panic)
	wg.Wait()

	// Check that there is an error
	select {
	case err := <-errCh:
		// Check that the error contains information about the panic
		if err == nil {
			t.Error("Expected non-nil error")
		} else if msg := err.Error(); msg == "" || msg == "nil" {
			t.Errorf("Expected non-empty error message, got: %v", msg)
		}
	default:
		t.Error("Expected an error, got none")
	}

	// Check that the output channel is empty
	select {
	case outputData := <-outputCh:
		t.Errorf("Expected no output data, got: %v", outputData)
	default:
		// No output data, as expected
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

	// Create a simple processor function that doubles the input
	processorFunc := func(ctx context.Context, data []byte) ([]byte, error) {
		result := make([]byte, len(data))
		for i, b := range data {
			result[i] = b * 2
		}
		return result, nil
	}

	// Create a simple stats update function
	updateStats := func(n uint64) {
		pipelineStats.UpdateOutputBytes(n)
	}

	// Start the processor stage in a goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		Stage(ctx, 0, "test", logger, inputCh, outputCh, errCh, pipelineStats, cancel, processorFunc, updateStats)
	}()

	// Send multiple chunks of test data
	numChunks := 5
	for i := 0; i < numChunks; i++ {
		inputData := []byte{byte(i + 1), byte(i + 2), byte(i + 3)}
		inputCh <- inputData
	}

	// Close the input channel to signal no more data
	close(inputCh)

	// Wait for the processor to finish
	wg.Wait()

	// Check that there are no errors
	select {
	case err := <-errCh:
		t.Errorf("Expected no errors, got: %v", err)
	default:
		// No errors, as expected
	}

	// Check that the output channel has the expected number of chunks
	var totalOutputBytes uint64
	for i := 0; i < numChunks; i++ {
		select {
		case outputData := <-outputCh:
			// Just count the bytes, we don't need to check the exact values
			totalOutputBytes += uint64(len(outputData))
		default:
			t.Errorf("Expected %d chunks, got %d", numChunks, i)
			break
		}
	}

	// Check that the stats were updated correctly
	if pipelineStats.OutputBytes.Load() != totalOutputBytes {
		t.Errorf("Expected OutputBytes to be %d, got %d", totalOutputBytes, pipelineStats.OutputBytes.Load())
	}
}

func TestStage_OutputChannelBlocked(t *testing.T) {
	// Skip this test in short mode as it involves waiting
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	// Create a logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()

	// Create a context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create channels - output channel with no buffer to cause blocking
	inputCh := make(chan []byte, 10)
	outputCh := make(chan []byte) // Unbuffered channel
	errCh := make(chan error, 10)

	// Create stats
	pipelineStats := stats.NewStats()

	// Create a simple processor function
	processorFunc := func(ctx context.Context, data []byte) ([]byte, error) {
		return data, nil
	}

	// Create a simple stats update function
	updateStats := func(n uint64) {
		pipelineStats.UpdateOutputBytes(n)
	}

	// Start the processor stage in a goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		Stage(ctx, 0, "test", logger, inputCh, outputCh, errCh, pipelineStats, cancel, processorFunc, updateStats)
	}()

	// Send test data
	inputCh <- []byte{1, 2, 3, 4, 5}

	// Don't read from the output channel to cause blocking
	// The processor should timeout after 5 seconds and report an error

	// Wait for the processor to finish (it should exit due to the timeout)
	wg.Wait()

	// Check that there is an error
	select {
	case err := <-errCh:
		// Check that the error contains information about the blocked stage
		if err == nil {
			t.Error("Expected non-nil error")
		} else {
			errMsg := err.Error()
			if errMsg == "" {
				t.Error("Expected non-empty error message")
			}
			if !containsAny(errMsg, "blocked", "timeout") {
				t.Errorf("Expected error message to mention blocking or timeout, got: %s", errMsg)
			}
		}
	default:
		t.Error("Expected an error, got none")
	}
}

// Helper function to check if a string contains any of the given substrings
func containsAny(s string, substrings ...string) bool {
	for _, sub := range substrings {
		if contains(s, sub) {
			return true
		}
	}
	return false
}

// Helper function to check if a string contains a substring
func contains(s, substring string) bool {
	for i := 0; i <= len(s)-len(substring); i++ {
		if s[i:i+len(substring)] == substring {
			return true
		}
	}
	return false
}

func TestStage_ContextCanceledDuringOutputSend(t *testing.T) {
	// Create a logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()

	// Create a context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create channels - output channel with no buffer to cause blocking
	inputCh := make(chan []byte, 10)
	outputCh := make(chan []byte) // Unbuffered channel
	errCh := make(chan error, 10)

	// Create stats
	pipelineStats := stats.NewStats()

	// Create a processor function that succeeds
	processorFunc := func(ctx context.Context, data []byte) ([]byte, error) {
		return data, nil
	}

	// Create a simple stats update function
	updateStats := func(n uint64) {
		pipelineStats.UpdateOutputBytes(n)
	}

	// Start the processor stage in a goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		Stage(ctx, 0, "test", logger, inputCh, outputCh, errCh, pipelineStats, cancel, processorFunc, updateStats)
	}()

	// Send test data
	inputCh <- []byte{1, 2, 3, 4, 5}

	// Wait a bit to ensure the processor is trying to send to the output channel
	time.Sleep(100 * time.Millisecond)

	// Cancel the context while the processor is blocked trying to send to the output channel
	cancel()

	// Wait for the processor to finish
	wg.Wait()

	// Check that there are no errors in the error channel
	select {
	case err := <-errCh:
		t.Errorf("Expected no errors in error channel, got: %v", err)
	default:
		// No errors, as expected
	}
}

func TestStage_ContextCanceledDuringProcessing(t *testing.T) {
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

	// Create a processor function that checks for context cancellation
	processorFunc := func(ctx context.Context, data []byte) ([]byte, error) {
		// Cancel the context while the processor is running
		cancel()

		// Small delay to ensure the cancellation is processed
		time.Sleep(10 * time.Millisecond)

		// Check if context is done
		select {
		case <-ctx.Done():
			return nil, ctx.Err() // This should be context.Canceled
		default:
			return data, nil
		}
	}

	// Create a simple stats update function
	updateStats := func(n uint64) {
		pipelineStats.UpdateOutputBytes(n)
	}

	// Start the processor stage in a goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		Stage(ctx, 0, "test", logger, inputCh, outputCh, errCh, pipelineStats, cancel, processorFunc, updateStats)
	}()

	// Send test data
	inputCh <- []byte{1, 2, 3, 4, 5}

	// Wait for the processor to finish
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

func TestStage_ErrorChannelFull(t *testing.T) {
	// Skip this test in short mode as it involves waiting
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	// Create a logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()

	// Create a context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create channels - error channel with buffer size 0 to cause blocking
	inputCh := make(chan []byte, 10)
	outputCh := make(chan []byte, 10)
	errCh := make(chan error) // Unbuffered channel

	// Create stats
	pipelineStats := stats.NewStats()

	// Create a processor function that returns an error
	processorFunc := func(ctx context.Context, data []byte) ([]byte, error) {
		return nil, errors.New("test error")
	}

	// Create a simple stats update function
	updateStats := func(n uint64) {
		pipelineStats.UpdateOutputBytes(n)
	}

	// Start the processor stage in a goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		Stage(ctx, 0, "test", logger, inputCh, outputCh, errCh, pipelineStats, cancel, processorFunc, updateStats)
	}()

	// Send test data
	inputCh <- []byte{1, 2, 3, 4, 5}

	// Don't read from the error channel to cause blocking
	// The processor should timeout after 100 milliseconds and continue

	// Wait for the processor to finish
	wg.Wait()

	// Check that the output channel is empty
	select {
	case outputData := <-outputCh:
		t.Errorf("Expected no output data, got: %v", outputData)
	default:
		// No output data, as expected
	}
}

func TestStage_ProcessorReturnsDeadlineExceeded(t *testing.T) {
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

	// Create a processor function that returns context.DeadlineExceeded
	processorFunc := func(ctx context.Context, data []byte) ([]byte, error) {
		return nil, context.DeadlineExceeded
	}

	// Create a simple stats update function
	updateStats := func(n uint64) {
		pipelineStats.UpdateOutputBytes(n)
	}

	// Start the processor stage in a goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		Stage(ctx, 0, "test", logger, inputCh, outputCh, errCh, pipelineStats, cancel, processorFunc, updateStats)
	}()

	// Send test data
	inputCh <- []byte{1, 2, 3, 4, 5}

	// Wait for the processor to finish
	wg.Wait()

	// Check that there is an error
	select {
	case err := <-errCh:
		// Check that the error contains information about the timeout
		if err == nil {
			t.Error("Expected non-nil error")
		} else {
			errMsg := err.Error()
			if errMsg == "" {
				t.Error("Expected non-empty error message")
			}
			if !containsAny(errMsg, "timed out", "timeout", "deadline") {
				t.Errorf("Expected error message to mention timeout or deadline, got: %s", errMsg)
			}
		}
	default:
		t.Error("Expected an error, got none")
	}

	// Check that the output channel is empty
	select {
	case outputData := <-outputCh:
		t.Errorf("Expected no output data, got: %v", outputData)
	default:
		// No output data, as expected
	}
}

func TestStage_ProcessorReturnsContextCanceled(t *testing.T) {
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

	// Create a processor function that returns context.Canceled
	processorFunc := func(ctx context.Context, data []byte) ([]byte, error) {
		return nil, context.Canceled
	}

	// Create a simple stats update function
	updateStats := func(n uint64) {
		pipelineStats.UpdateOutputBytes(n)
	}

	// Start the processor stage in a goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		Stage(ctx, 0, "test", logger, inputCh, outputCh, errCh, pipelineStats, cancel, processorFunc, updateStats)
	}()

	// Send test data
	inputCh <- []byte{1, 2, 3, 4, 5}

	// Wait for the processor to finish
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

func TestStage_ContextDeadlineExceeded(t *testing.T) {
	// Skip this test in short mode as it involves waiting
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	// Create a logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()

	// Create a context with a short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Create channels
	inputCh := make(chan []byte, 10)
	outputCh := make(chan []byte, 10)
	errCh := make(chan error, 10)

	// Create stats
	pipelineStats := stats.NewStats()

	// Create a processor function that takes longer than the context timeout
	processorFunc := func(ctx context.Context, data []byte) ([]byte, error) {
		// Sleep longer than the context timeout
		time.Sleep(200 * time.Millisecond)

		// Check if context is done
		select {
		case <-ctx.Done():
			return nil, ctx.Err() // This should be context.DeadlineExceeded
		default:
			return data, nil
		}
	}

	// Create a simple stats update function
	updateStats := func(n uint64) {
		pipelineStats.UpdateOutputBytes(n)
	}

	// Start the processor stage in a goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		Stage(ctx, 0, "test", logger, inputCh, outputCh, errCh, pipelineStats, cancel, processorFunc, updateStats)
	}()

	// Send test data
	inputCh <- []byte{1, 2, 3, 4, 5}

	// Wait for the processor to finish (it should exit due to the timeout)
	wg.Wait()

	// Check that there is an error
	select {
	case err := <-errCh:
		// Check that the error contains information about the timeout
		if err == nil {
			t.Error("Expected non-nil error")
		} else {
			errMsg := err.Error()
			if errMsg == "" {
				t.Error("Expected non-empty error message")
			}
			if !containsAny(errMsg, "timed out", "timeout", "deadline") {
				t.Errorf("Expected error message to mention timeout or deadline, got: %s", errMsg)
			}
		}
	default:
		t.Error("Expected an error, got none")
	}

	// Check that the output channel is empty
	select {
	case outputData := <-outputCh:
		t.Errorf("Expected no output data, got: %v", outputData)
	default:
		// No output data, as expected
	}
}
