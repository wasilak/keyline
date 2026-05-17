package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/wasilak/cachego"
	cachegoConfig "github.com/wasilak/cachego/config"
	"github.com/wasilak/keyline/internal/config"
	"github.com/wasilak/keyline/internal/session"
	"github.com/wasilak/keyline/internal/usermgmt"
	"golang.org/x/crypto/bcrypt"
)

// MockCache implements cachego.CacheInterface for testing
type MockCache struct {
	data map[string][]byte
}

func NewMockCache() *MockCache {
	return &MockCache{data: make(map[string][]byte)}
}

func (m *MockCache) Init() error { return nil }
func (m *MockCache) GetConfig() cachegoConfig.Config {
	return cachegoConfig.Config{}
}
func (m *MockCache) GetItemTTL(cacheKey string) (time.Duration, bool, error) {
	return 0, false, nil
}
func (m *MockCache) ExtendTTL(cacheKey string, item []byte) error { return nil }

func (m *MockCache) Get(key string) ([]byte, bool, error) {
	val, ok := m.data[key]
	return val, ok, nil
}

func (m *MockCache) Set(key string, value []byte) error {
	m.data[key] = value
	return nil
}

// Ensure MockCache implements cachego.CacheInterface
var _ cachego.CacheInterface = (*MockCache)(nil)

// MockUserManager implements usermgmt.Manager for testing
type MockUserManager struct {
	upsertFunc     func(ctx context.Context, user *usermgmt.AuthenticatedUser) (*usermgmt.Credentials, error)
	invalidateFunc func(ctx context.Context, username string) error
}

func (m *MockUserManager) UpsertUser(ctx context.Context, user *usermgmt.AuthenticatedUser) (*usermgmt.Credentials, error) {
	if m.upsertFunc != nil {
		return m.upsertFunc(ctx, user)
	}
	return &usermgmt.Credentials{
		Username: user.Username,
		Password: "test-password",
	}, nil
}

func (m *MockUserManager) InvalidateCache(ctx context.Context, username string) error {
	if m.invalidateFunc != nil {
		return m.invalidateFunc(ctx, username)
	}
	return nil
}

func (m *MockUserManager) GetUsernameFromAuthHeader(authHeader string) (string, error) {
	if !strings.HasPrefix(authHeader, "Basic ") {
		return "", fmt.Errorf("not basic auth")
	}
	encoded := strings.TrimPrefix(authHeader, "Basic ")
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid credentials format")
	}
	return parts[0], nil
}

// Ensure MockUserManager implements usermgmt.Manager
var _ usermgmt.Manager = (*MockUserManager)(nil)

// Test Engine creation
func TestNewEngine(t *testing.T) {
	tests := []struct {
		name        string
		config      *config.Config
		expectError bool
	}{
		{
			name: "Basic Auth only",
			config: &config.Config{
				LocalUsers: config.LocalUsersConfig{
					Enabled: true,
					Users: []config.LocalUser{
						{Username: "test", PasswordBcrypt: "$2a$10$test"},
					},
				},
				Session: config.SessionConfig{CookieName: "test_session"},
			},
			expectError: false,
		},
		{
			name: "LDAP only",
			config: &config.Config{
				LDAP: config.LDAPConfig{
					Enabled:      true,
					URL:          "ldap://localhost:389",
					BindDN:       "cn=admin,dc=test",
					BindPassword: "${LDAP_BIND_PASSWORD}",
					SearchBase:   "dc=test",
					SearchFilter: "(uid={username})",
				},
				Session: config.SessionConfig{CookieName: "test_session"},
			},
			expectError: false,
		},
		{
			name: "All auth methods",
			config: &config.Config{
				OIDC: config.OIDCConfig{
					Enabled:      true,
					IssuerURL:    "https://example.com",
					ClientID:     "test-client",
					ClientSecret: "${OIDC_SECRET}",
					RedirectURL:  "https://app.example.com/callback",
				},
				LocalUsers: config.LocalUsersConfig{
					Enabled: true,
					Users: []config.LocalUser{
						{Username: "test", PasswordBcrypt: "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"},
					},
				},
				LDAP: config.LDAPConfig{
					Enabled:      true,
					URL:          "ldap://localhost:389",
					BindDN:       "cn=admin,dc=test",
					BindPassword: "${LDAP_BIND_PASSWORD}",
					SearchBase:   "dc=test",
					SearchFilter: "(uid={username})",
				},
				Session: config.SessionConfig{CookieName: "test_session"},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment variables for LDAP tests
			if tt.config.LDAP.Enabled {
				t.Setenv("LDAP_BIND_PASSWORD", "test-ldap-password")
			}
			if tt.config.OIDC.Enabled {
				t.Setenv("OIDC_SECRET", "test-oidc-secret")
			}

			cache := NewMockCache()
			userManager := &MockUserManager{}

			engine, err := NewEngine(tt.config, cache, nil, userManager)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if !tt.expectError && engine == nil {
				t.Errorf("Expected engine but got nil")
			}
		})
	}
}

// Test session cookie authentication
func TestEngine_AuthenticateWithSession(t *testing.T) {
	ctx := context.Background()
	cache := NewMockCache()

	// Create a session
	sess := &session.Session{
		ID:        "test-session-id",
		Username:  "testuser",
		Email:     "test@example.com",
		FullName:  "Test User",
		Groups:    []string{"users"},
		Source:    "oidc",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
	}
	sessionData, _ := json.Marshal(sess)
	cache.Set("session:test-session-id", sessionData)

	cfg := &config.Config{
		Session: config.SessionConfig{
			CookieName: "keyline_session",
		},
		LocalUsers: config.LocalUsersConfig{Enabled: false},
		LDAP:       config.LDAPConfig{Enabled: false},
		OIDC:       config.OIDCConfig{Enabled: false},
	}

	userManager := &MockUserManager{
		upsertFunc: func(ctx context.Context, user *usermgmt.AuthenticatedUser) (*usermgmt.Credentials, error) {
			return &usermgmt.Credentials{
				Username: user.Username,
				Password: "dynamic-password",
			}, nil
		},
	}

	engine, _ := NewEngine(cfg, cache, nil, userManager)

	req := &EngineRequest{
		Method:   "GET",
		Path:     "/",
		Host:     "example.com",
		SourceIP: "127.0.0.1",
		Cookies: []*http.Cookie{
			{
				Name:  "keyline_session",
				Value: "test-session-id",
			},
		},
	}

	result := engine.Authenticate(ctx, req)

	if !result.Authenticated {
		t.Errorf("Expected authenticated but got: %v", result.Error)
	}
	if result.Username != "testuser" {
		t.Errorf("Expected username 'testuser' but got: %s", result.Username)
	}
	if result.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 but got: %d", result.StatusCode)
	}
	if result.ESAuthHeader == "" {
		t.Error("Expected ESAuthHeader to be set")
	}
}

// Test Basic Auth with local user - correct password
func TestEngine_AuthenticateWithBasicAuth_Success(t *testing.T) {
	ctx := context.Background()
	cache := NewMockCache()

	// Generate bcrypt hash for "password" dynamically
	bcryptHash, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("Failed to generate bcrypt hash: %v", err)
	}

	cfg := &config.Config{
		Session: config.SessionConfig{CookieName: "keyline_session"},
		LocalUsers: config.LocalUsersConfig{
			Enabled: true,
			Users: []config.LocalUser{
				{
					Username:       "testuser",
					PasswordBcrypt: string(bcryptHash),
					Groups:         []string{"users"},
					Email:          "test@example.com",
					FullName:       "Test User",
				},
			},
		},
		LDAP: config.LDAPConfig{Enabled: false},
		OIDC: config.OIDCConfig{Enabled: false},
	}

	userManager := &MockUserManager{
		upsertFunc: func(ctx context.Context, user *usermgmt.AuthenticatedUser) (*usermgmt.Credentials, error) {
			return &usermgmt.Credentials{
				Username: user.Username,
				Password: "dynamic-password",
			}, nil
		},
	}

	engine, err := NewEngine(cfg, cache, nil, userManager)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	// Create Basic Auth header: base64("testuser:password")
	authHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte("testuser:password"))

	req := &EngineRequest{
		Method:              "GET",
		Path:                "/",
		Host:                "example.com",
		SourceIP:            "127.0.0.1",
		AuthorizationHeader: authHeader,
	}

	result := engine.Authenticate(ctx, req)

	if !result.Authenticated {
		t.Errorf("Expected authenticated but got: %v", result.Error)
	}
	if result.Username != "testuser" {
		t.Errorf("Expected username 'testuser' but got: %s", result.Username)
	}
	if result.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 but got: %d", result.StatusCode)
	}
}

// Test Basic Auth with local user - wrong password
func TestEngine_AuthenticateWithBasicAuth_WrongPassword(t *testing.T) {
	ctx := context.Background()
	cache := NewMockCache()

	// Generate bcrypt hash for "correctpassword"
	bcryptHash, err := bcrypt.GenerateFromPassword([]byte("correctpassword"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("Failed to generate bcrypt hash: %v", err)
	}

	cfg := &config.Config{
		Session: config.SessionConfig{CookieName: "keyline_session"},
		LocalUsers: config.LocalUsersConfig{
			Enabled: true,
			Users: []config.LocalUser{
				{
					Username:       "testuser",
					PasswordBcrypt: string(bcryptHash),
					Groups:         []string{"users"},
				},
			},
		},
		LDAP: config.LDAPConfig{Enabled: false},
		OIDC: config.OIDCConfig{Enabled: false},
	}

	userManager := &MockUserManager{}
	engine, err := NewEngine(cfg, cache, nil, userManager)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	// Wrong password
	authHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte("testuser:wrongpassword"))

	req := &EngineRequest{
		Method:              "GET",
		Path:                "/",
		Host:                "example.com",
		SourceIP:            "127.0.0.1",
		AuthorizationHeader: authHeader,
	}

	result := engine.Authenticate(ctx, req)

	if result.Authenticated {
		t.Error("Expected not authenticated")
	}
	if result.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401 but got: %d", result.StatusCode)
	}
}

// Test Basic Auth - unknown user without LDAP
func TestEngine_AuthenticateWithBasicAuth_UnknownUser_NoLDAP(t *testing.T) {
	ctx := context.Background()
	cache := NewMockCache()

	// Generate bcrypt hash for existing user
	bcryptHash, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("Failed to generate bcrypt hash: %v", err)
	}

	cfg := &config.Config{
		Session: config.SessionConfig{CookieName: "keyline_session"},
		LocalUsers: config.LocalUsersConfig{
			Enabled: true,
			Users: []config.LocalUser{
				{
					Username:       "testuser",
					PasswordBcrypt: string(bcryptHash),
				},
			},
		},
		LDAP: config.LDAPConfig{Enabled: false},
		OIDC: config.OIDCConfig{Enabled: false},
	}

	userManager := &MockUserManager{}
	engine, err := NewEngine(cfg, cache, nil, userManager)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	// Unknown user
	authHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte("unknownuser:password"))

	req := &EngineRequest{
		Method:              "GET",
		Path:                "/",
		Host:                "example.com",
		SourceIP:            "127.0.0.1",
		AuthorizationHeader: authHeader,
	}

	result := engine.Authenticate(ctx, req)

	// Should fall through to 401 since user not found locally and LDAP is disabled
	if result.Authenticated {
		t.Error("Expected not authenticated")
	}
	if result.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401 but got: %d", result.StatusCode)
	}
}

// Test no authentication method available
func TestEngine_NoAuthMethodAvailable(t *testing.T) {
	ctx := context.Background()
	cache := NewMockCache()

	cfg := &config.Config{
		Session:    config.SessionConfig{CookieName: "keyline_session"},
		LocalUsers: config.LocalUsersConfig{Enabled: false},
		LDAP:       config.LDAPConfig{Enabled: false},
		OIDC:       config.OIDCConfig{Enabled: false},
	}

	userManager := &MockUserManager{}
	engine, _ := NewEngine(cfg, cache, nil, userManager)

	req := &EngineRequest{
		Method:   "GET",
		Path:     "/",
		Host:     "example.com",
		SourceIP: "127.0.0.1",
	}

	result := engine.Authenticate(ctx, req)

	if result.Authenticated {
		t.Error("Expected not authenticated")
	}
	if result.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401 but got: %d", result.StatusCode)
	}
}

// Test session not found
func TestEngine_SessionNotFound(t *testing.T) {
	ctx := context.Background()
	cache := NewMockCache()

	cfg := &config.Config{
		Session: config.SessionConfig{
			CookieName: "keyline_session",
		},
		LocalUsers: config.LocalUsersConfig{Enabled: false},
		LDAP:       config.LDAPConfig{Enabled: false},
		OIDC:       config.OIDCConfig{Enabled: false},
	}

	userManager := &MockUserManager{}
	engine, _ := NewEngine(cfg, cache, nil, userManager)

	req := &EngineRequest{
		Method:   "GET",
		Path:     "/",
		Host:     "example.com",
		SourceIP: "127.0.0.1",
		Cookies: []*http.Cookie{
			{
				Name:  "keyline_session",
				Value: "nonexistent-session",
			},
		},
	}

	result := engine.Authenticate(ctx, req)

	// Should fall through to 401 since session not found and no other auth
	if result.Authenticated {
		t.Error("Expected not authenticated")
	}
	if result.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401 but got: %d", result.StatusCode)
	}
}

// Test user manager failure
func TestEngine_UserManagerFailure(t *testing.T) {
	ctx := context.Background()
	cache := NewMockCache()

	// Create a session
	sess := &session.Session{
		ID:        "test-session-id",
		Username:  "testuser",
		Email:     "test@example.com",
		FullName:  "Test User",
		Groups:    []string{"users"},
		Source:    "oidc",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
	}
	sessionData, _ := json.Marshal(sess)
	cache.Set("session:test-session-id", sessionData)

	cfg := &config.Config{
		Session: config.SessionConfig{
			CookieName: "keyline_session",
		},
		LocalUsers: config.LocalUsersConfig{Enabled: false},
		LDAP:       config.LDAPConfig{Enabled: false},
		OIDC:       config.OIDCConfig{Enabled: false},
	}

	// User manager that fails
	userManager := &MockUserManager{
		upsertFunc: func(ctx context.Context, user *usermgmt.AuthenticatedUser) (*usermgmt.Credentials, error) {
			return nil, fmt.Errorf("user management error")
		},
	}

	engine, _ := NewEngine(cfg, cache, nil, userManager)

	req := &EngineRequest{
		Method:   "GET",
		Path:     "/",
		Host:     "example.com",
		SourceIP: "127.0.0.1",
		Cookies: []*http.Cookie{
			{
				Name:  "keyline_session",
				Value: "test-session-id",
			},
		},
	}

	result := engine.Authenticate(ctx, req)

	if result.Authenticated {
		t.Error("Expected not authenticated due to user manager failure")
	}
	if result.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status 500 but got: %d", result.StatusCode)
	}
}

// Test expired session
func TestEngine_ExpiredSession(t *testing.T) {
	ctx := context.Background()
	cache := NewMockCache()

	// Create an expired session
	sess := &session.Session{
		ID:        "expired-session-id",
		Username:  "testuser",
		Email:     "test@example.com",
		FullName:  "Test User",
		Groups:    []string{"users"},
		Source:    "oidc",
		CreatedAt: time.Now().Add(-2 * time.Hour),
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired
	}
	sessionData, _ := json.Marshal(sess)
	cache.Set("session:expired-session-id", sessionData)

	cfg := &config.Config{
		Session: config.SessionConfig{
			CookieName: "keyline_session",
		},
		LocalUsers: config.LocalUsersConfig{Enabled: false},
		LDAP:       config.LDAPConfig{Enabled: false},
		OIDC:       config.OIDCConfig{Enabled: false},
	}

	userManager := &MockUserManager{}
	engine, _ := NewEngine(cfg, cache, nil, userManager)

	req := &EngineRequest{
		Method:   "GET",
		Path:     "/",
		Host:     "example.com",
		SourceIP: "127.0.0.1",
		Cookies: []*http.Cookie{
			{
				Name:  "keyline_session",
				Value: "expired-session-id",
			},
		},
	}

	result := engine.Authenticate(ctx, req)

	// Should fall through to 401 since session is expired
	if result.Authenticated {
		t.Error("Expected not authenticated")
	}
	if result.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401 but got: %d", result.StatusCode)
	}
}

// Test hasLocalUser helper
func TestEngine_hasLocalUser(t *testing.T) {
	cfg := &config.Config{
		LocalUsers: config.LocalUsersConfig{
			Enabled: true,
			Users: []config.LocalUser{
				{Username: "alice"},
				{Username: "bob"},
			},
		},
	}

	engine := &Engine{
		config:       cfg,
		basicEnabled: true,
	}

	tests := []struct {
		name          string
		authHeader    string
		expectedLocal bool
	}{
		{
			name:          "Local user exists",
			authHeader:    "Basic " + base64.StdEncoding.EncodeToString([]byte("alice:password")),
			expectedLocal: true,
		},
		{
			name:          "Local user does not exist",
			authHeader:    "Basic " + base64.StdEncoding.EncodeToString([]byte("charlie:password")),
			expectedLocal: false,
		},
		{
			name:          "Malformed auth header",
			authHeader:    "Basic invalid-base64!!!",
			expectedLocal: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.hasLocalUser(tt.authHeader)
			if result != tt.expectedLocal {
				t.Errorf("hasLocalUser() = %v, expected %v", result, tt.expectedLocal)
			}
		})
	}
}
