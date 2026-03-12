package usermgmt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

// Encryptor provides methods for encrypting and decrypting passwords
// for secure cache storage using AES-256-GCM.
type Encryptor interface {
	// Encrypt encrypts plaintext and returns base64-encoded ciphertext
	Encrypt(plaintext string) (string, error)

	// Decrypt decrypts base64-encoded ciphertext and returns plaintext
	Decrypt(ciphertext string) (string, error)
}

// encryptor implements the Encryptor interface using AES-256-GCM
type encryptor struct {
	key []byte // 32 bytes for AES-256
}

// NewEncryptor creates a new Encryptor with the provided encryption key.
// The key must be exactly 32 bytes (256 bits) for AES-256.
func NewEncryptor(key []byte) (Encryptor, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("encryption key must be 32 bytes, got %d", len(key))
	}

	return &encryptor{key: key}, nil
}

// Encrypt encrypts the plaintext using AES-256-GCM with a random nonce.
// The nonce is prepended to the ciphertext, and the result is base64-encoded.
func (e *encryptor) Encrypt(plaintext string) (string, error) {
	// Create AES cipher
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate random nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt and prepend nonce
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// Encode as base64 for storage
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts the base64-encoded ciphertext using AES-256-GCM.
// The nonce is extracted from the beginning of the ciphertext.
func (e *encryptor) Decrypt(ciphertext string) (string, error) {
	// Decode from base64
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	// Create AES cipher
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Extract nonce
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]

	// Decrypt
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}
