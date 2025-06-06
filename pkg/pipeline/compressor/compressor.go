// Copyright (c) 2025 A Bit of Help, Inc.

// Package compressor provides the compressor stage for the processing pipeline.
//
// This package integrates the core compression functionality from pkg/compression
// into the pipeline architecture. It handles pipeline-specific concerns like
// channel communication, concurrency, error handling, and statistics.
//
// While the pkg/compression package implements the core compression algorithms,
// this package adapts that functionality to work within the pipeline framework.
package compressor

import (
	"context"

	"github.com/abitofhelp/multistage_pipeline_fanout/pkg/compression"
	"github.com/abitofhelp/multistage_pipeline_fanout/pkg/pipeline/processor"
	"github.com/abitofhelp/multistage_pipeline_fanout/pkg/stats"
	"go.uber.org/zap"
)

// Stage compresses data chunks and sends them to the next stage
func Stage(
	ctx context.Context,
	id int,
	logger *zap.Logger,
	readCh <-chan []byte,
	compressCh chan<- []byte,
	errCh chan<- error,
	pipelineStats *stats.Stats,
	cancelPipeline context.CancelFunc,
) {
	processor.Stage(
		ctx,
		id,
		"compressor",
		logger,
		readCh,
		compressCh,
		errCh,
		pipelineStats,
		cancelPipeline,
		compression.CompressDataWithContext,
		pipelineStats.UpdateCompressedBytes,
	)
}
