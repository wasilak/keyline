package transport

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/wasilak/cachego"
	"github.com/yourusername/keyline/internal/auth"
	"github.com/yourusername/keyline/internal/config"
)

// ForwardAuthAdapter handles Traefik/Nginx forwardAuth mode
type ForwardAuthAdapter struct {
	config     *config.Config
	cache      cachego.CacheInterface
	authEngine *auth.Engine
}

// NewForwardAuthAdapter creates a new ForwardAuth adapter
func NewForwardAuthAdapter(cfg *config.Config, cache cachego.CacheInterface, authEngine *auth.Engine) (*ForwardAuthAdapter, error) {
	return &ForwardAuthAdapter{
		config:     cfg,
		cache:      cache,
		authEngine: authEngine,
	}, nil
}

// Name returns the adapter name
func (a *ForwardAuthAdapter) Name() string {
	return "forward_auth"
}

// HandleRequest processes an incoming forwardAuth request
func (a *ForwardAuthAdapter) HandleRequest(c echo.Context) error {
	ctx := c.Request().Context()

	// Normalize headers to RequestContext
	reqCtx, err := a.normalizeHeaders(c)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to normalize headers",
			slog.String("error", err.Error()),
		)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request headers",
		})
	}

	slog.InfoContext(ctx, "ForwardAuth request received",
		slog.String("method", reqCtx.Method),
		slog.String("path", reqCtx.Path),
		slog.String("host", reqCtx.Host),
		slog.String("original_url", reqCtx.OriginalURL),
	)

	// Check if this is the OIDC callback path
	if a.isCallbackPath(reqCtx.Path) {
		return a.handleCallback(c, ctx)
	}

	// Build engine request
	engineReq := &auth.EngineRequest{
		Method:              reqCtx.Method,
		Path:                reqCtx.Path,
		Host:                reqCtx.Host,
		Headers:             reqCtx.Headers,
		Cookies:             reqCtx.Cookies,
		OriginalURL:         reqCtx.OriginalURL,
		AuthorizationHeader: c.Request().Header.Get("Authorization"),
	}

	// Delegate to auth engine
	result := a.authEngine.Authenticate(ctx, engineReq)

	// Handle authentication result
	return a.buildResponse(c, result)
}

// normalizeHeaders converts X-Forwarded-* or X-Original-* headers to RequestContext
func (a *ForwardAuthAdapter) normalizeHeaders(c echo.Context) (*RequestContext, error) {
	req := c.Request()
	headers := make(map[string]string)

	// Copy all headers
	for key, values := range req.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}

	reqCtx := &RequestContext{
		Headers: headers,
		Cookies: req.Cookies(),
	}

	// Try Traefik headers first (X-Forwarded-*)
	if method := req.Header.Get("X-Forwarded-Method"); method != "" {
		reqCtx.Method = method
		reqCtx.Path = req.Header.Get("X-Forwarded-Uri")
		reqCtx.Host = req.Header.Get("X-Forwarded-Host")

		// Build original URL
		proto := req.Header.Get("X-Forwarded-Proto")
		if proto == "" {
			proto = "https"
		}
		reqCtx.OriginalURL = fmt.Sprintf("%s://%s%s", proto, reqCtx.Host, reqCtx.Path)

		slog.DebugContext(c.Request().Context(), "Normalized Traefik headers",
			slog.String("method", reqCtx.Method),
			slog.String("path", reqCtx.Path),
			slog.String("host", reqCtx.Host),
		)

		return reqCtx, nil
	}

	// Try Nginx headers (X-Original-*)
	if method := req.Header.Get("X-Original-Method"); method != "" {
		reqCtx.Method = method
		reqCtx.Path = req.Header.Get("X-Original-URI")
		reqCtx.Host = req.Header.Get("X-Original-Host")

		// Build original URL
		proto := req.Header.Get("X-Original-Proto")
		if proto == "" {
			proto = "https"
		}
		reqCtx.OriginalURL = fmt.Sprintf("%s://%s%s", proto, reqCtx.Host, reqCtx.Path)

		slog.DebugContext(c.Request().Context(), "Normalized Nginx headers",
			slog.String("method", reqCtx.Method),
			slog.String("path", reqCtx.Path),
			slog.String("host", reqCtx.Host),
		)

		return reqCtx, nil
	}

	// No forwarded headers found, use direct request
	reqCtx.Method = req.Method
	reqCtx.Path = req.URL.Path
	reqCtx.Host = req.Host
	reqCtx.OriginalURL = req.URL.String()

	slog.DebugContext(c.Request().Context(), "Using direct request (no forwarded headers)",
		slog.String("method", reqCtx.Method),
		slog.String("path", reqCtx.Path),
		slog.String("host", reqCtx.Host),
	)

	return reqCtx, nil
}

// isCallbackPath checks if the path is the OIDC callback endpoint
func (a *ForwardAuthAdapter) isCallbackPath(path string) bool {
	return strings.HasSuffix(path, "/auth/callback")
}

// handleCallback processes the OIDC callback
func (a *ForwardAuthAdapter) handleCallback(c echo.Context, ctx context.Context) error {
	slog.InfoContext(ctx, "Processing OIDC callback in ForwardAuth mode")

	// Extract query parameters
	stateParam := c.QueryParam("state")
	code := c.QueryParam("code")
	errorParam := c.QueryParam("error")
	errorDesc := c.QueryParam("error_description")

	// We need access to the OIDC provider to complete the callback
	// This will be handled by adding a callback route in the server
	// For now, return an error indicating this should be handled by a dedicated route

	slog.ErrorContext(ctx, "OIDC callback should be handled by dedicated /auth/callback route")

	return c.JSON(http.StatusInternalServerError, map[string]string{
		"error":             "OIDC callback should be handled by dedicated route",
		"state":             stateParam,
		"code":              code,
		"error_param":       errorParam,
		"error_description": errorDesc,
	})
}

// buildResponse returns the appropriate response based on authentication result
func (a *ForwardAuthAdapter) buildResponse(c echo.Context, result *auth.EngineResult) error {
	ctx := c.Request().Context()

	if result.Authenticated {
		// Authentication successful - return 200 with X-Es-Authorization header
		slog.InfoContext(ctx, "ForwardAuth authentication successful",
			slog.String("username", result.Username),
			slog.String("es_user", result.ESUser),
		)

		// Set X-Es-Authorization header
		c.Response().Header().Set("X-Es-Authorization", result.ESAuthHeader)

		// Preserve Cookie headers from original request
		for _, cookie := range c.Request().Cookies() {
			c.Response().Header().Add("Set-Cookie", cookie.String())
		}

		return c.NoContent(http.StatusOK)
	}

	// Authentication failed or redirect needed
	if result.RedirectURL != "" {
		// OIDC flow - redirect to provider
		slog.InfoContext(ctx, "ForwardAuth redirecting to OIDC provider")
		return c.Redirect(http.StatusFound, result.RedirectURL)
	}

	// Basic Auth failure or other error
	slog.WarnContext(ctx, "ForwardAuth authentication failed",
		slog.Int("status_code", result.StatusCode),
	)

	// For Basic Auth failures, include WWW-Authenticate header
	if result.StatusCode == http.StatusUnauthorized {
		c.Response().Header().Set("WWW-Authenticate", `Basic realm="Keyline"`)
	}

	return c.JSON(result.StatusCode, map[string]string{
		"error": "Authentication required",
	})
}
