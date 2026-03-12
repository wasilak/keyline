# Dynamic Elasticsearch User Management - Design

## Overview

This design document describes the architecture and implementation approach for dynamic Elasticsearch user management in Keyline. The system automatically creates and manages ES users for all authenticated users (OIDC, Basic Auth, etc.) with role-based access control.

## Architecture

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         Keyline Proxy                            │
│                                                                   │
│  ┌──────────────┐      ┌──────────────┐      ┌──────────────┐  │
│  │ Auth Engine  │─────▶│ User Manager │─────▶│ ES API Client│  │
│  │ (OIDC/Basic) │      │              │      │              │  │
│  └──────────────┘      └──────┬───────┘      └──────────────┘  │
│                               │                                  │
│                               ▼                                  │
│                        ┌──────────────┐                          │
│                        │ Role Mapper  │                          │
│                        └──────────────┘                          │
│                               │                                  │
│                               ▼                                  │
│                        ┌──────────────┐                          │
│                        │ Cred Cache   │◀────▶ Redis/Memory      │
│                        │ (cachego)    │                          │
│                        └──────────────┘                          │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
                    ┌───────────────────────┐
                    │   Elasticsearch       │
                    │   Security API        │
                    │   /_security/user     │
                    └───────────────────────┘
```

### Component Interaction Flow

```
1. User authenticates (OIDC/Basic Auth)
   ↓
2. Auth Engine extracts user metadata (username, groups, email)
   ↓
3. User Manager checks credential cache
   ↓
4. If cache miss:
   a. Generate random password
   b. Role Mapper evaluates groups → ES roles
   c. ES API Client creates/updates user
   d. Cache credentials with TTL
   ↓
5. Return ES credentials for Authorization header
   ↓
6. Forward request to ES with credentials
```

## Component Design

### 1. ES API Client (`internal/elasticsearch/client.go`)

**Purpose**: Interact with Elasticsearch Security API for user management

**Uses**: `github.com/wasilak/otelgo` for OpenTelemetry tracing

**Interface**:
```go
type Client interface {
    // CreateOrUpdateUser creates or updates an ES user
    CreateOrUpdateUser(ctx context.Context, req *UserRequest) error
    
    // GetUser retrieves user information
    GetUser(ctx context.Context, username string) (*User, error)
    
    // DeleteUser deletes an ES user (optional, for cleanup)
    DeleteUser(ctx context.Context, username string) error
    
    // ValidateConnection validates admin credentials and connectivity
    ValidateConnection(ctx context.Context) error
}

type UserRequest struct {
    Username string
    Password string
    Roles    []string
    FullName string
    Email    string
    Metadata map[string]interface{}
}

type User struct {
    Username string
    Roles    []string
    FullName string
    Email    string
    Metadata map[string]interface{}
    Enabled  bool
}
```

**Implementation Details**:
- Uses `net/http` client with TLS configuration
- Admin credentials from config (`elasticsearch.admin_user`, `elasticsearch.admin_password`)
- Retry logic: 3 attempts with exponential backoff (1s, 2s, 4s)
- Circuit breaker pattern for ES unavailability
- Request timeout: 30 seconds
- API endpoint: `PUT /_security/user/{username}`
- **OpenTelemetry tracing**: Uses `otelgo` for distributed tracing of ES API calls

**Error Handling**:
- 401/403: Invalid admin credentials → log error, fail startup
- 404: User not found (on GET) → return nil
- 429: Rate limited → retry with backoff
- 5xx: ES unavailable → retry with backoff, circuit breaker
- Network errors: Retry with backoff

### 2. User Manager (`internal/usermgmt/manager.go`)

**Purpose**: Orchestrate user creation, role mapping, and credential caching

**Uses**: `github.com/wasilak/cachego` for credential caching

**Interface**:
```go
type Manager interface {
    // UpsertUser creates or updates an ES user for authenticated user
    // Returns ES credentials (username, password) for Authorization header
    UpsertUser(ctx context.Context, authUser *AuthenticatedUser) (*Credentials, error)
    
    // InvalidateCache invalidates cached credentials for a user
    InvalidateCache(ctx context.Context, username string) error
}

type AuthenticatedUser struct {
    Username string
    Groups   []string
    Email    string
    FullName string
    Source   string // "oidc:google", "basic_auth", etc.
}

type Credentials struct {
    Username string
    Password string
}
```

**Implementation**:
```go
type manager struct {
    esClient    elasticsearch.Client
    roleMapper  *RoleMapper
    cache       cachego.CacheInterface  // github.com/wasilak/cachego
    pwdGen      *PasswordGenerator
    encryptor   Encryptor               // For encrypting cached passwords
    cacheTTL    time.Duration
    config      *config.Config
    logger      *loggergo.Logger        // github.com/wasilak/loggergo
}

func (m *manager) UpsertUser(ctx context.Context, authUser *AuthenticatedUser) (*Credentials, error) {
    // 1. Check cache
    cacheKey := fmt.Sprintf("keyline:user:%s:password", authUser.Username)
    if encryptedPwd, err := m.cache.Get(cacheKey); err == nil {
        // Decrypt password from cache
        password, err := m.encryptor.Decrypt(encryptedPwd.(string))
        if err != nil {
            m.logger.Warn(ctx, "Failed to decrypt cached password, regenerating", "error", err.Error())
            // Fall through to generate new password
        } else {
            return &Credentials{
                Username: authUser.Username,
                Password: password,
            }, nil
        }
    }
    
    // 2. Generate new password
    password, err := m.pwdGen.Generate()
    if err != nil {
        return nil, fmt.Errorf("password generation failed: %w", err)
    }
    
    // 3. Map groups to roles
    roles, err := m.roleMapper.MapGroupsToRoles(ctx, authUser.Groups)
    if err != nil {
        return nil, fmt.Errorf("role mapping failed: %w", err)
    }
    
    // 4. Create/update ES user
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
        return nil, fmt.Errorf("ES user upsert failed: %w", err)
    }
    
    // 5. Encrypt and cache credentials
    encryptedPwd, err := m.encryptor.Encrypt(password)
    if err != nil {
        m.logger.Warn(ctx, "Failed to encrypt password for cache", "error", err.Error())
        // Don't fail - credentials are still valid, just not cached
    } else {
        if err := m.cache.Set(cacheKey, encryptedPwd, m.cacheTTL); err != nil {
            m.logger.Warn(ctx, "Failed to cache credentials", "error", err.Error())
            // Don't fail - credentials are still valid
        }
    }
    
    return &Credentials{
        Username: authUser.Username,
        Password: password,
    }, nil
}
```

### 3. Role Mapper (`internal/usermgmt/rolemapper.go`)

**Purpose**: Map user groups to Elasticsearch roles

**Interface**:
```go
type RoleMapper struct {
    config *config.Config
    logger *loggergo.Logger
}

func (rm *RoleMapper) MapGroupsToRoles(ctx context.Context, groups []string) ([]string, error)
```

**Implementation Logic**:
```go
func (rm *RoleMapper) MapGroupsToRoles(ctx context.Context, groups []string) ([]string, error) {
    rolesSet := make(map[string]bool)
    matched := false
    
    // Evaluate ALL role mappings
    for _, mapping := range rm.config.RoleMappings {
        for _, group := range groups {
            if rm.matchPattern(group, mapping.Pattern) {
                matched = true
                for _, role := range mapping.ESRoles {
                    rolesSet[role] = true
                }
                rm.logger.Debug(ctx, "Role mapping matched",
                    "group", group,
                    "pattern", mapping.Pattern,
                    "roles", mapping.ESRoles,
                )
            }
        }
    }
    
    // If no mappings matched, use default roles
    if !matched {
        if len(rm.config.DefaultESRoles) == 0 {
            return nil, fmt.Errorf("no role mappings matched and no default roles configured")
        }
        
        rm.logger.Info(ctx, "No role mappings matched, using default roles",
            "default_roles", rm.config.DefaultESRoles,
        )
        
        for _, role := range rm.config.DefaultESRoles {
            rolesSet[role] = true
        }
    }
    
    // Convert set to slice
    roles := make([]string, 0, len(rolesSet))
    for role := range rolesSet {
        roles = append(roles, role)
    }
    
    return roles, nil
}

func (rm *RoleMapper) matchPattern(value, pattern string) bool {
    // Exact match
    if value == pattern {
        return true
    }
    
    // Wildcard matching (reuse existing logic from mapper.matchPattern)
    // ...
}
```

### 4. Password Generator (`internal/usermgmt/password.go`)

**Purpose**: Generate cryptographically secure random passwords

**Interface**:
```go
type PasswordGenerator struct {
    length int
}

func NewPasswordGenerator(length int) *PasswordGenerator

func (pg *PasswordGenerator) Generate() (string, error)
```

**Implementation**:
```go
const (
    defaultPasswordLength = 32
    charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()-_=+[]{}|;:,.<>?"
)

func (pg *PasswordGenerator) Generate() (string, error) {
    password := make([]byte, pg.length)
    
    for i := range password {
        // Use crypto/rand for cryptographically secure randomness
        randomIndex, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
        if err != nil {
            return "", fmt.Errorf("failed to generate random number: %w", err)
        }
        password[i] = charset[randomIndex.Int64()]
    }
    
    return string(password), nil
}
```

**Security Considerations**:
- Uses `crypto/rand` (not `math/rand`)
- Minimum length: 32 characters
- Includes uppercase, lowercase, digits, special characters
- Passwords are never logged
- Passwords are encrypted before storing in cache

### 5. Credential Encryptor (`internal/usermgmt/encryptor.go`)

**Purpose**: Encrypt and decrypt passwords for cache storage using AES-256-GCM

**Interface**:
```go
type Encryptor interface {
    Encrypt(plaintext string) (string, error)
    Decrypt(ciphertext string) (string, error)
}

type encryptor struct {
    key []byte // 32 bytes for AES-256
}

func NewEncryptor(key []byte) (Encryptor, error)
```

**Implementation**:
```go
func NewEncryptor(key []byte) (Encryptor, error) {
    if len(key) != 32 {
        return nil, fmt.Errorf("encryption key must be 32 bytes, got %d", len(key))
    }
    
    return &encryptor{key: key}, nil
}

func (e *encryptor) Encrypt(plaintext string) (string, error) {
    // Create AES cipher
    block, err := aes.NewCipher(e.key)
    if err != nil {
        return "", fmt.Errorf("failed to create cipher: %w", err)
    }
    
    // Create GCM mode
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", fmt.Errorf("failed to create GCM: %w", err)
    }
    
    // Generate random nonce
    nonce := make([]byte, gcm.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return "", fmt.Errorf("failed to generate nonce: %w", err)
    }
    
    // Encrypt and prepend nonce
    ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
    
    // Encode as base64 for storage
    return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (e *encryptor) Decrypt(ciphertext string) (string, error) {
    // Decode from base64
    data, err := base64.StdEncoding.DecodeString(ciphertext)
    if err != nil {
        return "", fmt.Errorf("failed to decode ciphertext: %w", err)
    }
    
    // Create AES cipher
    block, err := aes.NewCipher(e.key)
    if err != nil {
        return "", fmt.Errorf("failed to create cipher: %w", err)
    }
    
    // Create GCM mode
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", fmt.Errorf("failed to create GCM: %w", err)
    }
    
    // Extract nonce
    nonceSize := gcm.NonceSize()
    if len(data) < nonceSize {
        return "", fmt.Errorf("ciphertext too short")
    }
    
    nonce, ciphertext := data[:nonceSize], data[nonceSize:]
    
    // Decrypt
    plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
    if err != nil {
        return "", fmt.Errorf("failed to decrypt: %w", err)
    }
    
    return string(plaintext), nil
}
```

**Security Considerations**:
- Uses AES-256-GCM (authenticated encryption)
- Random nonce for each encryption (prevents pattern analysis)
- Nonce prepended to ciphertext (standard practice)
- Base64 encoding for cache storage
- Key must be 32 bytes (256 bits)
- Same key required across all Keyline instances

## Configuration Changes

### New Config Structures

```go
// Add to internal/config/config.go

type Config struct {
    // ... existing fields ...
    RoleMappings   []RoleMapping `mapstructure:"role_mappings"`
    DefaultESRoles []string      `mapstructure:"default_es_roles"`
    UserManagement UserMgmtConfig `mapstructure:"user_management"`
    Cache          CacheConfig    `mapstructure:"cache"`
}

type RoleMapping struct {
    Claim    string   `mapstructure:"claim"`
    Pattern  string   `mapstructure:"pattern"`
    ESRoles  []string `mapstructure:"es_roles"`
}

type UserMgmtConfig struct {
    Enabled         bool          `mapstructure:"enabled"`
    PasswordLength  int           `mapstructure:"password_length"`
    CredentialTTL   time.Duration `mapstructure:"credential_ttl"`
}

type CacheConfig struct {
    Backend        string        `mapstructure:"backend"`         // "redis" or "memory"
    RedisURL       string        `mapstructure:"redis_url"`
    RedisPassword  string        `mapstructure:"redis_password"`
    RedisDB        int           `mapstructure:"redis_db"`
    CredentialTTL  time.Duration `mapstructure:"credential_ttl"`
    EncryptionKey  string        `mapstructure:"encryption_key"`  // 32 bytes for AES-256
}

type ElasticsearchConfig struct {
    // ... existing fields ...
    AdminUser     string        `mapstructure:"admin_user"`
    AdminPassword string        `mapstructure:"admin_password"`
    URL           string        `mapstructure:"url"`
    Timeout       time.Duration `mapstructure:"timeout"`
    InsecureSkipVerify bool     `mapstructure:"insecure_skip_verify"`
}

type LocalUser struct {
    Username       string   `mapstructure:"username"`
    PasswordBcrypt string   `mapstructure:"password_bcrypt"`
    Groups         []string `mapstructure:"groups"`
    Email          string   `mapstructure:"email"`
    FullName       string   `mapstructure:"full_name"`
    // Remove ESUser field - no longer needed
}
```

## Integration Points

### 1. Auth Engine Integration

**Location**: `internal/auth/engine.go`

**Changes**:
```go
type Engine struct {
    // ... existing fields ...
    userManager usermgmt.Manager
}

func (e *Engine) Authenticate(ctx context.Context, req *http.Request) (*AuthResult, error) {
    // ... existing authentication logic ...
    
    // After successful authentication, upsert ES user
    authUser := &usermgmt.AuthenticatedUser{
        Username: result.Username,
        Groups:   result.Groups,
        Email:    result.Email,
        FullName: result.FullName,
        Source:   result.Source,
    }
    
    creds, err := e.userManager.UpsertUser(ctx, authUser)
    if err != nil {
        return nil, fmt.Errorf("user management failed: %w", err)
    }
    
    // Use generated credentials for ES Authorization header
    result.ESUser = creds.Username
    result.ESPassword = creds.Password
    
    return result, nil
}
```

### 2. OIDC Provider Integration

**Location**: `internal/auth/oidc.go`

**Changes**:
```go
// Extract groups from OIDC claims
func (p *OIDCProvider) extractGroups(claims map[string]interface{}) []string {
    groups := []string{}
    
    // Try "groups" claim
    if groupsClaim, ok := claims["groups"]; ok {
        switch v := groupsClaim.(type) {
        case []interface{}:
            for _, g := range v {
                if str, ok := g.(string); ok {
                    groups = append(groups, str)
                }
            }
        case []string:
            groups = v
        case string:
            groups = []string{v}
        }
    }
    
    return groups
}
```

### 3. Basic Auth Integration

**Location**: `internal/auth/basic.go`

**Changes**:
```go
// Return groups from local user config
func (p *BasicAuthProvider) Authenticate(ctx context.Context, req *AuthRequest) (*AuthResult, error) {
    // ... existing authentication logic ...
    
    // Find user in config
    for _, user := range p.config.LocalUsers.Users {
        if user.Username == username {
            // ... password validation ...
            
            return &AuthResult{
                Username: user.Username,
                Groups:   user.Groups,  // NEW: Include groups
                Email:    user.Email,
                FullName: user.FullName,
                Source:   "basic_auth",
            }, nil
        }
    }
    
    return nil, fmt.Errorf("invalid credentials")
}
```

## Data Flow

### Successful Authentication Flow

```
1. User → Keyline: HTTP request with auth credentials
2. Keyline → Auth Engine: Authenticate(req)
3. Auth Engine → OIDC/Basic Provider: Validate credentials
4. Provider → Auth Engine: AuthResult{username, groups, email, ...}
5. Auth Engine → User Manager: UpsertUser(authUser)
6. User Manager → Cache: Check for cached credentials
7. Cache → User Manager: Cache miss
8. User Manager → Password Generator: Generate()
9. Password Generator → User Manager: Random password
10. User Manager → Role Mapper: MapGroupsToRoles(groups)
11. Role Mapper → User Manager: ES roles
12. User Manager → ES API Client: CreateOrUpdateUser(req)
13. ES API Client → Elasticsearch: PUT /_security/user/{username}
14. Elasticsearch → ES API Client: 200 OK
15. ES API Client → User Manager: Success
16. User Manager → Cache: Set(username, password, TTL)
17. User Manager → Auth Engine: Credentials{username, password}
18. Auth Engine → Transport: Forward with Authorization header
19. Transport → Elasticsearch: Proxied request with auth
20. Elasticsearch → User: Response
```

### Cached Credentials Flow

```
1-5. Same as above
6. User Manager → Cache: Check for cached credentials
7. Cache → User Manager: Cache hit (password)
8. User Manager → Auth Engine: Credentials{username, password}
9-11. Same as steps 18-20 above
```

## Error Handling Strategy

### ES API Errors

| Error | Handling |
|-------|----------|
| 401/403 (Invalid admin creds) | Log error, fail startup validation |
| 404 (User not found on GET) | Return nil (user doesn't exist) |
| 429 (Rate limited) | Retry with exponential backoff |
| 5xx (ES unavailable) | Retry with backoff, circuit breaker |
| Network timeout | Retry with backoff |

### Cache Errors

| Error | Handling |
|-------|----------|
| Cache unavailable | Log warning, continue (don't fail request) |
| Cache write failure | Log warning, continue (credentials still valid) |
| Cache read failure | Treat as cache miss, generate new credentials |

### Role Mapping Errors

| Error | Handling |
|-------|----------|
| No mappings matched, no default roles | Deny access, return 403 |
| Invalid pattern syntax | Log error at startup, skip mapping |
| Empty roles list | Deny access, return 403 |

## Performance Considerations

### Caching Strategy

- **Cache key**: `keyline:user:{username}:password`
- **Cache TTL**: Configurable (default: 1 hour)
- **Cache hit rate target**: > 95% for active users
- **Cache backend**: Redis (distributed) or Memory (single-node)

### Optimization Techniques

1. **Lazy user creation**: Only create ES user on first request
2. **Batch operations**: Not applicable (one user per request)
3. **Connection pooling**: HTTP client with connection pool
4. **Circuit breaker**: Prevent cascading failures when ES is down
5. **Async logging**: Don't block on log writes

### Performance Targets

- User upsert (cache miss): < 500ms (p95)
- User upsert (cache hit): < 10ms (p95)
- Cache hit rate: > 95%
- ES API call timeout: 30s
- Retry attempts: 3 with exponential backoff

## Security Considerations

### Password Security

- Generated using `crypto/rand` (cryptographically secure)
- Minimum length: 32 characters
- Character set includes uppercase, lowercase, digits, special chars
- Passwords never logged or exposed in errors
- **Passwords encrypted in cache using AES-256-GCM**
- **Encryption key must be 32 bytes (256 bits)**
- **Same encryption key required across all Keyline instances**
- Passwords only stored in encrypted form in cache

### Admin Credentials

- Stored in config (environment variables recommended)
- Never logged or exposed
- Validated on startup
- Requires `manage_security` privilege in ES

### TLS/SSL

- ES API calls use HTTPS in production
- `insecure_skip_verify` only for development/testing
- Certificate validation enabled by default

### Audit Trail

- All user creation/update operations logged
- ES audit logs show actual usernames (not shared accounts)
- Metadata includes source, groups, last_auth timestamp

## Testing Strategy

### Unit Tests

1. **Password Generator**:
   - Test password length
   - Test character set inclusion
   - Test randomness (no duplicates in 1000 generations)

2. **Role Mapper**:
   - Test exact match
   - Test wildcard patterns
   - Test multiple group matches
   - Test default roles fallback
   - Test no match, no default (error)

3. **ES API Client** (mocked):
   - Test successful user creation
   - Test user update
   - Test error handling (401, 404, 5xx)
   - Test retry logic

4. **User Manager**:
   - Test cache hit path
   - Test cache miss path
   - Test role mapping integration
   - Test ES API integration

### Integration Tests

1. **End-to-end OIDC flow**:
   - Authenticate with OIDC
   - Verify ES user created
   - Verify roles assigned correctly
   - Verify credentials cached

2. **End-to-end Basic Auth flow**:
   - Authenticate with Basic Auth
   - Verify ES user created with groups
   - Verify roles assigned correctly

3. **Cache expiration**:
   - Authenticate user
   - Wait for cache TTL
   - Authenticate again
   - Verify new password generated

4. **ES unavailability**:
   - Stop ES
   - Attempt authentication
   - Verify graceful error handling
   - Verify circuit breaker activates

### Property-Based Tests

1. **Password generation**:
   - Property: All generated passwords are valid (length, charset)
   - Property: No two passwords are identical in 10,000 generations

2. **Role mapping**:
   - Property: Mapping is deterministic (same groups → same roles)
   - Property: Multiple groups accumulate roles (no duplicates)

3. **Cache operations**:
   - Property: Set then Get returns same value
   - Property: Expired entries return cache miss

## Monitoring and Observability

### Logging with loggergo

**Uses**: `github.com/wasilak/loggergo` for structured logging

```go
// Initialize logger (already done in main.go)
logger := loggergo.LoggerInit(loggergo.LoggerConfig{
    LogLevel:  config.Observability.LogLevel,
    LogFormat: config.Observability.LogFormat,
})

// Structured logging in components
logger.Info(ctx, "ES user created",
    "username", username,
    "roles", roles,
    "source", source,
    "duration", duration,
)

logger.Warn(ctx, "ES API call failed, retrying",
    "error", err.Error(),
    "attempt", attempt,
    "backoff", backoff,
)

logger.Error(ctx, "User upsert failed",
    "username", username,
    "error", err.Error(),
)
```

### Tracing with otelgo

**Uses**: `github.com/wasilak/otelgo` for OpenTelemetry tracing

```go
// Initialize OpenTelemetry (already done in main.go)
otelShutdown, err := otelgo.InitOpenTelemetry(ctx, otelgo.OtelConfig{
    Enabled:        config.Observability.OTelEnabled,
    Endpoint:       config.Observability.OTelEndpoint,
    ServiceName:    config.Observability.OTelServiceName,
    ServiceVersion: config.Observability.OTelServiceVersion,
    Environment:    config.Observability.OTelEnvironment,
    TraceRatio:     config.Observability.OTelTraceRatio,
})

// Add tracing to ES API calls
func (c *client) CreateOrUpdateUser(ctx context.Context, req *UserRequest) error {
    ctx, span := otel.Tracer("keyline").Start(ctx, "elasticsearch.create_user")
    defer span.End()
    
    span.SetAttributes(
        attribute.String("es.username", req.Username),
        attribute.StringSlice("es.roles", req.Roles),
    )
    
    // ... ES API call ...
    
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return err
    }
    
    return nil
}
```

### Metrics

```go
// Prometheus metrics (already integrated)
var (
    userUpsertsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "keyline_user_upserts_total",
            Help: "Total number of ES user upserts",
        },
        []string{"status"}, // "success", "failure"
    )
    
    userUpsertDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "keyline_user_upsert_duration_seconds",
            Help: "Duration of ES user upsert operations",
            Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1.0, 5.0},
        },
        []string{"cache_status"}, // "hit", "miss"
    )
    
    credCacheHits = prometheus.NewCounter(
        prometheus.CounterOpts{
            Name: "keyline_cred_cache_hits_total",
            Help: "Total number of credential cache hits",
        },
    )
    
    credCacheMisses = prometheus.NewCounter(
        prometheus.CounterOpts{
            Name: "keyline_cred_cache_misses_total",
            Help: "Total number of credential cache misses",
        },
    )
    
    roleMappingMatches = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "keyline_role_mapping_matches_total",
            Help: "Total number of role mapping matches",
        },
        []string{"pattern"},
    )
)
```

## Migration Strategy

### Phase 1: Add User Management (Opt-in)

- Add `user_management.enabled` config flag (default: false)
- Implement all components
- Add comprehensive tests
- Deploy with feature disabled

### Phase 2: Testing and Validation

- Enable in development/staging environments
- Validate ES user creation
- Validate role mappings
- Monitor performance and cache hit rates

### Phase 3: Production Rollout

- Enable for subset of users (canary)
- Monitor metrics and logs
- Gradually increase rollout percentage
- Full rollout once validated

### Phase 4: Deprecate Static Mapping

- Update documentation
- Provide migration guide
- Eventually remove `elasticsearch.users` config (breaking change)

## Open Questions and Future Enhancements

### Open Questions

1. Should we support custom password policies (length, complexity)?
2. Should we implement user cleanup (delete inactive users)?
3. Should we support ES role creation (not just user creation)?
4. Should we cache role mappings separately from credentials?

### Future Enhancements

1. **User lifecycle management**: Automatic cleanup of inactive users
2. **Role synchronization**: Detect and update role changes
3. **Multi-cluster support**: Manage users across multiple ES clusters
4. **Audit logging**: Dedicated audit log for user management operations
5. **Admin API**: REST API for manual user management
6. **Metrics dashboard**: Grafana dashboard for user management metrics

## Conclusion

This design provides a comprehensive solution for dynamic Elasticsearch user management in Keyline. It maintains accountability through individual user accounts, provides flexible role-based access control, and ensures high performance through intelligent caching. The implementation follows Go best practices and integrates seamlessly with existing authentication flows.
