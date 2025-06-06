// Copyright (c) 2025 A Bit of Help, Inc.

// Package options provides configuration options for the processing pipeline.
package options

const (
	// DefaultChunkSize defines the size of data chunks read from the input file (32KB)
	DefaultChunkSize = 32 * 1024

	// DefaultChannelBufferSize defines the buffer size for pipeline channels
	// to prevent blocking between pipeline stages
	DefaultChannelBufferSize = 16

	// DefaultCompressorCount defines the number of concurrent compressor goroutines
	DefaultCompressorCount = 4

	// DefaultEncryptorCount defines the number of concurrent encryptor goroutines
	DefaultEncryptorCount = 4

	// ErrorChannelBufferSize defines the buffer size for the error channel
	ErrorChannelBufferSize = 4
)

// PipelineOptions contains configuration options for the processing pipeline
type PipelineOptions struct {
	// ChunkSize defines the size of data chunks read from the input file
	ChunkSize int

	// ChannelBufferSize defines the buffer size for pipeline channels
	ChannelBufferSize int

	// CompressorCount defines the number of concurrent compressor goroutines
	CompressorCount int

	// EncryptorCount defines the number of concurrent encryptor goroutines
	EncryptorCount int
}

// DefaultPipelineOptions returns a PipelineOptions with default values
func DefaultPipelineOptions() *PipelineOptions {
	return &PipelineOptions{
		ChunkSize:         DefaultChunkSize,
		ChannelBufferSize: DefaultChannelBufferSize,
		CompressorCount:   DefaultCompressorCount,
		EncryptorCount:    DefaultEncryptorCount,
	}
}
