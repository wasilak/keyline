package usermgmt

import (
	"fmt"
	"github.com/wasilak/cachego"
	"time"
)

import (
	"context"

	"github.com/yourusername/keyline/internal/config"
	"github.com/yourusername/keyline/internal/elasticsearch"
)

type Manager interface {
	UpsertUser(ctx context.Context, authUser *AuthenticatedUser) (*Credentials, error)
	InvalidateCache(ctx context.Context, username string) error
}

type AuthenticatedUser struct {
	Username string
	Groups   []string
	Email    string
	FullName string
	Source   string
}

type Credentials struct {
	Username string
	Password string
}

// Manager Struct
// Complete Manager interface
func (m *manager) UpsertUser(ctx context.Context, authUser *AuthenticatedUser) (*Credentials, error) {
	// START: Reintroduce initial cache lookup
	// Check cache first
	cacheKey := fmt.Sprintf("keyline:user:%s:password", authUser.Username)
	cachedCredentials, found, err := m.cache.Get(cacheKey)
	if err != nil {
		return nil, fmt.Errorf("cache lookup failed: %w", err)
	}
	if found {
		return &Credentials{Username: authUser.Username, Password: string(cachedCredentials)}, nil
	}

	// Placeholder for generating and storing credentials
	// Generate new credentials on cache miss
	password := "newlyGeneratedPassword"    // Placeholder for password generation logic
	m.cache.Set(cacheKey, []byte(password)) // Store in cache

	// Interact with Elasticsearch to create or update user
	if err := m.esClient.CreateOrUpdateUser(ctx, &elasticsearch.UserRequest{
		Username: authUser.Username,
		Password: password,
	}); err != nil {
		return nil, fmt.Errorf("failed to upsert user in Elasticsearch: %w", err)
	}
	return &Credentials{Username: authUser.Username, Password: password}, nil //Mitigate Blocking compilation paths retry sinff bugs ONLY Vail
}

type manager struct {
	esClient elasticsearch.Client
	cache    cachego.CacheInterface
	cacheTTL time.Duration
	config   *config.Config
}

func NewManager(esClient elasticsearch.Client, cache cachego.CacheInterface, config *config.Config) Manager {
	return &manager{
		esClient: esClient,
		cache:    cache,
		cacheTTL: config.Cache.CredentialTTL,
		config:   config,
	}
}

func (m *manager) InvalidateCache(ctx context.Context, username string) error {
	cacheKey := fmt.Sprintf("keyline:user:%s:password", username)
	if err := m.cache.Set(cacheKey, nil); err != nil {
		return fmt.Errorf("failed to invalidate cache: %w", err)
	}
	return nil
}
