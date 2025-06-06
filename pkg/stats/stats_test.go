// Copyright (c) 2025 A Bit of Help, Inc.

package stats

import (
	"os"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestNewStats(t *testing.T) {
	// Create a new Stats instance
	stats := NewStats()

	// Check that the instance is not nil
	if stats == nil {
		t.Error("Expected non-nil Stats instance, got nil")
	}

	// Check that the hash slices are initialized
	if stats.InputHash == nil {
		t.Error("Expected non-nil InputHash, got nil")
	}
	if stats.OutputHash == nil {
		t.Error("Expected non-nil OutputHash, got nil")
	}

	// Check that the counters are initialized to zero
	if stats.InputBytes.Load() != 0 {
		t.Errorf("Expected InputBytes to be 0, got %d", stats.InputBytes.Load())
	}
	if stats.CompressedBytes.Load() != 0 {
		t.Errorf("Expected CompressedBytes to be 0, got %d", stats.CompressedBytes.Load())
	}
	if stats.EncryptedBytes.Load() != 0 {
		t.Errorf("Expected EncryptedBytes to be 0, got %d", stats.EncryptedBytes.Load())
	}
	if stats.OutputBytes.Load() != 0 {
		t.Errorf("Expected OutputBytes to be 0, got %d", stats.OutputBytes.Load())
	}
	if stats.ChunksProcessed.Load() != 0 {
		t.Errorf("Expected ChunksProcessed to be 0, got %d", stats.ChunksProcessed.Load())
	}
	if stats.FinalChunkSize.Load() != 0 {
		t.Errorf("Expected FinalChunkSize to be 0, got %d", stats.FinalChunkSize.Load())
	}
}

func TestUpdateInputBytes(t *testing.T) {
	// Create a new Stats instance
	stats := NewStats()

	// Update input bytes
	stats.UpdateInputBytes(100)

	// Check that the value was updated
	if stats.InputBytes.Load() != 100 {
		t.Errorf("Expected InputBytes to be 100, got %d", stats.InputBytes.Load())
	}

	// Update again
	stats.UpdateInputBytes(50)

	// Check that the value was accumulated
	if stats.InputBytes.Load() != 150 {
		t.Errorf("Expected InputBytes to be 150, got %d", stats.InputBytes.Load())
	}
}

func TestUpdateOutputBytes(t *testing.T) {
	// Create a new Stats instance
	stats := NewStats()

	// Update output bytes
	stats.UpdateOutputBytes(100)

	// Check that the value was updated
	if stats.OutputBytes.Load() != 100 {
		t.Errorf("Expected OutputBytes to be 100, got %d", stats.OutputBytes.Load())
	}

	// Update again
	stats.UpdateOutputBytes(50)

	// Check that the value was accumulated
	if stats.OutputBytes.Load() != 150 {
		t.Errorf("Expected OutputBytes to be 150, got %d", stats.OutputBytes.Load())
	}
}

func TestUpdateCompressedBytes(t *testing.T) {
	// Create a new Stats instance
	stats := NewStats()

	// Update compressed bytes
	stats.UpdateCompressedBytes(100)

	// Check that the value was updated
	if stats.CompressedBytes.Load() != 100 {
		t.Errorf("Expected CompressedBytes to be 100, got %d", stats.CompressedBytes.Load())
	}

	// Update again
	stats.UpdateCompressedBytes(50)

	// Check that the value was accumulated
	if stats.CompressedBytes.Load() != 150 {
		t.Errorf("Expected CompressedBytes to be 150, got %d", stats.CompressedBytes.Load())
	}
}

func TestUpdateEncryptedBytes(t *testing.T) {
	// Create a new Stats instance
	stats := NewStats()

	// Update encrypted bytes
	stats.UpdateEncryptedBytes(100)

	// Check that the value was updated
	if stats.EncryptedBytes.Load() != 100 {
		t.Errorf("Expected EncryptedBytes to be 100, got %d", stats.EncryptedBytes.Load())
	}

	// Update again
	stats.UpdateEncryptedBytes(50)

	// Check that the value was accumulated
	if stats.EncryptedBytes.Load() != 150 {
		t.Errorf("Expected EncryptedBytes to be 150, got %d", stats.EncryptedBytes.Load())
	}
}

func TestIncrementChunksProcessed(t *testing.T) {
	// Create a new Stats instance
	stats := NewStats()

	// Increment chunks processed
	stats.IncrementChunksProcessed()

	// Check that the value was incremented
	if stats.ChunksProcessed.Load() != 1 {
		t.Errorf("Expected ChunksProcessed to be 1, got %d", stats.ChunksProcessed.Load())
	}

	// Increment again
	stats.IncrementChunksProcessed()

	// Check that the value was incremented
	if stats.ChunksProcessed.Load() != 2 {
		t.Errorf("Expected ChunksProcessed to be 2, got %d", stats.ChunksProcessed.Load())
	}
}

func TestSetFinalChunkSize(t *testing.T) {
	// Create a new Stats instance
	stats := NewStats()

	// Set final chunk size
	stats.SetFinalChunkSize(100)

	// Check that the value was set
	if stats.FinalChunkSize.Load() != 100 {
		t.Errorf("Expected FinalChunkSize to be 100, got %d", stats.FinalChunkSize.Load())
	}

	// Set again
	stats.SetFinalChunkSize(50)

	// Check that the value was overwritten
	if stats.FinalChunkSize.Load() != 50 {
		t.Errorf("Expected FinalChunkSize to be 50, got %d", stats.FinalChunkSize.Load())
	}
}

func TestConcurrentUpdates(t *testing.T) {
	// Skip this test in short mode as it involves concurrency
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	// Create a new Stats instance
	stats := NewStats()

	// Number of goroutines
	numGoroutines := 100

	// Number of updates per goroutine
	updatesPerGoroutine := 100

	// Wait group for synchronization
	var wg sync.WaitGroup
	wg.Add(numGoroutines * 6) // 6 types of updates

	// Start goroutines to update input bytes
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < updatesPerGoroutine; j++ {
				stats.UpdateInputBytes(1)
			}
		}()
	}

	// Start goroutines to update output bytes
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < updatesPerGoroutine; j++ {
				stats.UpdateOutputBytes(1)
			}
		}()
	}

	// Start goroutines to update compressed bytes
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < updatesPerGoroutine; j++ {
				stats.UpdateCompressedBytes(1)
			}
		}()
	}

	// Start goroutines to update encrypted bytes
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < updatesPerGoroutine; j++ {
				stats.UpdateEncryptedBytes(1)
			}
		}()
	}

	// Start goroutines to increment chunks processed
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < updatesPerGoroutine; j++ {
				stats.IncrementChunksProcessed()
			}
		}()
	}

	// Start goroutines to set final chunk size (this is a race, but we're just testing thread safety)
	for i := 0; i < numGoroutines; i++ {
		go func(i int) {
			defer wg.Done()
			stats.SetFinalChunkSize(uint64(i))
			// Sleep a bit to increase chance of race conditions
			time.Sleep(time.Millisecond)
		}(i)
	}

	// Wait for all goroutines to finish
	wg.Wait()

	// Check that the values were updated correctly
	expectedCount := uint64(numGoroutines * updatesPerGoroutine)
	if stats.InputBytes.Load() != expectedCount {
		t.Errorf("Expected InputBytes to be %d, got %d", expectedCount, stats.InputBytes.Load())
	}
	if stats.OutputBytes.Load() != expectedCount {
		t.Errorf("Expected OutputBytes to be %d, got %d", expectedCount, stats.OutputBytes.Load())
	}
	if stats.CompressedBytes.Load() != expectedCount {
		t.Errorf("Expected CompressedBytes to be %d, got %d", expectedCount, stats.CompressedBytes.Load())
	}
	if stats.EncryptedBytes.Load() != expectedCount {
		t.Errorf("Expected EncryptedBytes to be %d, got %d", expectedCount, stats.EncryptedBytes.Load())
	}
	if stats.ChunksProcessed.Load() != expectedCount {
		t.Errorf("Expected ChunksProcessed to be %d, got %d", expectedCount, stats.ChunksProcessed.Load())
	}
	// We can't check FinalChunkSize as it depends on the order of execution
}

func TestFormatDuration(t *testing.T) {
	testCases := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "hours",
			duration: 2*time.Hour + 30*time.Minute + 15*time.Second + 500*time.Millisecond,
			expected: "2h 30m 15s 500ms",
		},
		{
			name:     "minutes",
			duration: 30*time.Minute + 15*time.Second + 500*time.Millisecond,
			expected: "30m 15s 500ms",
		},
		{
			name:     "seconds",
			duration: 15*time.Second + 500*time.Millisecond,
			expected: "15s 500ms",
		},
		{
			name:     "milliseconds",
			duration: 500 * time.Millisecond,
			expected: "500ms",
		},
		{
			name:     "zero",
			duration: 0,
			expected: "0ms",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := FormatDuration(tc.duration)
			if result != tc.expected {
				t.Errorf("Expected %q, got %q", tc.expected, result)
			}
		})
	}
}

// Helper function for approximate floating point comparison
func approxEqual(a, b, epsilon float64) bool {
	return a-b < epsilon && b-a < epsilon
}

func TestCalculateRatios(t *testing.T) {
	testCases := []struct {
		name                 string
		stats                *Stats
		expectedCompRatio    float64
		expectedEncRatio     float64
		expectedOverallRatio float64
	}{
		{
			name: "normal case",
			stats: func() *Stats {
				s := NewStats()
				s.InputBytes.Store(1000)
				s.CompressedBytes.Store(500)
				s.EncryptedBytes.Store(600)
				s.OutputBytes.Store(600)
				return s
			}(),
			expectedCompRatio:    2.0,   // 1000/500
			expectedEncRatio:     0.833, // 500/600 (rounded)
			expectedOverallRatio: 40.0,  // (1 - (600/1000))*100
		},
		{
			name: "no compression",
			stats: func() *Stats {
				s := NewStats()
				s.InputBytes.Store(1000)
				s.CompressedBytes.Store(1000)
				s.EncryptedBytes.Store(1200)
				s.OutputBytes.Store(1200)
				return s
			}(),
			expectedCompRatio:    1.0,   // 1000/1000
			expectedEncRatio:     0.833, // 1000/1200 (rounded)
			expectedOverallRatio: -20.0, // (1 - (1200/1000))*100
		},
		{
			name: "zero values",
			stats: func() *Stats {
				s := NewStats()
				s.InputBytes.Store(0)
				s.CompressedBytes.Store(0)
				s.EncryptedBytes.Store(0)
				s.OutputBytes.Store(0)
				return s
			}(),
			expectedCompRatio:    0.0,
			expectedEncRatio:     0.0,
			expectedOverallRatio: 0.0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			compRatio, encRatio, overallRatio := tc.stats.CalculateRatios()

			// Use approximate comparison for floating point
			if !approxEqual(compRatio, tc.expectedCompRatio, 0.01) {
				t.Errorf("Expected compression ratio %.3f, got %.3f", tc.expectedCompRatio, compRatio)
			}

			if !approxEqual(encRatio, tc.expectedEncRatio, 0.01) {
				t.Errorf("Expected encryption ratio %.3f, got %.3f", tc.expectedEncRatio, encRatio)
			}

			if !approxEqual(overallRatio, tc.expectedOverallRatio, 0.01) {
				t.Errorf("Expected overall ratio %.3f, got %.3f", tc.expectedOverallRatio, overallRatio)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	if s == "" || substr == "" || len(s) < len(substr) {
		return false
	}

	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestDisplaySummary(t *testing.T) {
	// Create a logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()

	// Create stats
	testStats := NewStats()
	testStats.InputBytes.Store(1000)
	testStats.CompressedBytes.Store(500)
	testStats.EncryptedBytes.Store(600)
	testStats.OutputBytes.Store(600)
	testStats.ChunksProcessed.Store(5)
	testStats.FinalChunkSize.Store(100)
	testStats.ProcessingTime = 1 * time.Second
	testStats.InputHash = []byte{1, 2, 3, 4, 5}
	testStats.OutputHash = []byte{6, 7, 8, 9, 10}

	// Redirect stdout to capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Call DisplaySummary
	testStats.DisplaySummary(logger, "input.txt", "output.txt")

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read the captured output
	var buf [1024]byte
	n, _ := r.Read(buf[:])
	output := string(buf[:n])

	// Check that the output contains expected information
	expectedStrings := []string{
		"Processing Summary",
		"Input file: input.txt",
		"Output file: output.txt",
		"Total input bytes",
		"Input SHA256",
		"Total output bytes",
		"Output SHA256",
		"Total processing time",
		"Input to Compressed Ratio",
		"Compressed to Encrypted Ratio",
		"Compressing Input Saved Space",
		"Number of chunks processed",
		"Size of final chunk",
	}

	for _, s := range expectedStrings {
		if !contains(output, s) {
			t.Errorf("Expected output to contain %q, but it doesn't", s)
		}
	}
}
