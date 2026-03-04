package mapper

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"strings"

	"github.com/yourusername/keyline/internal/config"
)

// CredentialMapper maps authenticated users to Elasticsearch credentials
type CredentialMapper struct {
	config *config.Config
}

// NewCredentialMapper creates a new credential mapper
func NewCredentialMapper(cfg *config.Config) *CredentialMapper {
	return &CredentialMapper{
		config: cfg,
	}
}

// MapOIDCUser maps an OIDC user to an ES user based on claim extraction and pattern matching
func (m *CredentialMapper) MapOIDCUser(ctx context.Context, claims map[string]interface{}) (string, error) {
	slog.InfoContext(ctx, "Mapping OIDC user to ES user")

	// Evaluate each mapping in order
	for i, mapping := range m.config.OIDC.Mappings {
		// Extract claim value
		claimValue, ok := claims[mapping.Claim]
		if !ok {
			slog.DebugContext(ctx, "Claim not found in token",
				slog.String("claim", mapping.Claim),
				slog.Int("mapping_index", i),
			)
			continue
		}

		// Convert claim value to string
		claimStr, ok := claimValue.(string)
		if !ok {
			slog.DebugContext(ctx, "Claim value is not a string",
				slog.String("claim", mapping.Claim),
				slog.Any("value", claimValue),
			)
			continue
		}

		// Check if pattern matches
		if m.matchPattern(claimStr, mapping.Pattern) {
			slog.InfoContext(ctx, "OIDC user mapped to ES user",
				slog.String("claim", mapping.Claim),
				slog.String("claim_value", claimStr),
				slog.String("pattern", mapping.Pattern),
				slog.String("es_user", mapping.ESUser),
			)
			return mapping.ESUser, nil
		}
	}

	// No mapping matched, use default ES user
	if m.config.OIDC.DefaultESUser == "" {
		return "", fmt.Errorf("no OIDC mapping matched and no default ES user configured")
	}

	slog.InfoContext(ctx, "Using default ES user for OIDC user",
		slog.String("es_user", m.config.OIDC.DefaultESUser),
	)

	return m.config.OIDC.DefaultESUser, nil
}

// MapLocalUser maps a local user to an ES user (simple lookup)
func (m *CredentialMapper) MapLocalUser(ctx context.Context, username string) (string, error) {
	slog.InfoContext(ctx, "Mapping local user to ES user",
		slog.String("username", username),
	)

	// Find the local user
	for _, user := range m.config.LocalUsers.Users {
		if user.Username == username {
			slog.InfoContext(ctx, "Local user mapped to ES user",
				slog.String("username", username),
				slog.String("es_user", user.ESUser),
			)
			return user.ESUser, nil
		}
	}

	return "", fmt.Errorf("local user not found: %s", username)
}

// GetESCredentials retrieves ES username and password for the given ES user
func (m *CredentialMapper) GetESCredentials(ctx context.Context, esUser string) (username, password string, err error) {
	slog.InfoContext(ctx, "Retrieving ES credentials",
		slog.String("es_user", esUser),
	)

	// Find the ES user in configuration
	for _, user := range m.config.Elasticsearch.Users {
		if user.Username == esUser {
			slog.InfoContext(ctx, "ES credentials found",
				slog.String("es_user", esUser),
			)
			return user.Username, user.Password, nil
		}
	}

	slog.ErrorContext(ctx, "ES user not found in configuration",
		slog.String("es_user", esUser),
	)

	return "", "", fmt.Errorf("ES user not found in configuration: %s", esUser)
}

// EncodeESCredentials encodes ES credentials as Basic auth (base64 of username:password)
func (m *CredentialMapper) EncodeESCredentials(username, password string) string {
	credentials := username + ":" + password
	encoded := base64.StdEncoding.EncodeToString([]byte(credentials))
	return "Basic " + encoded
}

// GetESAuthorizationHeader retrieves and encodes ES credentials for the given ES user
// Returns the complete Authorization header value: "Basic {encoded_credentials}"
func (m *CredentialMapper) GetESAuthorizationHeader(ctx context.Context, esUser string) (string, error) {
	// Get ES credentials
	username, password, err := m.GetESCredentials(ctx, esUser)
	if err != nil {
		return "", err
	}

	// Encode credentials
	authHeader := m.EncodeESCredentials(username, password)

	// Never log credentials in plaintext
	slog.InfoContext(ctx, "ES authorization header generated",
		slog.String("es_user", esUser),
	)

	return authHeader, nil
}

// matchPattern performs wildcard pattern matching
// Supports * wildcard matching (e.g., "*@admin.example.com")
func (m *CredentialMapper) matchPattern(value, pattern string) bool {
	// Exact match
	if value == pattern {
		return true
	}

	// No wildcard, no match
	if !strings.Contains(pattern, "*") {
		return false
	}

	// Handle wildcard patterns
	// Split pattern on * to get prefix and suffix
	parts := strings.Split(pattern, "*")

	// Pattern starts with *
	if strings.HasPrefix(pattern, "*") {
		suffix := parts[1]
		return strings.HasSuffix(value, suffix)
	}

	// Pattern ends with *
	if strings.HasSuffix(pattern, "*") {
		prefix := parts[0]
		return strings.HasPrefix(value, prefix)
	}

	// Pattern has * in the middle
	if len(parts) == 2 {
		prefix := parts[0]
		suffix := parts[1]
		return strings.HasPrefix(value, prefix) && strings.HasSuffix(value, suffix)
	}

	// Multiple wildcards not supported
	return false
}
