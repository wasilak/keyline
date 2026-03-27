package config

import (
	"time"
)

// Config represents the complete Keyline configuration
type Config struct {
	Server         ServerConfig        `mapstructure:"server"`
	OIDC           OIDCConfig          `mapstructure:"oidc"`
	LocalUsers     LocalUsersConfig    `mapstructure:"local_users"`
	LDAP           LDAPConfig          `mapstructure:"ldap"`
	Session        SessionConfig       `mapstructure:"session"`
	Cache          CacheConfig         `mapstructure:"cache"`
	Elasticsearch  ElasticsearchConfig `mapstructure:"elasticsearch"`
	Upstream       UpstreamConfig      `mapstructure:"upstream"`
	Observability  ObservabilityConfig `mapstructure:"observability"`
	RoleMappings   []RoleMapping       `mapstructure:"role_mappings"`
	DefaultESRoles []string            `mapstructure:"default_es_roles"`
	UserManagement UserMgmtConfig      `mapstructure:"user_management"`
}

// ServerConfig contains server settings
type ServerConfig struct {
	Port          int           `mapstructure:"port"`
	Mode          string        `mapstructure:"mode"` // forward_auth, standalone
	ReadTimeout   time.Duration `mapstructure:"read_timeout"`
	WriteTimeout  time.Duration `mapstructure:"write_timeout"`
	MaxConcurrent int           `mapstructure:"max_concurrent"`
}

// OIDCConfig contains OIDC provider settings
type OIDCConfig struct {
	Enabled       bool          `mapstructure:"enabled"`
	IssuerURL     string        `mapstructure:"issuer_url"`
	ClientID      string        `mapstructure:"client_id"`
	ClientSecret  string        `mapstructure:"client_secret"`
	RedirectURL   string        `mapstructure:"redirect_url"`
	Scopes        []string      `mapstructure:"scopes"`
	Mappings      []OIDCMapping `mapstructure:"mappings"`
	DefaultESUser string        `mapstructure:"default_es_user"`
}

// OIDCMapping maps OIDC claims to ES users
type OIDCMapping struct {
	Claim   string `mapstructure:"claim"`
	Pattern string `mapstructure:"pattern"`
	ESUser  string `mapstructure:"es_user"`
}

// RoleMapping maps user groups/claims to Elasticsearch roles
// Used for dynamic user management across all authentication methods
type RoleMapping struct {
	Claim   string   `mapstructure:"claim"`    // Claim name (e.g., "groups", "email")
	Pattern string   `mapstructure:"pattern"`  // Pattern to match (supports wildcards)
	ESRoles []string `mapstructure:"es_roles"` // ES roles to assign when pattern matches
}

// LocalUsersConfig contains local user settings
type LocalUsersConfig struct {
	Enabled bool        `mapstructure:"enabled"`
	Users   []LocalUser `mapstructure:"users"`
}

// LocalUser represents a local user
type LocalUser struct {
	Username       string   `mapstructure:"username"`
	PasswordBcrypt string   `mapstructure:"password_bcrypt"`
	Groups         []string `mapstructure:"groups"`
	Email          string   `mapstructure:"email"`
	FullName       string   `mapstructure:"full_name"`
}

// LDAPConfig contains LDAP authentication settings
type LDAPConfig struct {
	Enabled bool `mapstructure:"enabled"`

	// Server connection
	URL               string        `mapstructure:"url"`                // ldap:// or ldaps://
	BindDN            string        `mapstructure:"bind_dn"`            // Service account DN
	BindPassword      string        `mapstructure:"bind_password"`      // Service account password
	ConnectionTimeout time.Duration `mapstructure:"connection_timeout"` // Default: 10s

	// TLS
	TLSMode       string `mapstructure:"tls_mode"`        // "none", "ldaps", "starttls"
	TLSSkipVerify bool   `mapstructure:"tls_skip_verify"` // Skip cert verification (dev only)

	// User search
	SearchBase   string `mapstructure:"search_base"`   // e.g. "DC=example,DC=com"
	SearchFilter string `mapstructure:"search_filter"` // e.g. "(sAMAccountName={username})"

	// Group search (optional — omit both to skip group fetching)
	GroupSearchBase   string `mapstructure:"group_search_base"`   // e.g. "DC=example,DC=com"
	GroupSearchFilter string `mapstructure:"group_search_filter"` // e.g. "(member={user_dn})"

	// Attribute mapping
	UsernameAttribute    string `mapstructure:"username_attribute"`     // Default: sAMAccountName
	EmailAttribute       string `mapstructure:"email_attribute"`        // Default: mail
	DisplayNameAttribute string `mapstructure:"display_name_attribute"` // Default: displayName
	GroupNameAttribute   string `mapstructure:"group_name_attribute"`   // Default: cn

	// Access control
	RequiredGroups []string `mapstructure:"required_groups"` // Optional — user must belong to at least one
}

// SessionConfig contains session management settings
type SessionConfig struct {
	TTL           time.Duration `mapstructure:"ttl"`
	CookieName    string        `mapstructure:"cookie_name"`
	CookieDomain  string        `mapstructure:"cookie_domain"`
	CookiePath    string        `mapstructure:"cookie_path"`
	SessionSecret string        `mapstructure:"session_secret"`
}

// CacheConfig contains cache backend settings (for cachego)
type CacheConfig struct {
	Backend       string        `mapstructure:"backend"`        // redis, memory
	RedisURL      string        `mapstructure:"redis_url"`      // Redis connection URL
	RedisPassword string        `mapstructure:"redis_password"` // Redis password
	RedisDB       int           `mapstructure:"redis_db"`       // Redis database number
	CredentialTTL time.Duration `mapstructure:"credential_ttl"` // TTL for cached credentials
	EncryptionKey string        `mapstructure:"encryption_key"` // 32-byte key for AES-256-GCM encryption
}

// UserMgmtConfig contains dynamic user management settings
// User management is always enabled - this is the only mode of operation
type UserMgmtConfig struct {
	PasswordLength int           `mapstructure:"password_length"` // Length of generated passwords (default: 32)
	CredentialTTL  time.Duration `mapstructure:"credential_ttl"`  // How long credentials are cached (default: 1h)
}

// ElasticsearchConfig contains ES credential settings and admin API configuration
type ElasticsearchConfig struct {
	AdminUser          string        `mapstructure:"admin_user"`           // Admin username for Security API calls
	AdminPassword      string        `mapstructure:"admin_password"`       // Admin password for Security API calls
	URL                string        `mapstructure:"url"`                  // Elasticsearch cluster URL
	Timeout            time.Duration `mapstructure:"timeout"`              // Timeout for API calls
	InsecureSkipVerify bool          `mapstructure:"insecure_skip_verify"` // Skip TLS certificate verification
}

// UpstreamConfig contains upstream proxy settings
type UpstreamConfig struct {
	URL                string        `mapstructure:"url"`
	Timeout            time.Duration `mapstructure:"timeout"`
	MaxIdleConns       int           `mapstructure:"max_idle_conns"`
	InsecureSkipVerify bool          `mapstructure:"insecure_skip_verify"` // Skip TLS certificate verification (default: false)
}

// ObservabilityConfig contains logging and tracing settings
type ObservabilityConfig struct {
	// loggergo settings
	LogLevel  string `mapstructure:"log_level"`
	LogFormat string `mapstructure:"log_format"` // json, text

	// otelgo settings
	OTelEnabled        bool    `mapstructure:"otel_enabled"`
	OTelEndpoint       string  `mapstructure:"otel_endpoint"`
	OTelServiceName    string  `mapstructure:"otel_service_name"`
	OTelServiceVersion string  `mapstructure:"otel_service_version"`
	OTelEnvironment    string  `mapstructure:"otel_environment"`
	OTelTraceRatio     float64 `mapstructure:"otel_trace_ratio"` // 0.0 to 1.0

	// Metrics
	MetricsEnabled bool `mapstructure:"metrics_enabled"`
}
