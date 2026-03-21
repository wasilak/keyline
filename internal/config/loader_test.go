package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_Success(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	configContent := `
server:
  port: 9000
  mode: forward_auth
  read_timeout: 30s
  write_timeout: 30s
  max_concurrent: 1000

oidc:
  enabled: true
  issuer_url: https://example.com
  client_id: test-client
  client_secret: test-secret
  redirect_url: https://auth.example.com/auth/callback
  scopes:
    - openid
    - email
  default_es_user: readonly

local_users:
  enabled: false
  users: []

session:
  ttl: 24h
  cookie_name: keyline_session
  cookie_domain: .example.com
  cookie_path: /
  session_secret: this-is-a-very-long-secret-key-that-is-at-least-32-bytes-long

cache:
  backend: memory

elasticsearch:
  users:
    - username: admin
      password: admin-password

upstream:
  url: http://kibana:5601
  timeout: 30s
  max_idle_conns: 100

observability:
  log_level: info
  log_format: json
  otel_enabled: false
  metrics_enabled: true
`

	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Load configuration
	cfg, err := Load(configFile)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Verify configuration values
	if cfg.Server.Port != 9000 {
		t.Errorf("Expected port 9000, got %d", cfg.Server.Port)
	}
	if cfg.Server.Mode != "forward_auth" {
		t.Errorf("Expected mode forward_auth, got %s", cfg.Server.Mode)
	}
	if !cfg.OIDC.Enabled {
		t.Error("Expected OIDC to be enabled")
	}
	if cfg.OIDC.IssuerURL != "https://example.com" {
		t.Errorf("Expected issuer URL https://example.com, got %s", cfg.OIDC.IssuerURL)
	}
	if cfg.Cache.Backend != "memory" {
		t.Errorf("Expected cache backend memory, got %s", cfg.Cache.Backend)
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/config.yaml")
	if err == nil {
		t.Error("Expected error for nonexistent config file")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	invalidYAML := `
server:
  port: not-a-number
  invalid yaml structure
`

	if err := os.WriteFile(configFile, []byte(invalidYAML), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	_, err := Load(configFile)
	if err == nil {
		t.Error("Expected error for invalid YAML")
	}
}

func TestLoad_EnvVarSubstitution(t *testing.T) {
	// Set environment variables
	os.Setenv("TEST_ISSUER_URL", "https://test-issuer.com")
	os.Setenv("TEST_CLIENT_ID", "test-client-id")
	os.Setenv("TEST_CLIENT_SECRET", "test-client-secret")
	os.Setenv("TEST_REDIRECT_URL", "https://test-redirect.com/callback")
	os.Setenv("TEST_SESSION_SECRET", "test-session-secret-that-is-at-least-32-bytes")
	os.Setenv("TEST_ES_ADMIN_USER", "test-es-admin")
	os.Setenv("TEST_ES_ADMIN_PASSWORD", "test-es-admin-password")
	defer func() {
		os.Unsetenv("TEST_ISSUER_URL")
		os.Unsetenv("TEST_CLIENT_ID")
		os.Unsetenv("TEST_CLIENT_SECRET")
		os.Unsetenv("TEST_REDIRECT_URL")
		os.Unsetenv("TEST_SESSION_SECRET")
		os.Unsetenv("TEST_ES_ADMIN_USER")
		os.Unsetenv("TEST_ES_ADMIN_PASSWORD")
	}()

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	configContent := `
server:
  port: 9000
  mode: forward_auth

oidc:
  enabled: true
  issuer_url: ${TEST_ISSUER_URL}
  client_id: ${TEST_CLIENT_ID}
  client_secret: ${TEST_CLIENT_SECRET}
  redirect_url: ${TEST_REDIRECT_URL}
  default_es_user: readonly

local_users:
  enabled: false

session:
  ttl: 24h
  session_secret: ${TEST_SESSION_SECRET}

cache:
  backend: memory
  encryption_key: ${TEST_SESSION_SECRET}

role_mappings:
  - claim: groups
    pattern: "admin"
    es_roles:
      - superuser

default_es_roles:
  - viewer

elasticsearch:
  admin_user: ${TEST_ES_ADMIN_USER}
  admin_password: ${TEST_ES_ADMIN_PASSWORD}
  url: http://elasticsearch:9200

upstream:
  url: http://kibana:5601

observability:
  log_level: info
`

	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := Load(configFile)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Verify environment variable substitution
	if cfg.OIDC.IssuerURL != "https://test-issuer.com" {
		t.Errorf("Expected issuer URL https://test-issuer.com, got %s", cfg.OIDC.IssuerURL)
	}
	if cfg.OIDC.ClientID != "test-client-id" {
		t.Errorf("Expected client ID test-client-id, got %s", cfg.OIDC.ClientID)
	}
	if cfg.OIDC.ClientSecret != "test-client-secret" {
		t.Errorf("Expected client secret test-client-secret, got %s", cfg.OIDC.ClientSecret)
	}
	if cfg.OIDC.RedirectURL != "https://test-redirect.com/callback" {
		t.Errorf("Expected redirect URL https://test-redirect.com/callback, got %s", cfg.OIDC.RedirectURL)
	}
	if cfg.Session.SessionSecret != "test-session-secret-that-is-at-least-32-bytes" {
		t.Errorf("Expected session secret test-session-secret-that-is-at-least-32-bytes, got %s", cfg.Session.SessionSecret)
	}
	if cfg.Elasticsearch.AdminUser != "test-es-admin" {
		t.Errorf("Expected ES admin user test-es-admin, got %s", cfg.Elasticsearch.AdminUser)
	}
}

func TestLoad_MissingEnvVar(t *testing.T) {
	// Ensure environment variable is not set
	os.Unsetenv("MISSING_VAR")

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	configContent := `
server:
  port: 9000

oidc:
  enabled: true
  issuer_url: ${MISSING_VAR}
  client_id: test
  client_secret: test
  redirect_url: https://test.com

local_users:
  enabled: false

session:
  session_secret: test-secret-that-is-at-least-32-bytes

cache:
  backend: memory

elasticsearch:
  users:
    - username: admin
      password: password

observability:
  log_level: info
`

	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	_, err := Load(configFile)
	if err == nil {
		t.Error("Expected error for missing environment variable")
	}
	if err != nil && !contains(err.Error(), "MISSING_VAR") {
		t.Errorf("Expected error message to mention MISSING_VAR, got: %v", err)
	}
}

func TestLoad_UnclosedEnvVar(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	configContent := `
server:
  port: 9000

oidc:
  enabled: true
  issuer_url: ${UNCLOSED_VAR
  client_id: test
  client_secret: test
  redirect_url: https://test.com

local_users:
  enabled: false

session:
  session_secret: test-secret-that-is-at-least-32-bytes

cache:
  backend: memory

elasticsearch:
  users:
    - username: admin
      password: password

observability:
  log_level: info
`

	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	_, err := Load(configFile)
	if err == nil {
		t.Error("Expected error for unclosed environment variable")
	}
	if err != nil && !contains(err.Error(), "unclosed") {
		t.Errorf("Expected error message to mention unclosed, got: %v", err)
	}
}

func TestLoad_ConfigFileEnvVar(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	configContent := `
server:
  port: 9000
  mode: forward_auth

oidc:
  enabled: false

local_users:
  enabled: true
  users:
    - username: test
      password_bcrypt: $2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy
      es_user: test

session:
  session_secret: test-secret-that-is-at-least-32-bytes

cache:
  backend: memory

elasticsearch:
  users:
    - username: admin
      password: password

observability:
  log_level: info
`

	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Set CONFIG_FILE environment variable
	os.Setenv("CONFIG_FILE", configFile)
	defer os.Unsetenv("CONFIG_FILE")

	// Load without specifying config file (should use CONFIG_FILE env var)
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.Server.Port != 9000 {
		t.Errorf("Expected port 9000, got %d", cfg.Server.Port)
	}
}

func TestSubstituteString(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		envVars   map[string]string
		expected  string
		expectErr bool
	}{
		{
			name:     "no substitution",
			input:    "plain-string",
			expected: "plain-string",
		},
		{
			name:     "single substitution",
			input:    "${VAR1}",
			envVars:  map[string]string{"VAR1": "value1"},
			expected: "value1",
		},
		{
			name:     "multiple substitutions",
			input:    "${VAR1}-${VAR2}",
			envVars:  map[string]string{"VAR1": "value1", "VAR2": "value2"},
			expected: "value1-value2",
		},
		{
			name:     "substitution with prefix and suffix",
			input:    "prefix-${VAR1}-suffix",
			envVars:  map[string]string{"VAR1": "value1"},
			expected: "prefix-value1-suffix",
		},
		{
			name:      "missing environment variable",
			input:     "${MISSING}",
			expectErr: true,
		},
		{
			name:      "unclosed substitution",
			input:     "${UNCLOSED",
			expectErr: true,
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for k, v := range tt.envVars {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			// Test substitution
			result := tt.input
			err := substituteString(&result)

			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("Expected %q, got %q", tt.expected, result)
				}
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
