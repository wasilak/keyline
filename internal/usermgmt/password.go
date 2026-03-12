package usermgmt

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

const (
	// defaultPasswordLength is the default length for generated passwords
	defaultPasswordLength = 32

	// charset contains all characters that can be used in generated passwords
	// Includes uppercase, lowercase, digits, and special characters
	charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()-_=+[]{}|;:,.<>?"
)

// PasswordGenerator generates cryptographically secure random passwords
type PasswordGenerator struct {
	length int
}

// NewPasswordGenerator creates a new password generator with the specified length
// If length is 0 or negative, defaultPasswordLength is used
func NewPasswordGenerator(length int) *PasswordGenerator {
	if length <= 0 {
		length = defaultPasswordLength
	}
	return &PasswordGenerator{
		length: length,
	}
}

// Generate creates a cryptographically secure random password
// Returns an error if random number generation fails
func (pg *PasswordGenerator) Generate() (string, error) {
	password := make([]byte, pg.length)

	for i := range password {
		// Use crypto/rand for cryptographically secure randomness
		randomIndex, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", fmt.Errorf("failed to generate random number: %w", err)
		}
		password[i] = charset[randomIndex.Int64()]
	}

	return string(password), nil
}
