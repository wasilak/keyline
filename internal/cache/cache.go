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
// Requirements: 16.1, 16.2, 16.9, 17.1
func InitCache(ctx context.Context, cfg *config.CacheConfig) (cachego.CacheInterface, error) {
	slog.InfoContext(ctx, "Initializing cache",
		slog.String("backend", cfg.Backend),
	)

	// Validate backend type
	if cfg.Backend != "redis" && cfg.Backend != "memory" {
		return nil, fmt.Errorf("invalid cache backend: %s (must be 'redis' or 'memory')", cfg.Backend)
	}

	cacheConfig := cachegoConfig.Config{
		Type: cfg.Backend,
		CTX:  ctx,
	}

	// Configure Redis backend if specified
	if cfg.Backend == "redis" {
		if cfg.RedisURL == "" {
			return nil, fmt.Errorf("redis_url is required when backend is redis")
		}

		// RedisHost can include authentication in the URL format:
		// redis://[:password@]host[:port][/database]
		// If RedisPassword is provided separately, we need to construct the URL
		redisURL := cfg.RedisURL
		if cfg.RedisPassword != "" && !containsAuth(redisURL) {
			// Insert password into URL if not already present
			redisURL = insertPasswordIntoURL(redisURL, cfg.RedisPassword)
		}

		cacheConfig.RedisHost = redisURL
		cacheConfig.RedisDB = cfg.RedisDB

		slog.InfoContext(ctx, "Configuring Redis cache",
			slog.String("redis_url", maskPassword(redisURL)),
			slog.Int("redis_db", cfg.RedisDB),
		)
	} else {
		slog.InfoContext(ctx, "Configuring in-memory cache")
	}

	// Initialize cache backend
	cache, err := cachego.CacheInit(ctx, cacheConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize cache: %w", err)
	}

	// Test connection by attempting a simple operation
	// Requirement 16.2: If the Redis connection fails at startup, refuse to start
	testKey := "healthcheck"
	testValue := []byte("ok")

	// Try to set a test value
	if err := cache.Set(testKey, testValue); err != nil {
		return nil, fmt.Errorf("cache connection test failed: %w", err)
	}

	// Try to get the test value back
	if _, _, err := cache.Get(testKey); err != nil {
		return nil, fmt.Errorf("cache connection test failed on read: %w", err)
	}

	// Clean up test key (best effort, ignore errors)
	_ = cache.Set(testKey, []byte{})

	slog.InfoContext(ctx, "Cache initialized and connection verified",
		slog.String("backend", cfg.Backend),
	)

	return cache, nil
}

// containsAuth checks if the Redis URL already contains authentication
func containsAuth(url string) bool {
	// Check if URL contains @ symbol indicating authentication
	for i := 0; i < len(url); i++ {
		if url[i] == '@' {
			return true
		}
		// Stop at first / after protocol
		if i > 8 && url[i] == '/' {
			return false
		}
	}
	return false
}

// insertPasswordIntoURL inserts password into Redis URL
func insertPasswordIntoURL(url, password string) string {
	// Format: redis://host:port -> redis://:password@host:port
	if len(url) > 8 && url[:8] == "redis://" {
		return "redis://:" + password + "@" + url[8:]
	}
	return url
}

// maskPassword masks the password in Redis URL for logging
func maskPassword(url string) string {
	// Find @ symbol
	atPos := -1
	for i := 0; i < len(url); i++ {
		if url[i] == '@' {
			atPos = i
			break
		}
	}

	if atPos == -1 {
		return url
	}

	// Find start of password (after ://)
	startPos := 0
	for i := 0; i < atPos; i++ {
		if i+2 < len(url) && url[i:i+3] == "://" {
			startPos = i + 3
			break
		}
	}

	if startPos == 0 {
		return url
	}

	// Mask the password portion
	return url[:startPos] + "****" + url[atPos:]
}
