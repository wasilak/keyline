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
			// Validate redirect URL is HTTPS
			if u, err := url.Parse(cfg.OIDC.RedirectURL); err != nil {
				errors = append(errors, fmt.Sprintf("oidc.redirect_url is not a valid URL: %v", err))
			} else if u.Scheme != "https" {
				errors = append(errors, "oidc.redirect_url must use HTTPS")
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
			if user.ESUser == "" {
				errors = append(errors, fmt.Sprintf("local_users.users[%d].es_user is required", i))
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

	// Validate Elasticsearch users
	if len(cfg.Elasticsearch.Users) == 0 {
		errors = append(errors, "at least one Elasticsearch user must be configured")
	}
	for i, user := range cfg.Elasticsearch.Users {
		if user.Username == "" {
			errors = append(errors, fmt.Sprintf("elasticsearch.users[%d].username is required", i))
		}
		if user.Password == "" {
			errors = append(errors, fmt.Sprintf("elasticsearch.users[%d].password is required", i))
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
