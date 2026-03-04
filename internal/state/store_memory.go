package state

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

// MemoryStore implements Store using an in-memory map
type MemoryStore struct {
	tokens map[string]*Token
	mu     sync.RWMutex
	logger *slog.Logger
}

// NewMemoryStore creates a new in-memory state token store
func NewMemoryStore(logger *slog.Logger) *MemoryStore {
	return &MemoryStore{
		tokens: make(map[string]*Token),
		logger: logger,
	}
}

// Store saves a state token
func (s *MemoryStore) Store(ctx context.Context, token *Token) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.tokens[token.ID] = token
	s.logger.Debug("State token stored",
		slog.String("token_id", token.ID),
		slog.String("original_url", token.OriginalURL),
	)

	return nil
}

// Get retrieves and marks a state token as used
func (s *MemoryStore) Get(ctx context.Context, tokenID string) (*Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	token, exists := s.tokens[tokenID]
	if !exists {
		return nil, fmt.Errorf("state token not found")
	}

	// Check if expired
	if token.IsExpired() {
		delete(s.tokens, tokenID)
		return nil, fmt.Errorf("state token expired")
	}

	// Check if already used
	if token.Used {
		return nil, fmt.Errorf("state token already used")
	}

	// Mark as used
	token.Used = true

	return token, nil
}

// Delete removes a state token
func (s *MemoryStore) Delete(ctx context.Context, tokenID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.tokens, tokenID)
	s.logger.Debug("State token deleted",
		slog.String("token_id", tokenID),
	)

	return nil
}
