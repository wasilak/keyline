package state

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

// NewRedisStore creates a new Redis state token store
func NewRedisStore(client *redis.Client, logger *slog.Logger) *RedisStore {
	return &RedisStore{
		client: client,
		logger: logger,
	}
}

// Store saves a state token in Redis
func (s *RedisStore) Store(ctx context.Context, token *Token) error {
	// Serialize token to JSON
	data, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("failed to marshal state token: %w", err)
	}

	// Store in Redis with 5-minute TTL
	key := fmt.Sprintf("state:%s", token.ID)
	if err := s.client.Set(ctx, key, data, 5*time.Minute).Err(); err != nil {
		return fmt.Errorf("failed to store state token in Redis: %w", err)
	}

	s.logger.Debug("State token stored in Redis",
		slog.String("token_id", token.ID),
		slog.String("original_url", token.OriginalURL),
	)

	return nil
}

// Get retrieves and marks a state token as used
func (s *RedisStore) Get(ctx context.Context, tokenID string) (*Token, error) {
	key := fmt.Sprintf("state:%s", tokenID)

	// Get from Redis
	data, err := s.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, fmt.Errorf("state token not found")
	} else if err != nil {
		return nil, fmt.Errorf("failed to get state token from Redis: %w", err)
	}

	// Deserialize token
	var token Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state token: %w", err)
	}

	// Check if already used
	if token.Used {
		return nil, fmt.Errorf("state token already used")
	}

	// Mark as used and update in Redis
	token.Used = true
	updatedData, err := json.Marshal(token)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal updated token: %w", err)
	}

	// Update with remaining TTL
	ttl, err := s.client.TTL(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get TTL: %w", err)
	}

	if err := s.client.Set(ctx, key, updatedData, ttl).Err(); err != nil {
		return nil, fmt.Errorf("failed to update state token: %w", err)
	}

	return &token, nil
}

// Delete removes a state token from Redis
func (s *RedisStore) Delete(ctx context.Context, tokenID string) error {
	key := fmt.Sprintf("state:%s", tokenID)

	if err := s.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete state token from Redis: %w", err)
	}

	s.logger.Debug("State token deleted from Redis",
		slog.String("token_id", tokenID),
	)

	return nil
}
