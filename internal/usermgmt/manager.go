package usermgmt

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/wasilak/cachego"

	"github.com/yourusername/keyline/internal/config"
	"github.com/yourusername/keyline/internal/elasticsearch"
)

type Manager interface {
	UpsertUser(ctx context.Context, authUser *AuthenticatedUser) (*Credentials, error)
	InvalidateCache(ctx context.Context, username string) error
	GetUsernameFromAuthHeader(authHeader string) (string, error)
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

func (m *manager) UpsertUser(ctx context.Context, authUser *AuthenticatedUser) (*Credentials, error) {
	cacheKey := fmt.Sprintf("keyline:user:%s:password", authUser.Username)

	cachedCredentials, found, err := m.cache.Get(cacheKey)
	if err != nil {
		return nil, fmt.Errorf("cache lookup failed: %w", err)
	}

	if found {
		remainingTTL, exists, ttlErr := m.cache.GetItemTTL(cacheKey)
		if ttlErr == nil && exists {
			threshold := m.calculateTTLThreshold()
			if remainingTTL < threshold {
				if err := m.invalidateAndRegenerate(ctx, authUser, cacheKey); err != nil {
					return nil, err
				}
				return m.getFreshCredentials(ctx, authUser, cacheKey)
			}
		}

		return &Credentials{Username: authUser.Username, Password: string(cachedCredentials)}, nil
	}

	newPassword, err := m.generatePassword()
	if err != nil {
		return nil, fmt.Errorf("failed to generate password: %w", err)
	}

	if err := m.createOrUpdateESUser(ctx, authUser, newPassword); err != nil {
		return nil, err
	}

	if err := m.cache.Set(cacheKey, []byte(newPassword)); err != nil {
		return nil, fmt.Errorf("failed to cache credentials: %w", err)
	}

	return &Credentials{Username: authUser.Username, Password: newPassword}, nil
}

func (m *manager) calculateTTLThreshold() time.Duration {
	threshold := m.cacheTTL / 10
	if threshold > time.Hour {
		return time.Hour
	}
	return threshold
}

func (m *manager) generatePassword() (string, error) {
	pg := NewPasswordGenerator(32)
	return pg.Generate()
}

func (m *manager) createOrUpdateESUser(ctx context.Context, authUser *AuthenticatedUser, password string) error {
	roles, err := m.roleMapper.MapGroupsToRoles(ctx, authUser.Groups)
	if err != nil {
		return fmt.Errorf("failed to map groups to roles: %w", err)
	}

	userReq := &elasticsearch.UserRequest{
		Username: authUser.Username,
		Password: password,
		Roles:    roles,
		FullName: authUser.FullName,
		Email:    authUser.Email,
	}
	return m.esClient.CreateOrUpdateUser(ctx, userReq)
}

func (m *manager) invalidateAndRegenerate(ctx context.Context, authUser *AuthenticatedUser, cacheKey string) error {
	if err := m.cache.Set(cacheKey, nil); err != nil {
		return fmt.Errorf("failed to invalidate cache: %w", err)
	}

	newPassword, err := m.generatePassword()
	if err != nil {
		return fmt.Errorf("failed to generate password: %w", err)
	}

	if err := m.createOrUpdateESUser(ctx, authUser, newPassword); err != nil {
		return err
	}

	if err := m.cache.Set(cacheKey, []byte(newPassword)); err != nil {
		return fmt.Errorf("failed to cache credentials: %w", err)
	}

	return nil
}

func (m *manager) getFreshCredentials(ctx context.Context, authUser *AuthenticatedUser, cacheKey string) (*Credentials, error) {
	cachedCredentials, found, err := m.cache.Get(cacheKey)
	if err != nil {
		return nil, fmt.Errorf("cache lookup failed: %w", err)
	}
	if !found {
		return m.UpsertUser(ctx, authUser)
	}
	return &Credentials{Username: authUser.Username, Password: string(cachedCredentials)}, nil
}

func (m *manager) GetUsernameFromAuthHeader(authHeader string) (string, error) {
	if authHeader == "" || !strings.HasPrefix(authHeader, "Basic ") {
		return "", fmt.Errorf("invalid auth header")
	}

	encoded := strings.TrimPrefix(authHeader, "Basic ")
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("failed to decode auth header: %w", err)
	}

	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid auth header format")
	}

	return parts[0], nil
}

type manager struct {
	esClient   elasticsearch.Client
	cache      cachego.CacheInterface
	cacheTTL   time.Duration
	config     *config.Config
	roleMapper *RoleMapper
}

func NewManager(esClient elasticsearch.Client, cache cachego.CacheInterface, config *config.Config) Manager {
	roleMapper := NewRoleMapper(config)
	return &manager{
		esClient:   esClient,
		cache:      cache,
		cacheTTL:   config.Cache.CredentialTTL,
		config:     config,
		roleMapper: roleMapper,
	}
}

func (m *manager) InvalidateCache(ctx context.Context, username string) error {
	cacheKey := fmt.Sprintf("keyline:user:%s:password", username)
	if err := m.cache.Set(cacheKey, nil); err != nil {
		return fmt.Errorf("failed to invalidate cache: %w", err)
	}
	return nil
}
