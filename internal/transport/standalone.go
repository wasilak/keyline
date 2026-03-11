package transport

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/wasilak/cachego"
	"github.com/yourusername/keyline/internal/auth"
	"github.com/yourusername/keyline/internal/config"
	"github.com/yourusername/keyline/internal/observability"
	"go.opentelemetry.io/otel"
)

// StandaloneProxyAdapter handles standalone proxy mode
type StandaloneProxyAdapter struct {
	config      *config.Config
	cache       cachego.CacheInterface
	authEngine  *auth.Engine
	proxy       *httputil.ReverseProxy
	upstreamURL *url.URL
}

// NewStandaloneProxyAdapter creates a new standalone proxy adapter
func NewStandaloneProxyAdapter(cfg *config.Config, cache cachego.CacheInterface, authEngine *auth.Engine) (*StandaloneProxyAdapter, error) {
	// Parse upstream URL
	upstreamURL, err := url.Parse(cfg.Upstream.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid upstream URL: %w", err)
	}

	adapter := &StandaloneProxyAdapter{
		config:      cfg,
		cache:       cache,
		authEngine:  authEngine,
		upstreamURL: upstreamURL,
	}

	// Initialize reverse proxy
	adapter.proxy = &httputil.ReverseProxy{
		Director: adapter.director,
		Transport: &http.Transport{
			MaxIdleConns:        cfg.Upstream.MaxIdleConns,
			MaxIdleConnsPerHost: cfg.Upstream.MaxIdleConns,
			IdleConnTimeout:     90 * time.Second,
			TLSHandshakeTimeout: 10 * time.Second,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: cfg.Upstream.InsecureSkipVerify,
			},
		},
		ErrorHandler: adapter.errorHandler,
	}

	return adapter, nil
}

// Name returns the adapter name
func (a *StandaloneProxyAdapter) Name() string {
	return "standalone"
}

// HandleRequest processes an incoming request in standalone mode
func (a *StandaloneProxyAdapter) HandleRequest(c echo.Context) error {
	ctx := c.Request().Context()
	path := c.Request().URL.Path

	slog.InfoContext(ctx, "Standalone proxy request received",
		slog.String("method", c.Request().Method),
		slog.String("path", path),
		slog.String("host", c.Request().Host),
	)

	// Don't proxy internal endpoints
	if a.isInternalEndpoint(path) {
		slog.DebugContext(ctx, "Internal endpoint, not proxying",
			slog.String("path", path),
		)
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Not found",
		})
	}

	// Build original URL
	originalURL := a.buildOriginalURL(c.Request())

	// Build engine request
	engineReq := &auth.EngineRequest{
		Method:              c.Request().Method,
		Path:                path,
		Host:                c.Request().Host,
		Headers:             a.extractHeaders(c.Request()),
		Cookies:             c.Request().Cookies(),
		OriginalURL:         originalURL,
		AuthorizationHeader: c.Request().Header.Get("Authorization"),
		SourceIP:            observability.ExtractSourceIP(c.Request()),
	}

	// Authenticate request
	result := a.authEngine.Authenticate(ctx, engineReq)

	if !result.Authenticated {
		// Handle authentication failure
		if result.RedirectURL != "" {
			// OIDC flow - redirect to provider
			slog.InfoContext(ctx, "Redirecting to OIDC provider")
			return c.Redirect(http.StatusFound, result.RedirectURL)
		}

		// Basic Auth failure or other error
		slog.WarnContext(ctx, "Authentication failed",
			slog.Int("status_code", result.StatusCode),
			slog.String("error", result.Error.Error()),
		)

		// Determine appropriate error message
		errorMessage := "Authentication required"
		if engineReq.AuthorizationHeader != "" {
			// Credentials were provided but invalid
			errorMessage = "Authentication failed: invalid credentials"
		}

		if result.StatusCode == http.StatusUnauthorized {
			c.Response().Header().Set("WWW-Authenticate", `Basic realm="Keyline"`)
		}

		return c.JSON(result.StatusCode, map[string]string{
			"error": errorMessage,
		})
	}

	// Authentication successful - add ES authorization header and proxy
	slog.InfoContext(ctx, "Authentication successful, proxying request",
		slog.String("username", result.Username),
		slog.String("es_user", result.ESUser),
		slog.String("upstream", a.upstreamURL.String()),
	)

	// Create span for upstream proxy request
	tracer := otel.Tracer("keyline")
	ctx, span := tracer.Start(ctx, "keyline.proxy.request")
	defer span.End()

	// Update request context with span
	c.SetRequest(c.Request().WithContext(ctx))

	// Add X-Es-Authorization header
	c.Request().Header.Set("X-Es-Authorization", result.ESAuthHeader)

	// Check for WebSocket upgrade
	if a.isWebSocketUpgrade(c.Request()) {
		return a.handleWebSocket(c, ctx, result.ESAuthHeader)
	}

	// Proxy the request
	a.proxy.ServeHTTP(c.Response(), c.Request())

	return nil
}

// director modifies the request before proxying
func (a *StandaloneProxyAdapter) director(req *http.Request) {
	// Preserve original request details
	req.URL.Scheme = a.upstreamURL.Scheme
	req.URL.Host = a.upstreamURL.Host
	req.Host = a.upstreamURL.Host

	// Replace Authorization header with ES credentials
	// The X-Es-Authorization header contains the ES user's credentials
	esAuth := req.Header.Get("X-Es-Authorization")
	if esAuth != "" {
		// Remove the original Authorization header (client's auth to Keyline)
		req.Header.Del("Authorization")
		// Set the Authorization header with ES credentials
		req.Header.Set("Authorization", esAuth)
		// Remove the X-Es-Authorization header (internal use only)
		req.Header.Del("X-Es-Authorization")
	}

	// Remove hop-by-hop headers
	a.removeHopByHopHeaders(req.Header)

	// Add X-Forwarded-* headers
	if req.Header.Get("X-Forwarded-For") == "" {
		req.Header.Set("X-Forwarded-For", req.RemoteAddr)
	}
	if req.Header.Get("X-Forwarded-Proto") == "" {
		if req.TLS != nil {
			req.Header.Set("X-Forwarded-Proto", "https")
		} else {
			req.Header.Set("X-Forwarded-Proto", "http")
		}
	}
	if req.Header.Get("X-Forwarded-Host") == "" {
		req.Header.Set("X-Forwarded-Host", req.Host)
	}
}

// errorHandler handles proxy errors
func (a *StandaloneProxyAdapter) errorHandler(w http.ResponseWriter, r *http.Request, err error) {
	ctx := r.Context()

	slog.ErrorContext(ctx, "Upstream proxy error",
		slog.String("upstream_url", a.upstreamURL.String()),
		slog.String("error", err.Error()),
	)

	// Determine error type and return appropriate status code and message
	if err == context.DeadlineExceeded {
		// Timeout
		slog.ErrorContext(ctx, "Upstream request timeout")
		w.WriteHeader(http.StatusGatewayTimeout)
		w.Write([]byte(`{"error":"Gateway Timeout: upstream request timed out"}`))
		return
	}

	// Check for TLS/SSL certificate errors
	if strings.Contains(err.Error(), "x509") || strings.Contains(err.Error(), "certificate") || strings.Contains(err.Error(), "tls") {
		slog.ErrorContext(ctx, "TLS/SSL certificate error - consider using insecure_skip_verify for self-signed certificates",
			slog.String("error_detail", err.Error()),
		)
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(`{"error":"TLS/SSL certificate verification failed. If using self-signed certificates, set 'insecure_skip_verify: true' in upstream config."}`))
		return
	}

	// Check if it's a connection error
	if netErr, ok := err.(net.Error); ok {
		if netErr.Timeout() {
			slog.ErrorContext(ctx, "Upstream connection timeout")
			w.WriteHeader(http.StatusGatewayTimeout)
			w.Write([]byte(`{"error":"Gateway Timeout: connection to upstream timed out"}`))
		} else {
			slog.ErrorContext(ctx, "Upstream connection failed")
			w.WriteHeader(http.StatusBadGateway)
			w.Write([]byte(`{"error":"Bad Gateway: failed to connect to upstream Elasticsearch"}`))
		}
		return
	}

	// Check for connection refused
	if strings.Contains(err.Error(), "connection refused") {
		slog.ErrorContext(ctx, "Upstream connection refused")
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(`{"error":"Bad Gateway: connection refused by upstream Elasticsearch"}`))
		return
	}

	// Generic error
	w.WriteHeader(http.StatusBadGateway)
	w.Write([]byte(fmt.Sprintf(`{"error":"Bad Gateway: %s"}`, err.Error())))
}

// isInternalEndpoint checks if the path is an internal endpoint
func (a *StandaloneProxyAdapter) isInternalEndpoint(path string) bool {
	internalPaths := []string{
		"/auth/callback",
		"/auth/logout",
		"/healthz",
		"/metrics",
	}

	for _, internal := range internalPaths {
		if path == internal || strings.HasPrefix(path, internal+"/") {
			return true
		}
	}

	return false
}

// buildOriginalURL builds the full original URL from the request
func (a *StandaloneProxyAdapter) buildOriginalURL(req *http.Request) string {
	scheme := "http"
	if req.TLS != nil {
		scheme = "https"
	}

	return fmt.Sprintf("%s://%s%s", scheme, req.Host, req.URL.RequestURI())
}

// extractHeaders extracts all headers from the request
func (a *StandaloneProxyAdapter) extractHeaders(req *http.Request) map[string]string {
	headers := make(map[string]string)
	for key, values := range req.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}
	return headers
}

// removeHopByHopHeaders removes hop-by-hop headers that shouldn't be proxied
func (a *StandaloneProxyAdapter) removeHopByHopHeaders(headers http.Header) {
	hopByHopHeaders := []string{
		"Connection",
		"Keep-Alive",
		"Proxy-Authenticate",
		"Proxy-Authorization",
		"Te",
		"Trailers",
		"Transfer-Encoding",
		"Upgrade",
	}

	for _, header := range hopByHopHeaders {
		headers.Del(header)
	}
}

// isWebSocketUpgrade checks if the request is a WebSocket upgrade
func (a *StandaloneProxyAdapter) isWebSocketUpgrade(req *http.Request) bool {
	return strings.ToLower(req.Header.Get("Upgrade")) == "websocket" &&
		strings.Contains(strings.ToLower(req.Header.Get("Connection")), "upgrade")
}

// handleWebSocket handles WebSocket upgrade requests
func (a *StandaloneProxyAdapter) handleWebSocket(c echo.Context, ctx context.Context, esAuthHeader string) error {
	slog.InfoContext(ctx, "Handling WebSocket upgrade request")

	// Build upstream WebSocket URL
	upstreamWSURL := *a.upstreamURL
	if upstreamWSURL.Scheme == "https" {
		upstreamWSURL.Scheme = "wss"
	} else {
		upstreamWSURL.Scheme = "ws"
	}
	upstreamWSURL.Path = c.Request().URL.Path
	upstreamWSURL.RawQuery = c.Request().URL.RawQuery

	// Create upstream connection
	upstreamConn, err := net.DialTimeout("tcp", a.upstreamURL.Host, a.config.Upstream.Timeout)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to connect to upstream for WebSocket",
			slog.String("error", err.Error()),
		)
		return c.JSON(http.StatusBadGateway, map[string]string{
			"error": "Failed to connect to upstream",
		})
	}
	defer upstreamConn.Close()

	// Hijack the client connection
	hijacker, ok := c.Response().Writer.(http.Hijacker)
	if !ok {
		slog.ErrorContext(ctx, "Response writer doesn't support hijacking")
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "WebSocket upgrade not supported",
		})
	}

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		slog.ErrorContext(ctx, "Failed to hijack connection",
			slog.String("error", err.Error()),
		)
		return err
	}
	defer clientConn.Close()

	// Forward upgrade request to upstream
	req := c.Request()
	req.Header.Set("X-Es-Authorization", esAuthHeader)

	if err := req.Write(upstreamConn); err != nil {
		slog.ErrorContext(ctx, "Failed to write upgrade request to upstream",
			slog.String("error", err.Error()),
		)
		return err
	}

	// Bidirectional copy
	errChan := make(chan error, 2)

	// Client -> Upstream
	go func() {
		_, err := io.Copy(upstreamConn, clientConn)
		errChan <- err
	}()

	// Upstream -> Client
	go func() {
		_, err := io.Copy(clientConn, upstreamConn)
		errChan <- err
	}()

	// Wait for either direction to complete
	err = <-errChan

	if err != nil && err != io.EOF {
		slog.WarnContext(ctx, "WebSocket connection error",
			slog.String("error", err.Error()),
		)
	} else {
		slog.InfoContext(ctx, "WebSocket connection closed")
	}

	return nil
}
