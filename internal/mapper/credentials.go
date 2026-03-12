package mapper

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"

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

// NOTE: MapOIDCUser and MapLocalUser methods have been removed.
// These methods are replaced by the dynamic user management system which uses:
// - RoleMapper to map groups to ES roles
// - UserManager to create/update ES users dynamically
// See internal/usermgmt package for the new implementation.

// GetESCredentials retrieves ES username and password for the given ES user
// NOTE: This method is kept for backward compatibility with static user mapping.
// When user management is enabled, credentials are generated dynamically by UserManager.
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

// NOTE: matchPattern method has been removed.
// Pattern matching is now handled by RoleMapper in internal/usermgmt package.
