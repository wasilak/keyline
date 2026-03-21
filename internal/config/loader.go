package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Load loads configuration from file and environment variables
func Load(configFile string) (*Config, error) {
	v := viper.New()

	// Set config file
	if configFile == "" {
		configFile = os.Getenv("CONFIG_FILE")
	}
	if configFile == "" {
		configFile = "config.yaml"
	}

	v.SetConfigFile(configFile)
	v.SetConfigType("yaml")

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Unmarshal into Config struct
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Perform environment variable substitution
	if err := substituteEnvVars(&cfg); err != nil {
		return nil, fmt.Errorf("environment variable substitution failed: %w", err)
	}

	return &cfg, nil
}

// substituteEnvVars replaces ${VAR_NAME} with environment variable values
func substituteEnvVars(cfg *Config) error {
	// OIDC config
	if err := substituteString(&cfg.OIDC.IssuerURL); err != nil {
		return err
	}
	if err := substituteString(&cfg.OIDC.ClientID); err != nil {
		return err
	}
	if err := substituteString(&cfg.OIDC.ClientSecret); err != nil {
		return err
	}
	if err := substituteString(&cfg.OIDC.RedirectURL); err != nil {
		return err
	}

	// Local users
	for i := range cfg.LocalUsers.Users {
		if err := substituteString(&cfg.LocalUsers.Users[i].PasswordBcrypt); err != nil {
			return err
		}
	}

	// Session config
	if err := substituteString(&cfg.Session.SessionSecret); err != nil {
		return err
	}

	// Cache config
	if err := substituteString(&cfg.Cache.RedisURL); err != nil {
		return err
	}
	if err := substituteString(&cfg.Cache.RedisPassword); err != nil {
		return err
	}
	if err := substituteString(&cfg.Cache.EncryptionKey); err != nil {
		return err
	}

	// Elasticsearch admin credentials
	if err := substituteString(&cfg.Elasticsearch.AdminUser); err != nil {
		return err
	}
	if err := substituteString(&cfg.Elasticsearch.AdminPassword); err != nil {
		return err
	}

	// Upstream config
	if err := substituteString(&cfg.Upstream.URL); err != nil {
		return err
	}

	// Observability
	if err := substituteString(&cfg.Observability.OTelEndpoint); err != nil {
		return err
	}

	return nil
}

// substituteString replaces ${VAR_NAME} with environment variable value
func substituteString(s *string) error {
	if s == nil || *s == "" {
		return nil
	}

	// Find all ${VAR_NAME} patterns
	result := *s
	for {
		start := strings.Index(result, "${")
		if start == -1 {
			break
		}

		end := strings.Index(result[start:], "}")
		if end == -1 {
			return fmt.Errorf("unclosed environment variable substitution in: %s", *s)
		}
		end += start

		varName := result[start+2 : end]
		varValue := os.Getenv(varName)
		if varValue == "" {
			return fmt.Errorf("environment variable %s is not set", varName)
		}

		result = result[:start] + varValue + result[end+1:]
	}

	*s = result
	return nil
}
