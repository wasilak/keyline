package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

// GenerateRandomBytes generates cryptographically secure random bytes
func GenerateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return b, nil
}

// GenerateStateToken generates a cryptographically secure state token (32 bytes)
func GenerateStateToken() (string, error) {
	b, err := GenerateRandomBytes(32)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// GenerateSessionID generates a cryptographically secure session ID (32 bytes)
func GenerateSessionID() (string, error) {
	b, err := GenerateRandomBytes(32)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// PKCEPair represents a PKCE code verifier and challenge pair
type PKCEPair struct {
	Verifier  string
	Challenge string
}

// GeneratePKCE generates a PKCE code verifier and challenge using S256 method
func GeneratePKCE() (*PKCEPair, error) {
	// Generate code_verifier (43-128 characters, URL-safe)
	// Using 32 bytes = 43 characters when base64url encoded
	verifierBytes, err := GenerateRandomBytes(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate code verifier: %w", err)
	}

	verifier := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(verifierBytes)

	// Derive code_challenge using S256 method (SHA256 hash, base64url-encoded)
	hash := sha256.Sum256([]byte(verifier))
	challenge := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(hash[:])

	return &PKCEPair{
		Verifier:  verifier,
		Challenge: challenge,
	}, nil
}
