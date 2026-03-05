package session

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wasilak/cachego"

	"github.com/yourusername/keyline/internal/cache"
	"github.com/yourusername/keyline/internal/config"
)

func setupTestCache(t *testing.T) cachego.CacheInterface {
	ctx := context.Background()
	cfg := &config.CacheConfig{
		Backend: "memory",
	}

	c, err := cache.InitCache(ctx, cfg)
	require.NoError(t, err)
	return c
}

func TestCreateSession(t *testing.T) {
	cache := setupTestCache(t)
	ctx := context.Background()

	session := &Session{
		ID:        "test-session-id",
		UserID:    "user123",
		Username:  "testuser",
		Email:     "test@example.com",
		ESUser:    "es_testuser",
		Claims:    map[string]interface{}{"role": "admin"},
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	err := CreateSession(ctx, cache, session)
	require.NoError(t, err)

	// Verify session was stored
	retrieved, err := GetSession(ctx, cache, session.ID)
	require.NoError(t, err)
	assert.Equal(t, session.ID, retrieved.ID)
	assert.Equal(t, session.Username, retrieved.Username)
	assert.Equal(t, session.ESUser, retrieved.ESUser)
}

func TestGetSession_ValidSession(t *testing.T) {
	cache := setupTestCache(t)
	ctx := context.Background()

	session := &Session{
		ID:        "valid-session",
		UserID:    "user456",
		Username:  "validuser",
		Email:     "valid@example.com",
		ESUser:    "es_validuser",
		Claims:    map[string]interface{}{},
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	err := CreateSession(ctx, cache, session)
	require.NoError(t, err)

	retrieved, err := GetSession(ctx, cache, session.ID)
	require.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, session.Username, retrieved.Username)
}

func TestGetSession_NonExistent(t *testing.T) {
	cache := setupTestCache(t)
	ctx := context.Background()

	retrieved, err := GetSession(ctx, cache, "nonexistent-session")
	assert.NoError(t, err)
	assert.Nil(t, retrieved)
}

func TestGetSession_ExpiredSession(t *testing.T) {
	cache := setupTestCache(t)
	ctx := context.Background()

	// Create session that's already expired
	session := &Session{
		ID:        "expired-session",
		UserID:    "user789",
		Username:  "expireduser",
		Email:     "expired@example.com",
		ESUser:    "es_expireduser",
		Claims:    map[string]interface{}{},
		CreatedAt: time.Now().Add(-2 * time.Hour),
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired 1 hour ago
	}

	err := CreateSession(ctx, cache, session)
	require.NoError(t, err)

	// Try to retrieve expired session - should return nil, nil (not an error)
	retrieved, err := GetSession(ctx, cache, session.ID)
	assert.NoError(t, err)
	assert.Nil(t, retrieved)
}

func TestDeleteSession(t *testing.T) {
	cache := setupTestCache(t)
	ctx := context.Background()

	session := &Session{
		ID:        "delete-test-session",
		UserID:    "user999",
		Username:  "deleteuser",
		Email:     "delete@example.com",
		ESUser:    "es_deleteuser",
		Claims:    map[string]interface{}{},
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	// Create session
	err := CreateSession(ctx, cache, session)
	require.NoError(t, err)

	// Verify it exists
	retrieved, err := GetSession(ctx, cache, session.ID)
	require.NoError(t, err)
	assert.NotNil(t, retrieved)

	// Delete session
	err = DeleteSession(ctx, cache, session.ID)
	require.NoError(t, err)

	// Verify it's gone
	retrieved, err = GetSession(ctx, cache, session.ID)
	assert.NoError(t, err)
	assert.Nil(t, retrieved)
}

func TestDeleteSession_NonExistent(t *testing.T) {
	cache := setupTestCache(t)
	ctx := context.Background()

	// Deleting non-existent session should not error
	err := DeleteSession(ctx, cache, "nonexistent-session")
	assert.NoError(t, err)
}

func TestSession_CryptographicRandomID(t *testing.T) {
	cache := setupTestCache(t)
	ctx := context.Background()

	// Create multiple sessions and verify IDs are unique
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		session := &Session{
			ID:        generateTestSessionID(),
			UserID:    "user",
			Username:  "testuser",
			Email:     "test@example.com",
			ESUser:    "es_user",
			Claims:    map[string]interface{}{},
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(1 * time.Hour),
		}

		// Verify ID is not empty and has sufficient length
		assert.NotEmpty(t, session.ID)
		assert.GreaterOrEqual(t, len(session.ID), 32, "Session ID should be at least 32 characters")

		// Verify uniqueness
		assert.False(t, ids[session.ID], "Session ID should be unique")
		ids[session.ID] = true

		err := CreateSession(ctx, cache, session)
		require.NoError(t, err)
	}
}

func TestSession_TTLRespected(t *testing.T) {
	cache := setupTestCache(t)
	ctx := context.Background()

	// Create session with short TTL
	session := &Session{
		ID:        "ttl-test-session",
		UserID:    "user",
		Username:  "testuser",
		Email:     "test@example.com",
		ESUser:    "es_user",
		Claims:    map[string]interface{}{},
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(100 * time.Millisecond),
	}

	err := CreateSession(ctx, cache, session)
	require.NoError(t, err)

	// Session should exist immediately
	retrieved, err := GetSession(ctx, cache, session.ID)
	require.NoError(t, err)
	assert.NotNil(t, retrieved)

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Session should be expired (returns nil, nil)
	retrieved, err = GetSession(ctx, cache, session.ID)
	assert.NoError(t, err)
	assert.Nil(t, retrieved)
}

func TestSession_AllFieldsPreserved(t *testing.T) {
	cache := setupTestCache(t)
	ctx := context.Background()

	claims := map[string]interface{}{
		"role":   "admin",
		"groups": []string{"admins", "users"},
		"level":  5,
	}

	session := &Session{
		ID:        "fields-test-session",
		UserID:    "user123",
		Username:  "testuser",
		Email:     "test@example.com",
		ESUser:    "es_testuser",
		Claims:    claims,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	err := CreateSession(ctx, cache, session)
	require.NoError(t, err)

	retrieved, err := GetSession(ctx, cache, session.ID)
	require.NoError(t, err)

	// Verify all fields are preserved
	assert.Equal(t, session.ID, retrieved.ID)
	assert.Equal(t, session.UserID, retrieved.UserID)
	assert.Equal(t, session.Username, retrieved.Username)
	assert.Equal(t, session.Email, retrieved.Email)
	assert.Equal(t, session.ESUser, retrieved.ESUser)
	assert.Equal(t, claims["role"], retrieved.Claims["role"])
	assert.WithinDuration(t, session.CreatedAt, retrieved.CreatedAt, time.Second)
	assert.WithinDuration(t, session.ExpiresAt, retrieved.ExpiresAt, time.Second)
}

// Helper function to generate test session IDs
// Uses crypto/rand for proper cryptographic randomness
func generateTestSessionID() string {
	// Generate 32 random bytes
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	// Encode as base64 URL-safe string (results in 43+ characters)
	return base64.URLEncoding.EncodeToString(b)
}
