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
				ESUser:         "es_testuser",
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
	assert.Equal(t, "es_testuser", result.ESUser)
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
				ESUser:         "es_testuser",
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
				ESUser:         "es_user",
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
				ESUser:         "es_user",
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
				ESUser:         "es_user",
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
				ESUser:         "es_user",
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
				ESUser:         "es_user",
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
				ESUser:         "es_user",
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
