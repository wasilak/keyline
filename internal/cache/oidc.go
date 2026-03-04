package cache

import (
	"encoding/json"
	"sync"
	"time"

	"gopkg.in/square/go-jose.v2"
)

// OIDCCache caches OIDC Discovery Document and JWKS
type OIDCCache struct {
	discoveryDoc *DiscoveryDocument
	jwks         *jose.JSONWebKeySet
	jwksExpiry   time.Time
	mu           sync.RWMutex
}

// DiscoveryDocument represents the OIDC discovery document
type DiscoveryDocument struct {
	Issuer                string   `json:"issuer"`
	AuthorizationEndpoint string   `json:"authorization_endpoint"`
	TokenEndpoint         string   `json:"token_endpoint"`
	UserinfoEndpoint      string   `json:"userinfo_endpoint"`
	JWKSURI               string   `json:"jwks_uri"`
	EndSessionEndpoint    string   `json:"end_session_endpoint,omitempty"`
	ScopesSupported       []string `json:"scopes_supported,omitempty"`
}

// NewOIDCCache creates a new OIDC cache
func NewOIDCCache() *OIDCCache {
	return &OIDCCache{}
}

// SetDiscoveryDoc stores the discovery document
func (c *OIDCCache) SetDiscoveryDoc(doc *DiscoveryDocument) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.discoveryDoc = doc
}

// GetDiscoveryDoc retrieves the cached discovery document
func (c *OIDCCache) GetDiscoveryDoc() *DiscoveryDocument {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.discoveryDoc
}

// SetJWKS stores the JWKS with expiry tracking
func (c *OIDCCache) SetJWKS(jwks *jose.JSONWebKeySet, expiryDuration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.jwks = jwks
	c.jwksExpiry = time.Now().Add(expiryDuration)
}

// GetJWKS retrieves the cached JWKS
func (c *OIDCCache) GetJWKS() (*jose.JSONWebKeySet, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.jwks == nil {
		return nil, false
	}

	// Check if expired
	if time.Now().After(c.jwksExpiry) {
		return c.jwks, false // Return JWKS but indicate it's expired
	}

	return c.jwks, true
}

// IsJWKSExpired checks if the JWKS cache has expired
func (c *OIDCCache) IsJWKSExpired() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return time.Now().After(c.jwksExpiry)
}

// RefreshJWKS updates the JWKS cache
// This method should be called by a background goroutine every 24 hours
func (c *OIDCCache) RefreshJWKS(fetchFunc func() (*jose.JSONWebKeySet, error)) error {
	jwks, err := fetchFunc()
	if err != nil {
		return err
	}

	c.SetJWKS(jwks, 24*time.Hour)
	return nil
}

// ParseJWKS parses a JWKS JSON response
func ParseJWKS(data []byte) (*jose.JSONWebKeySet, error) {
	var jwks jose.JSONWebKeySet
	if err := json.Unmarshal(data, &jwks); err != nil {
		return nil, err
	}
	return &jwks, nil
}
