// Copyright (c) 2025 A Bit of Help, Inc.

// Package pipeline provides the main processing pipeline for the application.
// It implements a multi-stage processing pipeline with concurrent execution
// of compression and encryption operations for improved performance.
//
// The pipeline architecture is designed to separate core functionality from pipeline integration:
//
//  1. Core packages in /pkg (like compression, encryption) implement fundamental algorithms
//     that can be used independently of the pipeline.
//
//  2. Pipeline packages in /pkg/pipeline (like compressor, encryptor) integrate these core
//     functionalities into the pipeline architecture, handling pipeline-specific concerns
//     like channel communication, concurrency, and error handling.
//
// This separation allows for better code organization, reusability, independent testing,
// and clearer separation of concerns.
package pipeline

import (
	"context"
	"crypto/sha256"
	"fmt"
	"hash"
	"os"
	"sync"
	"time"

	"github.com/abitofhelp/multistage_pipeline_fanout/pkg/encryption"
	customErrors "github.com/abitofhelp/multistage_pipeline_fanout/pkg/errors"
	"github.com/abitofhelp/multistage_pipeline_fanout/pkg/pipeline/compressor"
	"github.com/abitofhelp/multistage_pipeline_fanout/pkg/pipeline/encryptor"
	"github.com/abitofhelp/multistage_pipeline_fanout/pkg/pipeline/options"
	"github.com/abitofhelp/multistage_pipeline_fanout/pkg/pipeline/reader"
	"github.com/abitofhelp/multistage_pipeline_fanout/pkg/pipeline/writer"
	"github.com/abitofhelp/multistage_pipeline_fanout/pkg/stats"
	"github.com/google/tink/go/tink"
	"go.uber.org/zap"
)

// ProcessFile processes the input file through the pipeline stages.
func ProcessFile(ctx context.Context, logger *zap.Logger, inputPath, outputPath string) (*stats.Stats, error) {
	if err := validateInputs(ctx, logger, inputPath, outputPath); err != nil {
		return nil, err
	}

	if err := checkContext(ctx); err != nil {
		logContextError(logger, err, inputPath, outputPath)
		return nil, wrapPipelineError(err, "check_context", inputPath)
	}

	startTime := time.Now()
	pipelineStats := stats.NewStats()

	files, err := setupFiles(logger, inputPath, outputPath)
	if err != nil {
		return nil, err
	}
	defer closeFiles(logger, files)

	aead, err := initializeEncryption(logger)
	if err != nil {
		return nil, err
	}

	hashers := setupHashers()
	opts := options.DefaultPipelineOptions()
	channels := setupChannels(opts)

	pipelineCtx, cancelPipeline := context.WithCancel(ctx)
	defer cancelPipeline()

	errorCollector := customErrors.NewErrorCollector()

	if err := runPipeline(pipelineCtx, pipelineConfig{
		logger:         logger,
		files:          files,
		channels:       channels,
		opts:           opts,
		aead:           aead,
		pipelineStats:  pipelineStats,
		hashers:        hashers,
		cancelPipeline: cancelPipeline,
	}); err != nil {
		return nil, err
	}

	if err := collectErrors(channels.errCh, errorCollector); err != nil {
		logPipelineErrors(logger, err, inputPath, outputPath, startTime)
		return nil, wrapPipelineError(errorCollector, "process_file", inputPath)
	}

	if err := checkFinalContext(ctx, startTime, logger, inputPath, outputPath, pipelineStats); err != nil {
		return nil, err
	}

	finalizeStats(pipelineStats, hashers, startTime)
	return pipelineStats, nil
}

type fileHandlers struct {
	input  *os.File
	output *os.File
}

type pipelineChannels struct {
	readCh     chan []byte
	compressCh chan []byte
	encryptCh  chan []byte
	errCh      chan error
}

type pipelineHashers struct {
	input  hash.Hash
	output hash.Hash
}

type pipelineConfig struct {
	logger         *zap.Logger
	files          *fileHandlers
	channels       *pipelineChannels
	opts           *options.PipelineOptions
	aead           tink.AEAD
	pipelineStats  *stats.Stats
	hashers        *pipelineHashers
	cancelPipeline context.CancelFunc
}

func validateInputs(ctx context.Context, logger *zap.Logger, inputPath, outputPath string) error {
	if ctx == nil {
		return customErrors.NewPipelineError(fmt.Errorf("context cannot be nil"), "pipeline", 0, "validate_inputs", 0, "")
	}
	if logger == nil {
		return customErrors.NewPipelineError(fmt.Errorf("logger cannot be nil"), "pipeline", 0, "validate_inputs", 0, "")
	}
	if inputPath == "" {
		return customErrors.NewPipelineError(fmt.Errorf("input path cannot be empty"), "pipeline", 0, "validate_inputs", 0, "")
	}
	if outputPath == "" {
		return customErrors.NewPipelineError(fmt.Errorf("output path cannot be empty"), "pipeline", 0, "validate_inputs", 0, "")
	}
	return nil
}

func checkContext(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

func setupFiles(logger *zap.Logger, inputPath, outputPath string) (*fileHandlers, error) {
	input, err := os.Open(inputPath)
	if err != nil {
		return nil, wrapIOError(err, "open_input_file", inputPath)
	}

	output, err := os.Create(outputPath)
	if err != nil {
		input.Close()
		return nil, wrapIOError(err, "create_output_file", outputPath)
	}

	return &fileHandlers{input: input, output: output}, nil
}

func closeFiles(logger *zap.Logger, files *fileHandlers) {
	if err := files.input.Close(); err != nil {
		logger.Warn("Failed to close input file", zap.Error(err))
	}
	if err := files.output.Close(); err != nil {
		logger.Warn("Failed to close output file", zap.Error(err))
	}
}

func initializeEncryption(logger *zap.Logger) (tink.AEAD, error) {
	aead, err := encryption.InitEncryption(logger)
	if err != nil {
		return nil, wrapPipelineError(err, "init_encryption", "")
	}
	return aead, nil
}

func setupHashers() *pipelineHashers {
	return &pipelineHashers{
		input:  sha256.New(),
		output: sha256.New(),
	}
}

func setupChannels(opts *options.PipelineOptions) *pipelineChannels {
	return &pipelineChannels{
		readCh:     make(chan []byte, opts.ChannelBufferSize),
		compressCh: make(chan []byte, opts.ChannelBufferSize),
		encryptCh:  make(chan []byte, opts.ChannelBufferSize),
		errCh:      make(chan error, options.ErrorChannelBufferSize),
	}
}

func runPipeline(ctx context.Context, config pipelineConfig) error {
	var pipelineWg sync.WaitGroup

	// Start reader
	pipelineWg.Add(1)
	go func() {
		defer pipelineWg.Done()
		reader.Stage(ctx, config.logger, config.files.input, config.channels.readCh,
			config.channels.errCh, config.pipelineStats, config.hashers.input, config.opts.ChunkSize)
	}()

	// Start compressors
	startCompressors(ctx, &pipelineWg, config)

	// Close compressCh when all compressors are done
	go func() {
		pipelineWg.Wait()
		close(config.channels.compressCh)
	}()

	// Start encryptors
	var encWg sync.WaitGroup
	startEncryptors(ctx, &encWg, config)

	// Close encryptCh when all encryptors are done
	go func() {
		encWg.Wait()
		close(config.channels.encryptCh)
	}()

	// Start writer
	var writeWg sync.WaitGroup
	startWriter(ctx, &writeWg, config)

	writeWg.Wait()
	close(config.channels.errCh)

	return nil
}

// logContextError logs context-related errors
func logContextError(logger *zap.Logger, err error, inputPath, outputPath string) {
	logger.Error("Pipeline context error",
		zap.Error(err),
		zap.String("input_file", inputPath),
		zap.String("output_file", outputPath))
}

// collectErrors collects errors from the error channel
func collectErrors(errCh chan error, errorCollector *customErrors.ErrorCollector) error {
	for err := range errCh {
		if err != nil {
			errorCollector.Add(err)
		}
	}
	if errorCollector.HasErrors() {
		return errorCollector
	}
	return nil
}

// logPipelineErrors logs pipeline errors
func logPipelineErrors(logger *zap.Logger, err error, inputPath, outputPath string, startTime time.Time) {
	duration := time.Since(startTime)
	logger.Error("Pipeline processing failed",
		zap.Error(err),
		zap.String("input_file", inputPath),
		zap.String("output_file", outputPath),
		zap.Duration("duration", duration))
}

// checkFinalContext checks if the context is still valid at the end of processing
func checkFinalContext(ctx context.Context, startTime time.Time, logger *zap.Logger, inputPath, outputPath string, pipelineStats *stats.Stats) error {
	if err := checkContext(ctx); err != nil {
		duration := time.Since(startTime)
		logger.Error("Pipeline context error at finalization",
			zap.Error(err),
			zap.String("input_file", inputPath),
			zap.String("output_file", outputPath),
			zap.Duration("duration", duration))
		return wrapPipelineError(err, "finalize", inputPath)
	}
	return nil
}

// finalizeStats finalizes the pipeline statistics
func finalizeStats(pipelineStats *stats.Stats, hashers *pipelineHashers, startTime time.Time) {
	pipelineStats.InputHash = hashers.input.Sum(nil)
	pipelineStats.OutputHash = hashers.output.Sum(nil)
	pipelineStats.ProcessingTime = time.Since(startTime)
}

// startCompressors starts the compressor workers
func startCompressors(ctx context.Context, wg *sync.WaitGroup, config pipelineConfig) {
	for i := 0; i < config.opts.CompressorCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			compressor.Stage(ctx, id, config.logger, config.channels.readCh, config.channels.compressCh,
				config.channels.errCh, config.pipelineStats, config.cancelPipeline)
		}(i)
	}
}

// startEncryptors starts the encryptor workers
func startEncryptors(ctx context.Context, wg *sync.WaitGroup, config pipelineConfig) {
	for i := 0; i < config.opts.EncryptorCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			encryptor.Stage(ctx, id, config.aead, config.channels.compressCh, config.channels.encryptCh,
				config.channels.errCh, config.pipelineStats, config.cancelPipeline, config.logger)
		}(i)
	}
}

// startWriter starts the writer worker
func startWriter(ctx context.Context, wg *sync.WaitGroup, config pipelineConfig) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		writer.Stage(ctx, config.logger, config.files.output, config.channels.encryptCh,
			config.channels.errCh, config.pipelineStats, config.hashers.output, config.cancelPipeline)
	}()
}

func wrapPipelineError(err error, operation, path string) error {
	return customErrors.NewPipelineError(err, "pipeline", 0, operation, 0, path)
}

func wrapIOError(err error, operation, path string) error {
	return customErrors.NewPipelineError(
		fmt.Errorf("%w: %v", customErrors.ErrIOFailure, err),
		"pipeline",
		0,
		operation,
		0,
		path)
}

// ProcessFile processes the input file through the pipeline stages.
// It reads the input file, compresses and encrypts the data in parallel,
// and writes the result to the output file.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - logger: Logger for recording pipeline operations
//   - inputPath: Path to the input file
//   - outputPath: Path to the output file
//
// Returns:
//   - *stats.Stats: Statistics about the processing operation
//   - error: Any error that occurred during processing
/*func ProcessFile(ctx context.Context, logger *zap.Logger, inputPath, outputPath string) (*stats.Stats, error) {
	// Validate inputs with improved error context
	if ctx == nil {
		err := customErrors.NewPipelineError(
			fmt.Errorf("context cannot be nil"),
			"pipeline",
			0,
			"validate_inputs",
			0,
			"")
		return nil, err
	}
	if logger == nil {
		err := customErrors.NewPipelineError(
			fmt.Errorf("logger cannot be nil"),
			"pipeline",
			0,
			"validate_inputs",
			0,
			"")
		return nil, err
	}
	if inputPath == "" {
		err := customErrors.NewPipelineError(
			fmt.Errorf("input path cannot be empty"),
			"pipeline",
			0,
			"validate_inputs",
			0,
			"")
		return nil, err
	}
	if outputPath == "" {
		err := customErrors.NewPipelineError(
			fmt.Errorf("output path cannot be empty"),
			"pipeline",
			0,
			"validate_inputs",
			0,
			"")
		return nil, err
	}

	// Check if context is already done
	select {
	case <-ctx.Done():
		err := ctx.Err()
		if customErrors.IsTimeoutError(err) {
			logger.Error("Pipeline timed out before starting",
				zap.Error(err),
				zap.String("input_file", inputPath),
				zap.String("output_file", outputPath))

			timeoutErr := customErrors.NewPipelineError(
				err,
				"pipeline",
				0,
				"check_context",
				0,
				inputPath)

			return nil, timeoutErr
		} else if customErrors.IsCancellationError(err) {
			logger.Warn("Pipeline canceled by parent context before starting",
				zap.Error(err),
				zap.String("input_file", inputPath),
				zap.String("output_file", outputPath))

			cancelErr := customErrors.NewPipelineError(
				err,
				"pipeline",
				0,
				"check_context",
				0,
				inputPath)

			return nil, cancelErr
		} else {
			// Unknown context error
			return nil, customErrors.NewPipelineError(
				err,
				"pipeline",
				0,
				"check_context",
				0,
				inputPath)
		}
	default:
		// Context is not done, continue
	}

	// Record start time
	startTime := time.Now()

	// Open input file with improved error handling
	inputFile, err := os.Open(inputPath)
	if err != nil {
		ioErr := customErrors.NewPipelineError(
			fmt.Errorf("%w: %v", customErrors.ErrIOFailure, err),
			"pipeline",
			0,
			"open_input_file",
			0,
			inputPath)
		logger.Error("Failed to open input file",
			zap.Error(err),
			zap.String("path", inputPath))
		return nil, ioErr
	}
	defer func() {
		if err := inputFile.Close(); err != nil {
			logger.Warn("Failed to close input file",
				zap.Error(err),
				zap.String("path", inputPath))
		}
	}()

	// Create output file with improved error handling
	outputFile, err := os.Create(outputPath)
	if err != nil {
		ioErr := customErrors.NewPipelineError(
			fmt.Errorf("%w: %v", customErrors.ErrIOFailure, err),
			"pipeline",
			0,
			"create_output_file",
			0,
			outputPath)
		logger.Error("Failed to create output file",
			zap.Error(err),
			zap.String("path", outputPath))
		return nil, ioErr
	}
	defer func() {
		if err := outputFile.Close(); err != nil {
			logger.Warn("Failed to close output file",
				zap.Error(err),
				zap.String("path", outputPath))
		}
	}()

	// Initialize encryption with improved error handling
	aead, err := encryption.InitEncryption(logger)
	if err != nil {
		encryptErr := customErrors.NewPipelineError(
			err,
			"pipeline",
			0,
			"init_encryption",
			0,
			inputPath)
		logger.Error("Failed to initialize encryption", zap.Error(err))
		return nil, encryptErr
	}

	// Initialize stats and checksums
	pipelineStats := stats.NewStats()
	inputHasher := sha256.New()
	outputHasher := sha256.New()

	// Get default pipeline options
	opts := options.DefaultPipelineOptions()

	// Create buffered channels for the pipeline to prevent goroutine leaks
	readCh := make(chan []byte, opts.ChannelBufferSize)
	compressCh := make(chan []byte, opts.ChannelBufferSize)
	encryptCh := make(chan []byte, opts.ChannelBufferSize)

	// Create error channel with buffer to prevent goroutine leaks
	errCh := make(chan error, options.ErrorChannelBufferSize)

	// Create a context that can be canceled on error or parent context cancellation
	// We don't need to set up signal handling here as it's already done in main.go
	pipelineCtx, cancelPipeline := context.WithCancel(ctx)
	defer cancelPipeline()

	// Create an error collector to aggregate errors
	errorCollector := customErrors.NewErrorCollector()

	// Create a WaitGroup for pipeline synchronization
	var pipelineWg sync.WaitGroup

	// Start the reader goroutine
	pipelineWg.Add(1)
	go func() {
		defer pipelineWg.Done()
		reader.Stage(pipelineCtx, logger, inputFile, readCh, errCh, pipelineStats, inputHasher, opts.ChunkSize)
	}()

	// Start multiple compressor goroutines (fan-out)
	for i := 0; i < opts.CompressorCount; i++ {
		pipelineWg.Add(1)
		go func(id int) {
			defer pipelineWg.Done()
			compressor.Stage(pipelineCtx, id, logger, readCh, compressCh, errCh, pipelineStats, cancelPipeline)
		}(i)
	}

	// Start a goroutine to close compressCh when all readers and compressors are done
	go func() {
		// Wait for all compressors to finish
		pipelineWg.Wait()
		close(compressCh)
		logger.Debug("Compression stage completed, closed compress channel")
	}()

	// Start multiple encryptor goroutines (fan-out)
	var encWg sync.WaitGroup
	for i := 0; i < opts.EncryptorCount; i++ {
		encWg.Add(1)
		go func(id int) {
			defer encWg.Done()
			encryptor.Stage(pipelineCtx, id, aead, compressCh, encryptCh, errCh, pipelineStats, cancelPipeline, logger)
		}(i)
	}

	// Start a goroutine to close encryptCh when all encryptors are done
	go func() {
		encWg.Wait()
		close(encryptCh)
		logger.Debug("Encryption stage completed, closed encrypt channel")
	}()

	// Start the writer goroutine
	var writeWg sync.WaitGroup
	writeWg.Add(1)
	go func() {
		defer writeWg.Done()
		writer.Stage(pipelineCtx, logger, outputFile, encryptCh, errCh, pipelineStats, outputHasher, cancelPipeline)
	}()

	// Wait for the writer to finish
	writeWg.Wait()
	logger.Debug("Writing stage completed")

	// Close the error channel to prevent leaks
	close(errCh)

	// Collect all errors from the error channel
collectErrors:
	for {
		select {
		case err, ok := <-errCh:
			if !ok {
				// Channel closed, no more errors
				break collectErrors
			}
			// Add error to collector
			errorCollector.Add(err)
		default:
			// No more errors available right now
			break collectErrors
		}
	}

	// Check if there was an error or context cancellation
	if errorCollector.HasErrors() {
		// Log all collected errors
		logger.Error("Pipeline errors detected",
			zap.String("errors", errorCollector.Error()),
			zap.String("input_file", inputPath),
			zap.String("output_file", outputPath),
			zap.Duration("elapsed_time", time.Since(startTime)))

		// Return the aggregated errors
		return nil, customErrors.NewPipelineError(
			errorCollector,
			"pipeline",
			0,
			"process_file",
			int(pipelineStats.InputBytes.Load()),
			inputPath)
	}

	// Check for context cancellation
	select {
	case <-ctx.Done():
		err := ctx.Err()
		if customErrors.IsTimeoutError(err) {
			logger.Error("Pipeline timed out",
				zap.Error(err),
				zap.String("input_file", inputPath),
				zap.String("output_file", outputPath),
				zap.Duration("elapsed_time", time.Since(startTime)))

			timeoutErr := customErrors.NewPipelineError(
				err,
				"pipeline",
				0,
				"process_file",
				int(pipelineStats.InputBytes.Load()),
				inputPath)

			return nil, timeoutErr
		} else if customErrors.IsCancellationError(err) {
			logger.Warn("Pipeline canceled by parent context",
				zap.Error(err),
				zap.String("input_file", inputPath),
				zap.String("output_file", outputPath),
				zap.Duration("elapsed_time", time.Since(startTime)))

			cancelErr := customErrors.NewPipelineError(
				err,
				"pipeline",
				0,
				"process_file",
				int(pipelineStats.InputBytes.Load()),
				inputPath)

			return nil, cancelErr
		}
		return nil, err
	default:
		// No context cancellation, continue
	}

	// Finalize checksums
	pipelineStats.InputHash = inputHasher.Sum(nil)
	pipelineStats.OutputHash = outputHasher.Sum(nil)

	// Calculate processing time
	pipelineStats.ProcessingTime = time.Since(startTime)

	return pipelineStats, nil
}*/
