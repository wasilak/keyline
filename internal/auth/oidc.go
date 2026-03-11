package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/wasilak/cachego"
	"github.com/yourusername/keyline/internal/cache"
	"github.com/yourusername/keyline/internal/config"
	"github.com/yourusername/keyline/internal/mapper"
	"github.com/yourusername/keyline/internal/session"
	"github.com/yourusername/keyline/internal/state"
	pkgcrypto "github.com/yourusername/keyline/pkg/crypto"
	"go.opentelemetry.io/otel"
	"gopkg.in/square/go-jose.v2"
)

// OIDCProvider implements OIDC authentication
type OIDCProvider struct {
	config        *config.OIDCConfig
	sessionConfig *config.SessionConfig
	cache         *cache.OIDCCache
	httpClient    *http.Client
	mapper        *mapper.CredentialMapper
}

// NewOIDCProvider creates a new OIDC provider
func NewOIDCProvider(cfg *config.OIDCConfig, fullConfig *config.Config) (*OIDCProvider, error) {
	if !cfg.Enabled {
		return nil, fmt.Errorf("OIDC is not enabled")
	}

	// Create HTTP client with TLS certificate validation enabled
	// This ensures all OIDC provider requests use HTTPS with proper cert validation
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: nil, // nil means use default TLS config with cert validation
		},
	}

	provider := &OIDCProvider{
		config:        cfg,
		sessionConfig: &fullConfig.Session,
		cache:         cache.NewOIDCCache(),
		httpClient:    httpClient,
		mapper:        mapper.NewCredentialMapper(fullConfig),
	}

	// Perform discovery during initialization
	if err := provider.discover(context.Background()); err != nil {
		return nil, fmt.Errorf("OIDC discovery failed: %w", err)
	}

	// Fetch and cache JWKS during initialization
	if err := provider.fetchJWKS(context.Background()); err != nil {
		return nil, fmt.Errorf("JWKS fetch failed: %w", err)
	}

	// Start background JWKS refresh goroutine
	go provider.startJWKSRefresh()

	return provider, nil
}

// discover fetches and validates the OIDC discovery document
func (p *OIDCProvider) discover(ctx context.Context) error {
	// Validate that issuer URL uses HTTPS (or HTTP for localhost)
	if !isHTTPSOrLocalhostHTTP(p.config.IssuerURL) {
		return fmt.Errorf("OIDC issuer URL must use HTTPS (or HTTP for localhost/127.0.0.1): %s", p.config.IssuerURL)
	}

	discoveryURL := p.config.IssuerURL + "/.well-known/openid-configuration"

	slog.InfoContext(ctx, "Fetching OIDC discovery document",
		slog.String("url", discoveryURL),
	)

	var lastErr error

	// Retry up to 3 times with exponential backoff
	for attempt := 1; attempt <= 3; attempt++ {
		doc, err := p.fetchDiscoveryDocument(ctx, discoveryURL)
		if err == nil {
			// Validate issuer matches configuration
			if doc.Issuer != p.config.IssuerURL {
				return fmt.Errorf("issuer mismatch: expected %s, got %s", p.config.IssuerURL, doc.Issuer)
			}

			// Cache the discovery document
			p.cache.SetDiscoveryDoc(doc)

			slog.InfoContext(ctx, "OIDC discovery successful",
				slog.String("issuer", doc.Issuer),
				slog.String("authorization_endpoint", doc.AuthorizationEndpoint),
				slog.String("token_endpoint", doc.TokenEndpoint),
			)

			return nil
		}

		lastErr = err

		if attempt < 3 {
			// Exponential backoff: 1s, 2s, 4s
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			slog.WarnContext(ctx, "OIDC discovery attempt failed, retrying",
				slog.Int("attempt", attempt),
				slog.Duration("backoff", backoff),
				slog.String("error", err.Error()),
			)
			time.Sleep(backoff)
		}
	}

	return fmt.Errorf("OIDC discovery failed after 3 attempts: %w", lastErr)
}

// fetchDiscoveryDocument fetches the discovery document from the OIDC provider
func (p *OIDCProvider) fetchDiscoveryDocument(ctx context.Context, url string) (*cache.DiscoveryDocument, error) {
	// Create span for discovery document fetch
	tracer := otel.Tracer("keyline")
	ctx, span := tracer.Start(ctx, "keyline.oidc.discovery")
	defer span.End()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch discovery document: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var doc cache.DiscoveryDocument
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("failed to parse discovery document: %w", err)
	}

	// Validate required fields
	if doc.Issuer == "" {
		return nil, fmt.Errorf("discovery document missing issuer")
	}
	if doc.AuthorizationEndpoint == "" {
		return nil, fmt.Errorf("discovery document missing authorization_endpoint")
	}
	if doc.TokenEndpoint == "" {
		return nil, fmt.Errorf("discovery document missing token_endpoint")
	}
	if doc.JWKSURI == "" {
		return nil, fmt.Errorf("discovery document missing jwks_uri")
	}

	// Validate that all endpoints use HTTPS (or HTTP for localhost)
	if !isHTTPSOrLocalhostHTTP(doc.TokenEndpoint) {
		return nil, fmt.Errorf("token_endpoint must use HTTPS (or HTTP for localhost): %s", doc.TokenEndpoint)
	}
	if !isHTTPSOrLocalhostHTTP(doc.JWKSURI) {
		return nil, fmt.Errorf("jwks_uri must use HTTPS (or HTTP for localhost): %s", doc.JWKSURI)
	}
	if doc.UserinfoEndpoint != "" && !isHTTPSOrLocalhostHTTP(doc.UserinfoEndpoint) {
		return nil, fmt.Errorf("userinfo_endpoint must use HTTPS (or HTTP for localhost): %s", doc.UserinfoEndpoint)
	}

	return &doc, nil
}

// GetDiscoveryDoc returns the cached discovery document
func (p *OIDCProvider) GetDiscoveryDoc() *cache.DiscoveryDocument {
	return p.cache.GetDiscoveryDoc()
}

// fetchJWKS fetches and caches the JWKS from the OIDC provider
func (p *OIDCProvider) fetchJWKS(ctx context.Context) error {
	doc := p.cache.GetDiscoveryDoc()
	if doc == nil {
		return fmt.Errorf("discovery document not loaded")
	}

	slog.InfoContext(ctx, "Fetching JWKS",
		slog.String("jwks_uri", doc.JWKSURI),
	)

	var lastErr error

	// Retry up to 3 times with exponential backoff
	for attempt := 1; attempt <= 3; attempt++ {
		jwks, err := p.fetchJWKSFromURL(ctx, doc.JWKSURI)
		if err == nil {
			// Cache JWKS with 24-hour expiry
			p.cache.SetJWKS(jwks, 24*time.Hour)

			slog.InfoContext(ctx, "JWKS fetched and cached successfully",
				slog.Int("key_count", len(jwks.Keys)),
			)

			return nil
		}

		lastErr = err

		if attempt < 3 {
			// Exponential backoff: 1s, 2s, 4s
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			slog.WarnContext(ctx, "JWKS fetch attempt failed, retrying",
				slog.Int("attempt", attempt),
				slog.Duration("backoff", backoff),
				slog.String("error", err.Error()),
			)
			time.Sleep(backoff)
		}
	}

	return fmt.Errorf("JWKS fetch failed after 3 attempts: %w", lastErr)
}

// fetchJWKSFromURL fetches JWKS from the given URL
func (p *OIDCProvider) fetchJWKSFromURL(ctx context.Context, url string) (*jose.JSONWebKeySet, error) {
	// Create span for JWKS fetch
	tracer := otel.Tracer("keyline")
	ctx, span := tracer.Start(ctx, "keyline.oidc.jwks")
	defer span.End()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	jwks, err := cache.ParseJWKS(body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JWKS: %w", err)
	}

	return jwks, nil
}

// startJWKSRefresh starts a background goroutine to refresh JWKS every 24 hours
func (p *OIDCProvider) startJWKSRefresh() {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		ctx := context.Background()

		slog.InfoContext(ctx, "Starting JWKS background refresh")

		if err := p.fetchJWKS(ctx); err != nil {
			// Log warning but continue with cached JWKS
			slog.WarnContext(ctx, "JWKS refresh failed, continuing with cached JWKS",
				slog.String("error", err.Error()),
			)
		} else {
			slog.InfoContext(ctx, "JWKS refresh successful")
		}
	}
}

// GetJWKS returns the cached JWKS
func (p *OIDCProvider) GetJWKS() (*jose.JSONWebKeySet, bool) {
	return p.cache.GetJWKS()
}

// Authenticate initiates the OIDC authorization flow
func (p *OIDCProvider) Authenticate(ctx context.Context, cachego cachego.CacheInterface, originalURL string) (string, error) {
	// Generate state token
	stateID, err := pkgcrypto.GenerateStateToken()
	if err != nil {
		return "", fmt.Errorf("failed to generate state token: %w", err)
	}

	// Generate PKCE pair
	pkce, err := pkgcrypto.GeneratePKCE()
	if err != nil {
		return "", fmt.Errorf("failed to generate PKCE: %w", err)
	}

	// Create state token
	token := &state.Token{
		ID:           stateID,
		OriginalURL:  originalURL,
		CodeVerifier: pkce.Verifier,
		CreatedAt:    time.Now(),
		Used:         false,
	}

	// Store state token
	if err := state.StoreStateToken(ctx, cachego, token); err != nil {
		return "", fmt.Errorf("failed to store state token: %w", err)
	}

	// Build authorization URL
	authURL, err := p.buildAuthorizationURL(stateID, pkce.Challenge)
	if err != nil {
		return "", fmt.Errorf("failed to build authorization URL: %w", err)
	}

	slog.InfoContext(ctx, "OIDC authorization flow initiated",
		slog.String("state_token_id", stateID),
		slog.String("original_url", originalURL),
	)

	return authURL, nil
}

// buildAuthorizationURL builds the OIDC authorization URL with all required parameters
func (p *OIDCProvider) buildAuthorizationURL(state, codeChallenge string) (string, error) {
	doc := p.cache.GetDiscoveryDoc()
	if doc == nil {
		return "", fmt.Errorf("discovery document not loaded")
	}

	// Build query parameters
	params := url.Values{}
	params.Set("client_id", p.config.ClientID)
	params.Set("redirect_uri", p.config.RedirectURL)
	params.Set("response_type", "code")
	params.Set("scope", joinScopes(p.config.Scopes))
	params.Set("state", state)
	params.Set("code_challenge", codeChallenge)
	params.Set("code_challenge_method", "S256")

	authURL := doc.AuthorizationEndpoint + "?" + params.Encode()
	return authURL, nil
}

// joinScopes joins scope strings with spaces
func joinScopes(scopes []string) string {
	if len(scopes) == 0 {
		return "openid"
	}
	return joinStrings(scopes, " ")
}

// joinStrings joins strings with a separator
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}

// CallbackResult represents the result of processing an OIDC callback
type CallbackResult struct {
	StateToken   *state.Token
	Code         string
	Error        string
	ErrorMessage string
}

// HandleCallback processes the OIDC callback
func (p *OIDCProvider) HandleCallback(ctx context.Context, cachego cachego.CacheInterface, stateParam, code, errorParam, errorDesc string) (*CallbackResult, error) {
	result := &CallbackResult{
		Code:         code,
		Error:        errorParam,
		ErrorMessage: errorDesc,
	}

	// Check for error from OIDC provider
	if errorParam != "" {
		slog.WarnContext(ctx, "OIDC callback received error",
			slog.String("error", errorParam),
			slog.String("error_description", errorDesc),
		)
		return result, fmt.Errorf("OIDC provider error: %s - %s", errorParam, errorDesc)
	}

	// Validate state parameter
	if stateParam == "" {
		slog.WarnContext(ctx, "OIDC callback missing state parameter")
		return result, fmt.Errorf("missing state parameter")
	}

	// Retrieve state token
	token, err := state.GetStateToken(ctx, cachego, stateParam)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to retrieve state token",
			slog.String("error", err.Error()),
		)
		return result, fmt.Errorf("failed to retrieve state token: %w", err)
	}

	if token == nil {
		slog.WarnContext(ctx, "Invalid or expired state token",
			slog.String("state_token_id", stateParam),
		)
		return result, fmt.Errorf("invalid or expired state token")
	}

	// State token is automatically marked as used and deleted by GetStateToken
	result.StateToken = token

	// Validate code parameter
	if code == "" {
		slog.WarnContext(ctx, "OIDC callback missing code parameter")
		return result, fmt.Errorf("missing code parameter")
	}

	slog.InfoContext(ctx, "OIDC callback validated successfully",
		slog.String("state_token_id", stateParam),
		slog.String("original_url", token.OriginalURL),
	)

	return result, nil
}

// TokenResponse represents the OIDC token response
type TokenResponse struct {
	IDToken      string `json:"id_token"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

// ExchangeToken exchanges an authorization code for tokens
func (p *OIDCProvider) ExchangeToken(ctx context.Context, code, codeVerifier string) (*TokenResponse, error) {
	// Create span for token exchange
	tracer := otel.Tracer("keyline")
	ctx, span := tracer.Start(ctx, "keyline.oidc.token")
	defer span.End()

	doc := p.cache.GetDiscoveryDoc()
	if doc == nil {
		return nil, fmt.Errorf("discovery document not loaded")
	}

	slog.InfoContext(ctx, "Exchanging authorization code for tokens")

	// Build token request
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", p.config.RedirectURL)
	data.Set("client_id", p.config.ClientID)
	data.Set("client_secret", p.config.ClientSecret)
	data.Set("code_verifier", codeVerifier)

	// Create request with 30-second timeout
	reqCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, doc.TokenEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Body = io.NopCloser(bytes.NewBufferString(data.Encode()))

	// Make request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		slog.ErrorContext(ctx, "Token exchange request failed",
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("token exchange request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read token response: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		slog.ErrorContext(ctx, "Token exchange failed",
			slog.Int("status_code", resp.StatusCode),
			slog.String("response", string(body)),
		)
		return nil, fmt.Errorf("token exchange failed with status %d", resp.StatusCode)
	}

	// Parse response
	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	slog.InfoContext(ctx, "Token exchange successful")

	return &tokenResp, nil
}

// IDTokenClaims represents the claims in an ID token
type IDTokenClaims struct {
	Issuer    string                 `json:"iss"`
	Subject   string                 `json:"sub"`
	Audience  interface{}            `json:"aud"` // Can be string or []string
	ExpiresAt int64                  `json:"exp"`
	IssuedAt  int64                  `json:"iat"`
	Email     string                 `json:"email,omitempty"`
	Name      string                 `json:"name,omitempty"`
	Claims    map[string]interface{} `json:"-"` // Store all claims
}

// ValidateIDToken validates an ID token and returns the claims
func (p *OIDCProvider) ValidateIDToken(ctx context.Context, idToken string) (*IDTokenClaims, error) {
	slog.InfoContext(ctx, "Validating ID token")

	// Parse the JWT
	token, err := jose.ParseSigned(idToken)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ID token: %w", err)
	}

	// Get JWKS
	jwks, valid := p.cache.GetJWKS()
	if !valid || jwks == nil {
		return nil, fmt.Errorf("JWKS not available or expired")
	}

	// Verify signature with each key until one works
	var claims map[string]interface{}
	var verifyErr error

	for _, key := range jwks.Keys {
		output, err := token.Verify(&key)
		if err != nil {
			verifyErr = err
			continue
		}

		// Parse claims
		if err := json.Unmarshal(output, &claims); err != nil {
			return nil, fmt.Errorf("failed to parse claims: %w", err)
		}

		verifyErr = nil
		break
	}

	if verifyErr != nil {
		slog.ErrorContext(ctx, "ID token signature verification failed",
			slog.String("error", verifyErr.Error()),
		)
		return nil, fmt.Errorf("invalid token signature")
	}

	// Extract standard claims
	idClaims := &IDTokenClaims{
		Claims: claims,
	}

	// Extract issuer
	if iss, ok := claims["iss"].(string); ok {
		idClaims.Issuer = iss
	}

	// Extract subject
	if sub, ok := claims["sub"].(string); ok {
		idClaims.Subject = sub
	}

	// Extract audience (can be string or array)
	idClaims.Audience = claims["aud"]

	// Extract expiration
	if exp, ok := claims["exp"].(float64); ok {
		idClaims.ExpiresAt = int64(exp)
	}

	// Extract issued at
	if iat, ok := claims["iat"].(float64); ok {
		idClaims.IssuedAt = int64(iat)
	}

	// Extract email
	if email, ok := claims["email"].(string); ok {
		idClaims.Email = email
	}

	// Extract name
	if name, ok := claims["name"].(string); ok {
		idClaims.Name = name
	}

	// Validate issuer
	if idClaims.Issuer != p.config.IssuerURL {
		slog.ErrorContext(ctx, "ID token issuer mismatch",
			slog.String("expected", p.config.IssuerURL),
			slog.String("actual", idClaims.Issuer),
		)
		return nil, fmt.Errorf("invalid issuer")
	}

	// Validate audience
	if !p.validateAudience(idClaims.Audience) {
		slog.ErrorContext(ctx, "ID token audience mismatch",
			slog.String("expected", p.config.ClientID),
		)
		return nil, fmt.Errorf("invalid audience")
	}

	// Validate expiration
	now := time.Now().Unix()
	if idClaims.ExpiresAt <= now {
		slog.ErrorContext(ctx, "ID token expired",
			slog.Int64("exp", idClaims.ExpiresAt),
			slog.Int64("now", now),
		)
		return nil, fmt.Errorf("token expired")
	}

	slog.InfoContext(ctx, "ID token validated successfully",
		slog.String("subject", idClaims.Subject),
		slog.String("email", idClaims.Email),
	)

	return idClaims, nil
}

// validateAudience checks if the audience claim matches the client ID
func (p *OIDCProvider) validateAudience(aud interface{}) bool {
	switch v := aud.(type) {
	case string:
		return v == p.config.ClientID
	case []interface{}:
		for _, a := range v {
			if str, ok := a.(string); ok && str == p.config.ClientID {
				return true
			}
		}
	case []string:
		for _, a := range v {
			if a == p.config.ClientID {
				return true
			}
		}
	}
	return false
}

// CreateSessionFromClaims creates a session from validated ID token claims
func (p *OIDCProvider) CreateSessionFromClaims(ctx context.Context, cachego cachego.CacheInterface, claims *IDTokenClaims, sessionTTL time.Duration) (*session.Session, *http.Cookie, error) {
	// Generate cryptographically random session ID
	sessionID, err := pkgcrypto.GenerateSessionID()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate session ID: %w", err)
	}

	// Map OIDC user to ES user using credential mapper
	esUser, err := p.mapper.MapOIDCUser(ctx, claims.Claims)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to map OIDC user to ES user: %w", err)
	}

	// Create session
	now := time.Now()
	sess := &session.Session{
		ID:        sessionID,
		UserID:    claims.Subject,
		Username:  claims.Email,
		Email:     claims.Email,
		ESUser:    esUser,
		Claims:    claims.Claims,
		CreatedAt: now,
		ExpiresAt: now.Add(sessionTTL),
	}

	// Store session
	if err := session.CreateSession(ctx, cachego, sess); err != nil {
		return nil, nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Create session cookie
	// Use session config for cookie settings
	cookieName := p.sessionConfig.CookieName
	if cookieName == "" {
		cookieName = "keyline_session"
	}
	
	cookiePath := p.sessionConfig.CookiePath
	if cookiePath == "" {
		cookiePath = "/"
	}
	
	// For localhost testing, set Secure=false (cookies with Secure=true won't be sent over HTTP)
	isLocalhost := isHTTPSOrLocalhostHTTP(p.config.RedirectURL) && !strings.HasPrefix(p.config.RedirectURL, "https://")
	
	cookie := &http.Cookie{
		Name:     cookieName,
		Value:    sessionID,
		Path:     cookiePath,
		Domain:   p.sessionConfig.CookieDomain,
		HttpOnly: true,
		Secure:   !isLocalhost, // false for localhost HTTP, true for HTTPS
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(sessionTTL.Seconds()),
	}

	slog.InfoContext(ctx, "Session created from OIDC user",
		slog.String("user_id", claims.Subject),
		slog.String("email", claims.Email),
		slog.String("es_user", esUser),
		slog.Duration("ttl", sessionTTL),
	)

	return sess, cookie, nil
}

// CompleteCallback completes the OIDC callback flow and returns the redirect URL
// This method orchestrates the entire callback process:
// 1. Validates callback parameters and state token
// 2. Exchanges authorization code for tokens
// 3. Validates ID token
// 4. Creates session from user claims
// 5. Returns original URL for redirect
func (p *OIDCProvider) CompleteCallback(ctx context.Context, cachego cachego.CacheInterface, stateParam, code, errorParam, errorDesc string, sessionTTL time.Duration) (redirectURL string, cookie *http.Cookie, err error) {
	// Handle callback and validate state
	result, err := p.HandleCallback(ctx, cachego, stateParam, code, errorParam, errorDesc)
	if err != nil {
		return "", nil, err
	}

	// Exchange authorization code for tokens
	tokenResp, err := p.ExchangeToken(ctx, result.Code, result.StateToken.CodeVerifier)
	if err != nil {
		return "", nil, fmt.Errorf("token exchange failed: %w", err)
	}

	// Validate ID token
	claims, err := p.ValidateIDToken(ctx, tokenResp.IDToken)
	if err != nil {
		return "", nil, fmt.Errorf("ID token validation failed: %w", err)
	}

	// Create session from claims
	_, cookie, err = p.CreateSessionFromClaims(ctx, cachego, claims, sessionTTL)
	if err != nil {
		return "", nil, fmt.Errorf("session creation failed: %w", err)
	}

	// Return original URL from state token for redirect
	redirectURL = result.StateToken.OriginalURL

	slog.InfoContext(ctx, "OIDC callback completed successfully",
		slog.String("user_id", claims.Subject),
		slog.String("email", claims.Email),
		slog.String("redirect_url", redirectURL),
	)

	return redirectURL, cookie, nil
}

// isHTTPSOrLocalhostHTTP checks if a URL uses HTTPS or HTTP for localhost
// For production, HTTPS should be enforced, but for local testing we allow HTTP for localhost/127.0.0.1
func isHTTPSOrLocalhostHTTP(urlStr string) bool {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return false
	}
	
	// Allow HTTPS for all hosts
	if parsedURL.Scheme == "https" {
		return true
	}
	
	// Allow HTTP only for localhost and 127.0.0.1
	if parsedURL.Scheme == "http" {
		hostname := parsedURL.Hostname()
		return hostname == "localhost" || hostname == "127.0.0.1"
	}
	
	return false
}
