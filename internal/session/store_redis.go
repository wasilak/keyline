package session

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStore implements Store using Redis
type RedisStore struct {
	client *redis.Client
	logger *slog.Logger
}

// NewRedisStore creates a new Redis session store
func NewRedisStore(redisURL, password string, db int, logger *slog.Logger) (*RedisStore, error) {
	// Parse Redis URL
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	// Override password and DB if provided
	if password != "" {
		opts.Password = password
	}
	if db > 0 {
		opts.DB = db
	}

	// Configure connection pool
	opts.MinIdleConns = 5
	opts.PoolSize = 20

	// Create client
	client := redis.NewClient(opts)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.Info("Connected to Redis",
		slog.String("url", redisURL),
		slog.Int("db", opts.DB),
	)

	return &RedisStore{
		client: client,
		logger: logger,
	}, nil
}

// Create stores a new session in Redis
func (s *RedisStore) Create(ctx context.Context, session *Session) error {
	// Serialize session to JSON
	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	// Calculate TTL
	ttl := time.Until(session.ExpiresAt)
	if ttl <= 0 {
		return fmt.Errorf("session already expired")
	}

	// Store in Redis with TTL
	key := fmt.Sprintf("session:%s", session.ID)
	if err := s.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("failed to store session in Redis: %w", err)
	}

	s.logger.Debug("Session created in Redis",
		slog.String("session_id", session.ID),
		slog.String("username", session.Username),
		slog.Duration("ttl", ttl),
	)

	return nil
}

// Get retrieves a session from Redis
func (s *RedisStore) Get(ctx context.Context, sessionID string) (*Session, error) {
	key := fmt.Sprintf("session:%s", sessionID)

	// Get from Redis
	data, err := s.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, fmt.Errorf("session not found")
	} else if err != nil {
		return nil, fmt.Errorf("failed to get session from Redis: %w", err)
	}

	// Deserialize session
	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	// Check if expired (shouldn't happen with Redis TTL, but double-check)
	if session.IsExpired() {
		// Delete expired session
		s.client.Del(ctx, key)
		return nil, fmt.Errorf("session expired")
	}

	return &session, nil
}

// Delete removes a session from Redis
func (s *RedisStore) Delete(ctx context.Context, sessionID string) error {
	key := fmt.Sprintf("session:%s", sessionID)

	if err := s.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete session from Redis: %w", err)
	}

	s.logger.Debug("Session deleted from Redis",
		slog.String("session_id", sessionID),
	)

	return nil
}

// Cleanup is a no-op for Redis (TTL handles expiration)
func (s *RedisStore) Cleanup(ctx context.Context) error {
	// Redis automatically removes expired keys
	return nil
}

// Health checks if Redis is accessible
func (s *RedisStore) Health(ctx context.Context) error {
	if err := s.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("Redis health check failed: %w", err)
	}
	return nil
}

// Close closes the Redis connection
func (s *RedisStore) Close() error {
	return s.client.Close()
}
