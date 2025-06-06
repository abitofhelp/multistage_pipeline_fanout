// Copyright (c) 2025 A Bit of Help, Inc.

package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	customErrors "github.com/abitofhelp/multistage_pipeline_fanout/pkg/errors"
	"github.com/abitofhelp/multistage_pipeline_fanout/pkg/logger"
	"github.com/abitofhelp/multistage_pipeline_fanout/pkg/pipeline"
	"github.com/abitofhelp/multistage_pipeline_fanout/pkg/stats"
	"github.com/abitofhelp/multistage_pipeline_fanout/pkg/utils"
	"go.uber.org/zap"
)

// ExitFunc is a function that exits the program with a given status code
type ExitFunc func(int)

// DefaultExitFunc is the default implementation of ExitFunc
var DefaultExitFunc = os.Exit

// ProcessFileFunc is a function type for processing a file
type ProcessFileFunc func(ctx context.Context, log *zap.Logger, inputPath, outputPath string) (*stats.Stats, error)

// run is the main logic of the application, extracted for testability
func run(args []string, log *zap.Logger, exit ExitFunc, processFile ProcessFileFunc) {
	if len(args) != 2 {
		fmt.Println("Usage: program <input_file_path> <output_file_path>")
		exit(1)
		return
	}

	inputPath := args[0]
	outputPath := args[1]

	// Create a context with cancellation for safety
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup graceful shutdown with improved signal handling
	// Defer the cleanup function to ensure signal handling is properly cleaned up
	cleanup := utils.SetupGracefulShutdown(ctx, cancel, log)
	defer cleanup()

	// Process the file with improved error handling
	stats, err := processFile(ctx, log, inputPath, outputPath)
	if err != nil {
		if customErrors.IsCancellationError(err) {
			log.Warn("Processing was canceled", zap.Error(err))
		} else if customErrors.IsTimeoutError(err) {
			log.Error("Processing timed out", zap.Error(err))
		} else if customErrors.IsIOError(err) {
			log.Error("I/O error during processing", zap.Error(err))
		} else {
			log.Error("Failed to process file", zap.Error(err))
		}
		exit(1)
		return
	}

	// Display summary
	stats.DisplaySummary(log, inputPath, outputPath)
}

func main() {
	// Parse command line arguments
	flag.Parse()
	args := flag.Args()

	// Initialize zap logger
	log := logger.InitLogger()
	defer func() {
		// Ensure logger syncs before exit
		logger.SafeSync(log)
	}()

	// Run the application with the default exit function and pipeline.ProcessFile
	run(args, log, DefaultExitFunc, pipeline.ProcessFile)
}
