package auth

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/yourusername/keyline/internal/config"
)

func TestBasicAuthProvider_Authenticate_ValidCredentials(t *testing.T) {
	// Generate bcrypt hash for "password123"
	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	require.NoError(t, err)

	cfg := &config.LocalUsersConfig{
		Enabled: true,
		Users: []config.LocalUser{
			{
				Username:       "testuser",
				PasswordBcrypt: string(hash),
			},
		},
	}

	provider, err := NewBasicAuthProvider(cfg)
	require.NoError(t, err)

	// Create valid Basic Auth header
	credentials := base64.StdEncoding.EncodeToString([]byte("testuser:password123"))

	req := &AuthRequest{
		AuthorizationHeader: "Basic " + credentials,
		OriginalURL:         "https://example.com/test",
	}

	result := provider.Authenticate(context.Background(), req)
	assert.True(t, result.Authenticated)
	assert.Equal(t, "testuser", result.Username)
	assert.Nil(t, result.Error)
}

func TestBasicAuthProvider_Authenticate_InvalidPassword(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("correctpassword"), bcrypt.DefaultCost)
	require.NoError(t, err)

	cfg := &config.LocalUsersConfig{
		Enabled: true,
		Users: []config.LocalUser{
			{
				Username:       "testuser",
				PasswordBcrypt: string(hash),
			},
		},
	}

	provider, err := NewBasicAuthProvider(cfg)
	require.NoError(t, err)

	credentials := base64.StdEncoding.EncodeToString([]byte("testuser:wrongpassword"))

	req := &AuthRequest{
		AuthorizationHeader: "Basic " + credentials,
		OriginalURL:         "https://example.com/test",
	}

	result := provider.Authenticate(context.Background(), req)
	assert.False(t, result.Authenticated)
	assert.NotNil(t, result.Error)
}

func TestBasicAuthProvider_Authenticate_UserNotFound(t *testing.T) {
	cfg := &config.LocalUsersConfig{
		Enabled: true,
		Users: []config.LocalUser{
			{
				Username:       "existinguser",
				PasswordBcrypt: "$2a$10$test",
			},
		},
	}

	provider, err := NewBasicAuthProvider(cfg)
	require.NoError(t, err)

	credentials := base64.StdEncoding.EncodeToString([]byte("nonexistent:password"))

	req := &AuthRequest{
		AuthorizationHeader: "Basic " + credentials,
		OriginalURL:         "https://example.com/test",
	}

	result := provider.Authenticate(context.Background(), req)
	assert.False(t, result.Authenticated)
	assert.NotNil(t, result.Error)
}

func TestBasicAuthProvider_Authenticate_InvalidBase64(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	cfg := &config.LocalUsersConfig{
		Enabled: true,
		Users: []config.LocalUser{
			{
				Username:       "testuser",
				PasswordBcrypt: string(hash),
			},
		},
	}

	provider, err := NewBasicAuthProvider(cfg)
	require.NoError(t, err)

	req := &AuthRequest{
		AuthorizationHeader: "Basic invalid!!!base64",
		OriginalURL:         "https://example.com/test",
	}

	result := provider.Authenticate(context.Background(), req)
	assert.False(t, result.Authenticated)
	assert.NotNil(t, result.Error)
}

func TestBasicAuthProvider_Authenticate_MissingColon(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	cfg := &config.LocalUsersConfig{
		Enabled: true,
		Users: []config.LocalUser{
			{
				Username:       "testuser",
				PasswordBcrypt: string(hash),
			},
		},
	}

	provider, err := NewBasicAuthProvider(cfg)
	require.NoError(t, err)

	// Credentials without colon separator
	credentials := base64.StdEncoding.EncodeToString([]byte("usernameonly"))

	req := &AuthRequest{
		AuthorizationHeader: "Basic " + credentials,
		OriginalURL:         "https://example.com/test",
	}

	result := provider.Authenticate(context.Background(), req)
	assert.False(t, result.Authenticated)
	assert.NotNil(t, result.Error)
}

func TestBasicAuthProvider_Authenticate_EmptyCredentials(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	cfg := &config.LocalUsersConfig{
		Enabled: true,
		Users: []config.LocalUser{
			{
				Username:       "testuser",
				PasswordBcrypt: string(hash),
			},
		},
	}

	provider, err := NewBasicAuthProvider(cfg)
	require.NoError(t, err)

	credentials := base64.StdEncoding.EncodeToString([]byte(":"))

	req := &AuthRequest{
		AuthorizationHeader: "Basic " + credentials,
		OriginalURL:         "https://example.com/test",
	}

	result := provider.Authenticate(context.Background(), req)
	assert.False(t, result.Authenticated)
	assert.NotNil(t, result.Error)
}

func TestBasicAuthProvider_Authenticate_NoAuthHeader(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	cfg := &config.LocalUsersConfig{
		Enabled: true,
		Users: []config.LocalUser{
			{
				Username:       "testuser",
				PasswordBcrypt: string(hash),
			},
		},
	}

	provider, err := NewBasicAuthProvider(cfg)
	require.NoError(t, err)

	req := &AuthRequest{
		AuthorizationHeader: "",
		OriginalURL:         "https://example.com/test",
	}

	result := provider.Authenticate(context.Background(), req)
	assert.False(t, result.Authenticated)
	assert.NotNil(t, result.Error)
}

func TestBasicAuthProvider_Authenticate_WrongScheme(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	cfg := &config.LocalUsersConfig{
		Enabled: true,
		Users: []config.LocalUser{
			{
				Username:       "testuser",
				PasswordBcrypt: string(hash),
			},
		},
	}

	provider, err := NewBasicAuthProvider(cfg)
	require.NoError(t, err)

	req := &AuthRequest{
		AuthorizationHeader: "Bearer sometoken",
		OriginalURL:         "https://example.com/test",
	}

	result := provider.Authenticate(context.Background(), req)
	assert.False(t, result.Authenticated)
	assert.NotNil(t, result.Error)
}

func TestNewBasicAuthProvider_NotEnabled(t *testing.T) {
	cfg := &config.LocalUsersConfig{
		Enabled: false,
		Users:   []config.LocalUser{},
	}

	provider, err := NewBasicAuthProvider(cfg)
	assert.Error(t, err)
	assert.Nil(t, provider)
	assert.Contains(t, err.Error(), "not enabled")
}

func TestNewBasicAuthProvider_NoUsers(t *testing.T) {
	cfg := &config.LocalUsersConfig{
		Enabled: true,
		Users:   []config.LocalUser{},
	}

	provider, err := NewBasicAuthProvider(cfg)
	assert.Error(t, err)
	assert.Nil(t, provider)
	assert.Contains(t, err.Error(), "no local users")
}

func TestBasicAuthProvider_Authenticate_WithGroups(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	require.NoError(t, err)

	cfg := &config.LocalUsersConfig{
		Enabled: true,
		Users: []config.LocalUser{
			{
				Username:       "testuser",
				PasswordBcrypt: string(hash),
				Groups:         []string{"developers", "users"},
				Email:          "testuser@example.com",
				FullName:       "Test User",
			},
		},
	}

	provider, err := NewBasicAuthProvider(cfg)
	require.NoError(t, err)

	credentials := base64.StdEncoding.EncodeToString([]byte("testuser:password123"))

	req := &AuthRequest{
		AuthorizationHeader: "Basic " + credentials,
		OriginalURL:         "https://example.com/test",
	}

	result := provider.Authenticate(context.Background(), req)
	assert.True(t, result.Authenticated)
	assert.Equal(t, "testuser", result.Username)
	assert.Equal(t, "testuser@example.com", result.Email)
	assert.Equal(t, "Test User", result.FullName)
	assert.Equal(t, []string{"developers", "users"}, result.Groups)
	assert.Equal(t, "basic_auth", result.Source)
	assert.Nil(t, result.Error)
}

func TestBasicAuthProvider_Authenticate_WithNoGroups(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	require.NoError(t, err)

	cfg := &config.LocalUsersConfig{
		Enabled: true,
		Users: []config.LocalUser{
			{
				Username:       "viewer",
				PasswordBcrypt: string(hash),
				Groups:         []string{}, // No groups
				Email:          "viewer@example.com",
				FullName:       "Viewer User",
			},
		},
	}

	provider, err := NewBasicAuthProvider(cfg)
	require.NoError(t, err)

	credentials := base64.StdEncoding.EncodeToString([]byte("viewer:password123"))

	req := &AuthRequest{
		AuthorizationHeader: "Basic " + credentials,
		OriginalURL:         "https://example.com/test",
	}

	result := provider.Authenticate(context.Background(), req)
	assert.True(t, result.Authenticated)
	assert.Equal(t, "viewer", result.Username)
	assert.Equal(t, "viewer@example.com", result.Email)
	assert.Equal(t, "Viewer User", result.FullName)
	assert.Empty(t, result.Groups) // Empty groups array
	assert.Equal(t, "basic_auth", result.Source)
	assert.Nil(t, result.Error)
}

func TestBasicAuthProvider_Authenticate_WithMultipleGroups(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("adminpass"), bcrypt.DefaultCost)
	require.NoError(t, err)

	cfg := &config.LocalUsersConfig{
		Enabled: true,
		Users: []config.LocalUser{
			{
				Username:       "admin",
				PasswordBcrypt: string(hash),
				Groups:         []string{"admin", "superusers", "developers", "users"},
				Email:          "admin@example.com",
				FullName:       "Admin User",
			},
		},
	}

	provider, err := NewBasicAuthProvider(cfg)
	require.NoError(t, err)

	credentials := base64.StdEncoding.EncodeToString([]byte("admin:adminpass"))

	req := &AuthRequest{
		AuthorizationHeader: "Basic " + credentials,
		OriginalURL:         "https://example.com/test",
	}

	result := provider.Authenticate(context.Background(), req)
	assert.True(t, result.Authenticated)
	assert.Equal(t, "admin", result.Username)
	assert.Equal(t, "admin@example.com", result.Email)
	assert.Equal(t, "Admin User", result.FullName)
	assert.Equal(t, []string{"admin", "superusers", "developers", "users"}, result.Groups)
	assert.Equal(t, "basic_auth", result.Source)
	assert.Nil(t, result.Error)
}

func TestBasicAuthProvider_Authenticate_WithPartialMetadata(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	require.NoError(t, err)

	cfg := &config.LocalUsersConfig{
		Enabled: true,
		Users: []config.LocalUser{
			{
				Username:       "testuser",
				PasswordBcrypt: string(hash),
				Groups:         []string{"users"},
				// Email and FullName not provided
			},
		},
	}

	provider, err := NewBasicAuthProvider(cfg)
	require.NoError(t, err)

	credentials := base64.StdEncoding.EncodeToString([]byte("testuser:password123"))

	req := &AuthRequest{
		AuthorizationHeader: "Basic " + credentials,
		OriginalURL:         "https://example.com/test",
	}

	result := provider.Authenticate(context.Background(), req)
	assert.True(t, result.Authenticated)
	assert.Equal(t, "testuser", result.Username)
	assert.Empty(t, result.Email)    // Empty string
	assert.Empty(t, result.FullName) // Empty string
	assert.Equal(t, []string{"users"}, result.Groups)
	assert.Equal(t, "basic_auth", result.Source)
	assert.Nil(t, result.Error)
}
