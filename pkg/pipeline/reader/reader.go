// Copyright (c) 2025 A Bit of Help, Inc.

// Package reader provides the reader stage for the processing pipeline.
package reader

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	customErrors "github.com/abitofhelp/multistage_pipeline_fanout/pkg/errors"
	"github.com/abitofhelp/multistage_pipeline_fanout/pkg/stats"
	"go.uber.org/zap"
)

// Stage reads data from the input file in chunks and sends them to the next stage
func Stage(
	ctx context.Context,
	logger *zap.Logger,
	inputFile *os.File,
	readCh chan<- []byte,
	errCh chan<- error,
	pipelineStats *stats.Stats,
	inputHasher io.Writer,
	chunkSize int,
) {
	defer func() {
		close(readCh)
		logger.Debug("Reader goroutine completed")
	}()

	buffer := make([]byte, chunkSize)
	for {
		// Check if context is canceled with improved error handling
		select {
		case <-ctx.Done():
			err := ctx.Err()
			if customErrors.IsTimeoutError(err) {
				logger.Warn("Reader timed out", zap.Error(err))
			} else if customErrors.IsCancellationError(err) {
				logger.Debug("Reader canceled by context", zap.Error(err))
			} else {
				logger.Warn("Reader stopped by unknown context error", zap.Error(err))
			}
			return
		default:
		}

		// Read chunk with timeout to ensure responsiveness to cancellation
		readDone := make(chan struct{})
		var n int
		var readErr error

		go func() {
			n, readErr = inputFile.Read(buffer)
			close(readDone)
		}()

		select {
		case <-readDone:
			// Read completed
		case <-ctx.Done():
			// Context canceled during read
			err := ctx.Err()
			if customErrors.IsTimeoutError(err) {
				logger.Warn("Read operation timed out", zap.Error(err))
			} else if customErrors.IsCancellationError(err) {
				logger.Debug("Read operation canceled by context", zap.Error(err))
			} else {
				logger.Warn("Read operation stopped by unknown context error", zap.Error(err))
			}
			return
		case <-time.After(2 * time.Second): // Timeout for read operation
			// Create a custom error for blocked read
			baseErr := fmt.Errorf("%w: timeout after waiting 2 seconds for read operation",
				customErrors.ErrTimeout)

			timeoutErr := customErrors.NewPipelineError(
				baseErr,
				"reader",
				0,
				"read_data",
				0,
				"")

			logger.Error("Read operation timed out", zap.Error(timeoutErr))

			// Try to send error with improved timeout handling
			select {
			case errCh <- timeoutErr:
				logger.Debug("Read timeout error sent to error channel")
			case <-time.After(200 * time.Millisecond): // Timeout for sending to error channel
				logger.Warn("Error channel full, couldn't report read timeout", zap.Error(timeoutErr))
			}
			return
		}

		if readErr != nil && readErr != io.EOF {
			// Create a custom error with detailed context
			baseErr := fmt.Errorf("%w: %v", customErrors.ErrIOFailure, readErr)
			ioErr := customErrors.NewPipelineError(
				baseErr,
				"reader",
				0,
				"read_data",
				0,
				"")

			logger.Error("Read error", zap.Error(ioErr))

			// Try to send error with improved timeout handling
			select {
			case errCh <- ioErr:
				logger.Debug("Read error sent to error channel")
			case <-time.After(200 * time.Millisecond): // Timeout for sending to error channel
				logger.Warn("Error channel full, dropping read error", zap.Error(ioErr))
			}
			return
		}

		if n == 0 {
			logger.Debug("End of input file reached")
			break
		}

		// Make a copy of the buffer to avoid data races
		chunk := make([]byte, n)
		copy(chunk, buffer[:n])

		// Update input stats
		pipelineStats.UpdateInputBytes(uint64(n))
		inputHasher.Write(chunk)

		// Increment chunks processed count
		pipelineStats.IncrementChunksProcessed()

		// Update final chunk size (will be overwritten until the last chunk)
		pipelineStats.SetFinalChunkSize(uint64(n))

		// Send to compression stage with context awareness and improved timeout handling
		select {
		case readCh <- chunk:
			logger.Debug("Sent chunk to compression stage",
				zap.Uint64("chunk_number", pipelineStats.ChunksProcessed.Load()),
				zap.Int("chunk_size", n))
		case <-ctx.Done():
			err := ctx.Err()
			if customErrors.IsTimeoutError(err) {
				logger.Warn("Sending to compression stage timed out", zap.Error(err))
			} else if customErrors.IsCancellationError(err) {
				logger.Debug("Sending to compression stage canceled by context", zap.Error(err))
			} else {
				logger.Warn("Sending to compression stage stopped by unknown context error", zap.Error(err))
			}
			return
		case <-time.After(2 * time.Second): // Timeout for sending to next stage
			// Create a custom error for blocked pipeline
			baseErr := fmt.Errorf("%w: timeout after waiting 2 seconds to send to compression stage",
				customErrors.ErrPipelineBlocked)

			blockedErr := customErrors.NewPipelineError(
				baseErr,
				"reader",
				0,
				"send_to_compression_stage",
				n,
				"")

			logger.Error("Pipeline stage blocked", zap.Error(blockedErr))

			// Try to send error with improved timeout handling
			select {
			case errCh <- blockedErr:
				logger.Debug("Blocked pipeline error sent to error channel")
			case <-time.After(200 * time.Millisecond): // Timeout for sending to error channel
				logger.Warn("Error channel full, couldn't report blocked pipeline", zap.Error(blockedErr))
			}
			return
		}
	}
}
