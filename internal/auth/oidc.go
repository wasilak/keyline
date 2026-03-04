package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/yourusername/keyline/internal/cache"
	"github.com/yourusername/keyline/internal/config"
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
