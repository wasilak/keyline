package usermgmt

import (
"context"
"encoding/json"
"fmt"
"log/slog"
"time"

"github.com/wasilak/cachego"
"go.opentelemetry.io/otel"
"go.opentelemetry.io/otel/attribute"
"go.opentelemetry.io/otel/codes"
"go.opentelemetry.io/otel/metric"

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

type cachedCredentials struct {
EncryptedPassword string `json:"encrypted_password"`
}

type manager struct {
esClient         elasticsearch.Client
roleMapper       *RoleMapper
cache            cachego.CacheInterface
pwdGen           *PasswordGenerator
encryptor        Encryptor
cacheTTL         time.Duration
config           *config.Config
upsertsTotal     metric.Int64Counter
upsertDuration   metric.Float64Histogram
cacheHitsTotal   metric.Int64Counter
cacheMissesTotal metric.Int64Counter
}

func NewManager(
esClient elasticsearch.Client,
roleMapper *RoleMapper,
cache cachego.CacheInterface,
pwdGen *PasswordGenerator,
encryptor Encryptor,
config *config.Config,
) (Manager, error) {
meter := otel.Meter("keyline")
upsertsTotal, _ := meter.Int64Counter("keyline.user.upserts.total")
upsertDuration, _ := meter.Float64Histogram("keyline.user.upsert.duration")
cacheHitsTotal, _ := meter.Int64Counter("keyline.cred.cache.hits.total")
cacheMissesTotal, _ := meter.Int64Counter("keyline.cred.cache.misses.total")

return &manager{
esClient:         esClient,
roleMapper:       roleMapper,
cache:            cache,
pwdGen:           pwdGen,
encryptor:        encryptor,
cacheTTL:         config.Cache.CredentialTTL,
config:           config,
upsertsTotal:     upsertsTotal,
upsertDuration:   upsertDuration,
cacheHitsTotal:   cacheHitsTotal,
cacheMissesTotal: cacheMissesTotal,
}, nil
}

func (m *manager) UpsertUser(ctx context.Context, authUser *AuthenticatedUser) (*Credentials, error) {
startTime := time.Now()
ctx, span := otel.Tracer("keyline").Start(ctx, "usermgmt.upsert_user")
defer span.End()

span.SetAttributes(
attribute.String("user.username", authUser.Username),
attribute.String("user.source", authUser.Source),
attribute.StringSlice("user.groups", authUser.Groups),
)

cacheKey := fmt.Sprintf("keyline:user:%s:password", authUser.Username)
if cachedData, found, err := m.cache.Get(cacheKey); err == nil && found && len(cachedData) > 0 {
m.cacheHitsTotal.Add(ctx, 1)
span.SetAttributes(attribute.Bool("cache.hit", true))

var cached cachedCredentials
if err := json.Unmarshal(cachedData, &cached); err == nil {
if password, err := m.encryptor.Decrypt(cached.EncryptedPassword); err == nil {
duration := time.Since(startTime).Seconds()
m.upsertDuration.Record(ctx, duration, metric.WithAttributes(
attribute.String("cache_status", "hit"),
))
slog.DebugContext(ctx, "Using cached credentials",
slog.String("username", authUser.Username),
slog.Float64("duration_ms", duration*1000),
)
return &Credentials{Username: authUser.Username, Password: password}, nil
}
}
}

m.cacheMissesTotal.Add(ctx, 1)
span.SetAttributes(attribute.Bool("cache.hit", false))

password, err := m.pwdGen.Generate()
if err != nil {
m.upsertsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String("status", "failure")))
span.RecordError(err)
span.SetStatus(codes.Error, "password generation failed")
return nil, fmt.Errorf("password generation failed: %w", err)
}

roles, err := m.roleMapper.MapGroupsToRoles(ctx, authUser.Groups)
if err != nil {
m.upsertsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String("status", "failure")))
span.RecordError(err)
span.SetStatus(codes.Error, "role mapping failed")
return nil, fmt.Errorf("role mapping failed: %w", err)
}

span.SetAttributes(attribute.StringSlice("user.roles", roles))

req := &elasticsearch.UserRequest{
Username: authUser.Username,
Password: password,
Roles:    roles,
FullName: authUser.FullName,
Email:    authUser.Email,
Metadata: map[string]interface{}{
"source":     authUser.Source,
"groups":     authUser.Groups,
"last_auth":  time.Now().Unix(),
"managed_by": "keyline",
},
}

if err := m.esClient.CreateOrUpdateUser(ctx, req); err != nil {
m.upsertsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String("status", "failure")))
span.RecordError(err)
span.SetStatus(codes.Error, "ES user upsert failed")
return nil, fmt.Errorf("ES user upsert failed: %w", err)
}

slog.InfoContext(ctx, "ES user created/updated",
slog.String("username", authUser.Username),
slog.Any("roles", roles),
slog.String("source", authUser.Source),
)

encryptedPwd, err := m.encryptor.Encrypt(password)
if err == nil {
cached := cachedCredentials{EncryptedPassword: encryptedPwd}
if cachedData, err := json.Marshal(cached); err == nil {
m.cache.Set(cacheKey, cachedData)
}
}

duration := time.Since(startTime).Seconds()
m.upsertsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String("status", "success")))
m.upsertDuration.Record(ctx, duration, metric.WithAttributes(attribute.String("cache_status", "miss")))

slog.InfoContext(ctx, "User upsert completed",
slog.String("username", authUser.Username),
slog.Float64("duration_ms", duration*1000),
)

return &Credentials{Username: authUser.Username, Password: password}, nil
}

func (m *manager) InvalidateCache(ctx context.Context, username string) error {
ctx, span := otel.Tracer("keyline").Start(ctx, "usermgmt.invalidate_cache")
defer span.End()

span.SetAttributes(attribute.String("user.username", username))
cacheKey := fmt.Sprintf("keyline:user:%s:password", username)

if err := m.cache.Set(cacheKey, []byte{}); err != nil {
span.RecordError(err)
span.SetStatus(codes.Error, "cache invalidation failed")
return fmt.Errorf("failed to invalidate cache for user %s: %w", username, err)
}

slog.InfoContext(ctx, "Cache invalidated for user", slog.String("username", username))
return nil
}
