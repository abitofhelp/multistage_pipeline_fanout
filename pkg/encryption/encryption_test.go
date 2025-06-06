// Copyright (c) 2025 A Bit of Help, Inc.

package encryption

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestInitEncryption(t *testing.T) {
	// Create a logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()

	// Initialize encryption
	aead, err := InitEncryption(logger)

	// Check for errors
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Check that AEAD is not nil
	if aead == nil {
		t.Error("Expected non-nil AEAD, got nil")
	}
}

func TestEncryptDataWithContext_Success(t *testing.T) {
	// Create a logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()

	// Initialize encryption
	aead, err := InitEncryption(logger)
	if err != nil {
		t.Fatalf("Failed to initialize encryption: %v", err)
	}

	// Create a context
	ctx := context.Background()

	// Test data
	data := []byte("test data")

	// Encrypt the data
	encrypted, err := EncryptDataWithContext(ctx, aead, data)

	// Check for errors
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Check that encrypted data is not nil and not the same as input
	if encrypted == nil {
		t.Error("Expected non-nil encrypted data, got nil")
	}
	if len(encrypted) == 0 {
		t.Error("Expected non-empty encrypted data")
	}
	if len(encrypted) <= len(data) {
		t.Errorf("Expected encrypted data to be longer than input (due to added metadata), got %d <= %d", len(encrypted), len(data))
	}
}

func TestEncryptDataWithContext_CanceledContext(t *testing.T) {
	// Create a logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()

	// Initialize encryption
	aead, err := InitEncryption(logger)
	if err != nil {
		t.Fatalf("Failed to initialize encryption: %v", err)
	}

	// Create a canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Test data
	data := []byte("test data")

	// Encrypt the data
	encrypted, err := EncryptDataWithContext(ctx, aead, data)

	// Check for errors
	if err == nil {
		t.Error("Expected an error, got nil")
	}
	if encrypted != nil {
		t.Errorf("Expected nil encrypted data, got %v", encrypted)
	}
}

func TestEncryptDataWithContext_Timeout(t *testing.T) {
	// Skip this test in short mode as it involves waiting
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	// Create a logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()

	// Initialize encryption
	aead, err := InitEncryption(logger)
	if err != nil {
		t.Fatalf("Failed to initialize encryption: %v", err)
	}

	// Create a context with a very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Wait to ensure the timeout has occurred
	time.Sleep(10 * time.Millisecond)

	// Test data - make it large enough to potentially cause a timeout
	data := make([]byte, 1024*1024) // 1MB

	// Encrypt the data
	encrypted, err := EncryptDataWithContext(ctx, aead, data)

	// Check for errors
	if err == nil {
		t.Error("Expected an error, got nil")
	}
	if encrypted != nil {
		t.Errorf("Expected nil encrypted data, got data of length %d", len(encrypted))
	}
}

func TestEncryptAndDecrypt(t *testing.T) {
	// Create a logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()

	// Initialize encryption
	aead, err := InitEncryption(logger)
	if err != nil {
		t.Fatalf("Failed to initialize encryption: %v", err)
	}

	// Create a context
	ctx := context.Background()

	// Test data
	originalData := []byte("test data for encryption and decryption")

	// Encrypt the data
	encrypted, err := EncryptDataWithContext(ctx, aead, originalData)
	if err != nil {
		t.Fatalf("Failed to encrypt data: %v", err)
	}

	// Decrypt the data directly using the AEAD primitive
	decrypted, err := aead.Decrypt(encrypted, []byte{})
	if err != nil {
		t.Fatalf("Failed to decrypt data: %v", err)
	}

	// Check that decrypted data matches original
	if len(decrypted) != len(originalData) {
		t.Errorf("Expected decrypted data length %d, got %d", len(originalData), len(decrypted))
	}
	for i := range originalData {
		if decrypted[i] != originalData[i] {
			t.Errorf("Expected decrypted[%d] = %d, got %d", i, originalData[i], decrypted[i])
		}
	}
}
