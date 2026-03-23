package usermgmt

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	cachegoConfig "github.com/wasilak/cachego/config"
	"github.com/yourusername/keyline/internal/config"
	"github.com/yourusername/keyline/internal/elasticsearch"
)

type MockCache struct {
	mock.Mock
}

func (m *MockCache) Init() error {
	return nil
}

func (m *MockCache) GetConfig() cachegoConfig.Config {
	return cachegoConfig.Config{}
}

func (m *MockCache) Get(key string) ([]byte, bool, error) {
	args := m.Called(key)
	return args.Get(0).([]byte), args.Bool(1), args.Error(2)
}

func (m *MockCache) ExtendTTL(key string, value []byte) error {
	args := m.Called(key, value)
	return args.Error(0)
}

func (m *MockCache) GetItemTTL(key string) (time.Duration, bool, error) {
	args := m.Called(key)
	if duration, ok := args.Get(0).(time.Duration); ok {
		return duration, args.Bool(1), args.Error(2)
	}
	return 0, false, args.Error(2)
}

func (m *MockCache) Set(key string, value []byte) error {
	args := m.Called(key, value)
	return args.Error(0)
}

type MockElasticsearchClient struct {
	mock.Mock
}

func (m *MockElasticsearchClient) GetUser(ctx context.Context, username string) (*elasticsearch.User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*elasticsearch.User), args.Error(1)
}

func (m *MockElasticsearchClient) ValidateConnection(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)

}

func (m *MockElasticsearchClient) DeleteUser(ctx context.Context, username string) error {
	args := m.Called(ctx, username)
	return args.Error(0)
}

func (m *MockElasticsearchClient) RetryCreateOrUpdateUser(ctx context.Context, req *elasticsearch.UserRequest, attempts int) error {
	var err error
	for i := 0; i < attempts; i++ {
		args := m.Called(ctx, req)
		err = args.Error(0)
		if err == nil {
			break
		}
	}
	return err
}

func (m *MockElasticsearchClient) CreateOrUpdateUser(ctx context.Context, req *elasticsearch.UserRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func TestUpsertUser_CacheHit(t *testing.T) {
	mockCache := new(MockCache)
	mockES := new(MockElasticsearchClient)
	cacheTTL := 24 * time.Hour
	manager := &manager{
		cache:      mockCache,
		esClient:   mockES,
		cacheTTL:   cacheTTL,
		roleMapper: NewRoleMapper(&config.Config{}),
	}

	username := "testuser"
	cacheKey := "keyline:user:testuser:password"
	mockCache.On("Get", cacheKey).Return([]byte("cached-password"), true, nil)
	mockCache.On("GetItemTTL", cacheKey).Return(2*time.Hour, true, nil)

	credentials, err := manager.UpsertUser(context.TODO(), &AuthenticatedUser{Username: username})

	assert.NoError(t, err)
	assert.Equal(t, "testuser", credentials.Username)
	assert.Equal(t, "cached-password", credentials.Password)
	mockCache.AssertExpectations(t)
}

func TestUpsertUser_CacheMiss(t *testing.T) {
	mockCache := new(MockCache)
	mockES := new(MockElasticsearchClient)
	cacheTTL := 24 * time.Hour
	cfg := &config.Config{
		DefaultESRoles: []string{"viewer"},
	}
	manager := &manager{
		cache:      mockCache,
		esClient:   mockES,
		cacheTTL:   cacheTTL,
		roleMapper: NewRoleMapper(cfg),
	}

	username := "testuser"
	cacheKey := "keyline:user:testuser:password"

	mockCache.On("Get", cacheKey).Return([]byte{}, false, nil)
	mockCache.On("Set", cacheKey, mock.Anything).Return(nil)
	mockES.On("CreateOrUpdateUser", mock.Anything, mock.Anything).Return(nil)

	credentials, err := manager.UpsertUser(context.TODO(), &AuthenticatedUser{Username: username})
	assert.NoError(t, err)
	assert.Equal(t, "testuser", credentials.Username)
	assert.NotEmpty(t, credentials.Password)
	mockCache.AssertExpectations(t)
	mockES.AssertExpectations(t)
}
