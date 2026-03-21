package config

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// Validate validates the configuration
func Validate(cfg *Config) error {
	var errors []string

	// Validate at least one authentication method is enabled
	if !cfg.OIDC.Enabled && !cfg.LocalUsers.Enabled {
		errors = append(errors, "at least one authentication method must be enabled (OIDC or local users)")
	}

	// Validate OIDC configuration if enabled
	if cfg.OIDC.Enabled {
		if cfg.OIDC.IssuerURL == "" {
			errors = append(errors, "oidc.issuer_url is required when OIDC is enabled")
		}
		if cfg.OIDC.ClientID == "" {
			errors = append(errors, "oidc.client_id is required when OIDC is enabled")
		}
		if cfg.OIDC.ClientSecret == "" {
			errors = append(errors, "oidc.client_secret is required when OIDC is enabled")
		}
		if cfg.OIDC.RedirectURL == "" {
			errors = append(errors, "oidc.redirect_url is required when OIDC is enabled")
		} else {
			// Validate redirect URL is HTTPS (allow HTTP for localhost/127.0.0.1 for testing)
			if u, err := url.Parse(cfg.OIDC.RedirectURL); err != nil {
				errors = append(errors, fmt.Sprintf("oidc.redirect_url is not a valid URL: %v", err))
			} else if u.Scheme != "https" && u.Scheme != "http" {
				errors = append(errors, "oidc.redirect_url must use HTTP or HTTPS")
			} else if u.Scheme == "http" && u.Hostname() != "localhost" && u.Hostname() != "127.0.0.1" {
				errors = append(errors, "oidc.redirect_url must use HTTPS (HTTP only allowed for localhost/127.0.0.1)")
			}
		}
	}

	// Validate local users configuration if enabled
	if cfg.LocalUsers.Enabled {
		if len(cfg.LocalUsers.Users) == 0 {
			errors = append(errors, "at least one local user must be configured when local users are enabled")
		}
		for i, user := range cfg.LocalUsers.Users {
			if user.Username == "" {
				errors = append(errors, fmt.Sprintf("local_users.users[%d].username is required", i))
			}
			if user.PasswordBcrypt == "" {
				errors = append(errors, fmt.Sprintf("local_users.users[%d].password_bcrypt is required", i))
			} else {
				// Validate bcrypt hash
				if _, err := bcrypt.Cost([]byte(user.PasswordBcrypt)); err != nil {
					errors = append(errors, fmt.Sprintf("local_users.users[%d].password_bcrypt is not a valid bcrypt hash", i))
				}
			}
		}
	}

	// Validate session configuration
	if cfg.Session.SessionSecret == "" {
		errors = append(errors, "session.session_secret is required")
	} else {
		// Validate session secret is at least 32 bytes
		decoded, err := base64.StdEncoding.DecodeString(cfg.Session.SessionSecret)
		if err != nil {
			// Try as raw string
			if len(cfg.Session.SessionSecret) < 32 {
				errors = append(errors, "session.session_secret must be at least 32 bytes")
			}
		} else if len(decoded) < 32 {
			errors = append(errors, "session.session_secret must be at least 32 bytes when decoded")
		}
	}

	// Validate cache backend
	if cfg.Cache.Backend == "" {
		errors = append(errors, "cache.backend is required (redis or memory)")
	} else if cfg.Cache.Backend != "redis" && cfg.Cache.Backend != "memory" {
		errors = append(errors, "cache.backend must be either 'redis' or 'memory'")
	}

	// Validate Redis configuration if backend is redis
	if cfg.Cache.Backend == "redis" {
		if cfg.Cache.RedisURL == "" {
			errors = append(errors, "cache.redis_url is required when cache.backend is redis")
		}
	}

	// Validate user management configuration (always required - this is the only mode)
	// Validate admin credentials are provided
	if cfg.Elasticsearch.AdminUser == "" {
		errors = append(errors, "elasticsearch.admin_user is required (for dynamic user management)")
	}
	if cfg.Elasticsearch.AdminPassword == "" {
		errors = append(errors, "elasticsearch.admin_password is required (for dynamic user management)")
	}
	if cfg.Elasticsearch.URL == "" {
		errors = append(errors, "elasticsearch.url is required (for dynamic user management)")
	}

	// Validate encryption key (required for caching credentials)
	if cfg.Cache.EncryptionKey == "" {
		errors = append(errors, "cache.encryption_key is required (must be 32 bytes for AES-256)")
	} else {
		// Validate encryption key is 32 bytes
		// First try as raw string
		if len(cfg.Cache.EncryptionKey) == 32 {
			// Valid raw string
		} else {
			// Try as base64 encoded
			decoded, err := base64.StdEncoding.DecodeString(cfg.Cache.EncryptionKey)
			if err != nil {
				errors = append(errors, "cache.encryption_key must be exactly 32 bytes (raw or base64 encoded) for AES-256")
			} else if len(decoded) != 32 {
				errors = append(errors, fmt.Sprintf("cache.encryption_key must decode to exactly 32 bytes for AES-256 (got %d bytes)", len(decoded)))
			}
		}
	}

	// Validate role mappings (at least one mapping or default roles required)
	if len(cfg.RoleMappings) == 0 && len(cfg.DefaultESRoles) == 0 {
		errors = append(errors, "at least one role_mapping or default_es_roles must be configured")
	}

	for i, mapping := range cfg.RoleMappings {
		if mapping.Claim == "" {
			errors = append(errors, fmt.Sprintf("role_mappings[%d].claim is required", i))
		}
		if mapping.Pattern == "" {
			errors = append(errors, fmt.Sprintf("role_mappings[%d].pattern is required", i))
		}
		if len(mapping.ESRoles) == 0 {
			errors = append(errors, fmt.Sprintf("role_mappings[%d].es_roles must contain at least one role", i))
		}
		// Validate each role is not empty
		for j, role := range mapping.ESRoles {
			if role == "" {
				errors = append(errors, fmt.Sprintf("role_mappings[%d].es_roles[%d] cannot be empty", i, j))
			}
		}
	}

	// Validate password length
	if cfg.UserManagement.PasswordLength > 0 && cfg.UserManagement.PasswordLength < 16 {
		errors = append(errors, "user_management.password_length must be at least 16 characters")
	}

	// Validate standalone mode configuration
	if cfg.Server.Mode == "standalone" {
		if cfg.Upstream.URL == "" {
			errors = append(errors, "upstream.url is required when mode is standalone")
		}
	}

	// Return all errors
	if len(errors) > 0 {
		return fmt.Errorf("configuration validation failed:\n  - %s", strings.Join(errors, "\n  - "))
	}

	return nil
}
