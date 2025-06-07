# multistage_pipeline_fanout

[![Go Version](https://img.shields.io/badge/Go-1.24-blue.svg)](https://golang.org/doc/go1.24)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Test Coverage](https://img.shields.io/badge/Coverage-59.1%25-yellow.svg)](coverage.html)

A high-performance data processing pipeline implementation in Go that provides efficient file processing with parallel compression and encryption.

## Overview

multistage_pipeline_fanout implements a multi-stage processing pipeline with concurrent execution of compression and encryption operations for improved performance. The pipeline:

- Reads data from an input file in configurable chunks
- Compresses the data using parallel workers
- Encrypts the compressed data using parallel workers
- Writes the processed data to an output file
- Calculates SHA256 checksums on the fly for the input and output files to ensure data integrity
- Collects detailed statistics about the processing

The application is designed with a focus on performance, reliability, and proper resource management, including graceful shutdown handling.

## Project Structure

The project is organized into two main package hierarchies:

### `/pkg` - Core Functionality

The `/pkg` directory contains packages that implement core, reusable functionality:

- `/pkg/compression` - Provides core compression algorithms and utilities
- `/pkg/dataprocessor` - Provides generic data processing with context awareness
- `/pkg/encryption` - Provides core encryption algorithms and utilities
- `/pkg/errors` - Custom error types and error handling utilities
- `/pkg/logger` - Logging utilities
- `/pkg/stats` - Statistics collection and reporting
- `/pkg/utils` - General utility functions

### `/pkg/pipeline` - Pipeline Integration

The `/pkg/pipeline` directory contains packages that integrate the core functionality into a processing pipeline:

- `/pkg/pipeline/compressor` - Pipeline stage that uses the core compression functionality
- `/pkg/pipeline/encryptor` - Pipeline stage that uses the core encryption functionality
- `/pkg/pipeline/processor` - Generic pipeline stage processor
- `/pkg/pipeline/reader` - Pipeline stage for reading input data
- `/pkg/pipeline/writer` - Pipeline stage for writing output data
- `/pkg/pipeline/options` - Configuration options for the pipeline

## Why Similar Package Names Are Not Redundant

The packages in `/pkg` and `/pkg/pipeline` with similar names (e.g., `compression` vs `compressor`, `encryption` vs `encryptor`) serve different purposes and are not redundant:

1. **Core Packages (`/pkg`)**: 
   - Implement the fundamental algorithms and utilities
   - Are context-aware but not pipeline-specific
   - Can be used independently outside the pipeline
   - Focus on the core functionality (compression, encryption, etc.)

2. **Pipeline Packages (`/pkg/pipeline`)**: 
   - Integrate the core functionality into the pipeline architecture
   - Handle pipeline-specific concerns like channel communication
   - Manage concurrency, error handling, and statistics within the pipeline
   - Act as adapters between the core functionality and the pipeline framework

This separation allows for:
- Better code organization and maintainability
- Reuse of core functionality in different contexts
- Independent testing of core algorithms and pipeline integration
- Clearer separation of concerns

## Usage

### Prerequisites

- Go 1.24 or later
- Make

### Building and Running

The project includes a comprehensive Makefile that provides various commands for building, testing, and running the application.

#### Basic Commands

```bash
# Build the application
make build

# Run the application with default input and output files
make run

# Run all tests
make test

# Run short tests (faster)
make test-short

# Run unit tests only (excluding integration tests)
make test-unit

# Run integration tests
make test-integration

# Run tests for a specific package
make test-package PKG=./pkg/compression

# Run tests with race detection
make test-race

# Run tests with coverage analysis
make coverage

# Clean build artifacts
make clean

# Format code
make fmt

# Run linter
make lint

# Run security check
make sec

# Run vulnerability scanning
make vuln

# Check Go version compatibility
make check-go-version

# Generate CHANGELOG
make changelog

# Docker operations
make docker-build  # Build Docker image
make docker-run    # Run in Docker container
make docker        # Build and run in Docker

# Generate and serve documentation
make doc
```

#### Advanced Commands

```bash
# Install dependencies including linting and security tools
make deps

# Update dependencies
make update-deps

# Run tests with a specific tag
make test-tag TAG=unit

# Run tests for CI environments
make test-ci

# Generate mocks for testing
make mocks

# Install the binary to GOPATH/bin
make install

# Run vulnerability scanning
make vuln

# Check Go version compatibility
make check-go-version

# Generate CHANGELOG
make changelog

# Build Docker image
make docker-build

# Run in Docker container
make docker-run

# Show all available commands
make help
```

### Running the Application Directly

After building, you can run the application directly:

```bash
./build/multistage_pipeline_fanout <input_file_path> <output_file_path>
```

The application requires two command-line arguments:
1. `input_file_path`: Path to the file to be processed
2. `output_file_path`: Path where the processed data will be written

The application processes the input file through a pipeline that includes compression and encryption, then writes the result to the output file.

### Running with Docker

The project includes Docker support for containerized execution:

```bash
# Build the Docker image
make docker-build

# Run the application in a Docker container
make docker-run

# Or do both in one command
make docker
```

You can also use Docker commands directly:

```bash
# Build the image
docker build -t multistage_pipeline_fanout:latest .

# Run the container
docker run --rm -v $(pwd)/input_file.txt:/app/input.txt -v $(pwd):/app/output multistage_pipeline_fanout:latest input.txt /app/output/output.bin
```

### Configuration

The pipeline behavior can be configured through the `options.DefaultPipelineOptions()` function in the `pkg/pipeline/options/options.go` file. Key configurable parameters include:

- `ChunkSize`: Size of data chunks read from the input file (default: 32KB)
- `CompressorCount`: Number of parallel compression workers (default: 4)
- `EncryptorCount`: Number of parallel encryption workers (default: 4)
- `ChannelBufferSize`: Size of the channel buffers between pipeline stages (default: 16)

These options can be modified programmatically if you're using the pipeline as a library.

### Project Architecture

The application follows a modular architecture with clear separation of concerns:

1. **Entry Point**: The application entry point is in `cmd/main.go`, which parses command-line arguments and initializes the pipeline.

2. **Pipeline**: The core pipeline implementation in `pkg/pipeline/pipeline.go` orchestrates the data flow through multiple stages.

3. **Pipeline Stages**:
   - Reader (`pkg/pipeline/reader`): Reads data from the input file in chunks
   - Compressor (`pkg/pipeline/compressor`): Compresses data using the Brotli algorithm
   - Encryptor (`pkg/pipeline/encryptor`): Encrypts data using AES-GCM
   - Writer (`pkg/pipeline/writer`): Writes processed data to the output file

4. **Core Functionality**: Implemented in separate packages for reusability:
   - Compression (`pkg/compression`): Compression algorithms and utilities
   - Encryption (`pkg/encryption`): Encryption algorithms and utilities
   - Statistics (`pkg/stats`): Collection and reporting of processing statistics
   - Logging (`pkg/logger`): Structured logging using Zap

### Development Workflow

A typical development workflow might look like:

1. Make changes to the code
2. Format the code: `make fmt`
3. Run the linter: `make lint`
4. Run tests: `make test`
5. Build the application: `make build`
6. Run the application: `make run` or `./build/multistage_pipeline_fanout <input> <output>`

## Performance

multistage_pipeline_fanout is designed for high performance with parallel processing:

- Multiple compression workers process chunks concurrently
- Multiple encryption workers process compressed chunks concurrently
- Buffered channels prevent pipeline stalls
- Efficient memory management with controlled chunk sizes
- Optimized Brotli compression

Performance can be tuned by adjusting the configuration parameters in `options.DefaultPipelineOptions()`.

## Examples

### Basic File Processing

```bash
# Build the application
make build

# Process a 10MB JSON file
./build/multistage_pipeline_fanout input_10mb.jsonl output.bin

# Processing statistics are displayed automatically after completion
# No need to run additional commands to view statistics
```

### Using as a Library

```go
package main

import (
    "context"
    "log"

    "github.com/abitofhelp/multistage_pipeline_fanout/pkg/logger"
    "github.com/abitofhelp/multistage_pipeline_fanout/pkg/pipeline"
)

func main() {
    // Initialize logger
    log := logger.InitLogger()
    defer func() { logger.SafeSync(log) }()

    // Create context
    ctx := context.Background()

    // Process file
    stats, err := pipeline.ProcessFile(ctx, log, "input.txt", "output.bin")
    if err != nil {
        log.Fatal("Failed to process file", err)
    }

    // Use stats as needed
    log.Info("Processing complete", 
        "inputBytes", stats.InputBytes.Load(),
        "outputBytes", stats.OutputBytes.Load(),
        "compressionRatio", float64(stats.InputBytes.Load())/float64(stats.OutputBytes.Load()),
    )
}
```

## Error Handling and Reliability

multistage_pipeline_fanout implements robust error handling and reliability features:

### Custom Error Types

The application uses a custom error handling system (`pkg/errors`) that provides:

- Categorized error types (I/O errors, timeout errors, cancellation errors, etc.)
- Rich error context including stage, operation, time, and data size
- Error aggregation for collecting multiple errors
- Helper functions for error type checking

### Signal Handling

The application implements advanced signal handling for graceful shutdown:

- Handles SIGINT, SIGTERM, SIGHUP, and SIGQUIT
- Implements a two-phase shutdown (graceful on first signal, forced on second)
- Includes a 30-second timeout for graceful shutdown
- Properly cleans up resources during shutdown

### Context Propagation

All operations are context-aware, allowing for:

- Cancellation propagation throughout the pipeline
- Timeout handling at all stages
- Proper resource cleanup on cancellation

## Troubleshooting

### Common Issues

1. **"Failed to open input file"**: Ensure the input file exists and has proper read permissions.

2. **"Failed to create output file"**: Ensure the output directory exists and has proper write permissions.

3. **"Context deadline exceeded"**: The processing took longer than the context timeout. For large files, consider using a context with a longer timeout.

4. **"Out of memory"**: If processing very large files, try reducing the chunk size in the options to lower memory usage.

5. **"Pipeline stage blocked"**: A pipeline stage is waiting too long to send data to the next stage. This could indicate a bottleneck in the pipeline. Try adjusting the number of workers or chunk size.

6. **"Operation canceled"**: The processing was canceled, either by a signal (Ctrl+C) or programmatically. This is a normal part of graceful shutdown.

### Performance Issues

If you're experiencing performance issues:

1. Try adjusting the number of compressor and encryptor workers based on your system's capabilities
2. Experiment with different chunk sizes
3. Ensure your storage devices have sufficient I/O performance
4. Run with `GODEBUG=gctrace=1` to monitor garbage collection overhead

## Dependencies

This project relies on the following key dependencies:

- [github.com/andybalholm/brotli](https://github.com/andybalholm/brotli) - Brotli compression algorithm implementation
- [github.com/google/tink/go](https://github.com/google/tink/go) - Cryptographic API providing secure implementations of common cryptographic primitives
- [github.com/dustin/go-humanize](https://github.com/dustin/go-humanize) - Formatters for units to human-friendly sizes
- [go.uber.org/zap](https://github.com/uber-go/zap) - Structured, leveled logging
- [github.com/stretchr/testify](https://github.com/stretchr/testify) - Testing toolkit

For a complete list of dependencies, see the [go.mod](go.mod) file.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

Copyright (c) 2023 A Bit of Help, Inc.
