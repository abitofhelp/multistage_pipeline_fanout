// Copyright (c) 2025 A Bit of Help, Inc.

// Package encryptor provides the encryptor stage for the processing pipeline.
//
// This package integrates the core encryption functionality from pkg/encryption
// into the pipeline architecture. It handles pipeline-specific concerns like
// channel communication, concurrency, error handling, and statistics.
//
// While the pkg/encryption package implements the core encryption algorithms,
// this package adapts that functionality to work within the pipeline framework.
package encryptor

import (
	"context"

	"github.com/abitofhelp/multistage_pipeline_fanout/pkg/encryption"
	"github.com/abitofhelp/multistage_pipeline_fanout/pkg/pipeline/processor"
	"github.com/abitofhelp/multistage_pipeline_fanout/pkg/stats"
	"github.com/google/tink/go/tink"
	"go.uber.org/zap"
)

// Stage encrypts data chunks and sends them to the next stage
func Stage(
	ctx context.Context,
	id int,
	aead tink.AEAD,
	compressCh <-chan []byte,
	encryptCh chan<- []byte,
	errCh chan<- error,
	pipelineStats *stats.Stats,
	cancelPipeline context.CancelFunc,
	logger *zap.Logger,
) {
	// Create a closure that captures the AEAD primitive
	encryptFunc := func(ctx context.Context, data []byte) ([]byte, error) {
		return encryption.EncryptDataWithContext(ctx, aead, data)
	}

	processor.Stage(
		ctx,
		id,
		"encryptor",
		logger,
		compressCh,
		encryptCh,
		errCh,
		pipelineStats,
		cancelPipeline,
		encryptFunc,
		pipelineStats.UpdateEncryptedBytes,
	)
}
