// Copyright (c) 2025 A Bit of Help, Inc.

// Package errors provides custom error types and error handling utilities for the application.
package errors

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"
)

// Standard errors that can be used for comparison with errors.Is
var (
	// ErrTimeout indicates an operation timed out
	ErrTimeout = errors.New("operation timed out")

	// ErrCanceled indicates an operation was canceled
	ErrCanceled = errors.New("operation canceled")

	// ErrIOFailure indicates an I/O operation failed
	ErrIOFailure = errors.New("I/O operation failed")

	// ErrPipelineBlocked indicates a pipeline stage is blocked
	ErrPipelineBlocked = errors.New("pipeline stage blocked")

	// ErrPanic indicates a panic occurred
	ErrPanic = errors.New("panic occurred")
)

// PipelineError represents an error that occurred in the pipeline
type PipelineError struct {
	// Err is the underlying error
	Err error

	// Stage is the pipeline stage where the error occurred
	Stage string

	// StageID is the ID of the pipeline stage instance
	StageID int

	// Operation is the operation being performed
	Operation string

	// Time is when the error occurred
	Time time.Time

	// DataSize is the size of the data being processed
	DataSize int

	// FilePath is the path of the file being processed
	FilePath string
}

// Error implements the error interface
func (e *PipelineError) Error() string {
	return fmt.Sprintf("[%s] %s (stage=%s, id=%d, size=%d, file=%s): %v",
		e.Time.Format(time.RFC3339),
		e.Operation,
		e.Stage,
		e.StageID,
		e.DataSize,
		e.FilePath,
		e.Err)
}

// Unwrap returns the underlying error
func (e *PipelineError) Unwrap() error {
	return e.Err
}

// NewPipelineError creates a new PipelineError
func NewPipelineError(err error, stage string, stageID int, operation string, dataSize int, filePath string) *PipelineError {
	return &PipelineError{
		Err:       err,
		Stage:     stage,
		StageID:   stageID,
		Operation: operation,
		Time:      time.Now(),
		DataSize:  dataSize,
		FilePath:  filePath,
	}
}

// IsIOError checks if the error is an I/O error
func IsIOError(err error) bool {
	var pathErr *os.PathError
	return errors.Is(err, ErrIOFailure) || errors.As(err, &pathErr)
}

// IsTimeoutError checks if the error is a timeout error
func IsTimeoutError(err error) bool {
	return errors.Is(err, ErrTimeout) || errors.Is(err, context.DeadlineExceeded)
}

// IsCancellationError checks if the error is a cancellation error
func IsCancellationError(err error) bool {
	return errors.Is(err, ErrCanceled) || errors.Is(err, context.Canceled)
}

// ErrorCollector collects multiple errors
type ErrorCollector struct {
	errors []error
}

// NewErrorCollector creates a new ErrorCollector
func NewErrorCollector() *ErrorCollector {
	return &ErrorCollector{
		errors: make([]error, 0),
	}
}

// Add adds an error to the collector
func (c *ErrorCollector) Add(err error) {
	if err != nil {
		c.errors = append(c.errors, err)
	}
}

// HasErrors returns true if the collector has any errors
func (c *ErrorCollector) HasErrors() bool {
	return len(c.errors) > 0
}

// Error implements the error interface
func (c *ErrorCollector) Error() string {
	if len(c.errors) == 0 {
		return "no errors"
	}

	if len(c.errors) == 1 {
		return c.errors[0].Error()
	}

	msg := fmt.Sprintf("%d errors occurred:\n", len(c.errors))
	for i, err := range c.errors {
		msg += fmt.Sprintf("  %d: %v\n", i+1, err)
	}
	return msg
}

// Errors returns all collected errors
func (c *ErrorCollector) Errors() []error {
	return c.errors
}
