// Copyright (c) 2025 A Bit of Help, Inc.

// Package compression provides data compression functionality.
//
// This package implements the core compression algorithms and utilities that can be used
// independently of the pipeline. It is context-aware but not pipeline-specific.
//
// The corresponding package in the pipeline hierarchy is pkg/pipeline/compressor,
// which integrates this core functionality into the pipeline architecture.
package compression

import (
	"bytes"
	"context"
	"fmt"

	"github.com/abitofhelp/multistage_pipeline_fanout/pkg/dataprocessor"
	"github.com/andybalholm/brotli"
)

// compressData compresses data using Brotli compression
func compressData(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	compressor := brotli.NewWriter(&buf)

	if _, err := compressor.Write(data); err != nil {
		return nil, fmt.Errorf("failed to compress data: %w", err)
	}

	if err := compressor.Close(); err != nil {
		return nil, fmt.Errorf("failed to finalize compression: %w", err)
	}

	return buf.Bytes(), nil
}

// CompressDataWithContext compresses data with context awareness
func CompressDataWithContext(ctx context.Context, data []byte) ([]byte, error) {
	return dataprocessor.ProcessWithContext(ctx, compressData, data)
}
