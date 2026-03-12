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
			Enabled:      true,
			IssuerURL:    "https://example.com",
			ClientID:     "test-client",
			ClientSecret: "test-secret",
			RedirectURL:  "https://auth.example.com/callback",
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

func TestValidate_UserManagementMissingAdminUser(t *testing.T) {
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
			Backend:       "memory",
			EncryptionKey: "MTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTI=", // base64 encoded 32 bytes
		},
		UserManagement: UserMgmtConfig{
			Enabled: true,
		},
		Elasticsearch: ElasticsearchConfig{
			// Missing AdminUser
			AdminPassword: "admin-password",
			URL:           "https://elasticsearch:9200",
		},
	}

	err := Validate(cfg)
	if err == nil {
		t.Error("Expected error when user_management.enabled is true but admin_user is missing")
	}
	if !strings.Contains(err.Error(), "elasticsearch.admin_user is required") {
		t.Errorf("Expected error about admin_user, got: %v", err)
	}
}

func TestValidate_UserManagementMissingAdminPassword(t *testing.T) {
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
			Backend:       "memory",
			EncryptionKey: "MTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTI=", // base64 encoded 32 bytes
		},
		UserManagement: UserMgmtConfig{
			Enabled: true,
		},
		Elasticsearch: ElasticsearchConfig{
			AdminUser: "admin",
			// Missing AdminPassword
			URL: "https://elasticsearch:9200",
		},
	}

	err := Validate(cfg)
	if err == nil {
		t.Error("Expected error when user_management.enabled is true but admin_password is missing")
	}
	if !strings.Contains(err.Error(), "elasticsearch.admin_password is required") {
		t.Errorf("Expected error about admin_password, got: %v", err)
	}
}

func TestValidate_UserManagementMissingElasticsearchURL(t *testing.T) {
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
			Backend:       "memory",
			EncryptionKey: "MTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTI=", // base64 encoded 32 bytes
		},
		UserManagement: UserMgmtConfig{
			Enabled: true,
		},
		Elasticsearch: ElasticsearchConfig{
			AdminUser:     "admin",
			AdminPassword: "admin-password",
			// Missing URL
		},
	}

	err := Validate(cfg)
	if err == nil {
		t.Error("Expected error when user_management.enabled is true but elasticsearch.url is missing")
	}
	if !strings.Contains(err.Error(), "elasticsearch.url is required") {
		t.Errorf("Expected error about elasticsearch.url, got: %v", err)
	}
}

func TestValidate_UserManagementAllAdminCredentialsPresent(t *testing.T) {
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
			Backend:       "memory",
			EncryptionKey: "MTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTI=", // base64 encoded 32 bytes
		},
		UserManagement: UserMgmtConfig{
			Enabled: true,
		},
		Elasticsearch: ElasticsearchConfig{
			AdminUser:     "admin",
			AdminPassword: "admin-password",
			URL:           "https://elasticsearch:9200",
		},
		RoleMappings: []RoleMapping{
			{
				Claim:   "groups",
				Pattern: "admin",
				ESRoles: []string{"superuser"},
			},
		},
	}

	err := Validate(cfg)
	if err != nil {
		t.Errorf("Expected no error when all admin credentials are present, got: %v", err)
	}
}

func TestValidate_UserManagementDisabledNoAdminCredentialsRequired(t *testing.T) {
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
		UserManagement: UserMgmtConfig{
			Enabled: false,
		},
		Elasticsearch: ElasticsearchConfig{
			// No admin credentials needed when user management is disabled
			Users: []ESUser{
				{Username: "admin", Password: "admin-password"},
			},
		},
	}

	err := Validate(cfg)
	if err != nil {
		t.Errorf("Expected no error when user_management is disabled and static users are configured, got: %v", err)
	}
}

func TestValidate_EncryptionKeyMissing(t *testing.T) {
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
			// Missing EncryptionKey
		},
		UserManagement: UserMgmtConfig{
			Enabled: true,
		},
		Elasticsearch: ElasticsearchConfig{
			AdminUser:     "admin",
			AdminPassword: "admin-password",
			URL:           "https://elasticsearch:9200",
		},
		RoleMappings: []RoleMapping{
			{
				Claim:   "groups",
				Pattern: "admin",
				ESRoles: []string{"superuser"},
			},
		},
	}

	err := Validate(cfg)
	if err == nil {
		t.Error("Expected error when user_management.enabled is true but encryption_key is missing")
	}
	if !strings.Contains(err.Error(), "cache.encryption_key is required") {
		t.Errorf("Expected error about encryption_key, got: %v", err)
	}
}

func TestValidate_EncryptionKeyBase64Valid(t *testing.T) {
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
			Backend:       "memory",
			EncryptionKey: "MTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTI=", // base64 encoded 32 bytes
		},
		UserManagement: UserMgmtConfig{
			Enabled: true,
		},
		Elasticsearch: ElasticsearchConfig{
			AdminUser:     "admin",
			AdminPassword: "admin-password",
			URL:           "https://elasticsearch:9200",
		},
		RoleMappings: []RoleMapping{
			{
				Claim:   "groups",
				Pattern: "admin",
				ESRoles: []string{"superuser"},
			},
		},
	}

	err := Validate(cfg)
	if err != nil {
		t.Errorf("Expected no error with valid base64 encoded 32-byte encryption key, got: %v", err)
	}
}

func TestValidate_EncryptionKeyBase64TooShort(t *testing.T) {
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
			Backend:       "memory",
			EncryptionKey: "MTIzNDU2Nzg5MA==", // base64 encoded 10 bytes (too short)
		},
		UserManagement: UserMgmtConfig{
			Enabled: true,
		},
		Elasticsearch: ElasticsearchConfig{
			AdminUser:     "admin",
			AdminPassword: "admin-password",
			URL:           "https://elasticsearch:9200",
		},
		RoleMappings: []RoleMapping{
			{
				Claim:   "groups",
				Pattern: "admin",
				ESRoles: []string{"superuser"},
			},
		},
	}

	err := Validate(cfg)
	if err == nil {
		t.Error("Expected error when base64 decoded encryption key is too short")
	}
	if !strings.Contains(err.Error(), "must be exactly 32 bytes") {
		t.Errorf("Expected error about 32 bytes, got: %v", err)
	}
}

func TestValidate_EncryptionKeyRawStringValid(t *testing.T) {
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
			Backend:       "memory",
			EncryptionKey: "my-secret-key-is-exactly-32-byte", // raw 32 bytes (not valid base64)
		},
		UserManagement: UserMgmtConfig{
			Enabled: true,
		},
		Elasticsearch: ElasticsearchConfig{
			AdminUser:     "admin",
			AdminPassword: "admin-password",
			URL:           "https://elasticsearch:9200",
		},
		RoleMappings: []RoleMapping{
			{
				Claim:   "groups",
				Pattern: "admin",
				ESRoles: []string{"superuser"},
			},
		},
	}

	err := Validate(cfg)
	if err != nil {
		t.Errorf("Expected no error with valid raw 32-byte encryption key, got: %v", err)
	}
}

func TestValidate_EncryptionKeyRawStringTooShort(t *testing.T) {
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
			Backend:       "memory",
			EncryptionKey: "tooshort", // raw string less than 32 bytes
		},
		UserManagement: UserMgmtConfig{
			Enabled: true,
		},
		Elasticsearch: ElasticsearchConfig{
			AdminUser:     "admin",
			AdminPassword: "admin-password",
			URL:           "https://elasticsearch:9200",
		},
		RoleMappings: []RoleMapping{
			{
				Claim:   "groups",
				Pattern: "admin",
				ESRoles: []string{"superuser"},
			},
		},
	}

	err := Validate(cfg)
	if err == nil {
		t.Error("Expected error when raw string encryption key is too short")
	}
	if !strings.Contains(err.Error(), "must be exactly 32 bytes") {
		t.Errorf("Expected error about 32 bytes, got: %v", err)
	}
}

func TestValidate_EncryptionKeyRawStringTooLong(t *testing.T) {
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
			Backend:       "memory",
			EncryptionKey: "123456789012345678901234567890123", // raw string 33 bytes (too long)
		},
		UserManagement: UserMgmtConfig{
			Enabled: true,
		},
		Elasticsearch: ElasticsearchConfig{
			AdminUser:     "admin",
			AdminPassword: "admin-password",
			URL:           "https://elasticsearch:9200",
		},
		RoleMappings: []RoleMapping{
			{
				Claim:   "groups",
				Pattern: "admin",
				ESRoles: []string{"superuser"},
			},
		},
	}

	err := Validate(cfg)
	if err == nil {
		t.Error("Expected error when raw string encryption key is too long")
	}
	if !strings.Contains(err.Error(), "must be exactly 32 bytes") {
		t.Errorf("Expected error about 32 bytes, got: %v", err)
	}
}
