package cache

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/wasilak/cachego"
	cachegoConfig "github.com/wasilak/cachego/config"
	"github.com/yourusername/keyline/internal/config"
)

// InitCache initializes the cache backend based on configuration
func InitCache(ctx context.Context, cfg *config.CacheConfig) (cachego.CacheInterface, error) {
	slog.InfoContext(ctx, "Initializing cache",
		slog.String("backend", cfg.Backend),
	)

	cacheConfig := cachegoConfig.Config{
		Type: cfg.Backend,
		CTX:  ctx,
	}

	if cfg.Backend == "redis" {
		if cfg.RedisURL == "" {
			return nil, fmt.Errorf("redis_url is required when backend is redis")
		}
		cacheConfig.RedisHost = cfg.RedisURL
		cacheConfig.RedisDB = cfg.RedisDB
	}

	cache, err := cachego.CacheInit(ctx, cacheConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize cache: %w", err)
	}

	slog.InfoContext(ctx, "Cache initialized",
		slog.String("backend", cfg.Backend),
	)

	return cache, nil
}
