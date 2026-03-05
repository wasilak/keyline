package config

import (
	"strings"
	"testing"
	"time"
)

func TestValidate_Success(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{
			Port: 9000,
			Mode: "forward_auth",
		},
		OIDC: OIDCConfig{
			Enabled:       true,
			IssuerURL:     "https://example.com",
			ClientID:      "test-client",
			ClientSecret:  "test-secret",
			RedirectURL:   "https://auth.example.com/callback",
			DefaultESUser: "readonly",
		},
		LocalUsers: LocalUsersConfig{
			Enabled: false,
		},
		Session: SessionConfig{
			TTL:           24 * time.Hour,
			SessionSecret: "this-is-a-very-long-secret-key-that-is-at-least-32-bytes-long",
		},
		Cache: CacheConfig{
			Backend: "memory",
		},
		Elasticsearch: ElasticsearchConfig{
			Users: []ESUser{
				{Username: "admin", Password: "admin-password"},
			},
		},
		Upstream: UpstreamConfig{
			URL: "http://kibana:5601",
		},
	}

	err := Validate(cfg)
	if err != nil {
		t.Errorf("Validate() failed for valid config: %v", err)
	}
}

func TestValidate_NoAuthMethodEnabled(t *testing.T) {
	cfg := &Config{
		OIDC:       OIDCConfig{Enabled: false},
		LocalUsers: LocalUsersConfig{Enabled: false},
		Session: SessionConfig{
			SessionSecret: "this-is-a-very-long-secret-key-that-is-at-least-32-bytes-long",
		},
		Cache: CacheConfig{Backend: "memory"},
		Elasticsearch: ElasticsearchConfig{
			Users: []ESUser{{Username: "admin", Password: "password"}},
		},
	}

	err := Validate(cfg)
	if err == nil {
		t.Error("Expected error when no authentication method is enabled")
	}
	if !strings.Contains(err.Error(), "at least one authentication method") {
		t.Errorf("Expected error about authentication method, got: %v", err)
	}
}

func TestValidate_OIDCMissingIssuerURL(t *testing.T) {
	cfg := &Config{
		OIDC: OIDCConfig{
			Enabled:      true,
			ClientID:     "test-client",
			ClientSecret: "test-secret",
			RedirectURL:  "https://auth.example.com/callback",
		},
		Session: SessionConfig{
			SessionSecret: "this-is-a-very-long-secret-key-that-is-at-least-32-bytes-long",
		},
		Cache: CacheConfig{Backend: "memory"},
		Elasticsearch: ElasticsearchConfig{
			Users: []ESUser{{Username: "admin", Password: "password"}},
		},
	}

	err := Validate(cfg)
	if err == nil {
		t.Error("Expected error when OIDC issuer_url is missing")
	}
	if !strings.Contains(err.Error(), "issuer_url is required") {
		t.Errorf("Expected error about issuer_url, got: %v", err)
	}
}

func TestValidate_OIDCMissingClientID(t *testing.T) {
	cfg := &Config{
		OIDC: OIDCConfig{
			Enabled:      true,
			IssuerURL:    "https://example.com",
			ClientSecret: "test-secret",
			RedirectURL:  "https://auth.example.com/callback",
		},
		Session: SessionConfig{
			SessionSecret: "this-is-a-very-long-secret-key-that-is-at-least-32-bytes-long",
		},
		Cache: CacheConfig{Backend: "memory"},
		Elasticsearch: ElasticsearchConfig{
			Users: []ESUser{{Username: "admin", Password: "password"}},
		},
	}

	err := Validate(cfg)
	if err == nil {
		t.Error("Expected error when OIDC client_id is missing")
	}
	if !strings.Contains(err.Error(), "client_id is required") {
		t.Errorf("Expected error about client_id, got: %v", err)
	}
}

func TestValidate_OIDCMissingClientSecret(t *testing.T) {
	cfg := &Config{
		OIDC: OIDCConfig{
			Enabled:     true,
			IssuerURL:   "https://example.com",
			ClientID:    "test-client",
			RedirectURL: "https://auth.example.com/callback",
		},
		Session: SessionConfig{
			SessionSecret: "this-is-a-very-long-secret-key-that-is-at-least-32-bytes-long",
		},
		Cache: CacheConfig{Backend: "memory"},
		Elasticsearch: ElasticsearchConfig{
			Users: []ESUser{{Username: "admin", Password: "password"}},
		},
	}

	err := Validate(cfg)
	if err == nil {
		t.Error("Expected error when OIDC client_secret is missing")
	}
	if !strings.Contains(err.Error(), "client_secret is required") {
		t.Errorf("Expected error about client_secret, got: %v", err)
	}
}

func TestValidate_OIDCMissingRedirectURL(t *testing.T) {
	cfg := &Config{
		OIDC: OIDCConfig{
			Enabled:      true,
			IssuerURL:    "https://example.com",
			ClientID:     "test-client",
			ClientSecret: "test-secret",
		},
		Session: SessionConfig{
			SessionSecret: "this-is-a-very-long-secret-key-that-is-at-least-32-bytes-long",
		},
		Cache: CacheConfig{Backend: "memory"},
		Elasticsearch: ElasticsearchConfig{
			Users: []ESUser{{Username: "admin", Password: "password"}},
		},
	}

	err := Validate(cfg)
	if err == nil {
		t.Error("Expected error when OIDC redirect_url is missing")
	}
	if !strings.Contains(err.Error(), "redirect_url is required") {
		t.Errorf("Expected error about redirect_url, got: %v", err)
	}
}

func TestValidate_OIDCRedirectURLNotHTTPS(t *testing.T) {
	cfg := &Config{
		OIDC: OIDCConfig{
			Enabled:      true,
			IssuerURL:    "https://example.com",
			ClientID:     "test-client",
			ClientSecret: "test-secret",
			RedirectURL:  "http://auth.example.com/callback",
		},
		Session: SessionConfig{
			SessionSecret: "this-is-a-very-long-secret-key-that-is-at-least-32-bytes-long",
		},
		Cache: CacheConfig{Backend: "memory"},
		Elasticsearch: ElasticsearchConfig{
			Users: []ESUser{{Username: "admin", Password: "password"}},
		},
	}

	err := Validate(cfg)
	if err == nil {
		t.Error("Expected error when OIDC redirect_url is not HTTPS")
	}
	if !strings.Contains(err.Error(), "must use HTTPS") {
		t.Errorf("Expected error about HTTPS, got: %v", err)
	}
}

func TestValidate_OIDCRedirectURLInvalid(t *testing.T) {
	cfg := &Config{
		OIDC: OIDCConfig{
			Enabled:      true,
			IssuerURL:    "https://example.com",
			ClientID:     "test-client",
			ClientSecret: "test-secret",
			RedirectURL:  "not-a-valid-url",
		},
		Session: SessionConfig{
			SessionSecret: "this-is-a-very-long-secret-key-that-is-at-least-32-bytes-long",
		},
		Cache: CacheConfig{Backend: "memory"},
		Elasticsearch: ElasticsearchConfig{
			Users: []ESUser{{Username: "admin", Password: "password"}},
		},
	}

	err := Validate(cfg)
	if err == nil {
		t.Error("Expected error when OIDC redirect_url is invalid")
	}
}

func TestValidate_LocalUsersNoUsers(t *testing.T) {
	cfg := &Config{
		LocalUsers: LocalUsersConfig{
			Enabled: true,
			Users:   []LocalUser{},
		},
		Session: SessionConfig{
			SessionSecret: "this-is-a-very-long-secret-key-that-is-at-least-32-bytes-long",
		},
		Cache: CacheConfig{Backend: "memory"},
		Elasticsearch: ElasticsearchConfig{
			Users: []ESUser{{Username: "admin", Password: "password"}},
		},
	}

	err := Validate(cfg)
	if err == nil {
		t.Error("Expected error when local users enabled but no users configured")
	}
	if !strings.Contains(err.Error(), "at least one local user") {
		t.Errorf("Expected error about local users, got: %v", err)
	}
}

func TestValidate_LocalUserMissingUsername(t *testing.T) {
	cfg := &Config{
		LocalUsers: LocalUsersConfig{
			Enabled: true,
			Users: []LocalUser{
				{
					PasswordBcrypt: "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy",
					ESUser:         "test",
				},
			},
		},
		Session: SessionConfig{
			SessionSecret: "this-is-a-very-long-secret-key-that-is-at-least-32-bytes-long",
		},
		Cache: CacheConfig{Backend: "memory"},
		Elasticsearch: ElasticsearchConfig{
			Users: []ESUser{{Username: "admin", Password: "password"}},
		},
	}

	err := Validate(cfg)
	if err == nil {
		t.Error("Expected error when local user username is missing")
	}
	if !strings.Contains(err.Error(), "username is required") {
		t.Errorf("Expected error about username, got: %v", err)
	}
}

func TestValidate_LocalUserInvalidBcrypt(t *testing.T) {
	cfg := &Config{
		LocalUsers: LocalUsersConfig{
			Enabled: true,
			Users: []LocalUser{
				{
					Username:       "test",
					PasswordBcrypt: "not-a-bcrypt-hash",
					ESUser:         "test",
				},
			},
		},
		Session: SessionConfig{
			SessionSecret: "this-is-a-very-long-secret-key-that-is-at-least-32-bytes-long",
		},
		Cache: CacheConfig{Backend: "memory"},
		Elasticsearch: ElasticsearchConfig{
			Users: []ESUser{{Username: "admin", Password: "password"}},
		},
	}

	err := Validate(cfg)
	if err == nil {
		t.Error("Expected error when local user password_bcrypt is invalid")
	}
	if !strings.Contains(err.Error(), "not a valid bcrypt hash") {
		t.Errorf("Expected error about bcrypt hash, got: %v", err)
	}
}

func TestValidate_SessionSecretTooShort(t *testing.T) {
	cfg := &Config{
		OIDC: OIDCConfig{
			Enabled:      true,
			IssuerURL:    "https://example.com",
			ClientID:     "test-client",
			ClientSecret: "test-secret",
			RedirectURL:  "https://auth.example.com/callback",
		},
		Session: SessionConfig{
			SessionSecret: "short",
		},
		Cache: CacheConfig{Backend: "memory"},
		Elasticsearch: ElasticsearchConfig{
			Users: []ESUser{{Username: "admin", Password: "password"}},
		},
	}

	err := Validate(cfg)
	if err == nil {
		t.Error("Expected error when session secret is too short")
	}
	if !strings.Contains(err.Error(), "at least 32 bytes") {
		t.Errorf("Expected error about session secret length, got: %v", err)
	}
}

func TestValidate_SessionSecretMissing(t *testing.T) {
	cfg := &Config{
		OIDC: OIDCConfig{
			Enabled:      true,
			IssuerURL:    "https://example.com",
			ClientID:     "test-client",
			ClientSecret: "test-secret",
			RedirectURL:  "https://auth.example.com/callback",
		},
		Session: SessionConfig{
			SessionSecret: "",
		},
		Cache: CacheConfig{Backend: "memory"},
		Elasticsearch: ElasticsearchConfig{
			Users: []ESUser{{Username: "admin", Password: "password"}},
		},
	}

	err := Validate(cfg)
	if err == nil {
		t.Error("Expected error when session secret is missing")
	}
	if !strings.Contains(err.Error(), "session_secret is required") {
		t.Errorf("Expected error about session secret, got: %v", err)
	}
}

func TestValidate_CacheBackendMissing(t *testing.T) {
	cfg := &Config{
		OIDC: OIDCConfig{
			Enabled:      true,
			IssuerURL:    "https://example.com",
			ClientID:     "test-client",
			ClientSecret: "test-secret",
			RedirectURL:  "https://auth.example.com/callback",
		},
		Session: SessionConfig{
			SessionSecret: "this-is-a-very-long-secret-key-that-is-at-least-32-bytes-long",
		},
		Cache: CacheConfig{},
		Elasticsearch: ElasticsearchConfig{
			Users: []ESUser{{Username: "admin", Password: "password"}},
		},
	}

	err := Validate(cfg)
	if err == nil {
		t.Error("Expected error when cache backend is missing")
	}
	if !strings.Contains(err.Error(), "cache.backend is required") {
		t.Errorf("Expected error about cache backend, got: %v", err)
	}
}

func TestValidate_CacheBackendInvalid(t *testing.T) {
	cfg := &Config{
		OIDC: OIDCConfig{
			Enabled:      true,
			IssuerURL:    "https://example.com",
			ClientID:     "test-client",
			ClientSecret: "test-secret",
			RedirectURL:  "https://auth.example.com/callback",
		},
		Session: SessionConfig{
			SessionSecret: "this-is-a-very-long-secret-key-that-is-at-least-32-bytes-long",
		},
		Cache: CacheConfig{
			Backend: "invalid",
		},
		Elasticsearch: ElasticsearchConfig{
			Users: []ESUser{{Username: "admin", Password: "password"}},
		},
	}

	err := Validate(cfg)
	if err == nil {
		t.Error("Expected error when cache backend is invalid")
	}
	if !strings.Contains(err.Error(), "must be either 'redis' or 'memory'") {
		t.Errorf("Expected error about cache backend values, got: %v", err)
	}
}

func TestValidate_RedisMissingURL(t *testing.T) {
	cfg := &Config{
		OIDC: OIDCConfig{
			Enabled:      true,
			IssuerURL:    "https://example.com",
			ClientID:     "test-client",
			ClientSecret: "test-secret",
			RedirectURL:  "https://auth.example.com/callback",
		},
		Session: SessionConfig{
			SessionSecret: "this-is-a-very-long-secret-key-that-is-at-least-32-bytes-long",
		},
		Cache: CacheConfig{
			Backend: "redis",
		},
		Elasticsearch: ElasticsearchConfig{
			Users: []ESUser{{Username: "admin", Password: "password"}},
		},
	}

	err := Validate(cfg)
	if err == nil {
		t.Error("Expected error when Redis URL is missing")
	}
	if !strings.Contains(err.Error(), "redis_url is required") {
		t.Errorf("Expected error about redis_url, got: %v", err)
	}
}

func TestValidate_NoElasticsearchUsers(t *testing.T) {
	cfg := &Config{
		OIDC: OIDCConfig{
			Enabled:      true,
			IssuerURL:    "https://example.com",
			ClientID:     "test-client",
			ClientSecret: "test-secret",
			RedirectURL:  "https://auth.example.com/callback",
		},
		Session: SessionConfig{
			SessionSecret: "this-is-a-very-long-secret-key-that-is-at-least-32-bytes-long",
		},
		Cache: CacheConfig{
			Backend: "memory",
		},
		Elasticsearch: ElasticsearchConfig{
			Users: []ESUser{},
		},
	}

	err := Validate(cfg)
	if err == nil {
		t.Error("Expected error when no Elasticsearch users configured")
	}
	if !strings.Contains(err.Error(), "at least one Elasticsearch user") {
		t.Errorf("Expected error about Elasticsearch users, got: %v", err)
	}
}

func TestValidate_StandaloneMissingUpstreamURL(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{
			Mode: "standalone",
		},
		OIDC: OIDCConfig{
			Enabled:      true,
			IssuerURL:    "https://example.com",
			ClientID:     "test-client",
			ClientSecret: "test-secret",
			RedirectURL:  "https://auth.example.com/callback",
		},
		Session: SessionConfig{
			SessionSecret: "this-is-a-very-long-secret-key-that-is-at-least-32-bytes-long",
		},
		Cache: CacheConfig{
			Backend: "memory",
		},
		Elasticsearch: ElasticsearchConfig{
			Users: []ESUser{{Username: "admin", Password: "password"}},
		},
		Upstream: UpstreamConfig{},
	}

	err := Validate(cfg)
	if err == nil {
		t.Error("Expected error when standalone mode missing upstream URL")
	}
	if !strings.Contains(err.Error(), "upstream.url is required") {
		t.Errorf("Expected error about upstream URL, got: %v", err)
	}
}

func TestValidate_MultipleErrors(t *testing.T) {
	cfg := &Config{
		OIDC: OIDCConfig{
			Enabled: true,
			// Missing all required fields
		},
		LocalUsers: LocalUsersConfig{
			Enabled: false,
		},
		Session: SessionConfig{
			SessionSecret: "short",
		},
		Cache: CacheConfig{},
		Elasticsearch: ElasticsearchConfig{
			Users: []ESUser{},
		},
	}

	err := Validate(cfg)
	if err == nil {
		t.Error("Expected multiple validation errors")
	}

	// Check that error message contains multiple errors
	errMsg := err.Error()
	if !strings.Contains(errMsg, "issuer_url") {
		t.Error("Expected error about issuer_url")
	}
	if !strings.Contains(errMsg, "client_id") {
		t.Error("Expected error about client_id")
	}
	if !strings.Contains(errMsg, "client_secret") {
		t.Error("Expected error about client_secret")
	}
	if !strings.Contains(errMsg, "session_secret") {
		t.Error("Expected error about session_secret")
	}
	if !strings.Contains(errMsg, "cache.backend") {
		t.Error("Expected error about cache backend")
	}
	if !strings.Contains(errMsg, "Elasticsearch user") {
		t.Error("Expected error about Elasticsearch users")
	}
}
