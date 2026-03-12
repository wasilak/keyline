package usermgmt

import (
	"crypto/rand"
	"strings"
	"testing"
)

// TestNewEncryptor_ValidKey tests that NewEncryptor accepts a valid 32-byte key
func TestNewEncryptor_ValidKey(t *testing.T) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		t.Fatalf("Failed to generate random key: %v", err)
	}

	enc, err := NewEncryptor(key)
	if err != nil {
		t.Errorf("NewEncryptor() with valid 32-byte key failed: %v", err)
	}
	if enc == nil {
		t.Error("NewEncryptor() returned nil encryptor")
	}
}

// TestNewEncryptor_InvalidKeyLength tests that NewEncryptor rejects keys that are not 32 bytes
func TestNewEncryptor_InvalidKeyLength(t *testing.T) {
	testCases := []struct {
		name      string
		keyLength int
	}{
		{"16 bytes", 16},
		{"24 bytes", 24},
		{"31 bytes", 31},
		{"33 bytes", 33},
		{"0 bytes", 0},
		{"64 bytes", 64},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			key := make([]byte, tc.keyLength)
			_, err := rand.Read(key)
			if err != nil && tc.keyLength > 0 {
				t.Fatalf("Failed to generate random key: %v", err)
			}

			enc, err := NewEncryptor(key)
			if err == nil {
				t.Errorf("NewEncryptor() with %d-byte key should have failed", tc.keyLength)
			}
			if enc != nil {
				t.Error("NewEncryptor() should return nil encryptor on error")
			}
			if !strings.Contains(err.Error(), "must be 32 bytes") {
				t.Errorf("Error message should mention '32 bytes', got: %v", err)
			}
		})
	}
}

// TestEncryptDecrypt_RoundTrip tests that encryption and decryption work correctly
func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		t.Fatalf("Failed to generate random key: %v", err)
	}

	enc, err := NewEncryptor(key)
	if err != nil {
		t.Fatalf("NewEncryptor() failed: %v", err)
	}

	testCases := []struct {
		name      string
		plaintext string
	}{
		{"simple password", "mySecurePassword123!"},
		{"long password", strings.Repeat("a", 100)},
		{"empty string", ""},
		{"special characters", "!@#$%^&*()_+-=[]{}|;:,.<>?"},
		{"unicode", "こんにちは世界🌍"},
		{"whitespace", "   spaces   \t\n"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Encrypt
			ciphertext, err := enc.Encrypt(tc.plaintext)
			if err != nil {
				t.Fatalf("Encrypt() failed: %v", err)
			}
			if ciphertext == "" {
				t.Error("Encrypt() returned empty ciphertext")
			}

			// Decrypt
			decrypted, err := enc.Decrypt(ciphertext)
			if err != nil {
				t.Fatalf("Decrypt() failed: %v", err)
			}

			// Verify round-trip
			if decrypted != tc.plaintext {
				t.Errorf("Round-trip failed: got %q, want %q", decrypted, tc.plaintext)
			}
		})
	}
}

// TestEncrypt_RandomNonce tests that each encryption produces a different ciphertext
// due to random nonce generation
func TestEncrypt_RandomNonce(t *testing.T) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		t.Fatalf("Failed to generate random key: %v", err)
	}

	enc, err := NewEncryptor(key)
	if err != nil {
		t.Fatalf("NewEncryptor() failed: %v", err)
	}

	plaintext := "testPassword123"
	ciphertexts := make(map[string]bool)

	// Encrypt the same plaintext 100 times
	for i := 0; i < 100; i++ {
		ciphertext, err := enc.Encrypt(plaintext)
		if err != nil {
			t.Fatalf("Encrypt() failed on iteration %d: %v", i, err)
		}

		if ciphertexts[ciphertext] {
			t.Errorf("Duplicate ciphertext found on iteration %d", i)
		}
		ciphertexts[ciphertext] = true
	}

	// Verify all ciphertexts are unique
	if len(ciphertexts) != 100 {
		t.Errorf("Expected 100 unique ciphertexts, got %d", len(ciphertexts))
	}

	// Verify all ciphertexts decrypt to the same plaintext
	for ciphertext := range ciphertexts {
		decrypted, err := enc.Decrypt(ciphertext)
		if err != nil {
			t.Errorf("Decrypt() failed: %v", err)
		}
		if decrypted != plaintext {
			t.Errorf("Decrypted text doesn't match: got %q, want %q", decrypted, plaintext)
		}
	}
}

// TestDecrypt_WrongKey tests that decryption fails with a different key
func TestDecrypt_WrongKey(t *testing.T) {
	// Create first encryptor and encrypt
	key1 := make([]byte, 32)
	_, err := rand.Read(key1)
	if err != nil {
		t.Fatalf("Failed to generate random key: %v", err)
	}

	enc1, err := NewEncryptor(key1)
	if err != nil {
		t.Fatalf("NewEncryptor() failed: %v", err)
	}

	plaintext := "secretPassword"
	ciphertext, err := enc1.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt() failed: %v", err)
	}

	// Create second encryptor with different key
	key2 := make([]byte, 32)
	_, err = rand.Read(key2)
	if err != nil {
		t.Fatalf("Failed to generate random key: %v", err)
	}

	enc2, err := NewEncryptor(key2)
	if err != nil {
		t.Fatalf("NewEncryptor() failed: %v", err)
	}

	// Try to decrypt with wrong key
	_, err = enc2.Decrypt(ciphertext)
	if err == nil {
		t.Error("Decrypt() with wrong key should have failed")
	}
	if !strings.Contains(err.Error(), "failed to decrypt") {
		t.Errorf("Error message should mention 'failed to decrypt', got: %v", err)
	}
}

// TestDecrypt_CorruptedCiphertext tests that decryption fails with corrupted data
func TestDecrypt_CorruptedCiphertext(t *testing.T) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		t.Fatalf("Failed to generate random key: %v", err)
	}

	enc, err := NewEncryptor(key)
	if err != nil {
		t.Fatalf("NewEncryptor() failed: %v", err)
	}

	testCases := []struct {
		name       string
		ciphertext string
		errorMsg   string
	}{
		{"invalid base64", "not-valid-base64!", "failed to decode"},
		{"empty string", "", "ciphertext too short"},
		{"too short", "YWJj", "ciphertext too short"},
		{"random data", "YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXo=", "failed to decrypt"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := enc.Decrypt(tc.ciphertext)
			if err == nil {
				t.Error("Decrypt() with corrupted ciphertext should have failed")
			}
			if !strings.Contains(err.Error(), tc.errorMsg) {
				t.Errorf("Error message should contain %q, got: %v", tc.errorMsg, err)
			}
		})
	}
}

// TestDecrypt_ModifiedCiphertext tests that decryption fails when ciphertext is modified
func TestDecrypt_ModifiedCiphertext(t *testing.T) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		t.Fatalf("Failed to generate random key: %v", err)
	}

	enc, err := NewEncryptor(key)
	if err != nil {
		t.Fatalf("NewEncryptor() failed: %v", err)
	}

	plaintext := "originalPassword"
	ciphertext, err := enc.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt() failed: %v", err)
	}

	// Modify the ciphertext by changing one character
	modified := ciphertext[:len(ciphertext)-1] + "X"

	_, err = enc.Decrypt(modified)
	if err == nil {
		t.Error("Decrypt() with modified ciphertext should have failed")
	}
}

// TestEncrypt_EmptyPlaintext tests encryption of empty string
func TestEncrypt_EmptyPlaintext(t *testing.T) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		t.Fatalf("Failed to generate random key: %v", err)
	}

	enc, err := NewEncryptor(key)
	if err != nil {
		t.Fatalf("NewEncryptor() failed: %v", err)
	}

	ciphertext, err := enc.Encrypt("")
	if err != nil {
		t.Fatalf("Encrypt() failed: %v", err)
	}

	decrypted, err := enc.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt() failed: %v", err)
	}

	if decrypted != "" {
		t.Errorf("Expected empty string, got %q", decrypted)
	}
}

// TestEncryptor_ConcurrentUse tests that the encryptor is safe for concurrent use
func TestEncryptor_ConcurrentUse(t *testing.T) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		t.Fatalf("Failed to generate random key: %v", err)
	}

	enc, err := NewEncryptor(key)
	if err != nil {
		t.Fatalf("NewEncryptor() failed: %v", err)
	}

	// Run multiple goroutines encrypting and decrypting
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			plaintext := "password" + string(rune(id))
			for j := 0; j < 100; j++ {
				ciphertext, err := enc.Encrypt(plaintext)
				if err != nil {
					t.Errorf("Encrypt() failed in goroutine %d: %v", id, err)
					done <- false
					return
				}

				decrypted, err := enc.Decrypt(ciphertext)
				if err != nil {
					t.Errorf("Decrypt() failed in goroutine %d: %v", id, err)
					done <- false
					return
				}

				if decrypted != plaintext {
					t.Errorf("Round-trip failed in goroutine %d: got %q, want %q", id, decrypted, plaintext)
					done <- false
					return
				}
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}
