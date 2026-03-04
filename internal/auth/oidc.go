package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/wasilak/cachego"
	"github.com/yourusername/keyline/internal/cache"
	"github.com/yourusername/keyline/internal/config"
	"github.com/yourusername/keyline/internal/state"
	pkgcrypto "github.com/yourusername/keyline/pkg/crypto"
	"gopkg.in/square/go-jose.v2"
)

// OIDCProvider implements OIDC authentication
type OIDCProvider struct {
	config     *config.OIDCConfig
	cache      *cache.OIDCCache
	httpClient *http.Client
}

// NewOIDCProvider creates a new OIDC provider
func NewOIDCProvider(cfg *config.OIDCConfig) (*OIDCProvider, error) {
	if !cfg.Enabled {
		return nil, fmt.Errorf("OIDC is not enabled")
	}

	provider := &OIDCProvider{
		config: cfg,
		cache:  cache.NewOIDCCache(),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
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
