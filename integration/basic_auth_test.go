//go:build integration
// +build integration

package integration

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/yourusername/keyline/internal/auth"
	"github.com/yourusername/keyline/internal/config"
	"github.com/yourusername/keyline/internal/mapper"
	"golang.org/x/crypto/bcrypt"
)

func TestBasicAuthFlow_ValidCredentials(t *testing.T) {
	// Setup: Create configuration with local users
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("testpass123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	cfg := &config.Config{
		LocalUsers: config.LocalUsersConfig{
			Enabled: true,
			Users: []config.LocalUser{
				{
					Username:       "testuser",
					PasswordBcrypt: string(hashedPassword),
				},
			},
		},
		Elasticsearch: config.ElasticsearchConfig{
			Users: []config.ESUser{
				{
					Username: "es_testuser",
					Password: "es_password",
				},
			},
		},
	}

	// Create credential mapper
	credMapper := mapper.NewCredentialMapper(cfg)

	// Create Basic Auth provider
	basicProvider, err := auth.NewBasicAuthProvider(&cfg.LocalUsers)
	if err != nil {
		t.Fatalf("Failed to create Basic Auth provider: %v", err)
	}

	// Create request with valid credentials
	credentials := base64.StdEncoding.EncodeToString([]byte("testuser:testpass123"))
	authReq := &auth.AuthRequest{
		AuthorizationHeader: "Basic " + credentials,
		OriginalURL:         "/",
	}

	// Test: Authenticate with valid credentials
	ctx := context.Background()
	result := basicProvider.Authenticate(ctx, authReq)

	// Verify: Authentication succeeds
	if result.Error != nil {
		t.Fatalf("Authentication failed: %v", result.Error)
	}
	if !result.Authenticated {
		t.Error("Expected authentication to succeed")
	}
	if result.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", result.Username)
	}
	if result.ESUser != "es_testuser" {
		t.Errorf("Expected ES user 'es_testuser', got '%s'", result.ESUser)
	}

	// Verify ES credentials can be retrieved
	esUsername, esPassword, err := credMapper.GetESCredentials(ctx, result.ESUser)
	if err != nil {
		t.Fatalf("Failed to get ES credentials: %v", err)
	}
	if esUsername != "es_testuser" {
		t.Errorf("Expected ES username 'es_testuser', got '%s'", esUsername)
	}
	if esPassword != "es_password" {
		t.Errorf("Expected ES password 'es_password', got '%s'", esPassword)
	}
}

func TestBasicAuthFlow_InvalidUsername(t *testing.T) {
	// Setup
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("testpass123"), bcrypt.DefaultCost)
	cfg := &config.Config{
		LocalUsers: config.LocalUsersConfig{
			Enabled: true,
			Users: []config.LocalUser{
				{
					Username:       "testuser",
					PasswordBcrypt: string(hashedPassword),
				},
			},
		},
		Elasticsearch: config.ElasticsearchConfig{
			Users: []config.ESUser{
				{
					Username: "es_testuser",
					Password: "es_password",
				},
			},
		},
	}

	basicProvider, _ := auth.NewBasicAuthProvider(&cfg.LocalUsers)

	// Create request with invalid username
	credentials := base64.StdEncoding.EncodeToString([]byte("wronguser:testpass123"))
	authReq := &auth.AuthRequest{
		AuthorizationHeader: "Basic " + credentials,
		OriginalURL:         "/",
	}

	// Test: Authenticate with invalid username
	ctx := context.Background()
	result := basicProvider.Authenticate(ctx, authReq)

	// Verify: Returns 401
	if result.Error == nil {
		t.Error("Expected authentication to fail")
	}
	if result.Authenticated {
		t.Error("Expected authentication to fail")
	}
}

func TestBasicAuthFlow_InvalidPassword(t *testing.T) {
	// Setup
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("testpass123"), bcrypt.DefaultCost)
	cfg := &config.Config{
		LocalUsers: config.LocalUsersConfig{
			Enabled: true,
			Users: []config.LocalUser{
				{
					Username:       "testuser",
					PasswordBcrypt: string(hashedPassword),
				},
			},
		},
		Elasticsearch: config.ElasticsearchConfig{
			Users: []config.ESUser{
				{
					Username: "es_testuser",
					Password: "es_password",
				},
			},
		},
	}

	basicProvider, _ := auth.NewBasicAuthProvider(&cfg.LocalUsers)

	// Create request with invalid password
	credentials := base64.StdEncoding.EncodeToString([]byte("testuser:wrongpassword"))
	authReq := &auth.AuthRequest{
		AuthorizationHeader: "Basic " + credentials,
		OriginalURL:         "/",
	}

	// Test: Authenticate with invalid password
	ctx := context.Background()
	result := basicProvider.Authenticate(ctx, authReq)

	// Verify: Returns 401
	if result.Error == nil {
		t.Error("Expected authentication to fail")
	}
	if result.Authenticated {
		t.Error("Expected authentication to fail")
	}
}

func TestBasicAuthFlow_ESCredentialMapping(t *testing.T) {
	// Setup
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("testpass123"), bcrypt.DefaultCost)
	cfg := &config.Config{
		LocalUsers: config.LocalUsersConfig{
			Enabled: true,
			Users: []config.LocalUser{
				{
					Username:       "testuser",
					PasswordBcrypt: string(hashedPassword),
				},
			},
		},
		Elasticsearch: config.ElasticsearchConfig{
			Users: []config.ESUser{
				{
					Username: "es_testuser",
					Password: "es_password",
				},
			},
		},
	}

	credMapper := mapper.NewCredentialMapper(cfg)
	basicProvider, _ := auth.NewBasicAuthProvider(&cfg.LocalUsers)

	// Create request with valid credentials
	credentials := base64.StdEncoding.EncodeToString([]byte("testuser:testpass123"))
	authReq := &auth.AuthRequest{
		AuthorizationHeader: "Basic " + credentials,
		OriginalURL:         "/",
	}

	// Test: Authenticate and verify ES credential mapping
	ctx := context.Background()
	result := basicProvider.Authenticate(ctx, authReq)

	// Verify: ES credentials are mapped correctly
	if result.Error != nil {
		t.Fatalf("Authentication failed: %v", result.Error)
	}
	if result.ESUser != "es_testuser" {
		t.Errorf("Expected ES user 'es_testuser', got '%s'", result.ESUser)
	}

	// Verify ES credentials can be retrieved
	esUsername, esPassword, err := credMapper.GetESCredentials(ctx, result.ESUser)
	if err != nil {
		t.Fatalf("Failed to get ES credentials: %v", err)
	}
	if esUsername != "es_testuser" {
		t.Errorf("Expected ES username 'es_testuser', got '%s'", esUsername)
	}
	if esPassword != "es_password" {
		t.Errorf("Expected ES password 'es_password', got '%s'", esPassword)
	}
}
