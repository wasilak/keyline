package cache

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/wasilak/cachego"
	cachegoConfig "github.com/wasilak/cachego/config"
	"github.com/yourusername/keyline/internal/config"
)

// ExtendedCacheInterface extends cachego.CacheInterface with Delete functionality
type ExtendedCacheInterface interface {
	cachego.CacheInterface
	// Delete removes a key from the cache
	Delete(ctx context.Context, key string) error
	// GetUnderlying returns the underlying cache implementation (for advanced use)
	GetUnderlying() interface{}
}

// extendedCache wraps cachego with Delete functionality
type extendedCache struct {
	cachego.CacheInterface
	backendType string
	redisClient *redis.Client
	// For memory backend: track deleted keys with timestamps
	deletedKeys map[string]time.Time
	mu          sync.RWMutex
}

// Delete removes a key from the cache
func (c *extendedCache) Delete(ctx context.Context, key string) error {
	switch c.backendType {
	case "redis":
		if c.redisClient != nil {
			return c.redisClient.Del(ctx, key).Err()
		}
		// Fallback: set empty value if Redis client not available
		return c.CacheInterface.Set(key, []byte{})
	case "memory":
		// For memory backend, track deleted keys
		c.mu.Lock()
		c.deletedKeys[key] = time.Now()
		c.mu.Unlock()
		// Also set empty value for compatibility
		return c.CacheInterface.Set(key, []byte{})
	default:
		// Fallback: set empty value
		return c.CacheInterface.Set(key, []byte{})
	}
}

// Get retrieves a value, filtering out deleted keys for memory backend
func (c *extendedCache) Get(key string) ([]byte, bool, error) {
	// For memory backend, check if key was deleted
	if c.backendType == "memory" {
		c.mu.RLock()
		if _, deleted := c.deletedKeys[key]; deleted {
			c.mu.RUnlock()
			return nil, false, nil
		}
		c.mu.RUnlock()
	}

	return c.CacheInterface.Get(key)
}

// GetUnderlying returns the underlying cache implementation
func (c *extendedCache) GetUnderlying() interface{} {
	return c.CacheInterface
}

// cleanupDeletedKeys periodically removes old deleted key entries (for memory backend)
func (c *extendedCache) cleanupDeletedKeys() {
	if c.backendType != "memory" {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, deletedAt := range c.deletedKeys {
		// Remove entries older than 1 hour
		if now.Sub(deletedAt) > time.Hour {
			delete(c.deletedKeys, key)
		}
	}
}

// startCleanup starts the background cleanup goroutine
func (c *extendedCache) startCleanup(ctx context.Context) {
	if c.backendType != "memory" {
		return
	}

	ticker := time.NewTicker(5 * time.Minute)
	go func() {
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				c.cleanupDeletedKeys()
			}
		}
	}()
}

// redisConfig holds the parsed Redis configuration
type redisConfig struct {
	addr     string
	password string
	db       int
}

// parseRedisURL parses a Redis URL to extract connection details
func parseRedisURL(url string) redisConfig {
	cfg := redisConfig{}

	// Simple parser for redis://[:password@]host[:port][/database]
	// Remove protocol prefix
	if len(url) > 8 && url[:8] == "redis://" {
		url = url[8:]
	}

	// Extract password if present
	atIndex := -1
	for i := 0; i < len(url); i++ {
		if url[i] == '@' {
			atIndex = i
			break
		}
	}

	if atIndex != -1 {
		// Has password
		authPart := url[:atIndex]
		if len(authPart) > 0 && authPart[0] == ':' {
			authPart = authPart[1:] // Remove leading :
		}
		cfg.password = authPart
		url = url[atIndex+1:] // Remove auth part
	}

	// Extract database if present
	slashIndex := -1
	for i := 0; i < len(url); i++ {
		if url[i] == '/' {
			slashIndex = i
			break
		}
	}

	if slashIndex != -1 {
		// Has database
		fmt.Sscanf(url[slashIndex+1:], "%d", &cfg.db)
		url = url[:slashIndex]
	}

	cfg.addr = url
	return cfg
}

// InitExtendedCache initializes the cache backend with Delete support
func InitExtendedCache(ctx context.Context, cfg *config.CacheConfig) (ExtendedCacheInterface, error) {
	slog.InfoContext(ctx, "Initializing extended cache",
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

	var redisClient *redis.Client

	// Configure Redis backend if specified
	if cfg.Backend == "redis" {
		if cfg.RedisURL == "" {
			return nil, fmt.Errorf("redis_url is required when backend is redis")
		}

		// Parse Redis URL and create client
		redisURL := cfg.RedisURL
		if cfg.RedisPassword != "" && !containsAuth(redisURL) {
			redisURL = insertPasswordIntoURL(redisURL, cfg.RedisPassword)
		}

		cacheConfig.RedisHost = redisURL
		cacheConfig.RedisDB = cfg.RedisDB

		// Create go-redis client for Delete operations
		rcfg := parseRedisURL(redisURL)
		redisClient = redis.NewClient(&redis.Options{
			Addr:     rcfg.addr,
			Password: rcfg.password,
			DB:       rcfg.db,
		})

		// Test Redis connection
		if err := redisClient.Ping(ctx).Err(); err != nil {
			return nil, fmt.Errorf("failed to connect to Redis: %w", err)
		}

		slog.InfoContext(ctx, "Configured Redis cache with extended interface",
			slog.String("redis_url", maskPassword(redisURL)),
			slog.Int("redis_db", cfg.RedisDB),
		)
	} else {
		slog.InfoContext(ctx, "Configured in-memory cache with extended interface")
	}

	// Initialize cachego backend
	cachegoInterface, err := cachego.CacheInit(ctx, cacheConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize cache: %w", err)
	}

	// Test connection
	testKey := "healthcheck"
	testValue := []byte("ok")

	if err := cachegoInterface.Set(testKey, testValue); err != nil {
		return nil, fmt.Errorf("cache connection test failed: %w", err)
	}

	if _, _, err := cachegoInterface.Get(testKey); err != nil {
		return nil, fmt.Errorf("cache connection test failed on read: %w", err)
	}

	// Clean up test key
	_ = cachegoInterface.Set(testKey, []byte{})

	// Create extended cache wrapper
	extended := &extendedCache{
		CacheInterface: cachegoInterface,
		backendType:    cfg.Backend,
		redisClient:    redisClient,
		deletedKeys:    make(map[string]time.Time),
	}

	// Start background cleanup for memory backend
	if cfg.Backend == "memory" {
		extended.startCleanup(ctx)
	}

	slog.InfoContext(ctx, "Extended cache initialized",
		slog.String("backend", cfg.Backend),
	)

	return extended, nil
}
