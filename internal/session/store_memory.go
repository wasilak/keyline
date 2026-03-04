package session

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// MemoryStore implements Store using an in-memory map
type MemoryStore struct {
	sessions map[string]*Session
	mu       sync.RWMutex
	logger   *slog.Logger
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

// NewMemoryStore creates a new in-memory session store
func NewMemoryStore(logger *slog.Logger) *MemoryStore {
	store := &MemoryStore{
		sessions: make(map[string]*Session),
		logger:   logger,
		stopCh:   make(chan struct{}),
	}

	// Start background cleanup goroutine
	store.wg.Add(1)
	go store.cleanupLoop()

	return store
}

// Create stores a new session
func (s *MemoryStore) Create(ctx context.Context, session *Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessions[session.ID] = session
	s.logger.Debug("Session created",
		slog.String("session_id", session.ID),
		slog.String("username", session.Username),
		slog.Time("expires_at", session.ExpiresAt),
	)

	return nil
}

// Get retrieves a session by ID
func (s *MemoryStore) Get(ctx context.Context, sessionID string) (*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, exists := s.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found")
	}

	// Check if expired
	if session.IsExpired() {
		return nil, fmt.Errorf("session expired")
	}

	return session, nil
}

// Delete removes a session
func (s *MemoryStore) Delete(ctx context.Context, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.sessions, sessionID)
	s.logger.Debug("Session deleted",
		slog.String("session_id", sessionID),
	)

	return nil
}

// Cleanup removes expired sessions
func (s *MemoryStore) Cleanup(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	count := 0

	for id, session := range s.sessions {
		if now.After(session.ExpiresAt) {
			delete(s.sessions, id)
			count++
		}
	}

	if count > 0 {
		s.logger.Info("Cleaned up expired sessions",
			slog.Int("count", count),
		)
	}

	return nil
}

// Health checks if the store is accessible
func (s *MemoryStore) Health(ctx context.Context) error {
	// In-memory store is always healthy
	return nil
}

// Close stops the cleanup goroutine
func (s *MemoryStore) Close() error {
	close(s.stopCh)
	s.wg.Wait()
	return nil
}

// cleanupLoop runs periodic cleanup of expired sessions
func (s *MemoryStore) cleanupLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := s.Cleanup(context.Background()); err != nil {
				s.logger.Error("Failed to cleanup expired sessions",
					slog.String("error", err.Error()),
				)
			}
		case <-s.stopCh:
			return
		}
	}
}
