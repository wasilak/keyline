package mapper

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/yourusername/keyline/internal/config"
)

func TestMapOIDCUser(t *testing.T) {
	tests := []struct {
		name          string
		config        *config.Config
		claims        map[string]interface{}
		expectedUser  string
		expectedError bool
	}{
		{
			name: "exact match on email claim",
			config: &config.Config{
				OIDC: config.OIDCConfig{
					Mappings: []config.OIDCMapping{
						{Claim: "email", Pattern: "admin@example.com", ESUser: "admin"},
					},
				},
			},
			claims:       map[string]interface{}{"email": "admin@example.com"},
			expectedUser: "admin",
		},
		{
			name: "wildcard suffix match",
			config: &config.Config{
				OIDC: config.OIDCConfig{
					Mappings: []config.OIDCMapping{
						{Claim: "email", Pattern: "*@admin.example.com", ESUser: "admin"},
					},
				},
			},
			claims:       map[string]interface{}{"email": "user@admin.example.com"},
			expectedUser: "admin",
		},
		{
			name: "wildcard prefix match",
			config: &config.Config{
				OIDC: config.OIDCConfig{
					Mappings: []config.OIDCMapping{
						{Claim: "email", Pattern: "admin@*", ESUser: "admin"},
					},
				},
			},
			claims:       map[string]interface{}{"email": "admin@example.com"},
			expectedUser: "admin",
		},
		{
			name: "wildcard middle match",
			config: &config.Config{
				OIDC: config.OIDCConfig{
					Mappings: []config.OIDCMapping{
						{Claim: "email", Pattern: "admin@*.com", ESUser: "admin"},
					},
				},
			},
			claims:       map[string]interface{}{"email": "admin@example.com"},
			expectedUser: "admin",
		},
		{
			name: "first match wins",
			config: &config.Config{
				OIDC: config.OIDCConfig{
					Mappings: []config.OIDCMapping{
						{Claim: "email", Pattern: "*@admin.example.com", ESUser: "admin"},
						{Claim: "email", Pattern: "*@example.com", ESUser: "readonly"},
					},
				},
			},
			claims:       map[string]interface{}{"email": "user@admin.example.com"},
			expectedUser: "admin",
		},
		{
			name: "second mapping matches",
			config: &config.Config{
				OIDC: config.OIDCConfig{
					Mappings: []config.OIDCMapping{
						{Claim: "email", Pattern: "*@admin.example.com", ESUser: "admin"},
						{Claim: "email", Pattern: "*@example.com", ESUser: "readonly"},
					},
				},
			},
			claims:       map[string]interface{}{"email": "user@example.com"},
			expectedUser: "readonly",
		},
		{
			name: "default ES user when no match",
			config: &config.Config{
				OIDC: config.OIDCConfig{
					Mappings: []config.OIDCMapping{
						{Claim: "email", Pattern: "*@admin.example.com", ESUser: "admin"},
					},
					DefaultESUser: "readonly",
				},
			},
			claims:       map[string]interface{}{"email": "user@other.com"},
			expectedUser: "readonly",
		},
		{
			name: "error when no match and no default",
			config: &config.Config{
				OIDC: config.OIDCConfig{
					Mappings: []config.OIDCMapping{
						{Claim: "email", Pattern: "*@admin.example.com", ESUser: "admin"},
					},
				},
			},
			claims:        map[string]interface{}{"email": "user@other.com"},
			expectedError: true,
		},
		{
			name: "claim not found uses default",
			config: &config.Config{
				OIDC: config.OIDCConfig{
					Mappings: []config.OIDCMapping{
						{Claim: "email", Pattern: "*@example.com", ESUser: "admin"},
					},
					DefaultESUser: "readonly",
				},
			},
			claims:       map[string]interface{}{"sub": "12345"},
			expectedUser: "readonly",
		},
		{
			name: "non-string claim value uses default",
			config: &config.Config{
				OIDC: config.OIDCConfig{
					Mappings: []config.OIDCMapping{
						{Claim: "email", Pattern: "*@example.com", ESUser: "admin"},
					},
					DefaultESUser: "readonly",
				},
			},
			claims:       map[string]interface{}{"email": 12345},
			expectedUser: "readonly",
		},
		{
			name: "different claim name",
			config: &config.Config{
				OIDC: config.OIDCConfig{
					Mappings: []config.OIDCMapping{
						{Claim: "preferred_username", Pattern: "admin", ESUser: "admin"},
					},
				},
			},
			claims:       map[string]interface{}{"preferred_username": "admin"},
			expectedUser: "admin",
		},
		{
			name: "array claim with match",
			config: &config.Config{
				OIDC: config.OIDCConfig{
					Mappings: []config.OIDCMapping{
						{Claim: "groups", Pattern: "admin", ESUser: "admin"},
					},
				},
			},
			claims:       map[string]interface{}{"groups": []interface{}{"users", "admin", "developers"}},
			expectedUser: "admin",
		},
		{
			name: "array claim with wildcard match",
			config: &config.Config{
				OIDC: config.OIDCConfig{
					Mappings: []config.OIDCMapping{
						{Claim: "groups", Pattern: "*-admins", ESUser: "admin"},
					},
				},
			},
			claims:       map[string]interface{}{"groups": []interface{}{"users", "elasticsearch-admins", "developers"}},
			expectedUser: "admin",
		},
		{
			name: "array claim no match",
			config: &config.Config{
				OIDC: config.OIDCConfig{
					Mappings: []config.OIDCMapping{
						{Claim: "groups", Pattern: "admin", ESUser: "admin"},
					},
					DefaultESUser: "readonly",
				},
			},
			claims:       map[string]interface{}{"groups": []interface{}{"users", "developers"}},
			expectedUser: "readonly",
		},
		{
			name: "string array claim with match",
			config: &config.Config{
				OIDC: config.OIDCConfig{
					Mappings: []config.OIDCMapping{
						{Claim: "groups", Pattern: "admin", ESUser: "admin"},
					},
				},
			},
			claims:       map[string]interface{}{"groups": []string{"users", "admin", "developers"}},
			expectedUser: "admin",
		},
		{
			name: "array claim first match wins",
			config: &config.Config{
				OIDC: config.OIDCConfig{
					Mappings: []config.OIDCMapping{
						{Claim: "groups", Pattern: "admin", ESUser: "superuser"},
						{Claim: "groups", Pattern: "developers", ESUser: "dev_user"},
					},
				},
			},
			claims:       map[string]interface{}{"groups": []interface{}{"developers", "admin"}},
			expectedUser: "superuser",
		},
		{
			name: "empty array claim uses default",
			config: &config.Config{
				OIDC: config.OIDCConfig{
					Mappings: []config.OIDCMapping{
						{Claim: "groups", Pattern: "admin", ESUser: "admin"},
					},
					DefaultESUser: "readonly",
				},
			},
			claims:       map[string]interface{}{"groups": []interface{}{}},
			expectedUser: "readonly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapper := NewCredentialMapper(tt.config)
			ctx := context.Background()

			esUser, err := mapper.MapOIDCUser(ctx, tt.claims)

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

			if esUser != tt.expectedUser {
				t.Errorf("expected ES user %q, got %q", tt.expectedUser, esUser)
			}
		})
	}
}

func TestMapLocalUser(t *testing.T) {
	tests := []struct {
		name          string
		config        *config.Config
		username      string
		expectedUser  string
		expectedError bool
	}{
		{
			name: "user found",
			config: &config.Config{
				LocalUsers: config.LocalUsersConfig{
					Users: []config.LocalUser{
						{Username: "ci-pipeline"},
						{Username: "monitoring"},
					},
				},
			},
			username:     "ci-pipeline",
			expectedUser: "ci-pipeline", // Now returns username directly
		},
		{
			name: "second user found",
			config: &config.Config{
				LocalUsers: config.LocalUsersConfig{
					Users: []config.LocalUser{
						{Username: "ci-pipeline"},
						{Username: "monitoring"},
					},
				},
			},
			username:     "monitoring",
			expectedUser: "monitoring", // Now returns username directly
		},
		{
			name: "user not found",
			config: &config.Config{
				LocalUsers: config.LocalUsersConfig{
					Users: []config.LocalUser{
						{Username: "ci-pipeline"},
					},
				},
			},
			username:      "unknown",
			expectedUser:  "unknown", // Now returns username even if not in config
			expectedError: false,     // No longer an error - returns username
		},
		{
			name: "empty user list",
			config: &config.Config{
				LocalUsers: config.LocalUsersConfig{
					Users: []config.LocalUser{},
				},
			},
			username:      "ci-pipeline",
			expectedUser:  "ci-pipeline", // Now returns username even with empty list
			expectedError: false,         // No longer an error - returns username
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapper := NewCredentialMapper(tt.config)
			ctx := context.Background()

			esUser, err := mapper.MapLocalUser(ctx, tt.username)

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

			if esUser != tt.expectedUser {
				t.Errorf("expected ES user %q, got %q", tt.expectedUser, esUser)
			}
		})
	}
}

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

func TestMatchPattern(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		pattern  string
		expected bool
	}{
		{
			name:     "exact match",
			value:    "admin@example.com",
			pattern:  "admin@example.com",
			expected: true,
		},
		{
			name:     "no match",
			value:    "user@example.com",
			pattern:  "admin@example.com",
			expected: false,
		},
		{
			name:     "wildcard suffix match",
			value:    "user@admin.example.com",
			pattern:  "*@admin.example.com",
			expected: true,
		},
		{
			name:     "wildcard suffix no match",
			value:    "user@example.com",
			pattern:  "*@admin.example.com",
			expected: false,
		},
		{
			name:     "wildcard prefix match",
			value:    "admin@example.com",
			pattern:  "admin@*",
			expected: true,
		},
		{
			name:     "wildcard prefix no match",
			value:    "user@example.com",
			pattern:  "admin@*",
			expected: false,
		},
		{
			name:     "wildcard middle match",
			value:    "admin@example.com",
			pattern:  "admin@*.com",
			expected: true,
		},
		{
			name:     "wildcard middle no match prefix",
			value:    "user@example.com",
			pattern:  "admin@*.com",
			expected: false,
		},
		{
			name:     "wildcard middle no match suffix",
			value:    "admin@example.org",
			pattern:  "admin@*.com",
			expected: false,
		},
		{
			name:     "wildcard only",
			value:    "anything",
			pattern:  "*",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapper := NewCredentialMapper(&config.Config{})

			result := mapper.matchPattern(tt.value, tt.pattern)

			if result != tt.expected {
				t.Errorf("matchPattern(%q, %q) = %v, expected %v", tt.value, tt.pattern, result, tt.expected)
			}
		})
	}
}

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
