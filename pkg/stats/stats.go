// Copyright (c) 2025 A Bit of Help, Inc.

// Package stats provides functionality for tracking pipeline processing statistics
package stats

import (
	"encoding/hex"
	"fmt"
	"github.com/dustin/go-humanize"
	"go.uber.org/zap"
	"sync/atomic"
	"time"
)

// Stats tracks pipeline processing statistics with thread-safe access methods
type Stats struct {
	// Byte counts for different stages
	InputBytes      atomic.Uint64
	CompressedBytes atomic.Uint64
	EncryptedBytes  atomic.Uint64
	OutputBytes     atomic.Uint64

	// Cryptographic hashes for verification
	InputHash  []byte
	OutputHash []byte

	// Performance metrics
	ProcessingTime  time.Duration
	ChunksProcessed atomic.Uint64
	FinalChunkSize  atomic.Uint64
}

// UpdateInputBytes safely adds n bytes to the input byte count
func (s *Stats) UpdateInputBytes(n uint64) {
	s.InputBytes.Add(n)
}

// UpdateOutputBytes safely adds n bytes to the output byte count
func (s *Stats) UpdateOutputBytes(n uint64) {
	s.OutputBytes.Add(n)
}

// UpdateCompressedBytes safely adds n bytes to the compressed byte count
func (s *Stats) UpdateCompressedBytes(n uint64) {
	s.CompressedBytes.Add(n)
}

// UpdateEncryptedBytes safely adds n bytes to the encrypted byte count
func (s *Stats) UpdateEncryptedBytes(n uint64) {
	s.EncryptedBytes.Add(n)
}

// IncrementChunksProcessed safely increments the chunks processed counter
func (s *Stats) IncrementChunksProcessed() {
	s.ChunksProcessed.Add(1)
}

// SetFinalChunkSize safely sets the size of the final chunk processed
func (s *Stats) SetFinalChunkSize(n uint64) {
	s.FinalChunkSize.Store(n)
}

// NewStats creates a new Stats instance with initialized fields
func NewStats() *Stats {
	return &Stats{
		InputHash:  make([]byte, 0),
		OutputHash: make([]byte, 0),
	}
}

// FormatDuration formats a duration in a human-readable way
func FormatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60
	milliseconds := int(d.Milliseconds()) % 1000

	if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds %dms", hours, minutes, seconds, milliseconds)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm %ds %dms", minutes, seconds, milliseconds)
	} else if seconds > 0 {
		return fmt.Sprintf("%ds %dms", seconds, milliseconds)
	}
	return fmt.Sprintf("%dms", milliseconds)
}

// CalculateRatios calculates compression and encryption ratios
func (s *Stats) CalculateRatios() (float64, float64, float64) {
	// Avoid division by zero
	if s.CompressedBytes.Load() == 0 || s.InputBytes.Load() == 0 || s.EncryptedBytes.Load() == 0 {
		return 0, 0, 0
	}

	// Calculate ratios as input:output (higher is better compression)
	uncompressedToCompressedRatio := float64(s.InputBytes.Load()) / float64(s.CompressedBytes.Load())
	unencryptedToEncryptedRatio := float64(s.CompressedBytes.Load()) / float64(s.EncryptedBytes.Load())

	// Percentage of space saved = (1 - (Compressed size / Original size)) * 100
	// Example:
	// Percentage of space saved = (1 - (25 MB / 100 MB)) * 100
	// Percentage of space saved = (1 - 0.25) * 100
	// Percentage of space saved = 0.75 * 100 = 75%.
	// This indicates that the compression saved 75% of the space.
	compressionPercentage := (1 - (float64(s.OutputBytes.Load()) / float64(s.InputBytes.Load()))) * 100

	return uncompressedToCompressedRatio, unencryptedToEncryptedRatio, compressionPercentage
}

// DisplaySummary prints and logs a summary of the processing results
func (s *Stats) DisplaySummary(logger *zap.Logger, inputPath, outputPath string) {
	// Format the processing time
	timeFormatted := FormatDuration(s.ProcessingTime)

	// Calculate ratios
	uncompressedToCompressedRatio, unencryptedToEncryptedRatio, compressionPercentage := s.CalculateRatios()

	// Get atomic values
	inputBytes := s.InputBytes.Load()
	outputBytes := s.OutputBytes.Load()
	chunksProcessed := s.ChunksProcessed.Load()
	finalChunkSize := s.FinalChunkSize.Load()
	compressedBytes := s.CompressedBytes.Load()
	encryptedBytes := s.EncryptedBytes.Load()

	// Print summary to console
	fmt.Println("\n==================")
	fmt.Println("Processing Summary")
	fmt.Println("==================")
	fmt.Printf("Input file: %s\n", inputPath)
	fmt.Printf("Output file: %s\n", outputPath)
	fmt.Println("------------------")
	fmt.Printf("Total input bytes: %s (%d bytes)\n", humanize.Bytes(inputBytes), inputBytes)
	fmt.Printf("Input SHA256: %s\n", hex.EncodeToString(s.InputHash))
	fmt.Printf("Total output bytes: %s (%d bytes)\n", humanize.Bytes(outputBytes), outputBytes)
	fmt.Printf("Output SHA256: %s\n", hex.EncodeToString(s.OutputHash))
	fmt.Println("------------------")
	fmt.Printf("Input to Compressed Ratio: %.2f:1\n", uncompressedToCompressedRatio)
	fmt.Printf("Compressed to Encrypted Ratio: %.2f:1\n", unencryptedToEncryptedRatio)
	fmt.Printf("Compressing Input Saved Space: %.2f%%\n", compressionPercentage)
	fmt.Printf("Number of chunks processed: %d\n", chunksProcessed)
	fmt.Printf("Size of final chunk: %s (%d bytes)\n", humanize.Bytes(finalChunkSize), finalChunkSize)
	fmt.Println("------------------")
	fmt.Printf("Total processing time: %s (%v)\n", timeFormatted, s.ProcessingTime)
	fmt.Println("==================")

	// Log detailed summary
	logger.Debug("Processing completed successfully",
		zap.String("input_file", inputPath),
		zap.String("output_file", outputPath),
		zap.Uint64("total_input_bytes", inputBytes),
		zap.String("input_sha256_hash", hex.EncodeToString(s.InputHash)),
		zap.Uint64("total_output_bytes", outputBytes),
		zap.String("output_sha256_hash", hex.EncodeToString(s.OutputHash)),
		zap.Uint64("compressed_bytes", compressedBytes),
		zap.Float64("compression_ratio", uncompressedToCompressedRatio),
		zap.Uint64("encrypted_bytes", encryptedBytes),
		zap.Float64("encryption_ratio", unencryptedToEncryptedRatio),
		zap.Float64("compressing_saved_space", compressionPercentage),
		zap.Uint64("chunks_processed", chunksProcessed),
		zap.Uint64("final_chunk_size", finalChunkSize),
		zap.Duration("processing_time", s.ProcessingTime),
		zap.String("formatted_processing_time", timeFormatted))
}
