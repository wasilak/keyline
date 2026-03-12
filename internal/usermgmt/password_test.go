package usermgmt

import (
	"strings"
	"testing"
)

func TestNewPasswordGenerator(t *testing.T) {
	tests := []struct {
		name           string
		length         int
		expectedLength int
	}{
		{
			name:           "default length when 0",
			length:         0,
			expectedLength: defaultPasswordLength,
		},
		{
			name:           "default length when negative",
			length:         -1,
			expectedLength: defaultPasswordLength,
		},
		{
			name:           "custom length",
			length:         64,
			expectedLength: 64,
		},
		{
			name:           "minimum length",
			length:         32,
			expectedLength: 32,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pg := NewPasswordGenerator(tt.length)
			if pg.length != tt.expectedLength {
				t.Errorf("NewPasswordGenerator() length = %d, want %d", pg.length, tt.expectedLength)
			}
		})
	}
}

func TestPasswordGenerator_Generate(t *testing.T) {
	t.Run("generates password with correct length", func(t *testing.T) {
		lengths := []int{32, 64, 128}
		for _, length := range lengths {
			pg := NewPasswordGenerator(length)
			password, err := pg.Generate()
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}
			if len(password) != length {
				t.Errorf("Generate() password length = %d, want %d", len(password), length)
			}
		}
	})

	t.Run("generates password with valid characters", func(t *testing.T) {
		pg := NewPasswordGenerator(100)
		password, err := pg.Generate()
		if err != nil {
			t.Fatalf("Generate() error = %v", err)
		}

		// Check that all characters are from the charset
		for _, char := range password {
			if !strings.ContainsRune(charset, char) {
				t.Errorf("Generate() password contains invalid character: %c", char)
			}
		}
	})

	t.Run("generates password with character set diversity", func(t *testing.T) {
		pg := NewPasswordGenerator(100)
		password, err := pg.Generate()
		if err != nil {
			t.Fatalf("Generate() error = %v", err)
		}

		// Check for presence of different character types
		hasLower := false
		hasUpper := false
		hasDigit := false
		hasSpecial := false

		for _, char := range password {
			switch {
			case char >= 'a' && char <= 'z':
				hasLower = true
			case char >= 'A' && char <= 'Z':
				hasUpper = true
			case char >= '0' && char <= '9':
				hasDigit = true
			case strings.ContainsRune("!@#$%^&*()-_=+[]{}|;:,.<>?", char):
				hasSpecial = true
			}
		}

		// With 100 characters, we should have at least some diversity
		// This is probabilistic but should pass consistently
		if !hasLower || !hasUpper || !hasDigit {
			t.Errorf("Generate() password lacks character diversity: hasLower=%v, hasUpper=%v, hasDigit=%v, hasSpecial=%v",
				hasLower, hasUpper, hasDigit, hasSpecial)
		}
	})

	t.Run("generates unique passwords", func(t *testing.T) {
		pg := NewPasswordGenerator(32)
		passwords := make(map[string]bool)
		iterations := 1000

		for i := 0; i < iterations; i++ {
			password, err := pg.Generate()
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}
			if passwords[password] {
				t.Errorf("Generate() produced duplicate password after %d iterations", i+1)
			}
			passwords[password] = true
		}

		if len(passwords) != iterations {
			t.Errorf("Generate() produced %d unique passwords, want %d", len(passwords), iterations)
		}
	})

	t.Run("generates different passwords on consecutive calls", func(t *testing.T) {
		pg := NewPasswordGenerator(32)
		password1, err := pg.Generate()
		if err != nil {
			t.Fatalf("Generate() error = %v", err)
		}

		password2, err := pg.Generate()
		if err != nil {
			t.Fatalf("Generate() error = %v", err)
		}

		if password1 == password2 {
			t.Errorf("Generate() produced identical passwords: %s", password1)
		}
	})
}

func TestPasswordGenerator_Generate_Randomness(t *testing.T) {
	t.Run("password has sufficient entropy", func(t *testing.T) {
		pg := NewPasswordGenerator(32)
		password, err := pg.Generate()
		if err != nil {
			t.Fatalf("Generate() error = %v", err)
		}

		// Count character frequency
		charCount := make(map[rune]int)
		for _, char := range password {
			charCount[char]++
		}

		// Check that no character appears too frequently
		// With 32 characters and ~90 possible characters, no char should appear > 5 times
		for char, count := range charCount {
			if count > 5 {
				t.Errorf("Generate() character %c appears %d times, which suggests poor randomness", char, count)
			}
		}
	})
}

func BenchmarkPasswordGenerator_Generate(b *testing.B) {
	pg := NewPasswordGenerator(32)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := pg.Generate()
		if err != nil {
			b.Fatalf("Generate() error = %v", err)
		}
	}
}

func BenchmarkPasswordGenerator_Generate_Long(b *testing.B) {
	pg := NewPasswordGenerator(128)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := pg.Generate()
		if err != nil {
			b.Fatalf("Generate() error = %v", err)
		}
	}
}
