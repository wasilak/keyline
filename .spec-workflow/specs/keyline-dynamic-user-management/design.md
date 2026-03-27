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
    CreateOrUpdateUser(ctx context.Context, req *UserRequest) error
    GetUser(ctx context.Context, username string) (*User, error)
    DeleteUser(ctx context.Context, username string) error
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

**Error Handling**:
| Error | Handling |
|-------|----------|
| 401/403 | Invalid admin credentials → log error, fail startup |
| 404 | User not found → return nil |
| 429 | Rate limited → retry with backoff |
| 5xx | ES unavailable → retry with backoff, circuit breaker |
| Network errors | Retry with backoff |

### 2. User Manager (`internal/usermgmt/manager.go`)

**Purpose**: Orchestrate user creation, role mapping, and credential caching

**Uses**: `github.com/wasilak/cachego` for credential caching

**Interface**:
```go
type Manager interface {
    UpsertUser(ctx context.Context, authUser *AuthenticatedUser) (*Credentials, error)
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

**Key Operations**:
1. Check cache for existing credentials
2. If cache hit: decrypt and return
3. If cache miss: generate password, map roles, create ES user, cache
4. Encrypt passwords before caching (AES-256-GCM)
5. Return credentials for Authorization header

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

**Mapping Logic**:
1. Evaluate ALL role mappings in order
2. For each group, check if it matches any pattern
3. Collect ALL matching ES roles (deduplicate)
4. If at least one match: use collected roles
5. If no matches AND default_es_roles defined: use defaults
6. If no matches AND no defaults: return error (deny access)

**Pattern Matching**:
- Exact match: `admin` == `admin`
- Wildcard prefix: `admin@*` matches `admin@example.com`
- Wildcard suffix: `*@example.com` matches `user@example.com`
- Wildcard middle: `admin@*.com` matches `admin@example.com`

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

**Security**:
- Uses `crypto/rand` (not `math/rand`)
- Minimum length: 32 characters
- Character set: uppercase, lowercase, digits, special characters
- Passwords never logged

### 5. Credential Encryptor (`internal/usermgmt/encryptor.go`)

**Purpose**: Encrypt and decrypt passwords for cache storage

**Interface**:
```go
type Encryptor interface {
    Encrypt(plaintext string) (string, error)
    Decrypt(ciphertext string) (string, error)
}
```

**Security**:
- AES-256-GCM (authenticated encryption)
- Random nonce for each encryption
- Key must be 32 bytes (256 bits)
- Base64 encoding for cache storage
- Same key required across all Keyline instances

## Configuration Changes

### New Config Structures

```go
type Config struct {
    RoleMappings   []RoleMapping
    DefaultESRoles []string
    UserManagement UserMgmtConfig
    Cache          CacheConfig
}

type RoleMapping struct {
    Claim    string
    Pattern  string
    ESRoles  []string
}

type CacheConfig struct {
    Backend       string        // "redis" or "memory"
    RedisURL      string
    RedisPassword string
    RedisDB       int
    CredentialTTL time.Duration
    EncryptionKey string        // 32 bytes for AES-256
}

type LocalUser struct {
    Username       string
    PasswordBcrypt string
    Groups         []string
    Email          string
    FullName       string
}
```

## Integration Points

### 1. Auth Engine Integration

**Location**: `internal/auth/engine.go`

**Changes**:
- Add `userManager usermgmt.Manager` field
- After authentication, call `UpsertUser()` to get ES credentials
- Use generated credentials for Authorization header

### 2. OIDC Provider Integration

**Location**: `internal/auth/oidc.go`

**Changes**:
- Extract groups from OIDC claims (`groups` claim)
- Handle multiple formats: `[]interface{}`, `[]string`, `string`
- Return groups in AuthResult

### 3. Basic Auth Integration

**Location**: `internal/auth/basic.go`

**Changes**:
- Return groups from local user config
- Include email, full_name in AuthResult

## Data Flow

### Successful Authentication Flow

```
User → Keyline: HTTP request with auth credentials
  ↓
Auth Engine: Authenticate(req)
  ↓
Provider: Validate credentials → AuthResult{username, groups, email, ...}
  ↓
User Manager: UpsertUser(authUser)
  ↓
Cache: Check for cached credentials (miss)
  ↓
Password Generator: Generate()
  ↓
Role Mapper: MapGroupsToRoles(groups)
  ↓
ES API Client: CreateOrUpdateUser(req)
  ↓
Elasticsearch: PUT /_security/user/{username} → 200 OK
  ↓
Cache: Set(username, encrypted_password, TTL)
  ↓
Return Credentials{username, password}
  ↓
Transport: Forward with Authorization header
  ↓
Elasticsearch: Response
```

### Cached Credentials Flow

```
User → Keyline: HTTP request
  ↓
Auth Engine: Authenticate(req)
  ↓
User Manager: UpsertUser(authUser)
  ↓
Cache: Check for cached credentials (hit)
  ↓
Decrypt password
  ↓
Return Credentials{username, password}
  ↓
Transport: Forward with Authorization header
```

## Error Handling Strategy

### ES API Errors
| Error | Handling |
|-------|----------|
| 401/403 | Log error, fail startup validation |
| 404 | Return nil (user doesn't exist) |
| 429 | Retry with exponential backoff |
| 5xx | Retry with backoff, circuit breaker |
| Network timeout | Retry with backoff |

### Cache Errors
| Error | Handling |
|-------|----------|
| Cache unavailable | Log warning, continue |
| Cache write failure | Log warning, continue |
| Cache read failure | Treat as cache miss |

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

### Performance Targets
| Metric | Target |
|--------|--------|
| User upsert (cache miss) | < 500ms (p95) |
| User upsert (cache hit) | < 10ms (p95) |
| Cache hit rate | > 95% |
| ES API call timeout | 30s |
| Retry attempts | 3 with exponential backoff |

## Security Considerations

### Password Security
- Generated using `crypto/rand`
- Minimum length: 32 characters
- Character set includes uppercase, lowercase, digits, special chars
- Passwords never logged or exposed
- **Passwords encrypted in cache using AES-256-GCM**
- Encryption key must be 32 bytes (256 bits)
- Same encryption key across all instances

### Admin Credentials
- Stored in config (environment variables recommended)
- Never logged or exposed
- Validated on startup
- Requires `manage_security` privilege in ES

### TLS/SSL
- ES API calls use HTTPS in production
- `insecure_skip_verify` only for development/testing
- Certificate validation enabled by default

## Testing Strategy

### Unit Tests
1. **Password Generator**: length, charset, randomness
2. **Role Mapper**: pattern matching, multiple groups, defaults
3. **ES API Client**: mocked responses, error handling, retry
4. **User Manager**: cache hit/miss, encryption, error paths

### Integration Tests
1. End-to-end OIDC flow with user creation
2. End-to-end Basic Auth with groups
3. Cache expiration and refresh
4. ES unavailability handling

### Property-Based Tests
1. Password generation: always valid, no duplicates
2. Role mapping: deterministic, accumulative
3. Cache operations: idempotent

## Monitoring and Observability

### Logging with loggergo

```go
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
```

### Tracing with otelgo

- User upsert span
- ES API call spans
- Role mapping span
- Cache operation spans

### Prometheus Metrics

| Metric | Type | Labels |
|--------|------|--------|
| `keyline_user_upserts_total` | Counter | status |
| `keyline_user_upsert_duration_seconds` | Histogram | cache_status |
| `keyline_cred_cache_hits_total` | Counter | - |
| `keyline_cred_cache_misses_total` | Counter | - |
| `keyline_role_mapping_matches_total` | Counter | pattern |
| `keyline_es_api_calls_total` | Counter | operation, status |

## Out of Scope

- Automatic ES role creation (roles must exist in ES)
- User deletion/cleanup (manual process)
- Password rotation policies (handled by cache TTL)
- Multi-cluster support (single ES cluster only)
- Custom password policies (fixed secure policy)
