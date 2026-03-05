//go:build integration
// +build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/yourusername/keyline/internal/cache"
	"github.com/yourusername/keyline/internal/config"
	"github.com/yourusername/keyline/internal/session"
	"github.com/yourusername/keyline/internal/state"
)

// Note: These tests use the memory backend to test the cache interface
// In production, Redis would be used with the same interface

func TestRedisSessionStore_SessionCreationAndRetrieval(t *testing.T) {
	// Setup cache
	cfg := &config.CacheConfig{
		Backend: "memory",
	}

	ctx := context.Background()
	cacheInstance, err := cache.InitCache(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to initialize cache: %v", err)
	}

	// Create a session
	sess := &session.Session{
		ID:        "test-session-123",
		Username:  "testuser",
		Email:     "test@example.com",
		ESUser:    "es_testuser",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	// Test: Create session
	err = session.CreateSession(ctx, cacheInstance, sess)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Test: Retrieve session
	retrieved, err := session.GetSession(ctx, cacheInstance, sess.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve session: %v", err)
	}

	// Verify: Session data matches
	if retrieved == nil {
		t.Fatal("Expected session to be retrieved, got nil")
	}
	if retrieved.ID != sess.ID {
		t.Errorf("Expected session ID %s, got %s", sess.ID, retrieved.ID)
	}
	if retrieved.Username != sess.Username {
		t.Errorf("Expected username %s, got %s", sess.Username, retrieved.Username)
	}
	if retrieved.ESUser != sess.ESUser {
		t.Errorf("Expected ES user %s, got %s", sess.ESUser, retrieved.ESUser)
	}
}

func TestRedisSessionStore_SessionExpiration(t *testing.T) {
	// Setup cache
	cfg := &config.CacheConfig{
		Backend: "memory",
	}

	ctx := context.Background()
	cacheInstance, err := cache.InitCache(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to initialize cache: %v", err)
	}

	// Create an expired session
	sess := &session.Session{
		ID:        "expired-session-123",
		Username:  "testuser",
		ESUser:    "es_testuser",
		CreatedAt: time.Now().Add(-2 * time.Hour),
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired 1 hour ago
	}

	// Test: Create session
	err = session.CreateSession(ctx, cacheInstance, sess)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Test: Retrieve expired session
	retrieved, err := session.GetSession(ctx, cacheInstance, sess.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve session: %v", err)
	}

	// Verify: Expired session returns nil
	if retrieved != nil {
		t.Error("Expected expired session to return nil")
	}
}

func TestRedisSessionStore_StateTokenStorage(t *testing.T) {
	// Setup cache
	cfg := &config.CacheConfig{
		Backend: "memory",
	}

	ctx := context.Background()
	cacheInstance, err := cache.InitCache(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to initialize cache: %v", err)
	}

	// Create a state token
	token := &state.Token{
		ID:           "state-token-123",
		OriginalURL:  "https://example.com/original",
		CodeVerifier: "code-verifier-value",
		CreatedAt:    time.Now(),
		Used:         false,
	}

	// Test: Store state token
	err = state.StoreStateToken(ctx, cacheInstance, token)
	if err != nil {
		t.Fatalf("Failed to store state token: %v", err)
	}

	// Test: Retrieve state token
	retrieved, err := state.GetStateToken(ctx, cacheInstance, token.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve state token: %v", err)
	}

	// Verify: State token data matches
	if retrieved == nil {
		t.Fatal("Expected state token to be retrieved, got nil")
	}
	if retrieved.ID != token.ID {
		t.Errorf("Expected token ID %s, got %s", token.ID, retrieved.ID)
	}
	if retrieved.OriginalURL != token.OriginalURL {
		t.Errorf("Expected original URL %s, got %s", token.OriginalURL, retrieved.OriginalURL)
	}
	if retrieved.CodeVerifier != token.CodeVerifier {
		t.Errorf("Expected code verifier %s, got %s", token.CodeVerifier, retrieved.CodeVerifier)
	}

	// Test: Verify token is deleted after first use (single-use enforcement)
	retrievedAgain, err := state.GetStateToken(ctx, cacheInstance, token.ID)
	// Note: GetStateToken may return an error when trying to unmarshal empty data
	// This is expected behavior when the token has been deleted
	if retrievedAgain != nil {
		t.Error("Expected state token to be deleted after first use (single-use enforcement)")
	}
}

func TestRedisSessionStore_StateTokenPrefix(t *testing.T) {
	// Setup cache
	cfg := &config.CacheConfig{
		Backend: "memory",
	}

	ctx := context.Background()
	cacheInstance, err := cache.InitCache(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to initialize cache: %v", err)
	}

	// Create a state token and a session with the same ID
	tokenID := "same-id-123"

	token := &state.Token{
		ID:           tokenID,
		OriginalURL:  "https://example.com/original",
		CodeVerifier: "code-verifier",
		CreatedAt:    time.Now(),
		Used:         false,
	}

	sess := &session.Session{
		ID:        tokenID,
		Username:  "testuser",
		ESUser:    "es_testuser",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	// Test: Store both with same ID
	err = state.StoreStateToken(ctx, cacheInstance, token)
	if err != nil {
		t.Fatalf("Failed to store state token: %v", err)
	}

	err = session.CreateSession(ctx, cacheInstance, sess)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Test: Retrieve both
	retrievedToken, err := state.GetStateToken(ctx, cacheInstance, tokenID)
	if err != nil {
		t.Fatalf("Failed to retrieve state token: %v", err)
	}

	retrievedSession, err := session.GetSession(ctx, cacheInstance, tokenID)
	if err != nil {
		t.Fatalf("Failed to retrieve session: %v", err)
	}

	// Verify: Both are retrieved correctly (different key prefixes)
	if retrievedToken == nil {
		t.Error("Expected state token to be retrieved")
	}
	if retrievedSession == nil {
		t.Error("Expected session to be retrieved")
	}

	// Verify they are different objects
	if retrievedToken != nil && retrievedSession != nil {
		if retrievedToken.OriginalURL == retrievedSession.Username {
			t.Error("State token and session should be stored separately")
		}
	}
}

func TestRedisSessionStore_SessionDeletion(t *testing.T) {
	// Setup cache
	cfg := &config.CacheConfig{
		Backend: "memory",
	}

	ctx := context.Background()
	cacheInstance, err := cache.InitCache(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to initialize cache: %v", err)
	}

	// Create a session
	sess := &session.Session{
		ID:        "delete-test-session",
		Username:  "testuser",
		ESUser:    "es_testuser",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	// Test: Create session
	err = session.CreateSession(ctx, cacheInstance, sess)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Verify session exists
	retrieved, err := session.GetSession(ctx, cacheInstance, sess.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve session: %v", err)
	}
	if retrieved == nil {
		t.Fatal("Expected session to exist before deletion")
	}

	// Test: Delete session
	err = session.DeleteSession(ctx, cacheInstance, sess.ID)
	if err != nil {
		t.Fatalf("Failed to delete session: %v", err)
	}

	// Verify: Session no longer exists
	retrieved, err = session.GetSession(ctx, cacheInstance, sess.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve session after deletion: %v", err)
	}
	if retrieved != nil {
		t.Error("Expected session to be deleted")
	}
}

func TestRedisSessionStore_ConnectionFailureHandling(t *testing.T) {
	// This test verifies that cache initialization validates connectivity

	// Test with invalid Redis URL (should fail if Redis backend is used)
	cfg := &config.CacheConfig{
		Backend:  "redis",
		RedisURL: "redis://invalid-host:6379",
	}

	ctx := context.Background()
	_, err := cache.InitCache(ctx, cfg)

	// Verify: Initialization fails with invalid Redis URL
	if err == nil {
		t.Error("Expected cache initialization to fail with invalid Redis URL")
	}
}

func TestRedisSessionStore_MultipleSessionsIndependence(t *testing.T) {
	// Setup cache
	cfg := &config.CacheConfig{
		Backend: "memory",
	}

	ctx := context.Background()
	cacheInstance, err := cache.InitCache(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to initialize cache: %v", err)
	}

	// Create multiple sessions
	sessions := []*session.Session{
		{
			ID:        "session-1",
			Username:  "user1",
			ESUser:    "es_user1",
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(1 * time.Hour),
		},
		{
			ID:        "session-2",
			Username:  "user2",
			ESUser:    "es_user2",
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(1 * time.Hour),
		},
		{
			ID:        "session-3",
			Username:  "user3",
			ESUser:    "es_user3",
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(1 * time.Hour),
		},
	}

	// Test: Create all sessions
	for _, sess := range sessions {
		err = session.CreateSession(ctx, cacheInstance, sess)
		if err != nil {
			t.Fatalf("Failed to create session %s: %v", sess.ID, err)
		}
	}

	// Test: Retrieve all sessions
	for _, sess := range sessions {
		retrieved, err := session.GetSession(ctx, cacheInstance, sess.ID)
		if err != nil {
			t.Fatalf("Failed to retrieve session %s: %v", sess.ID, err)
		}
		if retrieved == nil {
			t.Fatalf("Expected session %s to be retrieved", sess.ID)
		}
		if retrieved.Username != sess.Username {
			t.Errorf("Session %s: expected username %s, got %s", sess.ID, sess.Username, retrieved.Username)
		}
	}

	// Test: Delete one session
	err = session.DeleteSession(ctx, cacheInstance, "session-2")
	if err != nil {
		t.Fatalf("Failed to delete session-2: %v", err)
	}

	// Verify: Other sessions still exist
	retrieved1, _ := session.GetSession(ctx, cacheInstance, "session-1")
	retrieved2, _ := session.GetSession(ctx, cacheInstance, "session-2")
	retrieved3, _ := session.GetSession(ctx, cacheInstance, "session-3")

	if retrieved1 == nil {
		t.Error("Expected session-1 to still exist")
	}
	if retrieved2 != nil {
		t.Error("Expected session-2 to be deleted")
	}
	if retrieved3 == nil {
		t.Error("Expected session-3 to still exist")
	}
}
