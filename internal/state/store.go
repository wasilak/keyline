package state

import (
	"context"
	"time"
)

// Store defines the interface for OIDC state token storage
type Store interface {
	// Store saves a state token with original URL
	Store(ctx context.Context, token *Token) error

	// Get retrieves and marks a state token as used
	Get(ctx context.Context, tokenID string) (*Token, error)

	// Delete removes a state token
	Delete(ctx context.Context, tokenID string) error
}

// Token represents an OIDC state token
type Token struct {
	ID           string    `json:"id"`
	OriginalURL  string    `json:"original_url"`
	CodeVerifier string    `json:"code_verifier"`
	CreatedAt    time.Time `json:"created_at"`
	Used         bool      `json:"used"`
}

// IsExpired checks if the token has expired (5-minute TTL)
func (t *Token) IsExpired() bool {
	return time.Since(t.CreatedAt) > 5*time.Minute
}
