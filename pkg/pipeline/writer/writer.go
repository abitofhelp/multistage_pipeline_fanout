// Copyright (c) 2025 A Bit of Help, Inc.

// Package writer provides the writer stage for the processing pipeline.
package writer

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

// Stage writes encrypted data to the output file
func Stage(
	ctx context.Context,
	logger *zap.Logger,
	outputFile *os.File,
	encryptCh <-chan []byte,
	errCh chan<- error,
	pipelineStats *stats.Stats,
	outputHasher io.Writer,
	cancelPipeline context.CancelFunc,
) {
	defer func() {
		logger.Debug("Writer goroutine completed")
	}()

	for encryptedData := range encryptCh {
		// Check if context is canceled with improved error handling
		select {
		case <-ctx.Done():
			err := ctx.Err()
			if customErrors.IsTimeoutError(err) {
				logger.Warn("Writer timed out", zap.Error(err))
			} else if customErrors.IsCancellationError(err) {
				logger.Debug("Writer canceled by context", zap.Error(err))
			} else {
				logger.Warn("Writer stopped by unknown context error", zap.Error(err))
			}
			return
		default:
		}

		// Update output checksum
		outputHasher.Write(encryptedData)

		// Write to output file with context awareness
		writeDone := make(chan struct{})
		var written int
		var writeErr error

		go func() {
			written, writeErr = outputFile.Write(encryptedData)
			close(writeDone)
		}()

		select {
		case <-writeDone:
			// Write completed
		case <-ctx.Done():
			// Context canceled during write
			err := ctx.Err()
			if customErrors.IsTimeoutError(err) {
				logger.Warn("Write operation timed out", zap.Error(err))
			} else if customErrors.IsCancellationError(err) {
				logger.Debug("Write operation canceled by context", zap.Error(err))
			} else {
				logger.Warn("Write operation stopped by unknown context error", zap.Error(err))
			}
			return
		case <-time.After(5 * time.Second): // Timeout for write operation
			// Create a custom error for blocked write
			baseErr := fmt.Errorf("%w: timeout after waiting 5 seconds for write operation",
				customErrors.ErrTimeout)

			timeoutErr := customErrors.NewPipelineError(
				baseErr,
				"writer",
				0,
				"write_data",
				len(encryptedData),
				"")

			logger.Error("Write operation timed out", zap.Error(timeoutErr))

			// Try to send error with improved timeout handling
			select {
			case errCh <- timeoutErr:
				logger.Debug("Write timeout error sent to error channel")
			case <-time.After(200 * time.Millisecond): // Timeout for sending to error channel
				logger.Warn("Error channel full, couldn't report write timeout", zap.Error(timeoutErr))
			}
			cancelPipeline()
			return
		}

		if writeErr != nil {
			// Create a custom error with detailed context
			baseErr := fmt.Errorf("%w: %v", customErrors.ErrIOFailure, writeErr)
			ioErr := customErrors.NewPipelineError(
				baseErr,
				"writer",
				0,
				"write_data",
				len(encryptedData),
				"")

			logger.Error("Write error", zap.Error(ioErr))

			// Try to send error with improved timeout handling
			select {
			case errCh <- ioErr:
				logger.Debug("Write error sent to error channel")
			case <-time.After(200 * time.Millisecond): // Timeout for sending to error channel
				logger.Warn("Error channel full, dropping write error", zap.Error(ioErr))
			}
			cancelPipeline()
			return
		}

		pipelineStats.UpdateOutputBytes(uint64(written))
	}
}
