// Copyright (c) 2025 A Bit of Help, Inc.

package options

import (
	"testing"
)

func TestDefaultPipelineOptions(t *testing.T) {
	// Get default options
	opts := DefaultPipelineOptions()

	// Check that the options are not nil
	if opts == nil {
		t.Fatal("Expected non-nil options")
	}

	// Check that the default values match the constants
	if opts.ChunkSize != DefaultChunkSize {
		t.Errorf("Expected ChunkSize to be %d, got %d", DefaultChunkSize, opts.ChunkSize)
	}

	if opts.ChannelBufferSize != DefaultChannelBufferSize {
		t.Errorf("Expected ChannelBufferSize to be %d, got %d", DefaultChannelBufferSize, opts.ChannelBufferSize)
	}

	if opts.CompressorCount != DefaultCompressorCount {
		t.Errorf("Expected CompressorCount to be %d, got %d", DefaultCompressorCount, opts.CompressorCount)
	}

	if opts.EncryptorCount != DefaultEncryptorCount {
		t.Errorf("Expected EncryptorCount to be %d, got %d", DefaultEncryptorCount, opts.EncryptorCount)
	}
}

func TestConstantValues(t *testing.T) {
	// Test that the constants have the expected values
	if DefaultChunkSize != 32*1024 {
		t.Errorf("Expected DefaultChunkSize to be %d, got %d", 32*1024, DefaultChunkSize)
	}

	if DefaultChannelBufferSize != 16 {
		t.Errorf("Expected DefaultChannelBufferSize to be %d, got %d", 16, DefaultChannelBufferSize)
	}

	if DefaultCompressorCount != 4 {
		t.Errorf("Expected DefaultCompressorCount to be %d, got %d", 4, DefaultCompressorCount)
	}

	if DefaultEncryptorCount != 4 {
		t.Errorf("Expected DefaultEncryptorCount to be %d, got %d", 4, DefaultEncryptorCount)
	}

	if ErrorChannelBufferSize != 4 {
		t.Errorf("Expected ErrorChannelBufferSize to be %d, got %d", 4, ErrorChannelBufferSize)
	}
}

func TestPipelineOptionsStruct(t *testing.T) {
	// Create a custom options struct
	opts := &PipelineOptions{
		ChunkSize:         64 * 1024,
		ChannelBufferSize: 32,
		CompressorCount:   8,
		EncryptorCount:    8,
	}

	// Check that the values are set correctly
	if opts.ChunkSize != 64*1024 {
		t.Errorf("Expected ChunkSize to be %d, got %d", 64*1024, opts.ChunkSize)
	}

	if opts.ChannelBufferSize != 32 {
		t.Errorf("Expected ChannelBufferSize to be %d, got %d", 32, opts.ChannelBufferSize)
	}

	if opts.CompressorCount != 8 {
		t.Errorf("Expected CompressorCount to be %d, got %d", 8, opts.CompressorCount)
	}

	if opts.EncryptorCount != 8 {
		t.Errorf("Expected EncryptorCount to be %d, got %d", 8, opts.EncryptorCount)
	}
}
