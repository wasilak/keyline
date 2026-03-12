package session

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/wasilak/cachego"
	"github.com/yourusername/keyline/internal/observability"
	"go.opentelemetry.io/otel"
)

const keyPrefix = "session:"

// CreateSession stores a new session in the cache
func CreateSession(ctx context.Context, cache cachego.CacheInterface, session *Session) error {
	tracer := otel.Tracer("keyline")
	ctx, span := tracer.Start(ctx, "session.create")
	defer span.End()

	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	key := keyPrefix + session.ID

	if err := cache.Set(key, data); err != nil {
		slog.ErrorContext(ctx, "Failed to store session",
			slog.String("session_id", session.ID),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to store session: %w", err)
	}

	slog.InfoContext(ctx, "Session created",
		slog.String("session_id_hash", observability.HashSessionID(session.ID)),
		slog.String("username", session.Username),
		slog.String("action", "created"),
		slog.Duration("ttl", time.Until(session.ExpiresAt)),
	)

	return nil
}

// GetSession retrieves a session from the cache
func GetSession(ctx context.Context, cache cachego.CacheInterface, sessionID string) (*Session, error) {
	tracer := otel.Tracer("keyline")
	ctx, span := tracer.Start(ctx, "session.get")
	defer span.End()

	key := keyPrefix + sessionID

	data, found, err := cache.Get(key)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to retrieve session",
			slog.String("session_id", sessionID),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("failed to retrieve session: %w", err)
	}

	if !found || len(data) == 0 {
		slog.InfoContext(ctx, "Session not found",
			slog.String("session_id_hash", observability.HashSessionID(sessionID)),
			slog.String("action", "not_found"),
		)
		return nil, nil
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	// Check if session is expired
	if session.IsExpired() {
		slog.InfoContext(ctx, "Session expired, deleting",
			slog.String("session_id_hash", observability.HashSessionID(sessionID)),
			slog.String("username", session.Username),
			slog.String("action", "expired"),
		)
		_ = DeleteSession(ctx, cache, sessionID)
		return nil, nil
	}

	slog.InfoContext(ctx, "Session retrieved",
		slog.String("session_id_hash", observability.HashSessionID(session.ID)),
		slog.String("username", session.Username),
		slog.String("action", "validated"),
	)

	return &session, nil
}

// DeleteSession removes a session from the cache
func DeleteSession(ctx context.Context, cache cachego.CacheInterface, sessionID string) error {
	tracer := otel.Tracer("keyline")
	ctx, span := tracer.Start(ctx, "session.delete")
	defer span.End()

	key := keyPrefix + sessionID

	// cachego doesn't have a Delete method, so we set an empty value
	// This effectively removes the session from the cache
	if err := cache.Set(key, []byte{}); err != nil {
		slog.ErrorContext(ctx, "Failed to delete session",
			slog.String("session_id", sessionID),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to delete session: %w", err)
	}

	slog.InfoContext(ctx, "Session deleted",
		slog.String("session_id_hash", observability.HashSessionID(sessionID)),
		slog.String("action", "deleted"),
	)

	return nil
}
