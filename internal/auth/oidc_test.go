package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/yourusername/keyline/internal/cache"
	"github.com/yourusername/keyline/internal/config"
	"gopkg.in/square/go-jose.v2"
)

// Mock OIDC server for testing
func setupMockOIDCServer(t *testing.T) (*httptest.Server, string) {
	mux := http.NewServeMux()

	// Discovery endpoint
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		doc := cache.DiscoveryDocument{
			Issuer:                "https://test-issuer.com",
			AuthorizationEndpoint: "https://test-issuer.com/authorize",
			TokenEndpoint:         "https://test-issuer.com/token",
			JWKSURI:               "https://test-issuer.com/jwks",
			UserinfoEndpoint:      "https://test-issuer.com/userinfo",
		}
		json.NewEncoder(w).Encode(doc)
	})

	// JWKS endpoint
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, r *http.Request) {
		// Return a minimal JWKS for testing
		jwks := map[string]interface{}{
			"keys": []map[string]interface{}{
				{
					"kty": "RSA",
					"use": "sig",
					"kid": "test-key-1",
					"n":   "test-modulus",
					"e":   "AQAB",
				},
			},
		}
		json.NewEncoder(w).Encode(jwks)
	})

	server := httptest.NewTLSServer(mux)
	return server, server.URL
}

func TestNewOIDCProvider_Disabled(t *testing.T) {
	cfg := &config.OIDCConfig{
		Enabled: false,
	}

	_, err := NewOIDCProvider(cfg, &config.Config{})
	if err == nil {
		t.Error("Expected error when OIDC is disabled")
	}
	if !strings.Contains(err.Error(), "not enabled") {
		t.Errorf("Expected error about OIDC not enabled, got: %v", err)
	}
}

func TestDiscover_NonHTTPSIssuer(t *testing.T) {
	cfg := &config.OIDCConfig{
		Enabled:   true,
		IssuerURL: "http://insecure-issuer.com",
	}

	provider := &OIDCProvider{
		config:     cfg,
		cache:      cache.NewOIDCCache(),
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}

	err := provider.discover(context.Background())
	if err == nil {
		t.Error("Expected error for non-HTTPS issuer URL")
	}
	if !strings.Contains(err.Error(), "must use HTTPS") {
		t.Errorf("Expected error about HTTPS, got: %v", err)
	}
}

func TestDiscover_IssuerMismatch(t *testing.T) {
	// Create a mock server that returns a different issuer
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		doc := cache.DiscoveryDocument{
			Issuer:                "https://different-issuer.com",
			AuthorizationEndpoint: "https://different-issuer.com/authorize",
			TokenEndpoint:         "https://different-issuer.com/token",
			JWKSURI:               "https://different-issuer.com/jwks",
		}
		json.NewEncoder(w).Encode(doc)
	})

	server := httptest.NewTLSServer(mux)
	defer server.Close()

	cfg := &config.OIDCConfig{
		Enabled:   true,
		IssuerURL: server.URL,
	}

	provider := &OIDCProvider{
		config:     cfg,
		cache:      cache.NewOIDCCache(),
		httpClient: server.Client(),
	}

	err := provider.discover(context.Background())
	if err == nil {
		t.Error("Expected error for issuer mismatch")
	}
	if !strings.Contains(err.Error(), "issuer mismatch") {
		t.Errorf("Expected error about issuer mismatch, got: %v", err)
	}
}

func TestDiscover_MissingRequiredFields(t *testing.T) {
	tests := []struct {
		name string
		doc  cache.DiscoveryDocument
		want string
	}{
		{
			name: "missing issuer",
			doc: cache.DiscoveryDocument{
				AuthorizationEndpoint: "https://test.com/authorize",
				TokenEndpoint:         "https://test.com/token",
				JWKSURI:               "https://test.com/jwks",
			},
			want: "missing issuer",
		},
		{
			name: "missing authorization_endpoint",
			doc: cache.DiscoveryDocument{
				Issuer:        "https://test.com",
				TokenEndpoint: "https://test.com/token",
				JWKSURI:       "https://test.com/jwks",
			},
			want: "missing authorization_endpoint",
		},
		{
			name: "missing token_endpoint",
			doc: cache.DiscoveryDocument{
				Issuer:                "https://test.com",
				AuthorizationEndpoint: "https://test.com/authorize",
				JWKSURI:               "https://test.com/jwks",
			},
			want: "missing token_endpoint",
		},
		{
			name: "missing jwks_uri",
			doc: cache.DiscoveryDocument{
				Issuer:                "https://test.com",
				AuthorizationEndpoint: "https://test.com/authorize",
				TokenEndpoint:         "https://test.com/token",
			},
			want: "missing jwks_uri",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
				json.NewEncoder(w).Encode(tt.doc)
			})

			server := httptest.NewTLSServer(mux)
			defer server.Close()

			cfg := &config.OIDCConfig{
				Enabled:   true,
				IssuerURL: server.URL,
			}

			provider := &OIDCProvider{
				config:     cfg,
				cache:      cache.NewOIDCCache(),
				httpClient: server.Client(),
			}

			err := provider.discover(context.Background())
			if err == nil {
				t.Errorf("Expected error for %s", tt.name)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("Expected error containing %q, got: %v", tt.want, err)
			}
		})
	}
}

func TestDiscover_NonHTTPSEndpoints(t *testing.T) {
	tests := []struct {
		name string
		doc  cache.DiscoveryDocument
		want string
	}{
		{
			name: "non-HTTPS token_endpoint",
			doc: cache.DiscoveryDocument{
				Issuer:                "https://test.com",
				AuthorizationEndpoint: "https://test.com/authorize",
				TokenEndpoint:         "http://test.com/token",
				JWKSURI:               "https://test.com/jwks",
			},
			want: "token_endpoint must use HTTPS",
		},
		{
			name: "non-HTTPS jwks_uri",
			doc: cache.DiscoveryDocument{
				Issuer:                "https://test.com",
				AuthorizationEndpoint: "https://test.com/authorize",
				TokenEndpoint:         "https://test.com/token",
				JWKSURI:               "http://test.com/jwks",
			},
			want: "jwks_uri must use HTTPS",
		},
		{
			name: "non-HTTPS userinfo_endpoint",
			doc: cache.DiscoveryDocument{
				Issuer:                "https://test.com",
				AuthorizationEndpoint: "https://test.com/authorize",
				TokenEndpoint:         "https://test.com/token",
				JWKSURI:               "https://test.com/jwks",
				UserinfoEndpoint:      "http://test.com/userinfo",
			},
			want: "userinfo_endpoint must use HTTPS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
				json.NewEncoder(w).Encode(tt.doc)
			})

			server := httptest.NewTLSServer(mux)
			defer server.Close()

			cfg := &config.OIDCConfig{
				Enabled:   true,
				IssuerURL: server.URL,
			}

			provider := &OIDCProvider{
				config:     cfg,
				cache:      cache.NewOIDCCache(),
				httpClient: server.Client(),
			}

			err := provider.discover(context.Background())
			if err == nil {
				t.Errorf("Expected error for %s", tt.name)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("Expected error containing %q, got: %v", tt.want, err)
			}
		})
	}
}

func TestBuildAuthorizationURL(t *testing.T) {
	cfg := &config.OIDCConfig{
		ClientID:    "test-client",
		RedirectURL: "https://app.example.com/callback",
		Scopes:      []string{"openid", "email", "profile"},
	}

	provider := &OIDCProvider{
		config: cfg,
		cache:  cache.NewOIDCCache(),
	}

	// Set discovery document
	doc := &cache.DiscoveryDocument{
		Issuer:                "https://test-issuer.com",
		AuthorizationEndpoint: "https://test-issuer.com/authorize",
		TokenEndpoint:         "https://test-issuer.com/token",
		JWKSURI:               "https://test-issuer.com/jwks",
	}
	provider.cache.SetDiscoveryDoc(doc)

	authURL, err := provider.buildAuthorizationURL("test-state", "test-challenge")
	if err != nil {
		t.Fatalf("buildAuthorizationURL() failed: %v", err)
	}

	// Verify URL contains required parameters
	if !strings.Contains(authURL, "client_id=test-client") {
		t.Error("Authorization URL missing client_id")
	}
	if !strings.Contains(authURL, "redirect_uri=https%3A%2F%2Fapp.example.com%2Fcallback") {
		t.Error("Authorization URL missing redirect_uri")
	}
	if !strings.Contains(authURL, "response_type=code") {
		t.Error("Authorization URL missing response_type")
	}
	if !strings.Contains(authURL, "scope=openid+email+profile") {
		t.Error("Authorization URL missing scope")
	}
	if !strings.Contains(authURL, "state=test-state") {
		t.Error("Authorization URL missing state")
	}
	if !strings.Contains(authURL, "code_challenge=test-challenge") {
		t.Error("Authorization URL missing code_challenge")
	}
	if !strings.Contains(authURL, "code_challenge_method=S256") {
		t.Error("Authorization URL missing code_challenge_method")
	}
}

func TestBuildAuthorizationURL_NoDiscoveryDoc(t *testing.T) {
	cfg := &config.OIDCConfig{
		ClientID:    "test-client",
		RedirectURL: "https://app.example.com/callback",
	}

	provider := &OIDCProvider{
		config: cfg,
		cache:  cache.NewOIDCCache(),
	}

	_, err := provider.buildAuthorizationURL("test-state", "test-challenge")
	if err == nil {
		t.Error("Expected error when discovery document not loaded")
	}
	if !strings.Contains(err.Error(), "discovery document not loaded") {
		t.Errorf("Expected error about discovery document, got: %v", err)
	}
}

func TestJoinScopes(t *testing.T) {
	tests := []struct {
		name   string
		scopes []string
		want   string
	}{
		{
			name:   "multiple scopes",
			scopes: []string{"openid", "email", "profile"},
			want:   "openid email profile",
		},
		{
			name:   "single scope",
			scopes: []string{"openid"},
			want:   "openid",
		},
		{
			name:   "empty scopes",
			scopes: []string{},
			want:   "openid",
		},
		{
			name:   "nil scopes",
			scopes: nil,
			want:   "openid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := joinScopes(tt.scopes)
			if got != tt.want {
				t.Errorf("joinScopes() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValidateAudience(t *testing.T) {
	provider := &OIDCProvider{
		config: &config.OIDCConfig{
			ClientID: "test-client",
		},
	}

	tests := []struct {
		name string
		aud  interface{}
		want bool
	}{
		{
			name: "string match",
			aud:  "test-client",
			want: true,
		},
		{
			name: "string no match",
			aud:  "other-client",
			want: false,
		},
		{
			name: "array with match",
			aud:  []interface{}{"test-client", "other-client"},
			want: true,
		},
		{
			name: "array without match",
			aud:  []interface{}{"other-client", "another-client"},
			want: false,
		},
		{
			name: "string array with match",
			aud:  []string{"test-client", "other-client"},
			want: true,
		},
		{
			name: "string array without match",
			aud:  []string{"other-client", "another-client"},
			want: false,
		},
		{
			name: "invalid type",
			aud:  123,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := provider.validateAudience(tt.aud)
			if got != tt.want {
				t.Errorf("validateAudience() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsHTTPS(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want bool
	}{
		{
			name: "HTTPS URL",
			url:  "https://example.com",
			want: true,
		},
		{
			name: "HTTP URL",
			url:  "http://example.com",
			want: false,
		},
		{
			name: "HTTPS with path",
			url:  "https://example.com/path",
			want: true,
		},
		{
			name: "invalid URL",
			url:  "not-a-url",
			want: false,
		},
		{
			name: "empty URL",
			url:  "",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isHTTPS(tt.url)
			if got != tt.want {
				t.Errorf("isHTTPS(%q) = %v, want %v", tt.url, got, tt.want)
			}
		})
	}
}

func TestHandleCallback_MissingState(t *testing.T) {
	provider := &OIDCProvider{
		config: &config.OIDCConfig{},
	}

	_, err := provider.HandleCallback(context.Background(), nil, "", "test-code", "", "")
	if err == nil {
		t.Error("Expected error for missing state parameter")
	}
	if !strings.Contains(err.Error(), "missing state parameter") {
		t.Errorf("Expected error about missing state, got: %v", err)
	}
}

func TestHandleCallback_MissingCode(t *testing.T) {
	// This test requires a valid state token in cache, which we don't have
	// So we expect an error about failed to retrieve state token
	// Skip this test as it requires integration with cache
	t.Skip("Requires integration with cache - covered by integration tests")
}

func TestHandleCallback_OIDCProviderError(t *testing.T) {
	provider := &OIDCProvider{
		config: &config.OIDCConfig{},
	}

	_, err := provider.HandleCallback(context.Background(), nil, "test-state", "test-code", "access_denied", "User denied access")
	if err == nil {
		t.Error("Expected error when OIDC provider returns error")
	}
	if !strings.Contains(err.Error(), "access_denied") {
		t.Errorf("Expected error about access_denied, got: %v", err)
	}
}

func TestFetchJWKS_NoDiscoveryDoc(t *testing.T) {
	provider := &OIDCProvider{
		config:     &config.OIDCConfig{},
		cache:      cache.NewOIDCCache(),
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}

	err := provider.fetchJWKS(context.Background())
	if err == nil {
		t.Error("Expected error when discovery document not loaded")
	}
	if !strings.Contains(err.Error(), "discovery document not loaded") {
		t.Errorf("Expected error about discovery document, got: %v", err)
	}
}

func TestParseJWKS(t *testing.T) {
	validJWKS := `{
		"keys": [
			{
				"kty": "RSA",
				"use": "sig",
				"kid": "test-key",
				"n": "test-modulus",
				"e": "AQAB"
			}
		]
	}`

	jwks, err := cache.ParseJWKS([]byte(validJWKS))
	if err != nil {
		t.Fatalf("ParseJWKS() failed: %v", err)
	}

	if len(jwks.Keys) != 1 {
		t.Errorf("Expected 1 key, got %d", len(jwks.Keys))
	}
}

func TestParseJWKS_Invalid(t *testing.T) {
	invalidJWKS := `{invalid json}`

	_, err := cache.ParseJWKS([]byte(invalidJWKS))
	if err == nil {
		t.Error("Expected error for invalid JWKS JSON")
	}
}

// Helper function to create a test JWKS
func createTestJWKS() *jose.JSONWebKeySet {
	return &jose.JSONWebKeySet{
		Keys: []jose.JSONWebKey{
			{
				KeyID: "test-key-1",
				Use:   "sig",
			},
		},
	}
}
