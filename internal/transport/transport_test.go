package transport

import (
"encoding/base64"
"net/http"
"net/http/httptest"
"testing"

"github.com/labstack/echo/v4"
"github.com/stretchr/testify/assert"
"github.com/stretchr/testify/require"
"github.com/wasilak/cachego"
"github.com/yourusername/keyline/internal/auth"
"github.com/yourusername/keyline/internal/config"
"github.com/yourusername/keyline/internal/session"
"golang.org/x/crypto/bcrypt"
)

func setupTestAuthEngine(t *testing.T, cfg *config.Config, cache cachego.CacheInterface) *auth.Engine {
	engine, err := auth.NewEngine(cfg, cache, nil, nil)
	require.NoError(t, err)
	return engine
}

func createTestSession(t *testing.T, cache cachego.CacheInterface, sessionID string, username string, groups []string) {
	sess := &session.Session{
		ID:       sessionID,
		Username: username,
		Groups:   groups,
		Email:    username + "@example.com",
		FullName: "Test User",
		Source:   "test",
	}
	err := session.SaveSession(cache, sess)
	require.NoError(t, err)
}

func TestForwardAuthAdapter_SuccessfulAuthentication(t *testing.T) {
	cache := cachego.NewMapCache()
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
		UserManagement: config.UserManagementConfig{
			Enabled: false,
		},
	}

	sessionID := "test-session-123"
	createTestSession(t, cache, sessionID, "testuser", []string{"admin"})

	authEngine := setupTestAuthEngine(t, cfg, cache)
	adapter, err := NewForwardAuthAdapter(cfg, cache, authEngine)
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
	c := e.NewContext(req, rec)

	err = adapter.HandleRequest(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.NotEmpty(t, rec.Header().Get("Authorization"))
}

func TestForwardAuthAdapter_AuthenticationFailure(t *testing.T) {
	cache := cachego.NewMapCache()
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

	authEngine := setupTestAuthEngine(t, cfg, cache)
	adapter, err := NewForwardAuthAdapter(cfg, cache, authEngine)
	require.NoError(t, err)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/_auth", nil)
	req.Header.Set("X-Forwarded-Method", "GET")
	req.Header.Set("X-Forwarded-Uri", "/test")
	req.Header.Set("X-Forwarded-Host", "example.com")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err = adapter.HandleRequest(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestStandaloneProxyAdapter_Director(t *testing.T) {
	cfg := &config.Config{
		Upstream: config.UpstreamConfig{
			URL: "https://elasticsearch:9200",
		},
	}

	adapter, err := NewStandaloneProxyAdapter(cfg, nil, nil)
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

	adapter, err := NewStandaloneProxyAdapter(cfg, nil, nil)
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
			c := e.NewContext(req, rec)

			err := adapter.HandleRequest(c)

			assert.NoError(t, err)
			assert.Equal(t, http.StatusNotFound, rec.Code)
		})
	}
}
