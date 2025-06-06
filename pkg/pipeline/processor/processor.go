// Copyright (c) 2025 A Bit of Help, Inc.

// Package processor provides a generic processor stage for the processing pipeline.
package processor

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"

	customErrors "github.com/abitofhelp/multistage_pipeline_fanout/pkg/errors"
	"github.com/abitofhelp/multistage_pipeline_fanout/pkg/stats"
	"go.uber.org/zap"
)

// ProcessorFunc is a function that processes data with context awareness
type ProcessorFunc func(ctx context.Context, data []byte) ([]byte, error)

// Stage processes data chunks and sends them to the next stage
func Stage(
	ctx context.Context,
	id int,
	name string,
	logger *zap.Logger,
	inputCh <-chan []byte,
	outputCh chan<- []byte,
	errCh chan<- error,
	pipelineStats *stats.Stats,
	cancelPipeline context.CancelFunc,
	processorFunc ProcessorFunc,
	updateStats func(uint64),
) {
	// Recover from panics in the main goroutine with improved error handling
	defer func() {
		if r := recover(); r != nil {
			stack := debug.Stack()

			// Create a custom error with the panic information
			baseErr := fmt.Errorf("%w: %v", customErrors.ErrPanic, r)
			panicErr := customErrors.NewPipelineError(
				baseErr,
				name,
				id,
				"process_data",
				0, // We don't know the data size here
				"")

			logger.Error("Panic in processor stage",
				zap.Int(fmt.Sprintf("%s_id", name), id),
				zap.String("stack", string(stack)),
				zap.Any("panic_value", r),
				zap.Error(panicErr))

			// Try to send error to error channel with improved timeout handling
			select {
			case errCh <- panicErr:
				logger.Debug("Panic error sent to error channel",
					zap.Int(fmt.Sprintf("%s_id", name), id))
			case <-time.After(200 * time.Millisecond): // Increased timeout
				logger.Error("Error channel full or blocked, couldn't report panic",
					zap.Int(fmt.Sprintf("%s_id", name), id),
					zap.Error(panicErr))
			}

			// Cancel the pipeline
			cancelPipeline()
		}

		logger.Debug(fmt.Sprintf("%s goroutine completed", name), zap.Int(fmt.Sprintf("%s_id", name), id))
	}()

	for data := range inputCh {
		// Check if context is canceled with improved error handling
		select {
		case <-ctx.Done():
			err := ctx.Err()
			if customErrors.IsTimeoutError(err) {
				logger.Warn(fmt.Sprintf("%s timed out", name),
					zap.Int(fmt.Sprintf("%s_id", name), id),
					zap.Error(err))

				// We don't send this to errCh because we're returning immediately
				// and the pipeline will be canceled by the parent context
			} else if customErrors.IsCancellationError(err) {
				logger.Debug(fmt.Sprintf("%s canceled by context", name),
					zap.Int(fmt.Sprintf("%s_id", name), id),
					zap.Error(err))
			} else {
				logger.Warn(fmt.Sprintf("%s stopped by unknown context error", name),
					zap.Int(fmt.Sprintf("%s_id", name), id),
					zap.Error(err))
			}
			return
		default:
		}

		// Process data with context awareness and improved error handling
		processedData, err := processorFunc(ctx, data)
		if err != nil {
			// Check for context cancellation first
			if customErrors.IsCancellationError(err) {
				logger.Debug(fmt.Sprintf("%s canceled", name),
					zap.Int(fmt.Sprintf("%s_id", name), id),
					zap.Error(err))
				return
			}

			// Create a custom error with detailed context
			var processorErr error

			if customErrors.IsTimeoutError(err) {
				logger.Warn(fmt.Sprintf("%s timed out", name),
					zap.Int(fmt.Sprintf("%s_id", name), id),
					zap.Error(err))

				baseErr := fmt.Errorf("%w: %v", customErrors.ErrTimeout, err)
				processorErr = customErrors.NewPipelineError(
					baseErr,
					name,
					id,
					"process_data",
					len(data),
					"")
			} else {
				// Generic processing error
				processorErr = customErrors.NewPipelineError(
					err,
					name,
					id,
					"process_data",
					len(data),
					"")
			}

			logger.Error(fmt.Sprintf("%s error", name),
				zap.Int(fmt.Sprintf("%s_id", name), id),
				zap.Int("data_size", len(data)),
				zap.Error(processorErr))

			// Try to send error with improved timeout handling
			select {
			case errCh <- processorErr:
				logger.Debug("Error sent to error channel",
					zap.Int(fmt.Sprintf("%s_id", name), id))
			case <-time.After(200 * time.Millisecond): // Increased timeout
				logger.Warn(fmt.Sprintf("Error channel full or blocked, dropping %s error", name),
					zap.Int(fmt.Sprintf("%s_id", name), id),
					zap.Error(processorErr))
			}

			cancelPipeline()
			return
		}

		logger.Debug(fmt.Sprintf("Processed data chunk (%s)", name),
			zap.Int(fmt.Sprintf("%s_id", name), id),
			zap.Int("input_size", len(data)),
			zap.Int("output_size", len(processedData)))

		// Update processed data stats
		updateStats(uint64(len(processedData)))

		// Send to next stage with context awareness and improved timeout handling
		select {
		case outputCh <- processedData:
			// Data sent successfully
		case <-ctx.Done():
			err := ctx.Err()
			if customErrors.IsTimeoutError(err) {
				logger.Warn(fmt.Sprintf("Sending to next stage timed out (%s)", name),
					zap.Int(fmt.Sprintf("%s_id", name), id),
					zap.Error(err))
			} else if customErrors.IsCancellationError(err) {
				logger.Debug(fmt.Sprintf("Sending to next stage canceled by context (%s)", name),
					zap.Int(fmt.Sprintf("%s_id", name), id),
					zap.Error(err))
			} else {
				logger.Warn(fmt.Sprintf("Sending to next stage stopped by unknown context error (%s)", name),
					zap.Int(fmt.Sprintf("%s_id", name), id),
					zap.Error(err))
			}
			return
		case <-time.After(5 * time.Second): // Timeout for sending to next stage
			// Create a custom error for blocked pipeline
			baseErr := fmt.Errorf("%w: timeout after waiting 5 seconds to send to next stage",
				customErrors.ErrPipelineBlocked)

			blockedErr := customErrors.NewPipelineError(
				baseErr,
				name,
				id,
				"send_to_next_stage",
				len(processedData),
				"")

			logger.Error("Pipeline stage blocked",
				zap.Int(fmt.Sprintf("%s_id", name), id),
				zap.Error(blockedErr))

			// Try to send error with improved timeout handling
			select {
			case errCh <- blockedErr:
				logger.Debug("Blocked pipeline error sent to error channel",
					zap.Int(fmt.Sprintf("%s_id", name), id))
			case <-time.After(200 * time.Millisecond): // Increased timeout
				logger.Warn("Error channel full, couldn't report blocked pipeline",
					zap.Int(fmt.Sprintf("%s_id", name), id),
					zap.Error(blockedErr))
			}

			cancelPipeline()
			return
		}
	}
}
