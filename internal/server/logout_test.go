package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yourusername/keyline/internal/cache"
	"github.com/yourusername/keyline/internal/config"
	"github.com/yourusername/keyline/internal/session"
)

func setupTestServer(t *testing.T) (*Server, context.Context) {
	ctx := context.Background()

	cfg := &config.Config{
		Session: config.SessionConfig{
			CookieName:   "test_session",
			CookiePath:   "/",
			CookieDomain: "example.com",
			TTL:          24 * time.Hour,
		},
		Cache: config.CacheConfig{
			Backend: "memory",
		},
		OIDC: config.OIDCConfig{
			Enabled: false,
		},
		LocalUsers: config.LocalUsersConfig{
			Enabled: false,
		},
	}

	testCache, err := cache.InitCache(ctx, &cfg.Cache)
	require.NoError(t, err)

	server := &Server{
		config:       cfg,
		cache:        testCache,
		oidcProvider: nil,
	}

	return server, ctx
}

func TestHandleLogout_WithValidSession(t *testing.T) {
	server, ctx := setupTestServer(t)

	// Create a test session
	testSession := &session.Session{
		ID:        "test-session-123",
		UserID:    "user123",
		Username:  "testuser",
		Email:     "test@example.com",
		ESUser:    "es_testuser",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	err := session.CreateSession(ctx, server.cache, testSession)
	require.NoError(t, err)

	// Create Echo context with session cookie
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/auth/logout", nil)
	req.AddCookie(&http.Cookie{
		Name:  server.config.Session.CookieName,
		Value: testSession.ID,
	})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Call logout handler
	err = server.handleLogout(c)
	require.NoError(t, err)

	// Verify response
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Logged out")

	// Verify cookie was cleared
	cookies := rec.Result().Cookies()
	var clearCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == server.config.Session.CookieName {
			clearCookie = cookie
			break
		}
	}
	require.NotNil(t, clearCookie, "Clear cookie should be set")
	assert.Equal(t, "", clearCookie.Value)
	assert.Equal(t, -1, clearCookie.MaxAge)
	assert.True(t, clearCookie.HttpOnly)
	assert.True(t, clearCookie.Secure)
	assert.Equal(t, http.SameSiteLaxMode, clearCookie.SameSite)

	// Verify session was deleted from store
	retrievedSession, err := session.GetSession(ctx, server.cache, testSession.ID)
	assert.Nil(t, retrievedSession)
	// Session not found is not an error in GetSession - it just returns nil
}

func TestHandleLogout_WithoutSession(t *testing.T) {
	server, _ := setupTestServer(t)

	// Create Echo context without session cookie
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/auth/logout", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Call logout handler
	err := server.handleLogout(c)
	require.NoError(t, err)

	// Verify response
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "No active session")

	// Verify no clear cookie was set (since there was no session)
	cookies := rec.Result().Cookies()
	assert.Empty(t, cookies)
}

func TestHandleLogout_WithEmptySessionCookie(t *testing.T) {
	server, _ := setupTestServer(t)

	// Create Echo context with empty session cookie
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/auth/logout", nil)
	req.AddCookie(&http.Cookie{
		Name:  server.config.Session.CookieName,
		Value: "",
	})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Call logout handler
	err := server.handleLogout(c)
	require.NoError(t, err)

	// Verify response
	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify cookie was cleared
	cookies := rec.Result().Cookies()
	var clearCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == server.config.Session.CookieName {
			clearCookie = cookie
			break
		}
	}
	require.NotNil(t, clearCookie, "Clear cookie should be set")
	assert.Equal(t, "", clearCookie.Value)
	assert.Equal(t, -1, clearCookie.MaxAge)
}

func TestHandleLogout_WithNonExistentSession(t *testing.T) {
	server, _ := setupTestServer(t)

	// Create Echo context with non-existent session ID
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/auth/logout", nil)
	req.AddCookie(&http.Cookie{
		Name:  server.config.Session.CookieName,
		Value: "non-existent-session-id",
	})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Call logout handler
	err := server.handleLogout(c)
	require.NoError(t, err)

	// Verify response (should still succeed and clear cookie)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Logged out")

	// Verify cookie was cleared
	cookies := rec.Result().Cookies()
	var clearCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == server.config.Session.CookieName {
			clearCookie = cookie
			break
		}
	}
	require.NotNil(t, clearCookie, "Clear cookie should be set")
	assert.Equal(t, "", clearCookie.Value)
	assert.Equal(t, -1, clearCookie.MaxAge)
}

func TestHandleLogout_CookieAttributes(t *testing.T) {
	server, ctx := setupTestServer(t)

	// Create a test session
	testSession := &session.Session{
		ID:        "test-session-456",
		UserID:    "user456",
		Username:  "testuser2",
		Email:     "test2@example.com",
		ESUser:    "es_testuser2",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	err := session.CreateSession(ctx, server.cache, testSession)
	require.NoError(t, err)

	// Create Echo context with session cookie
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/auth/logout", nil)
	req.AddCookie(&http.Cookie{
		Name:  server.config.Session.CookieName,
		Value: testSession.ID,
	})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Call logout handler
	err = server.handleLogout(c)
	require.NoError(t, err)

	// Verify cookie attributes
	cookies := rec.Result().Cookies()
	var clearCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == server.config.Session.CookieName {
			clearCookie = cookie
			break
		}
	}

	require.NotNil(t, clearCookie)
	assert.Equal(t, server.config.Session.CookieName, clearCookie.Name)
	assert.Equal(t, "", clearCookie.Value)
	assert.Equal(t, server.config.Session.CookiePath, clearCookie.Path)
	assert.Equal(t, server.config.Session.CookieDomain, clearCookie.Domain)
	assert.Equal(t, -1, clearCookie.MaxAge)
	assert.True(t, clearCookie.HttpOnly)
	assert.True(t, clearCookie.Secure)
	assert.Equal(t, http.SameSiteLaxMode, clearCookie.SameSite)
}

func TestHandleLogout_WithoutOIDCProvider(t *testing.T) {
	server, ctx := setupTestServer(t)

	// Server has no OIDC provider (oidcProvider is nil)
	assert.Nil(t, server.oidcProvider)

	// Create a test session
	testSession := &session.Session{
		ID:        "test-session-789",
		UserID:    "user789",
		Username:  "testuser3",
		Email:     "test3@example.com",
		ESUser:    "es_testuser3",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	err := session.CreateSession(ctx, server.cache, testSession)
	require.NoError(t, err)

	// Create Echo context with session cookie
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/auth/logout", nil)
	req.AddCookie(&http.Cookie{
		Name:  server.config.Session.CookieName,
		Value: testSession.ID,
	})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Call logout handler
	err = server.handleLogout(c)
	require.NoError(t, err)

	// Without OIDC provider, should return success without redirect
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Logged out")

	// Verify cookie was cleared
	cookies := rec.Result().Cookies()
	var clearCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == server.config.Session.CookieName {
			clearCookie = cookie
			break
		}
	}
	require.NotNil(t, clearCookie)
	assert.Equal(t, "", clearCookie.Value)
	assert.Equal(t, -1, clearCookie.MaxAge)
}

func TestHandleLogout_SessionDeletionFailure(t *testing.T) {
	server, ctx := setupTestServer(t)

	// Create a test session
	testSession := &session.Session{
		ID:        "test-session-999",
		UserID:    "user999",
		Username:  "testuser4",
		Email:     "test4@example.com",
		ESUser:    "es_testuser4",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	err := session.CreateSession(ctx, server.cache, testSession)
	require.NoError(t, err)

	// Create Echo context with session cookie
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/auth/logout", nil)
	req.AddCookie(&http.Cookie{
		Name:  server.config.Session.CookieName,
		Value: testSession.ID,
	})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Call logout handler
	err = server.handleLogout(c)
	require.NoError(t, err)

	// Even if session deletion fails, logout should succeed and clear cookie
	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify cookie was cleared
	cookies := rec.Result().Cookies()
	var clearCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == server.config.Session.CookieName {
			clearCookie = cookie
			break
		}
	}
	require.NotNil(t, clearCookie)
	assert.Equal(t, "", clearCookie.Value)
	assert.Equal(t, -1, clearCookie.MaxAge)
}

func TestHandleLogout_SubsequentRequestsUnauthenticated(t *testing.T) {
	server, ctx := setupTestServer(t)

	// Create a test session
	testSession := &session.Session{
		ID:        "test-session-subsequent",
		UserID:    "user-subsequent",
		Username:  "testuser-subsequent",
		Email:     "subsequent@example.com",
		ESUser:    "es_subsequent",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	err := session.CreateSession(ctx, server.cache, testSession)
	require.NoError(t, err)

	// Verify session exists before logout
	retrievedSession, err := session.GetSession(ctx, server.cache, testSession.ID)
	require.NoError(t, err)
	assert.NotNil(t, retrievedSession)

	// Create Echo context with session cookie
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/auth/logout", nil)
	req.AddCookie(&http.Cookie{
		Name:  server.config.Session.CookieName,
		Value: testSession.ID,
	})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Call logout handler
	err = server.handleLogout(c)
	require.NoError(t, err)

	// Verify session no longer exists after logout
	retrievedSession, err = session.GetSession(ctx, server.cache, testSession.ID)
	assert.Nil(t, retrievedSession)
	// Session not found is not an error - GetSession returns nil

	// Subsequent requests with the same session ID should be treated as unauthenticated
	retrievedSession, err = session.GetSession(ctx, server.cache, testSession.ID)
	assert.Nil(t, retrievedSession)
	// Session not found is not an error - GetSession returns nil
}
