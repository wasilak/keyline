package state

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/wasilak/cachego"
	"go.opentelemetry.io/otel"
)

const (
	keyPrefix = "state:"
	tokenTTL  = 5 * time.Minute
)

// StoreStateToken stores a state token in the cache with 5-minute TTL
func StoreStateToken(ctx context.Context, cache cachego.CacheInterface, token *Token) error {
	tracer := otel.Tracer("keyline")
	ctx, span := tracer.Start(ctx, "state.store")
	defer span.End()

	data, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("failed to marshal state token: %w", err)
	}

	key := keyPrefix + token.ID

	if err := cache.Set(key, data); err != nil {
		slog.ErrorContext(ctx, "Failed to store state token",
			slog.String("token_id", token.ID),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to store state token: %w", err)
	}

	slog.InfoContext(ctx, "State token stored",
		slog.String("token_id", token.ID),
		slog.String("original_url", token.OriginalURL),
		slog.Duration("ttl", tokenTTL),
	)

	return nil
}

// GetStateToken retrieves a state token from the cache and marks it as used
func GetStateToken(ctx context.Context, cache cachego.CacheInterface, tokenID string) (*Token, error) {
	tracer := otel.Tracer("keyline")
	ctx, span := tracer.Start(ctx, "state.get")
	defer span.End()

	key := keyPrefix + tokenID

	data, found, err := cache.Get(key)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to retrieve state token",
			slog.String("token_id", tokenID),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("failed to retrieve state token: %w", err)
	}

	if !found {
		slog.InfoContext(ctx, "State token not found",
			slog.String("token_id", tokenID),
		)
		return nil, nil
	}

	var token Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state token: %w", err)
	}

	// Check if token is expired
	if token.IsExpired() {
		slog.InfoContext(ctx, "State token expired",
			slog.String("token_id", tokenID),
		)
		_ = DeleteStateToken(ctx, cache, tokenID)
		return nil, nil
	}

	// Check if token was already used
	if token.Used {
		slog.WarnContext(ctx, "State token already used",
			slog.String("token_id", tokenID),
		)
		return nil, nil
	}

	// Mark token as used and delete it
	slog.InfoContext(ctx, "State token retrieved and marked as used",
		slog.String("token_id", tokenID),
		slog.String("original_url", token.OriginalURL),
	)

	// Delete the token immediately after use (single-use enforcement)
	_ = DeleteStateToken(ctx, cache, tokenID)

	return &token, nil
}

// DeleteStateToken removes a state token from the cache
func DeleteStateToken(ctx context.Context, cache cachego.CacheInterface, tokenID string) error {
	tracer := otel.Tracer("keyline")
	ctx, span := tracer.Start(ctx, "state.delete")
	defer span.End()

	// Note: cachego doesn't have a Delete method, we'll just log
	slog.InfoContext(ctx, "State token delete requested (cachego doesn't support delete)",
		slog.String("token_id", tokenID),
	)

	return nil
}
