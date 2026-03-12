package mapper

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/yourusername/keyline/internal/config"
)

// NOTE: Tests for MapOIDCUser and MapLocalUser have been removed.
// These methods are replaced by the dynamic user management system.
// See internal/usermgmt package tests for the new implementation.

func TestGetESCredentials(t *testing.T) {
	tests := []struct {
		name             string
		config           *config.Config
		esUser           string
		expectedUsername string
		expectedPassword string
		expectedError    bool
	}{
		{
			name: "credentials found",
			config: &config.Config{
				Elasticsearch: config.ElasticsearchConfig{
					Users: []config.ESUser{
						{Username: "admin", Password: "admin_password"},
						{Username: "readonly", Password: "readonly_password"},
					},
				},
			},
			esUser:           "admin",
			expectedUsername: "admin",
			expectedPassword: "admin_password",
		},
		{
			name: "second user credentials found",
			config: &config.Config{
				Elasticsearch: config.ElasticsearchConfig{
					Users: []config.ESUser{
						{Username: "admin", Password: "admin_password"},
						{Username: "readonly", Password: "readonly_password"},
					},
				},
			},
			esUser:           "readonly",
			expectedUsername: "readonly",
			expectedPassword: "readonly_password",
		},
		{
			name: "credentials not found",
			config: &config.Config{
				Elasticsearch: config.ElasticsearchConfig{
					Users: []config.ESUser{
						{Username: "admin", Password: "admin_password"},
					},
				},
			},
			esUser:        "unknown",
			expectedError: true,
		},
		{
			name: "empty user list",
			config: &config.Config{
				Elasticsearch: config.ElasticsearchConfig{
					Users: []config.ESUser{},
				},
			},
			esUser:        "admin",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapper := NewCredentialMapper(tt.config)
			ctx := context.Background()

			username, password, err := mapper.GetESCredentials(ctx, tt.esUser)

			if tt.expectedError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if username != tt.expectedUsername {
				t.Errorf("expected username %q, got %q", tt.expectedUsername, username)
			}

			if password != tt.expectedPassword {
				t.Errorf("expected password %q, got %q", tt.expectedPassword, password)
			}
		})
	}
}

func TestEncodeESCredentials(t *testing.T) {
	tests := []struct {
		name     string
		username string
		password string
		expected string
	}{
		{
			name:     "basic encoding",
			username: "admin",
			password: "password",
			expected: "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:password")),
		},
		{
			name:     "special characters",
			username: "user@example.com",
			password: "p@ssw0rd!",
			expected: "Basic " + base64.StdEncoding.EncodeToString([]byte("user@example.com:p@ssw0rd!")),
		},
		{
			name:     "empty password",
			username: "admin",
			password: "",
			expected: "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapper := NewCredentialMapper(&config.Config{})

			result := mapper.EncodeESCredentials(tt.username, tt.password)

			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestGetESAuthorizationHeader(t *testing.T) {
	tests := []struct {
		name          string
		config        *config.Config
		esUser        string
		expectedError bool
	}{
		{
			name: "successful header generation",
			config: &config.Config{
				Elasticsearch: config.ElasticsearchConfig{
					Users: []config.ESUser{
						{Username: "admin", Password: "admin_password"},
					},
				},
			},
			esUser: "admin",
		},
		{
			name: "user not found",
			config: &config.Config{
				Elasticsearch: config.ElasticsearchConfig{
					Users: []config.ESUser{
						{Username: "admin", Password: "admin_password"},
					},
				},
			},
			esUser:        "unknown",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapper := NewCredentialMapper(tt.config)
			ctx := context.Background()

			header, err := mapper.GetESAuthorizationHeader(ctx, tt.esUser)

			if tt.expectedError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Verify header starts with "Basic "
			if len(header) < 6 || header[:6] != "Basic " {
				t.Errorf("expected header to start with 'Basic ', got %q", header)
			}

			// Verify header can be decoded
			encoded := header[6:]
			decoded, err := base64.StdEncoding.DecodeString(encoded)
			if err != nil {
				t.Errorf("failed to decode header: %v", err)
			}

			// Verify decoded format is username:password
			decodedStr := string(decoded)
			if len(decodedStr) == 0 || !contains(decodedStr, ":") {
				t.Errorf("decoded header has invalid format: %q", decodedStr)
			}
		})
	}
}

// NOTE: Tests for matchPattern have been removed.
// Pattern matching is now handled by RoleMapper in internal/usermgmt package.

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
