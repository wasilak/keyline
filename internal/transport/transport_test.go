package transport

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wasilak/cachego"
	"github.com/yourusername/keyline/internal/auth"
	"github.com/yourusername/keyline/internal/cache"
	"github.com/yourusername/keyline/internal/config"
	"github.com/yourusername/keyline/internal/session"
	"github.com/yourusername/keyline/internal/usermgmt"
)

func setupTestCache(t *testing.T) cachego.CacheInterface {
	cfg := &config.CacheConfig{
		Backend: "memory",
	}
	c, err := cache.InitCache(context.Background(), cfg)
	require.NoError(t, err)
	return c
}

// mockUserManager is a simple mock for testing
type mockUserManager struct{}

func (m *mockUserManager) UpsertUser(ctx context.Context, user *usermgmt.AuthenticatedUser) (*usermgmt.Credentials, error) {
	return &usermgmt.Credentials{
		Username: user.Username,
		Password: "mock-es-password",
	}, nil
}

func (m *mockUserManager) InvalidateCache(ctx context.Context, username string) error {
	return nil
}

func (m *mockUserManager) GetUsernameFromAuthHeader(authHeader string) (string, error) {
	return "", nil
}

func setupTestAuthEngine(t *testing.T, cfg *config.Config, c cachego.CacheInterface) *auth.Engine {
	userManager := &mockUserManager{}
	engine, err := auth.NewEngine(cfg, c, nil, userManager)
	require.NoError(t, err)
	return engine
}

func createTestSession(t *testing.T, c cachego.CacheInterface, sessionID string, username string, groups []string) {
	sess := &session.Session{
		ID:        sessionID,
		Username:  username,
		Groups:    groups,
		Email:     username + "@example.com",
		FullName:  "Test User",
		Source:    "test",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	err := session.CreateSession(context.Background(), c, sess)
	require.NoError(t, err)
}

func TestForwardAuthAdapter_SuccessfulAuthentication(t *testing.T) {
	c := setupTestCache(t)
	cfg := &config.Config{
		Session: config.SessionConfig{
			CookieName: "keyline_session",
			TTL:        3600,
		},
		LocalUsers: config.LocalUsersConfig{
			Enabled: false,
		},
		OIDC: config.OIDCConfig{
			Enabled: false,
		},
		UserManagement: config.UserMgmtConfig{
			PasswordLength: 32,
			CredentialTTL:  time.Hour,
		},
		Elasticsearch: config.ElasticsearchConfig{
			AdminUser:     "test-admin",
			AdminPassword: "test-pass",
			URL:           "http://localhost:9200",
		},
		Cache: config.CacheConfig{
			Backend:       "memory",
			EncryptionKey: "MTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTI=",
		},
		RoleMappings: []config.RoleMapping{
			{Claim: "groups", Pattern: "admin", ESRoles: []string{"superuser"}},
		},
	}

	sessionID := "test-session-123"
	createTestSession(t, c, sessionID, "testuser", []string{"admin"})

	authEngine := setupTestAuthEngine(t, cfg, c)
	adapter, err := NewForwardAuthAdapter(cfg, c, authEngine)
	require.NoError(t, err)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/_auth", nil)
	req.Header.Set("X-Forwarded-Method", "GET")
	req.Header.Set("X-Forwarded-Uri", "/test")
	req.Header.Set("X-Forwarded-Host", "example.com")
	req.Header.Set("X-Forwarded-Proto", "https")
	req.AddCookie(&http.Cookie{
		Name:  "keyline_session",
		Value: sessionID,
	})
	rec := httptest.NewRecorder()
	echoCtx := e.NewContext(req, rec)

	err = adapter.HandleRequest(echoCtx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.NotEmpty(t, rec.Header().Get("Authorization"))
}

func TestForwardAuthAdapter_AuthenticationFailure(t *testing.T) {
	c := setupTestCache(t)
	cfg := &config.Config{
		Session: config.SessionConfig{
			CookieName: "keyline_session",
			TTL:        3600,
		},
		LocalUsers: config.LocalUsersConfig{
			Enabled: false,
		},
		OIDC: config.OIDCConfig{
			Enabled: false,
		},
	}

	authEngine := setupTestAuthEngine(t, cfg, c)
	adapter, err := NewForwardAuthAdapter(cfg, c, authEngine)
	require.NoError(t, err)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/_auth", nil)
	req.Header.Set("X-Forwarded-Method", "GET")
	req.Header.Set("X-Forwarded-Uri", "/test")
	req.Header.Set("X-Forwarded-Host", "example.com")
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	err = adapter.HandleRequest(ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestStandaloneProxyAdapter_Director(t *testing.T) {
	cfg := &config.Config{
		Upstream: config.UpstreamConfig{
			URL: "https://elasticsearch:9200",
		},
	}

	adapter, err := NewStandaloneProxyAdapter(cfg, nil, nil, nil)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Basic user:pass")

	esUser := "esuser"
	esPassword := "espass"
	esAuthHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte(esUser+":"+esPassword))
	req.Header.Set("X-Es-Authorization", esAuthHeader)

	adapter.director(req)

	assert.Equal(t, "https", req.URL.Scheme)
	assert.Equal(t, "elasticsearch:9200", req.URL.Host)
	assert.Equal(t, "elasticsearch:9200", req.Host)
	assert.Equal(t, esAuthHeader, req.Header.Get("Authorization"))
	assert.Empty(t, req.Header.Get("X-Es-Authorization"))
}

func TestStandaloneProxyAdapter_InternalEndpoint(t *testing.T) {
	cfg := &config.Config{
		Upstream: config.UpstreamConfig{
			URL: "http://localhost:9200",
		},
	}

	adapter, err := NewStandaloneProxyAdapter(cfg, nil, nil, nil)
	require.NoError(t, err)

	internalPaths := []string{
		"/auth/callback",
		"/auth/logout",
		"/healthz",
		"/metrics",
	}

	for _, path := range internalPaths {
		t.Run(path, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, path, nil)
			rec := httptest.NewRecorder()
			ctx := e.NewContext(req, rec)

			err := adapter.HandleRequest(ctx)

			assert.NoError(t, err)
			assert.Equal(t, http.StatusNotFound, rec.Code)
		})
	}
}
