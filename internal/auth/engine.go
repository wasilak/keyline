package auth

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/wasilak/cachego"
	"github.com/yourusername/keyline/internal/config"
	"github.com/yourusername/keyline/internal/mapper"
	"github.com/yourusername/keyline/internal/session"
)

// Engine handles authentication with precedence logic
type Engine struct {
	config         *config.Config
	cache          cachego.CacheInterface
	oidcProvider   *OIDCProvider
	basicProvider  *BasicAuthProvider
	mapper         *mapper.CredentialMapper
	sessionEnabled bool
	oidcEnabled    bool
	basicEnabled   bool
}

// NewEngine creates a new authentication engine
func NewEngine(cfg *config.Config, cache cachego.CacheInterface, oidcProvider *OIDCProvider) (*Engine, error) {
	engine := &Engine{
		config:         cfg,
		cache:          cache,
		oidcProvider:   oidcProvider,
		mapper:         mapper.NewCredentialMapper(cfg),
		sessionEnabled: true, // Sessions are always enabled
		oidcEnabled:    cfg.OIDC.Enabled,
		basicEnabled:   cfg.LocalUsers.Enabled,
	}

	// Initialize Basic Auth provider if enabled
	if cfg.LocalUsers.Enabled {
		basicProvider, err := NewBasicAuthProvider(&cfg.LocalUsers)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize Basic Auth provider: %w", err)
		}
		engine.basicProvider = basicProvider
	}

	return engine, nil
}

// EngineRequest contains authentication request data
type EngineRequest struct {
	Method              string
	Path                string
	Host                string
	Headers             map[string]string
	Cookies             []*http.Cookie
	OriginalURL         string
	AuthorizationHeader string
}

// EngineResult contains authentication result
type EngineResult struct {
	Authenticated bool
	Username      string
	ESUser        string
	ESAuthHeader  string // X-Es-Authorization header value
	RedirectURL   string // For OIDC flow
	SetCookie     *http.Cookie
	StatusCode    int
	Error         error
}

// Authenticate performs authentication with precedence logic:
// 1. Check session cookie first
// 2. Then Basic Auth header
// 3. Then initiate OIDC flow
func (e *Engine) Authenticate(ctx context.Context, req *EngineRequest) *EngineResult {
	slog.InfoContext(ctx, "Authentication engine processing request",
		slog.String("method", req.Method),
		slog.String("path", req.Path),
		slog.String("host", req.Host),
	)

	// 1. Check session cookie first (highest precedence)
	if e.sessionEnabled {
		result := e.authenticateWithSession(ctx, req)
		if result != nil {
			return result
		}
	}

	// 2. Check Basic Auth header (second precedence)
	if e.basicEnabled && req.AuthorizationHeader != "" {
		result := e.authenticateWithBasicAuth(ctx, req)
		if result != nil {
			return result
		}
	}

	// 3. Initiate OIDC flow (lowest precedence, fallback)
	if e.oidcEnabled {
		return e.initiateOIDCFlow(ctx, req)
	}

	// No authentication method available
	slog.WarnContext(ctx, "No authentication method available")
	return &EngineResult{
		Authenticated: false,
		StatusCode:    http.StatusUnauthorized,
		Error:         fmt.Errorf("no authentication method available"),
	}
}

// authenticateWithSession validates session cookie
func (e *Engine) authenticateWithSession(ctx context.Context, req *EngineRequest) *EngineResult {
	// Extract session cookie
	var sessionCookie *http.Cookie
	for _, cookie := range req.Cookies {
		if cookie.Name == e.config.Session.CookieName {
			sessionCookie = cookie
			break
		}
	}

	if sessionCookie == nil {
		slog.DebugContext(ctx, "No session cookie found")
		return nil
	}

	sessionID := sessionCookie.Value
	if sessionID == "" {
		slog.DebugContext(ctx, "Empty session cookie value")
		return nil
	}

	// Retrieve session from store
	sess, err := session.GetSession(ctx, e.cache, sessionID)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to retrieve session",
			slog.String("error", err.Error()),
		)
		return nil
	}

	if sess == nil {
		slog.InfoContext(ctx, "Session not found or expired",
			slog.String("session_id", sessionID),
		)
		return nil
	}

	// Session is valid, get ES authorization header
	esAuthHeader, err := e.mapper.GetESAuthorizationHeader(ctx, sess.ESUser)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to get ES authorization header",
			slog.String("es_user", sess.ESUser),
			slog.String("error", err.Error()),
		)
		return &EngineResult{
			Authenticated: false,
			StatusCode:    http.StatusInternalServerError,
			Error:         fmt.Errorf("failed to get ES credentials"),
		}
	}

	slog.InfoContext(ctx, "Session authentication successful",
		slog.String("username", sess.Username),
		slog.String("es_user", sess.ESUser),
	)

	return &EngineResult{
		Authenticated: true,
		Username:      sess.Username,
		ESUser:        sess.ESUser,
		ESAuthHeader:  esAuthHeader,
		StatusCode:    http.StatusOK,
	}
}

// authenticateWithBasicAuth validates Basic Auth credentials
func (e *Engine) authenticateWithBasicAuth(ctx context.Context, req *EngineRequest) *EngineResult {
	slog.InfoContext(ctx, "Attempting Basic Auth authentication")

	authReq := &AuthRequest{
		AuthorizationHeader: req.AuthorizationHeader,
		OriginalURL:         req.OriginalURL,
	}

	authResult := e.basicProvider.Authenticate(ctx, authReq)

	if !authResult.Authenticated {
		slog.WarnContext(ctx, "Basic Auth authentication failed",
			slog.String("username", authResult.Username),
		)
		return &EngineResult{
			Authenticated: false,
			StatusCode:    http.StatusUnauthorized,
			Error:         authResult.Error,
		}
	}

	// Get ES authorization header
	esAuthHeader, err := e.mapper.GetESAuthorizationHeader(ctx, authResult.ESUser)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to get ES authorization header",
			slog.String("es_user", authResult.ESUser),
			slog.String("error", err.Error()),
		)
		return &EngineResult{
			Authenticated: false,
			StatusCode:    http.StatusInternalServerError,
			Error:         fmt.Errorf("failed to get ES credentials"),
		}
	}

	slog.InfoContext(ctx, "Basic Auth authentication successful",
		slog.String("username", authResult.Username),
		slog.String("es_user", authResult.ESUser),
	)

	return &EngineResult{
		Authenticated: true,
		Username:      authResult.Username,
		ESUser:        authResult.ESUser,
		ESAuthHeader:  esAuthHeader,
		StatusCode:    http.StatusOK,
	}
}

// initiateOIDCFlow starts the OIDC authorization flow
func (e *Engine) initiateOIDCFlow(ctx context.Context, req *EngineRequest) *EngineResult {
	slog.InfoContext(ctx, "Initiating OIDC authorization flow",
		slog.String("original_url", req.OriginalURL),
	)

	// Generate authorization URL
	authURL, err := e.oidcProvider.Authenticate(ctx, e.cache, req.OriginalURL)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to initiate OIDC flow",
			slog.String("error", err.Error()),
		)
		return &EngineResult{
			Authenticated: false,
			StatusCode:    http.StatusInternalServerError,
			Error:         fmt.Errorf("failed to initiate OIDC flow"),
		}
	}

	slog.InfoContext(ctx, "OIDC flow initiated, redirecting to provider")

	return &EngineResult{
		Authenticated: false,
		RedirectURL:   authURL,
		StatusCode:    http.StatusFound, // 302
	}
}
