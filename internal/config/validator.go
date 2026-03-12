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
		errors = append(errors, "at least one authentication method must be enabled")
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

	// Validate cache configuration
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

	// Validate user management configuration
	if cfg.UserManagement.Enabled {
		// Validate admin credentials are provided
		if cfg.Elasticsearch.AdminUser == "" {
			errors = append(errors, "elasticsearch.admin_user is required when user_management.enabled is true")
		}
		if cfg.Elasticsearch.AdminPassword == "" {
			errors = append(errors, "elasticsearch.admin_password is required when user_management.enabled is true")
		}
		if cfg.Elasticsearch.URL == "" {
			errors = append(errors, "elasticsearch.url is required when user_management.enabled is true")
		}

		// Validate encryption key
		if cfg.Cache.EncryptionKey == "" {
			errors = append(errors, "cache.encryption_key is required when user_management.enabled is true")
		} else {
			// Validate encryption key is 32 bytes when decoded
			decoded, err := base64.StdEncoding.DecodeString(cfg.Cache.EncryptionKey)
			if err != nil {
				// Try as raw string
				if len(cfg.Cache.EncryptionKey) != 32 {
					errors = append(errors, "cache.encryption_key must be exactly 32 bytes for AES-256")
				}
			} else if len(decoded) != 32 {
				errors = append(errors, "cache.encryption_key must be exactly 32 bytes when decoded for AES-256")
			}
		}

		// Validate role mappings
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
	} else {
		// If user management is disabled, validate static Elasticsearch users
		if len(cfg.Elasticsearch.Users) == 0 {
			errors = append(errors, "at least one Elasticsearch user must be configured when user_management.enabled is false")
		}
		for i, user := range cfg.Elasticsearch.Users {
			if user.Username == "" {
				errors = append(errors, fmt.Sprintf("elasticsearch.users[%d].username is required", i))
			}
			if user.Password == "" {
				errors = append(errors, fmt.Sprintf("elasticsearch.users[%d].password is required", i))
			}
		}
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
