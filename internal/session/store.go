package session

import (
	"context"
	"time"
)

// Store defines the interface for session storage
type Store interface {
	// Create stores a new session
	Create(ctx context.Context, session *Session) error

	// Get retrieves a session by ID
	Get(ctx context.Context, sessionID string) (*Session, error)

	// Delete removes a session
	Delete(ctx context.Context, sessionID string) error

	// Cleanup removes expired sessions (for in-memory store)
	Cleanup(ctx context.Context) error

	// Health checks if the store is accessible
	Health(ctx context.Context) error
}

// Session represents a user session
type Session struct {
	ID        string                 `json:"id"`
	UserID    string                 `json:"user_id"`
	Username  string                 `json:"username"`
	Email     string                 `json:"email"`
	FullName  string                 `json:"full_name"`
	Groups    []string               `json:"groups"`
	Source    string                 `json:"source"` // "oidc", "basic_auth", etc.
	Claims    map[string]interface{} `json:"claims"`
	CreatedAt time.Time              `json:"created_at"`
	ExpiresAt time.Time              `json:"expires_at"`
}

// IsExpired checks if the session has expired
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}
