// Copyright (c) 2025 A Bit of Help, Inc.

// Package encryption provides data encryption functionality.
//
// This package implements the core encryption algorithms and utilities that can be used
// independently of the pipeline. It is context-aware but not pipeline-specific.
//
// The corresponding package in the pipeline hierarchy is pkg/pipeline/encryptor,
// which integrates this core functionality into the pipeline architecture.
package encryption

import (
	"context"
	"fmt"

	"github.com/abitofhelp/multistage_pipeline_fanout/pkg/dataprocessor"
	"github.com/google/tink/go/aead"
	"github.com/google/tink/go/keyset"
	"github.com/google/tink/go/tink"
	"go.uber.org/zap"
)

// InitEncryption initializes the encryption primitive with proper error handling
func InitEncryption(logger *zap.Logger) (tink.AEAD, error) {
	// For demonstration purposes, we're generating a new key
	// In a real application, this should be securely managed and possibly loaded from a secure storage
	kh, err := keyset.NewHandle(aead.AES256GCMKeyTemplate())
	if err != nil {
		return nil, fmt.Errorf("failed to create keyset handle: %w", err)
	}

	a, err := aead.New(kh)
	if err != nil {
		return nil, fmt.Errorf("failed to create AEAD primitive: %w", err)
	}

	logger.Info("Encryption initialized successfully")
	return a, nil
}

// EncryptDataWithContext encrypts data with context awareness
func EncryptDataWithContext(ctx context.Context, a tink.AEAD, data []byte) ([]byte, error) {
	// Create a closure that captures the AEAD primitive
	encryptFunc := func(data []byte) ([]byte, error) {
		// Using empty associated data for simplicity
		return a.Encrypt(data, []byte{})
	}

	return dataprocessor.ProcessWithContext(ctx, encryptFunc, data)
}
