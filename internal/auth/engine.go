package auth

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/wasilak/cachego"
	"github.com/yourusername/keyline/internal/config"
	"github.com/yourusername/keyline/internal/session"
	"github.com/yourusername/keyline/internal/usermgmt"
)

// Engine handles authentication with dynamic user management
type Engine struct {
	config       *config.Config
	cache        cachego.CacheInterface
	oidcProvider *OIDCProvider
	basicProvider *BasicAuthProvider
	userManager  usermgmt.Manager
	sessionEnabled bool
	oidcEnabled    bool
	basicEnabled   bool
}

// NewEngine creates a new authentication engine with dynamic user management
func NewEngine(cfg *config.Config, cache cachego.CacheInterface, oidcProvider *OIDCProvider, userManager usermgmt.Manager) (*Engine, error) {
	engine := &Engine{
		config:       cfg,
		cache:        cache,
		oidcProvider: oidcProvider,
		userManager:  userManager,
		sessionEnabled: true,
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
	SourceIP            string // Source IP address for logging
}

// EngineResult contains authentication result
type EngineResult struct {
	Authenticated bool
	Username      string
	ESUser        string
	ESPassword    string // ES password for dynamic user management
	ESAuthHeader  string // X-Es-Authorization header value (base64 encoded username:password)
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
		slog.String("source_ip", req.SourceIP),
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

	// Upsert user with dynamic credentials
	authUser := &usermgmt.AuthenticatedUser{
		Username: sess.Username,
		Groups:   sess.Groups,
		Email:    sess.Email,
		FullName: sess.FullName,
		Source:   sess.Source,
	}

	creds, err := e.userManager.UpsertUser(ctx, authUser)
	if err != nil {
		slog.ErrorContext(ctx, "User management failed for session",
			slog.String("username", sess.Username),
			slog.String("error", err.Error()),
		)
		return &EngineResult{
			Authenticated: false,
			StatusCode:    http.StatusInternalServerError,
			Error:         fmt.Errorf("user management failed: %w", err),
		}
	}

	// Create ES Authorization header with dynamic credentials
	esAuthHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte(creds.Username+":"+creds.Password))

	slog.InfoContext(ctx, "Session authentication successful with dynamic user management",
		slog.String("username", sess.Username),
		slog.String("method", "session"),
		slog.String("source_ip", req.SourceIP),
		slog.String("result", "success"),
		slog.String("es_user", creds.Username),
	)

	return &EngineResult{
		Authenticated: true,
		Username:      sess.Username,
		ESUser:        creds.Username,
		ESPassword:    creds.Password,
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

	// Get user metadata from local user config
	var userGroups []string
	var userEmail string
	var userFullName string

	for _, user := range e.config.LocalUsers.Users {
		if user.Username == authResult.Username {
			userGroups = user.Groups
			userEmail = user.Email
			userFullName = user.FullName
			break
		}
	}

	// Upsert user with dynamic credentials
	authUser := &usermgmt.AuthenticatedUser{
		Username: authResult.Username,
		Groups:   userGroups,
		Email:    userEmail,
		FullName: userFullName,
		Source:   "basic_auth",
	}

	creds, err := e.userManager.UpsertUser(ctx, authUser)
	if err != nil {
		slog.ErrorContext(ctx, "User management failed for Basic Auth",
			slog.String("username", authResult.Username),
			slog.String("error", err.Error()),
		)
		return &EngineResult{
			Authenticated: false,
			StatusCode:    http.StatusInternalServerError,
			Error:         fmt.Errorf("user management failed: %w", err),
		}
	}

	// Create ES Authorization header with dynamic credentials
	esAuthHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte(creds.Username+":"+creds.Password))

	slog.InfoContext(ctx, "Basic Auth authentication successful with dynamic user management",
		slog.String("username", authResult.Username),
		slog.String("method", "basic"),
		slog.String("source_ip", req.SourceIP),
		slog.String("result", "success"),
		slog.String("es_user", creds.Username),
		slog.Any("groups", userGroups),
	)

	return &EngineResult{
		Authenticated: true,
		Username:      authResult.Username,
		ESUser:        creds.Username,
		ESPassword:    creds.Password,
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
