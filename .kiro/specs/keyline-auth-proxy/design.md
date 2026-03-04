# Design Document: Keyline Authentication Proxy

## Overview

Keyline is a unified authentication proxy service that replaces the existing Authelia + elastauth stack. It provides dual authentication modes (OIDC and Basic Auth) simultaneously, supports three deployment modes (forwardAuth, auth_request, standalone proxy), and automatically injects Elasticsearch credentials into authenticated requests.

### Design Goals

- **Unified Service**: Single binary replacing two-service architecture
- **Dual Authentication**: Support both interactive (OIDC) and programmatic (Basic Auth) access simultaneously
- **Deployment Flexibility**: Work with Traefik, Nginx, or as standalone proxy
- **Security First**: Implement PKCE, secure session management, and cryptographic best practices
- **Production Ready**: Built-in observability, health checks, and graceful shutdown
- **Full Observability**: OpenTelemetry tracing and structured logging from day one
- **Unified Caching**: Single cache interface for sessions, state tokens, and OIDC data

### Technology Stack

- **Language**: Go 1.22+
- **Web Framework**: Echo v4
- **Configuration**: Viper
- **Cache Layer**: cachego (unified interface for Redis/in-memory)
- **Logging**: loggergo (global slog setup)
- **Echo Logging**: slog-echo (request logging middleware)
- **Tracing**: otelgo (OpenTelemetry setup)
- **Echo Tracing**: otelecho (request tracing middleware)
- **OIDC**: coreos/go-oidc v3 + golang.org/x/oauth2
- **Proxy**: net/http/httputil.ReverseProxy
- **Crypto**: crypto/rand, bcrypt


## Architecture

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         Keyline Service                          │
├─────────────────────────────────────────────────────────────────┤
│  ┌───────────────────────────────────────────────────────────┐  │
│  │              Observability Layer                           │  │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐    │  │
│  │  │   otelecho   │  │  slog-echo   │  │   loggergo   │    │  │
│  │  │ (tracing MW) │  │ (logging MW) │  │ (global slog)│    │  │
│  │  └──────────────┘  └──────────────┘  └──────────────┘    │  │
│  └───────────────────────────────────────────────────────────┘  │
│                                                                   │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │              Transport Adapter Layer                       │  │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐    │  │
│  │  │ ForwardAuth  │  │ Auth_Request │  │  Standalone  │    │  │
│  │  │   Adapter    │  │   Adapter    │  │    Proxy     │    │  │
│  │  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘    │  │
│  └─────────┼──────────────────┼──────────────────┼───────────┘  │
│            │                  │                  │               │
│  ┌─────────┴──────────────────┴──────────────────┴───────────┐  │
│  │              Core Authentication Engine                    │  │
│  │                                                             │  │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐    │  │
│  │  │     OIDC     │  │  Basic Auth  │  │   Session    │    │  │
│  │  │   Handler    │  │  Validator   │  │   Manager    │    │  │
│  │  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘    │  │
│  │         │                  │                  │             │  │
│  │  ┌──────┴──────────────────┴──────────────────┴─────────┐ │  │
│  │  │         ES Credential Mapper & Injector              │ │  │
│  │  └──────────────────────────────────────────────────────┘ │  │
│  └─────────────────────────────────────────────────────────────┘  │
│                                                                   │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │                  Cache Layer (cachego)                     │  │
│  │  ┌──────────────────────────────────────────────────────┐ │  │
│  │  │  Unified Cache Interface (Redis or Memory backend)   │ │  │
│  │  │  - Sessions (session:{id})                           │ │  │
│  │  │  - State Tokens (state:{id})                         │ │  │
│  │  │  - OIDC Discovery (oidc:discovery:{issuer})          │ │  │
│  │  │  - JWKS (oidc:jwks:{issuer})                         │ │  │
│  │  └──────────────────────────────────────────────────────┘ │  │
│  └───────────────────────────────────────────────────────────┘  │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘
         │                      │                      │
         ▼                      ▼                      ▼
   OIDC Provider          Cache Backend          Protected Service
   (Okta, etc.)          (Redis/Memory)         (Kibana, ES, etc.)
```

### Component Responsibilities

#### Observability Layer
- **otelecho**: Automatic OpenTelemetry tracing for all HTTP requests
- **slog-echo**: Automatic structured logging for all HTTP requests with trace correlation
- **loggergo**: Global slog configuration (JSON/text format, log levels)

#### Transport Adapter Layer
- **ForwardAuth Adapter**: Handles Traefik X-Forwarded-* headers, returns auth decisions
- **Auth_Request Adapter**: Handles Nginx X-Original-* headers, returns auth decisions
- **Standalone Proxy**: Proxies authenticated requests to upstream, handles WebSocket upgrades

#### Core Authentication Engine
- **OIDC Handler**: Manages authorization flow, token exchange, ID token validation (with manual spans)
- **Basic Auth Validator**: Validates local user credentials using bcrypt
- **Session Manager**: Creates, validates, extends, and deletes user sessions (with manual spans)
- **ES Credential Mapper**: Maps authenticated users to Elasticsearch credentials

#### Cache Layer (cachego)
- **Unified Interface**: Single cache interface for all storage needs
- **Sessions**: Stores user sessions with TTL (key: `session:{id}`)
- **State Tokens**: Stores OIDC CSRF tokens with 5-minute TTL (key: `state:{id}`)
- **OIDC Discovery**: Caches discovery documents (key: `oidc:discovery:{issuer}`)
- **JWKS**: Caches JSON Web Key Sets (key: `oidc:jwks:{issuer}`)
- **Backend Agnostic**: Supports Redis or in-memory backends via configuration


### Deployment Mode Flows

#### ForwardAuth Mode (Traefik)

```
┌─────────┐      ┌─────────┐      ┌─────────┐      ┌──────────┐
│ Browser │─────▶│ Traefik │─────▶│ Keyline │      │ Kibana   │
└─────────┘      └─────────┘      └─────────┘      └──────────┘
                      │                 │                │
                      │  1. Forward     │                │
                      │     Auth Check  │                │
                      │────────────────▶│                │
                      │                 │                │
                      │  2. 200 + Header│                │
                      │◀────────────────│                │
                      │                 │                │
                      │  3. Proxy with  │                │
                      │     ES Auth     │                │
                      │─────────────────┼───────────────▶│
                      │                 │                │
                      │  4. Response    │                │
                      │◀────────────────┼────────────────│
```

#### Standalone Mode

```
┌─────────┐      ┌─────────┐      ┌──────────┐
│ Browser │─────▶│ Keyline │─────▶│ Kibana   │
└─────────┘      └─────────┘      └──────────┘
                      │                 │
                      │  1. Request     │
                      │                 │
                      │  2. Auth Check  │
                      │     (internal)  │
                      │                 │
                      │  3. Proxy with  │
                      │     ES Auth     │
                      │────────────────▶│
                      │                 │
                      │  4. Response    │
                      │◀────────────────│
```

### OIDC Authorization Flow

```
┌─────────┐   ┌─────────┐   ┌──────────┐   ┌──────────┐
│ Browser │   │ Keyline │   │   OIDC   │   │ Session  │
│         │   │         │   │ Provider │   │  Store   │
└────┬────┘   └────┬────┘   └────┬─────┘   └────┬─────┘
     │             │              │              │
     │ 1. Request  │              │              │
     │────────────▶│              │              │
     │             │              │              │
     │             │ 2. Generate  │              │
     │             │    state +   │              │
     │             │    PKCE      │              │
     │             │              │              │
     │             │ 3. Store     │              │
     │             │    state     │              │
     │             │──────────────┼─────────────▶│
     │             │              │              │
     │ 4. Redirect │              │              │
     │    to OIDC  │              │              │
     │◀────────────│              │              │
     │             │              │              │
     │ 5. Auth     │              │              │
     │────────────────────────────▶│              │
     │             │              │              │
     │ 6. Callback │              │              │
     │    with code│              │              │
     │────────────▶│              │              │
     │             │              │              │
     │             │ 7. Validate  │              │
     │             │    state     │              │
     │             │──────────────┼─────────────▶│
     │             │              │              │
     │             │ 8. Exchange  │              │
     │             │    code for  │              │
     │             │    tokens    │              │
     │             │─────────────▶│              │
     │             │              │              │
     │             │ 9. ID Token  │              │
     │             │◀─────────────│              │
     │             │              │              │
     │             │ 10. Validate │              │
     │             │     signature│              │
     │             │     & claims │              │
     │             │              │              │
     │             │ 11. Create   │              │
     │             │     session  │              │
     │             │──────────────┼─────────────▶│
     │             │              │              │
     │ 12. Redirect│              │              │
     │     with    │              │              │
     │     cookie  │              │              │
     │◀────────────│              │              │
```


## Components and Interfaces

### Core Interfaces

#### AuthProvider Interface

```go
// AuthProvider defines the interface for authentication methods
type AuthProvider interface {
    // Authenticate validates credentials and returns user info
    Authenticate(ctx context.Context, req *AuthRequest) (*AuthResult, error)
    
    // Name returns the provider type (oidc, basic)
    Name() string
}

// AuthRequest contains authentication request data
type AuthRequest struct {
    Method          string            // HTTP method
    Path            string            // Request path
    Headers         map[string]string // Request headers
    Cookies         []*http.Cookie    // Request cookies
    OriginalURL     string            // Original request URL
}

// AuthResult contains authentication result
type AuthResult struct {
    Authenticated   bool              // Whether auth succeeded
    User            *User             // Authenticated user info
    ESUser          string            // Mapped Elasticsearch user
    SessionID       string            // Session identifier (if created)
    RedirectURL     string            // Redirect URL (for OIDC flow)
    SetCookie       *http.Cookie      // Cookie to set (if any)
}

// User represents an authenticated user
type User struct {
    ID              string            // User identifier
    Username        string            // Username
    Email           string            // Email address
    Claims          map[string]any    // Additional claims
    AuthMethod      string            // Authentication method used
}
```

#### Cache Layer (cachego)

```go
// Keyline uses cachego for all caching needs (sessions, state tokens, OIDC data)
// The cache backend (Redis or memory) is configured at startup

// Session operations
func CreateSession(ctx context.Context, cache cachego.Cache, session *Session) error {
    ctx, span := tracer.Start(ctx, "session.create")
    defer span.End()
    
    data, _ := json.Marshal(session)
    key := fmt.Sprintf("session:%s", session.ID)
    ttl := time.Until(session.ExpiresAt)
    
    slog.InfoContext(ctx, "Creating session",
        slog.String("username", session.Username),
        slog.String("es_user", session.ESUser),
    )
    
    return cache.Set(ctx, key, data, ttl)
}

// State token operations
func StoreStateToken(ctx context.Context, cache cachego.Cache, token *StateToken) error {
    ctx, span := tracer.Start(ctx, "state.store")
    defer span.End()
    
    data, _ := json.Marshal(token)
    key := fmt.Sprintf("state:%s", token.ID)
    
    return cache.Set(ctx, key, data, 5*time.Minute)
}

// OIDC cache operations
func CacheDiscoveryDocument(ctx context.Context, cache cachego.Cache, issuer string, doc *DiscoveryDocument) error {
    data, _ := json.Marshal(doc)
    key := fmt.Sprintf("oidc:discovery:%s", issuer)
    
    return cache.Set(ctx, key, data, 24*time.Hour)
}

// Session represents a user session
type Session struct {
    ID              string                 `json:"id"`
    UserID          string                 `json:"user_id"`
    Username        string                 `json:"username"`
    Email           string                 `json:"email"`
    ESUser          string                 `json:"es_user"`
    Claims          map[string]interface{} `json:"claims"`
    CreatedAt       time.Time              `json:"created_at"`
    ExpiresAt       time.Time              `json:"expires_at"`
}

// StateToken represents an OIDC state token
type StateToken struct {
    ID           string    `json:"id"`
    OriginalURL  string    `json:"original_url"`
    CodeVerifier string    `json:"code_verifier"`
    CreatedAt    time.Time `json:"created_at"`
    Used         bool      `json:"used"`
}
```

#### TransportAdapter Interface

```go
// TransportAdapter defines the interface for deployment mode adapters
type TransportAdapter interface {
    // HandleRequest processes an incoming request
    HandleRequest(c echo.Context) error
    
    // Name returns the adapter name
    Name() string
}

// RequestContext contains normalized request information
type RequestContext struct {
    Method          string            // HTTP method
    Path            string            // Request path
    Host            string            // Request host
    Headers         map[string]string // Request headers
    Cookies         []*http.Cookie    // Request cookies
    OriginalURL     string            // Full original URL
}
```

### Component Implementations

#### OIDCProvider

```go
// OIDCProvider implements OIDC authentication
type OIDCProvider struct {
    config       *OIDCConfig
    provider     *oidc.Provider
    oauth2Config *oauth2.Config
    verifier     *oidc.IDTokenVerifier
    cache        cachego.Cache
    mapper       *CredentialMapper
}

// Key methods (all take context.Context as first parameter):
// - Authenticate: Initiates OIDC flow or handles callback (with manual span)
// - HandleCallback: Processes OIDC callback (with manual span)
// - exchangeToken: Exchanges authorization code for tokens (with manual span)
// - validateIDToken: Validates ID token signature and claims (with manual span)
// - generateState: Creates cryptographically secure state token
// - generatePKCE: Creates PKCE code verifier and challenge
//
// All methods use slog.InfoContext(ctx, ...) for logging
// All critical operations create manual spans for tracing
```

#### BasicAuthProvider

```go
// BasicAuthProvider implements Basic Auth for local users
type BasicAuthProvider struct {
    config *LocalUsersConfig
    mapper *CredentialMapper
}

// Key methods (all take context.Context as first parameter):
// - Authenticate: Validates Basic Auth credentials (with manual span)
// - validatePassword: Uses bcrypt timing-safe comparison
// - findUser: Looks up user by username
//
// All methods use slog.InfoContext(ctx, ...) for logging
```

#### SessionManager

```go
// SessionManager handles session lifecycle
type SessionManager struct {
    cache  cachego.Cache
    config *SessionConfig
}

// Key methods (all take context.Context as first parameter):
// - CreateSession: Generates session ID and stores in cache (with manual span)
// - ValidateSession: Retrieves and validates session (with manual span)
// - DeleteSession: Removes session from cache (with manual span)
// - ExtendSession: Updates session expiration (optional)
//
// All methods use slog.InfoContext(ctx, ...) for logging
// All operations create manual spans for tracing
```

#### CredentialMapper

```go
// CredentialMapper maps users to Elasticsearch credentials
type CredentialMapper struct {
    config          *Config
    logger          *slog.Logger
}

// Key methods:
// - MapOIDCUser: Maps OIDC claims to ES user
// - MapLocalUser: Maps local user to ES user
// - GetESCredentials: Retrieves ES credentials for user
// - matchPattern: Performs wildcard pattern matching
```


#### ForwardAuthAdapter

```go
// ForwardAuthAdapter handles Traefik/Nginx forwardAuth mode
type ForwardAuthAdapter struct {
    authEngine      *AuthEngine
    logger          *slog.Logger
}

// Key methods:
// - HandleRequest: Normalizes headers and delegates to auth engine
// - normalizeHeaders: Converts X-Forwarded-* or X-Original-* to RequestContext
// - buildResponse: Returns 200 with headers or 401/302 for auth
```

#### StandaloneProxyAdapter

```go
// StandaloneProxyAdapter handles standalone proxy mode
type StandaloneProxyAdapter struct {
    authEngine      *AuthEngine
    proxy           *httputil.ReverseProxy
    config          *UpstreamConfig
    logger          *slog.Logger
}

// Key methods:
// - HandleRequest: Authenticates then proxies request
// - proxyRequest: Forwards authenticated request to upstream
// - handleWebSocket: Handles WebSocket upgrade requests
// - injectESCredentials: Adds X-Es-Authorization header
```

#### RedisSessionStore

```go
// RedisSessionStore implements SessionStore using Redis
type RedisSessionStore struct {
    client          *redis.Client
    keyPrefix       string
    logger          *slog.Logger
}

// Key methods:
// - Create: Stores session as JSON with TTL
// - Get: Retrieves and deserializes session
// - Delete: Removes session key
// - Health: Pings Redis
```

#### InMemorySessionStore

```go
// InMemorySessionStore implements SessionStore using in-memory map
type InMemorySessionStore struct {
    sessions        map[string]*Session
    mu              sync.RWMutex
    logger          *slog.Logger
}

// Key methods:
// - Create: Stores session in map
// - Get: Retrieves session with expiration check
// - Delete: Removes session from map
// - Cleanup: Removes expired sessions (background goroutine)
```

#### OIDCCache

```go
// OIDCCache caches Discovery Document and JWKS
type OIDCCache struct {
    discoveryDoc    *DiscoveryDocument
    jwks            *jose.JSONWebKeySet
    jwksExpiry      time.Time
    mu              sync.RWMutex
    logger          *slog.Logger
}

// Key methods:
// - GetDiscoveryDoc: Returns cached discovery document
// - RefreshJWKS: Fetches and caches JWKS
// - GetJWKS: Returns cached JWKS
```


## Data Models

### Configuration Schema

```go
// Config represents the complete Keyline configuration
type Config struct {
    Server          ServerConfig          `mapstructure:"server"`
    OIDC            OIDCConfig            `mapstructure:"oidc"`
    LocalUsers      LocalUsersConfig      `mapstructure:"local_users"`
    Session         SessionConfig         `mapstructure:"session"`
    Cache           CacheConfig           `mapstructure:"cache"`
    Elasticsearch   ElasticsearchConfig   `mapstructure:"elasticsearch"`
    Upstream        UpstreamConfig        `mapstructure:"upstream"`
    Observability   ObservabilityConfig   `mapstructure:"observability"`
}

// ServerConfig contains server settings
type ServerConfig struct {
    Port            int                   `mapstructure:"port"`
    Mode            string                `mapstructure:"mode"` // forward_auth, standalone
    ReadTimeout     time.Duration         `mapstructure:"read_timeout"`
    WriteTimeout    time.Duration         `mapstructure:"write_timeout"`
    MaxConcurrent   int                   `mapstructure:"max_concurrent"`
}

// OIDCConfig contains OIDC provider settings
type OIDCConfig struct {
    Enabled         bool                  `mapstructure:"enabled"`
    IssuerURL       string                `mapstructure:"issuer_url"`
    ClientID        string                `mapstructure:"client_id"`
    ClientSecret    string                `mapstructure:"client_secret"`
    RedirectURL     string                `mapstructure:"redirect_url"`
    Scopes          []string              `mapstructure:"scopes"`
    Mappings        []OIDCMapping         `mapstructure:"mappings"`
    DefaultESUser   string                `mapstructure:"default_es_user"`
}

// OIDCMapping maps OIDC claims to ES users
type OIDCMapping struct {
    Claim           string                `mapstructure:"claim"`
    Pattern         string                `mapstructure:"pattern"`
    ESUser          string                `mapstructure:"es_user"`
}

// LocalUsersConfig contains local user settings
type LocalUsersConfig struct {
    Enabled         bool                  `mapstructure:"enabled"`
    Users           []LocalUser           `mapstructure:"users"`
}

// LocalUser represents a local user
type LocalUser struct {
    Username        string                `mapstructure:"username"`
    PasswordBcrypt  string                `mapstructure:"password_bcrypt"`
    ESUser          string                `mapstructure:"es_user"`
}

// SessionConfig contains session management settings
type SessionConfig struct {
    Store           string                `mapstructure:"store"` // redis, memory (for cachego backend)
    TTL             time.Duration         `mapstructure:"ttl"`
    CookieName      string                `mapstructure:"cookie_name"`
    CookieDomain    string                `mapstructure:"cookie_domain"`
    CookiePath      string                `mapstructure:"cookie_path"`
    SessionSecret   string                `mapstructure:"session_secret"`
}

// CacheConfig contains cache backend settings (for cachego)
type CacheConfig struct {
    Backend         string                `mapstructure:"backend"` // redis, memory
    RedisURL        string                `mapstructure:"redis_url"`
    RedisPassword   string                `mapstructure:"redis_password"`
    RedisDB         int                   `mapstructure:"redis_db"`
}

// ObservabilityConfig contains logging and tracing settings
type ObservabilityConfig struct {
    // loggergo settings
    LogLevel        string                `mapstructure:"log_level"`
    LogFormat       string                `mapstructure:"log_format"` // json, text
    
    // otelgo settings
    OTelEnabled     bool                  `mapstructure:"otel_enabled"`
    OTelEndpoint    string                `mapstructure:"otel_endpoint"`
    OTelServiceName string                `mapstructure:"otel_service_name"`
    OTelServiceVersion string             `mapstructure:"otel_service_version"`
    OTelEnvironment string                `mapstructure:"otel_environment"`
    OTelTraceRatio  float64               `mapstructure:"otel_trace_ratio"` // 0.0 to 1.0
}
```

### Example Configuration File

```yaml
server:
  port: 9000
  mode: forward_auth  # forward_auth or standalone
  read_timeout: 30s
  write_timeout: 30s
  max_concurrent: 1000

oidc:
  enabled: true
  issuer_url: ${OIDC_ISSUER_URL}
  client_id: ${OIDC_CLIENT_ID}
  client_secret: ${OIDC_CLIENT_SECRET}
  redirect_url: https://auth.example.com/auth/callback
  scopes:
    - openid
    - email
    - profile
  mappings:
    - claim: email
      pattern: "*@admin.example.com"
      es_user: admin
    - claim: email
      pattern: "*@example.com"
      es_user: readonly
  default_es_user: readonly

local_users:
  enabled: true
  users:
    - username: ci-pipeline
      password_bcrypt: ${CI_PASSWORD_BCRYPT}
      es_user: ci_user
    - username: monitoring
      password_bcrypt: ${MONITORING_PASSWORD_BCRYPT}
      es_user: monitoring_user

session:
  ttl: 24h
  cookie_name: keyline_session
  cookie_domain: .example.com
  cookie_path: /
  session_secret: ${SESSION_SECRET}  # Must be at least 32 bytes

cache:
  backend: redis  # redis or memory (for cachego)
  redis_url: ${REDIS_URL}
  redis_password: ${REDIS_PASSWORD}
  redis_db: 0

elasticsearch:
  users:
    - username: admin
      password: ${ES_ADMIN_PASSWORD}
    - username: readonly
      password: ${ES_READONLY_PASSWORD}
    - username: ci_user
      password: ${ES_CI_PASSWORD}
    - username: monitoring_user
      password: ${ES_MONITORING_PASSWORD}

upstream:
  url: http://kibana:5601
  timeout: 30s
  max_idle_conns: 100

observability:
  # loggergo settings
  log_level: info
  log_format: json
  
  # otelgo settings
  otel_enabled: true
  otel_endpoint: http://otel-collector:4318
  otel_service_name: keyline
  otel_service_version: ${VERSION}
  otel_environment: production
  otel_trace_ratio: 1.0  # 0.0 to 1.0 (1.0 = 100% sampling)
```


## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

### Acceptance Criteria Testing Prework

Before defining correctness properties, I analyzed each acceptance criterion to determine testability:

**Requirement 1: OIDC Auto-Discovery**
1.1. Discovery document fetch on startup
  Thoughts: This is a specific startup behavior that can be tested with a mock OIDC provider
  Testable: yes - example

1.2. Startup failure on discovery fetch error
  Thoughts: This is testing error handling for a specific case
  Testable: yes - example

1.3. Extraction of endpoints from discovery document
  Thoughts: For any valid discovery document, we should extract all required fields
  Testable: yes - property

1.4. Issuer validation
  Thoughts: For any discovery document where issuer doesn't match config, startup should fail
  Testable: yes - property

1.5. Discovery document caching
  Thoughts: After successful fetch, the document should be available in memory
  Testable: yes - example

1.6. JWKS refresh every 24 hours
  Thoughts: This is a time-based behavior that's difficult to test in unit tests
  Testable: no

1.7. JWKS refresh failure handling
  Thoughts: When refresh fails, system should continue with cached JWKS
  Testable: yes - example

**Requirement 2: Dual Authentication Mode Selection**
2.1. Session cookie takes precedence
  Thoughts: For any request with valid session cookie, session auth should be used
  Testable: yes - property

2.2. Basic auth when no session cookie
  Thoughts: For any request with Authorization header and no session, Basic auth should be attempted
  Testable: yes - property

2.3. OIDC flow when no credentials
  Thoughts: For any request without session or auth header, OIDC flow should start
  Testable: yes - property

2.4. Callback path handling
  Thoughts: The /auth/callback path should always be processed as OIDC callback
  Testable: yes - example

2.5. Simultaneous mode support
  Thoughts: This is an architectural requirement, not a functional test
  Testable: no

**Requirement 3: OIDC Authorization Flow with PKCE**
3.1. State token generation
  Thoughts: For any unauthenticated request, a cryptographically random state token should be generated
  Testable: yes - property

3.2. State token storage
  Thoughts: For any generated state token, it should be stored with original URL and 5-minute TTL
  Testable: yes - property

3.3. PKCE generation
  Thoughts: For any OIDC redirect, PKCE values should be generated
  Testable: yes - property

3.4. Authorization redirect
  Thoughts: The redirect URL should contain all required parameters
  Testable: yes - property

3.5. State validation on callback
  Thoughts: For any callback, the state must match an unused token
  Testable: yes - property

3.6-3.7. Invalid state handling
  Thoughts: For any invalid/expired/used state, return 400
  Testable: yes - property

3.8. State token marking and deletion
  Thoughts: After validation, state should be marked used and deleted
  Testable: yes - property

3.9. Token exchange
  Thoughts: For any valid authorization code, token exchange should be attempted
  Testable: yes - property

3.10. Token exchange failure
  Thoughts: Failed token exchange should return 401
  Testable: yes - example

3.11. ID token signature validation
  Thoughts: For any ID token, signature must be validated using JWKS
  Testable: yes - property

3.12. Invalid signature handling
  Thoughts: Invalid signatures should return 401
  Testable: yes - example

3.13. Claims validation (iss, aud)
  Thoughts: For any ID token, iss and aud claims must match configuration
  Testable: yes - property

3.14. Invalid claims handling
  Thoughts: Invalid claims should return 401
  Testable: yes - example

3.15. Session creation
  Thoughts: For any validated ID token, a session should be created
  Testable: yes - property

3.16. Redirect to original URL
  Thoughts: After session creation, redirect to stored original URL
  Testable: yes - property

**Requirement 4: Session Management**
4.1. Session ID generation
  Thoughts: For any authenticated OIDC user, a cryptographically random session ID should be generated
  Testable: yes - property

4.2. Session storage
  Thoughts: For any created session, all required fields should be stored
  Testable: yes - property

4.3. Session expiration time
  Thoughts: For any session, expiration should be set based on configured TTL
  Testable: yes - property

4.4. Session cookie attributes
  Thoughts: For any session cookie, it should have HttpOnly, Secure, SameSite=Lax
  Testable: yes - property

4.5. Session retrieval
  Thoughts: For any request with session cookie, session should be retrieved
  Testable: yes - property

4.6. Non-existent session handling
  Thoughts: For any non-existent session ID, treat as unauthenticated
  Testable: yes - property

4.7. Expired session handling
  Thoughts: For any expired session, delete and treat as unauthenticated
  Testable: yes - property

4.8. Valid session usage
  Thoughts: For any valid non-expired session, use stored user identity
  Testable: yes - property

4.9-4.10. Session store backend
  Thoughts: This is configuration-based behavior, tested through integration
  Testable: no

**Requirement 5: Local User Authentication**
5.1. Basic auth decoding
  Thoughts: For any request with Authorization header, credentials should be decoded
  Testable: yes - property

5.2. Decode failure handling
  Thoughts: For any malformed Authorization header, return 401
  Testable: yes - property

5.3. Credential extraction
  Thoughts: For any decoded credentials, extract username and password
  Testable: yes - property

5.4. Username lookup
  Thoughts: For any username, search configured local users
  Testable: yes - property

5.5. Unknown username handling
  Thoughts: For any unknown username, return 401
  Testable: yes - property

5.6. Password validation
  Thoughts: For any local user, validate password using bcrypt
  Testable: yes - property

5.7. Invalid password handling
  Thoughts: For any invalid password, return 401
  Testable: yes - property

5.8. ES user mapping
  Thoughts: For any authenticated local user, retrieve mapped ES user
  Testable: yes - property

5.9. No session creation
  Thoughts: For any Basic auth success, proceed without creating session
  Testable: yes - property

**Requirement 6: Elasticsearch Credential Mapping and Injection**
6.1. OIDC claim extraction
  Thoughts: For any OIDC user, extract claims according to configured mappings
  Testable: yes - property

6.2. Mapping evaluation order
  Thoughts: For any OIDC user, evaluate mappings in order until match
  Testable: yes - property

6.3-6.4. Claim matching
  Thoughts: For any claim value and pattern, wildcard matching should work correctly
  Testable: yes - property

6.5. Mapping match usage
  Thoughts: For any matched mapping, use corresponding ES user
  Testable: yes - property

6.6. Default ES user fallback
  Thoughts: For any OIDC user with no mapping match, use default ES user
  Testable: yes - property

6.7. Local user ES mapping
  Thoughts: For any local user, use configured ES user
  Testable: yes - property

6.8. ES credentials retrieval
  Thoughts: For any ES user, retrieve corresponding credentials
  Testable: yes - property

6.9. Missing credentials handling
  Thoughts: For any ES user without credentials, return 500
  Testable: yes - example

6.10. Credentials encoding
  Thoughts: For any ES credentials, encode as Basic auth
  Testable: yes - property

6.11. Header injection
  Thoughts: For any authenticated request, add X-Es-Authorization header
  Testable: yes - property

6.12. Credential security
  Thoughts: ES credentials should never be logged in plaintext
  Testable: no (security audit)

**Requirement 7: ForwardAuth Mode (Traefik)**
7.1-7.3. Header reading
  Thoughts: For any forwardAuth request, read X-Forwarded-* headers
  Testable: yes - property

7.4. Success response
  Thoughts: For any successful auth in forwardAuth mode, return 200 with header
  Testable: yes - property

7.5. Failure response
  Thoughts: For any failed auth in forwardAuth mode, return appropriate status
  Testable: yes - property

7.6. Callback handling
  Thoughts: For any callback path in forwardAuth mode, process and return 302
  Testable: yes - example

7.7. No proxying
  Thoughts: In forwardAuth mode, never proxy to upstream
  Testable: yes - example

7.8. Cookie preservation
  Thoughts: For any forwardAuth request, preserve Cookie headers
  Testable: yes - property

**Requirement 8: Auth_Request Mode (Nginx)**
8.1-8.3. Nginx header support
  Thoughts: For any request with X-Original-* headers, normalize to internal format
  Testable: yes - property

8.4. Header normalization
  Thoughts: For any Traefik or Nginx headers, produce same internal representation
  Testable: yes - property

8.5-8.6. Consistent behavior
  Thoughts: For any Nginx request, apply same auth logic and responses as Traefik
  Testable: yes - property

**Requirement 9: Standalone Proxy Mode**
9.1. Request proxying
  Thoughts: For any authenticated request in standalone mode, proxy to upstream
  Testable: yes - property

9.2-9.4. Internal endpoint handling
  Thoughts: Callback, logout, healthz should not be proxied
  Testable: yes - example

9.5. Header injection before forwarding
  Thoughts: For any proxied request, add X-Es-Authorization before forwarding
  Testable: yes - property

9.6-9.7. Request preservation
  Thoughts: For any proxied request, preserve headers, method, path, query, body
  Testable: yes - property

9.8-9.10. Response preservation
  Thoughts: For any upstream response, preserve body, headers, status code
  Testable: yes - property

9.11-9.12. Error handling
  Thoughts: For any upstream failure, return appropriate error status
  Testable: yes - example

9.13. WebSocket support
  Thoughts: For any WebSocket upgrade request, establish bidirectional connection
  Testable: yes - example

9.14. Timeout configuration
  Thoughts: For any upstream request, use configured timeout
  Testable: yes - property

**Requirement 10: Logout Functionality**
10.1. Session extraction
  Thoughts: For any logout request, extract session ID from cookie
  Testable: yes - property

10.2. Session deletion
  Thoughts: For any found session ID, delete from store
  Testable: yes - property

10.3. Cookie clearing
  Thoughts: For any session deletion, return Set-Cookie with Max-Age=0
  Testable: yes - property

10.4. OIDC provider logout
  Thoughts: For any logout with end_session_endpoint, redirect to provider
  Testable: yes - example

10.5. Fallback logout
  Thoughts: Without end_session_endpoint, redirect to configured URL or return 200
  Testable: yes - example

10.6. No session handling
  Thoughts: For any logout without session, return 200
  Testable: yes - example

**Requirement 11: Health Check Endpoint**
11.1. Unauthenticated endpoint
  Thoughts: /healthz should always be accessible without auth
  Testable: yes - example

11.2. Success response
  Thoughts: For any healthy system, return 200 with status and version
  Testable: yes - example

11.3. Session store check
  Thoughts: For any healthz request, verify session store accessibility
  Testable: yes - property

11.4. Unhealthy session store
  Thoughts: For any inaccessible session store, return 503
  Testable: yes - example

11.5-11.6. OIDC health check
  Thoughts: When OIDC enabled, verify discovery document loaded
  Testable: yes - example

**Requirement 12: Configuration Loading**
12.1. Config file loading
  Thoughts: For any startup, load config from specified file
  Testable: yes - example

12.2. Environment variable substitution
  Thoughts: For any config value with ${VAR}, replace with env var value
  Testable: yes - property

12.3. Variable replacement
  Thoughts: This is the same as 12.2
  Testable: yes - property

12.4. Missing variable handling
  Thoughts: For any missing env var, refuse to start
  Testable: yes - example

12.5. Required field validation
  Thoughts: For any startup, validate all required fields present
  Testable: yes - property

12.6. Missing field error
  Thoughts: For any missing required field, log error and refuse to start
  Testable: yes - example

12.7-12.10. Specific validations
  Thoughts: For any config, validate session_secret length, bcrypt hashes, ES credentials, redirect URL
  Testable: yes - property

12.11. Validation failure
  Thoughts: For any validation failure, log error and refuse to start
  Testable: yes - example

**Requirement 13: Security Controls**
13.1-13.2. Cryptographic randomness
  Thoughts: For any state token or session ID, use crypto/rand
  Testable: yes - property

13.3. Bcrypt timing-safe comparison
  Thoughts: For any password validation, use timing-safe bcrypt comparison
  Testable: yes - property

13.4. No plaintext logging
  Thoughts: For any log entry, sensitive values should not appear
  Testable: no (security audit)

13.5-13.7. Cookie security attributes
  Thoughts: For any session cookie, verify HttpOnly, Secure, SameSite=Lax
  Testable: yes - property

13.8. Cookie content restriction
  Thoughts: For any session cookie, only store session ID
  Testable: yes - property

13.9. ID token signature validation
  Thoughts: For any ID token, validate signature using JWKS
  Testable: yes - property

13.10-13.12. ID token claim validation
  Thoughts: For any ID token, validate iss, aud, exp claims
  Testable: yes - property

13.13. Invalid token rejection
  Thoughts: For any invalid ID token, return 401
  Testable: yes - example

13.14. State token single-use
  Thoughts: For any state token, reject reuse attempts
  Testable: yes - property

13.15. State token deletion
  Thoughts: For any used or expired state token, delete from store
  Testable: yes - property

13.16. PKCE usage
  Thoughts: For any OIDC flow, use PKCE
  Testable: yes - property

13.17-13.18. HTTPS and TLS
  Thoughts: For any OIDC provider request, use HTTPS with cert validation
  Testable: yes - property

**Requirement 14: Logging and Observability**
14.1. Structured logging
  Thoughts: For any log entry, include timestamp, level, message, context
  Testable: yes - property

14.2-14.6. Context fields
  Thoughts: For any specific event type, include appropriate context fields
  Testable: yes - property

14.7-14.9. Log levels
  Thoughts: For any event, use appropriate log level
  Testable: yes - property

14.10. No sensitive logging
  Thoughts: For any log entry, exclude sensitive values
  Testable: no (security audit)

**Requirement 15: Error Handling and Resilience**
15.1. Session store failure
  Thoughts: For any session store connection failure, return 503
  Testable: yes - example

15.2. OIDC provider unreachable
  Thoughts: For any unreachable token endpoint, return 502
  Testable: yes - example

15.3. OIDC provider error
  Thoughts: For any OIDC provider error response, return 401
  Testable: yes - example

15.4. JWKS fetch retry
  Thoughts: For any JWKS fetch failure at startup, retry 3 times with backoff
  Testable: yes - example

15.5. JWKS refresh failure
  Thoughts: For any JWKS refresh failure during operation, log warning and continue
  Testable: yes - example

15.6. Discovery document retry
  Thoughts: For any discovery fetch failure at startup, retry 3 times with backoff
  Testable: yes - example

15.7. Upstream unreachable
  Thoughts: For any unreachable upstream in standalone mode, return 502
  Testable: yes - example

15.8. Unexpected error handling
  Thoughts: For any unexpected error, log with stack trace and return 500
  Testable: yes - example

15.9. Graceful shutdown
  Thoughts: For any shutdown signal, wait for in-flight requests
  Testable: yes - example

15.10. Shutdown timeout
  Thoughts: For any shutdown, wait up to 30 seconds for requests
  Testable: yes - example

**Requirement 16: Redis Session Store Integration**
16.1. Redis connection at startup
  Thoughts: When Redis configured, connect at startup
  Testable: yes - example

16.2. Connection failure
  Thoughts: For any Redis connection failure at startup, refuse to start
  Testable: yes - example

16.3. Session serialization
  Thoughts: For any session stored in Redis, serialize as JSON with session ID as key
  Testable: yes - property

16.4. Redis TTL
  Thoughts: For any session stored in Redis, set TTL to match expiration
  Testable: yes - property

16.5. Session deserialization
  Thoughts: For any session retrieved from Redis, deserialize JSON to session object
  Testable: yes - property

16.6. Redis operation failure
  Thoughts: For any Redis operation failure during request, return 503
  Testable: yes - example

16.7. State token key prefix
  Thoughts: For any state token in Redis, use "state:" prefix
  Testable: yes - property

16.8. State token TTL
  Thoughts: For any state token in Redis, set 5-minute TTL
  Testable: yes - property

16.9. Connection pooling
  Thoughts: Redis client should use connection pool with min 5, max 20
  Testable: yes - example

16.10. Automatic reconnection
  Thoughts: For any lost Redis connection, reconnect with exponential backoff
  Testable: yes - example

**Requirement 17: Performance and Resource Management**
17.1-17.3. Caching
  Thoughts: Discovery document, JWKS, and parsed keys should be cached in memory
  Testable: yes - example

17.4. Expired session cleanup
  Thoughts: For in-memory store, cleanup expired sessions every 5 minutes
  Testable: yes - example

17.5. Concurrent request limit
  Thoughts: For any time, limit concurrent requests to 1000
  Testable: yes - example

17.6. Overload handling
  Thoughts: When limit reached, return 503
  Testable: yes - example

17.7. Request timeouts
  Thoughts: For any OIDC provider request, use 30-second timeout
  Testable: yes - example

17.8. Connection pooling
  Thoughts: Use connection pool with keep-alive for OIDC provider
  Testable: yes - example

17.9. Request body size limit
  Thoughts: For any request, limit body to 1MB
  Testable: yes - example

17.10. Body size exceeded
  Thoughts: For any request exceeding 1MB, return 413
  Testable: yes - example

**Requirement 18: Metrics and Monitoring**
18.1-18.9. Prometheus metrics
  Thoughts: Various metrics should be exposed at /metrics endpoint
  Testable: yes - example

18.10. Unauthenticated metrics
  Thoughts: /metrics endpoint should require no authentication
  Testable: yes - example

**Requirement 19: OpenTelemetry Integration**
19.1-19.8. Tracing
  Thoughts: When enabled, create spans for various operations
  Testable: yes - example

19.9. Initialization failure
  Thoughts: For any OTel init failure, log warning and continue
  Testable: yes - example

**Requirement 20: Configuration Validation and Documentation**
20.1-20.9. Validation error messages
  Thoughts: For any specific configuration error, print clear error message
  Testable: yes - example

20.10. Validate-config flag
  Thoughts: --validate-config should validate and exit
  Testable: yes - example

20.11-20.12. Validation exit codes
  Thoughts: Exit 0 on success, 1 on failure
  Testable: yes - example


### Property Reflection

After analyzing all acceptance criteria, I identified several areas where properties can be consolidated:

**Redundancy Analysis:**

1. **State Token Properties**: Multiple criteria (3.1, 3.2, 3.5, 3.7, 3.8, 13.14, 13.15) all relate to state token lifecycle. These can be combined into comprehensive properties about state token generation, validation, and single-use enforcement.

2. **Session Properties**: Criteria 4.1-4.8 all relate to session lifecycle. These can be consolidated into properties about session creation, validation, and expiration.

3. **ID Token Validation**: Criteria 3.11, 3.13, 13.9-13.12 all relate to ID token validation. These can be combined into a comprehensive ID token validation property.

4. **Authentication Mode Selection**: Criteria 2.1-2.3 describe a precedence order that can be expressed as a single property about authentication method selection.

5. **Header Normalization**: Criteria 7.1-7.3, 8.1-8.4 describe header normalization that can be expressed as a property about consistent internal representation.

6. **Request/Response Preservation**: Criteria 9.6-9.10 describe request and response preservation that can be combined into properties about proxy transparency.

7. **ES Credential Mapping**: Criteria 6.1-6.8 describe the mapping process that can be consolidated into properties about mapping evaluation and credential retrieval.

8. **Configuration Validation**: Multiple criteria in requirements 12 and 20 describe validation that can be grouped into properties about configuration completeness and correctness.

**Properties to Define:**

After consolidation, the key properties to test are:

1. State token generation and single-use enforcement
2. Session lifecycle (creation, validation, expiration)
3. ID token signature and claims validation
4. Authentication method precedence
5. OIDC authorization flow completeness
6. ES credential mapping evaluation order
7. Header normalization consistency
8. Proxy request/response transparency
9. Configuration validation completeness
10. Security attribute enforcement (cookies, PKCE, HTTPS)
11. Error handling consistency
12. Basic auth credential validation


### Correctness Properties

### Property 1: State Token Single-Use Enforcement

*For any* OIDC state token, after it is successfully validated once, any subsequent attempt to use the same state token should be rejected with HTTP 400.

**Validates: Requirements 3.5, 3.6, 3.7, 3.8, 13.14, 13.15**

### Property 2: State Token Lifecycle

*For any* generated state token, it should be stored with the original request URL, have a 5-minute TTL, be cryptographically random (32 bytes), and be deleted after successful use or expiration.

**Validates: Requirements 3.1, 3.2, 13.1, 13.15**

### Property 3: Session Creation and Storage

*For any* successfully authenticated OIDC user, a session should be created with a cryptographically random session ID, stored with user identity and mapped ES user, have an expiration time based on configured TTL, and return a cookie with HttpOnly, Secure, and SameSite=Lax attributes.

**Validates: Requirements 4.1, 4.2, 4.3, 4.4, 13.2, 13.5, 13.6, 13.7, 13.8**

### Property 4: Session Validation and Expiration

*For any* request with a session cookie, if the session exists and is not expired, the stored user identity should be used; if the session is expired, it should be deleted and the request treated as unauthenticated; if the session does not exist, the request should be treated as unauthenticated.

**Validates: Requirements 4.5, 4.6, 4.7, 4.8**

### Property 5: Authentication Method Precedence

*For any* incoming request, authentication should be attempted in this order: (1) valid session cookie, (2) Basic Authorization header, (3) OIDC flow initiation, with the first successful method being used and subsequent methods skipped.

**Validates: Requirements 2.1, 2.2, 2.3**

### Property 6: ID Token Validation Completeness

*For any* ID token received from the OIDC provider, it should be validated for: (1) signature using JWKS public keys, (2) issuer claim matching configured issuer_url, (3) audience claim matching configured client_id, (4) expiration claim being in the future, and any validation failure should result in HTTP 401.

**Validates: Requirements 3.11, 3.13, 13.9, 13.10, 13.11, 13.12, 13.13**

### Property 7: OIDC Authorization Flow Completeness

*For any* unauthenticated request initiating OIDC flow, the system should: (1) generate state token and PKCE values, (2) store state with original URL, (3) redirect to authorization_endpoint with all required parameters, (4) validate state on callback, (5) exchange code for tokens with PKCE verifier, (6) validate ID token, (7) create session, (8) redirect to original URL.

**Validates: Requirements 3.1, 3.2, 3.3, 3.4, 3.5, 3.9, 3.11, 3.13, 3.15, 3.16, 13.16**

### Property 8: PKCE Generation and Validation

*For any* OIDC authorization flow, a code_verifier should be generated using cryptographically secure randomness, a code_challenge should be derived using S256 method, the verifier should be stored with the state token, and the verifier should be included in the token exchange request.

**Validates: Requirements 3.3, 13.16**

### Property 9: Basic Auth Credential Validation

*For any* request with Basic Authorization header, the credentials should be base64-decoded, the username should be looked up in configured local users, the password should be validated using bcrypt timing-safe comparison, and authentication should succeed only if both username exists and password matches.

**Validates: Requirements 5.1, 5.2, 5.3, 5.4, 5.5, 5.6, 13.3**

### Property 10: ES Credential Mapping Evaluation Order

*For any* authenticated OIDC user, the system should evaluate oidc_mappings in configuration order, extract the claim specified by each mapping, perform wildcard pattern matching, use the es_user from the first matching mapping, or use default_es_user if no mappings match.

**Validates: Requirements 6.1, 6.2, 6.3, 6.4, 6.5, 6.6**

### Property 11: ES Credential Injection

*For any* authenticated request (OIDC or Basic), the system should: (1) determine the mapped ES user, (2) retrieve ES credentials for that user from configuration, (3) encode credentials as Basic auth (base64 of username:password), (4) add X-Es-Authorization header with value "Basic {encoded_credentials}".

**Validates: Requirements 6.7, 6.8, 6.10, 6.11**

### Property 12: Header Normalization Consistency

*For any* request in forwardAuth mode, whether it contains Traefik headers (X-Forwarded-*) or Nginx headers (X-Original-*), the system should normalize them to the same internal RequestContext representation with method, path, host, and original URL.

**Validates: Requirements 7.1, 7.2, 7.3, 8.1, 8.2, 8.3, 8.4**

### Property 13: ForwardAuth Response Format

*For any* authenticated request in forwardAuth mode, the system should return HTTP 200 with X-Es-Authorization header and never proxy the request to upstream; for unauthenticated requests, it should return HTTP 401 for Basic auth failures or HTTP 302 for OIDC redirects.

**Validates: Requirements 7.4, 7.5, 7.7**

### Property 14: Standalone Proxy Request Preservation

*For any* authenticated request in standalone mode (excluding /auth/callback, /auth/logout, /healthz), the system should proxy to upstream preserving: (1) HTTP method, (2) path and query parameters, (3) request headers (except hop-by-hop), (4) request body, and add X-Es-Authorization header before forwarding.

**Validates: Requirements 9.1, 9.2, 9.3, 9.4, 9.5, 9.6, 9.7**

### Property 15: Standalone Proxy Response Preservation

*For any* upstream response in standalone mode, the system should preserve: (1) response status code, (2) response headers (except hop-by-hop), (3) response body streamed to client.

**Validates: Requirements 9.8, 9.9, 9.10**

### Property 16: Configuration Environment Variable Substitution

*For any* configuration value containing ${VAR_NAME} syntax, the system should replace it with the value of the VAR_NAME environment variable at startup, and refuse to start if the environment variable is not set.

**Validates: Requirements 12.2, 12.3, 12.4**

### Property 17: Configuration Validation Completeness

*For any* startup, the system should validate: (1) all required fields are present, (2) session_secret is at least 32 bytes, (3) all password_bcrypt values are valid bcrypt hashes, (4) redirect_url is a valid HTTPS URL, (5) at least one authentication method is enabled, (6) at least one ES user is configured, and refuse to start with a descriptive error if any validation fails.

**Validates: Requirements 12.5, 12.6, 12.7, 12.8, 12.9, 12.10, 12.11, 20.1, 20.2, 20.3, 20.4, 20.5, 20.6, 20.7, 20.8, 20.9**

### Property 18: Discovery Document Validation

*For any* successfully fetched Discovery Document, the issuer value in the document must match the configured issuer_url, or the system should refuse to start with an error.

**Validates: Requirements 1.4**

### Property 19: Redis Session Serialization Round-Trip

*For any* session stored in Redis, serializing to JSON then deserializing should produce an equivalent session object with all fields preserved.

**Validates: Requirements 16.3, 16.5**

### Property 20: Redis Key TTL Consistency

*For any* session stored in Redis, the Redis key TTL should match the session expiration time; for any state token stored in Redis, the Redis key TTL should be 5 minutes.

**Validates: Requirements 16.4, 16.8**

### Property 21: Logout Session Cleanup

*For any* logout request with a valid session cookie, the session should be deleted from the session store, a Set-Cookie header with Max-Age=0 should be returned, and subsequent requests with that cookie should be treated as unauthenticated.

**Validates: Requirements 10.1, 10.2, 10.3**

### Property 22: Cryptographic Randomness

*For any* generated state token or session ID, it should be generated using crypto/rand (cryptographically secure random number generator) and have sufficient entropy (minimum 32 bytes for state tokens, minimum 32 bytes for session IDs).

**Validates: Requirements 13.1, 13.2**

### Property 23: HTTPS Enforcement for OIDC

*For any* request to the OIDC provider (token_endpoint, userinfo_endpoint, jwks_uri), the system should use HTTPS and validate TLS certificates.

**Validates: Requirements 13.17, 13.18**


## Error Handling

### Error Classification

Keyline errors are classified into four categories:

1. **Client Errors (4xx)**: Invalid requests, authentication failures, authorization failures
2. **Server Errors (5xx)**: Internal errors, dependency failures, configuration errors
3. **Startup Errors**: Configuration validation failures, dependency connection failures
4. **Runtime Errors**: Transient failures, timeout errors, connection errors

### Error Handling Strategy

#### Authentication Errors

```go
// Authentication failure scenarios
type AuthError struct {
    Code    int    // HTTP status code
    Message string // User-facing message
    Internal error // Internal error (logged, not exposed)
}

// Error types:
// - Invalid credentials: 401 Unauthorized
// - Expired session: 401 Unauthorized (triggers re-auth)
// - Invalid state token: 400 Bad Request
// - OIDC provider error: 502 Bad Gateway
// - Token validation failure: 401 Unauthorized
```

#### Dependency Failures

```go
// Dependency failure handling
type DependencyError struct {
    Service  string // redis, oidc_provider, upstream
    Operation string // connect, read, write
    Err      error  // Underlying error
}

// Handling strategy:
// - Session store failure: Return 503, log error, continue serving (degraded)
// - OIDC provider failure: Return 502, log error, retry with backoff
// - Upstream failure (standalone): Return 502/504, log error
// - Redis connection loss: Attempt reconnection with exponential backoff
```

#### Configuration Errors

```go
// Configuration validation errors
type ConfigError struct {
    Field   string // Configuration field name
    Value   string // Invalid value (sanitized)
    Reason  string // Why validation failed
}

// Handling strategy:
// - Missing required field: Print error, exit code 1
// - Invalid value: Print error with expected format, exit code 1
// - Missing environment variable: Print error with variable name, exit code 1
// - Validation failure: Print all errors, exit code 1
```

#### Retry Strategy

```go
// Retry configuration for transient failures
type RetryConfig struct {
    MaxAttempts int           // Maximum retry attempts
    InitialDelay time.Duration // Initial backoff delay
    MaxDelay    time.Duration // Maximum backoff delay
    Multiplier  float64       // Backoff multiplier
}

// Retry scenarios:
// - Discovery Document fetch: 3 attempts, exponential backoff (1s, 2s, 4s)
// - JWKS fetch: 3 attempts, exponential backoff (1s, 2s, 4s)
// - Redis connection: Infinite attempts, exponential backoff (max 30s)
// - OIDC token exchange: No retry (single attempt)
// - Upstream proxy: No retry (single attempt with timeout)
```

### Graceful Degradation

Keyline implements graceful degradation for non-critical failures:

1. **JWKS Refresh Failure**: Continue using cached JWKS, log warning
2. **Metrics Collection Failure**: Continue serving requests, log error
3. **Tracing Failure**: Continue serving requests, log warning
4. **Session Store Failure (read)**: Treat as unauthenticated, log error
5. **Session Store Failure (write)**: Return 503, log error (cannot create session)

### Graceful Shutdown

```go
// Shutdown sequence
func (s *Server) Shutdown(ctx context.Context) error {
    // 1. Stop accepting new connections
    s.listener.Close()
    
    // 2. Wait for in-flight requests (max 30 seconds)
    shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()
    
    // 3. Close session store connections
    s.sessionStore.Close()
    
    // 4. Close OIDC provider connections
    s.oidcProvider.Close()
    
    // 5. Flush metrics and traces
    s.metrics.Flush()
    s.tracer.Flush()
    
    return s.server.Shutdown(shutdownCtx)
}
```

### Error Response Format

```go
// Standard error response
type ErrorResponse struct {
    Error   string `json:"error"`           // Error type
    Message string `json:"message"`         // User-facing message
    Code    int    `json:"code,omitempty"`  // Application error code
}

// Examples:
// {"error": "unauthorized", "message": "Invalid credentials"}
// {"error": "bad_request", "message": "Invalid or expired state token"}
// {"error": "service_unavailable", "message": "Session store temporarily unavailable"}
// {"error": "bad_gateway", "message": "Authentication provider unavailable"}
```

### Logging Strategy

```go
// Error logging with context
logger.Error("authentication failed",
    slog.String("method", "oidc"),
    slog.String("error", err.Error()),
    slog.String("user_id", userID),
    slog.String("source_ip", sourceIP),
)

// Startup error logging
logger.Error("configuration validation failed",
    slog.String("field", "oidc.issuer_url"),
    slog.String("reason", "missing required field"),
)

// Dependency error logging
logger.Error("session store connection failed",
    slog.String("store", "redis"),
    slog.String("url", redisURL),
    slog.String("error", err.Error()),
)
```


## Testing Strategy

### Dual Testing Approach

Keyline testing combines unit tests and property-based tests for comprehensive coverage:

- **Unit Tests**: Verify specific examples, edge cases, error conditions, and integration points
- **Property Tests**: Verify universal properties across all inputs using randomized testing

Both approaches are complementary and necessary for production readiness.

### Property-Based Testing

#### Framework Selection

- **Language**: Go
- **Library**: [gopter](https://github.com/leanovate/gopter) - Property-based testing for Go
- **Configuration**: Minimum 100 iterations per property test

#### Property Test Structure

```go
import (
    "testing"
    "github.com/leanovate/gopter"
    "github.com/leanovate/gopter/gen"
    "github.com/leanovate/gopter/prop"
)

// Feature: keyline-auth-proxy, Property 1: State Token Single-Use Enforcement
func TestProperty_StateTokenSingleUse(t *testing.T) {
    properties := gopter.NewProperties(nil)
    properties.Property("state token can only be used once", prop.ForAll(
        func(stateToken string, originalURL string) bool {
            // Generate state token
            token := &StateToken{
                ID:          stateToken,
                OriginalURL: originalURL,
                Used:        false,
            }
            
            // Store token
            store.Store(ctx, token)
            
            // First use should succeed
            retrieved1, err1 := store.Get(ctx, stateToken)
            if err1 != nil || retrieved1 == nil {
                return false
            }
            
            // Second use should fail
            retrieved2, err2 := store.Get(ctx, stateToken)
            if err2 == nil || retrieved2 != nil {
                return false // Should have failed
            }
            
            return true
        },
        gen.Identifier(),      // Random state token
        gen.AnyString(),       // Random original URL
    ))
    
    properties.TestingRun(t, gopter.ConsoleReporter(t))
}

// Feature: keyline-auth-proxy, Property 10: ES Credential Mapping Evaluation Order
func TestProperty_ESCredentialMappingOrder(t *testing.T) {
    properties := gopter.NewProperties(nil)
    properties.Property("mappings evaluated in order until first match", prop.ForAll(
        func(email string) bool {
            // Setup mappings
            mappings := []OIDCMapping{
                {Claim: "email", Pattern: "*@admin.example.com", ESUser: "admin"},
                {Claim: "email", Pattern: "*@example.com", ESUser: "readonly"},
            }
            
            // Create user with email claim
            user := &User{
                Claims: map[string]any{"email": email},
            }
            
            // Map user
            esUser := mapper.MapOIDCUser(user, mappings, "default")
            
            // Verify correct mapping
            if strings.HasSuffix(email, "@admin.example.com") {
                return esUser == "admin"
            } else if strings.HasSuffix(email, "@example.com") {
                return esUser == "readonly"
            } else {
                return esUser == "default"
            }
        },
        gen.OneGenOf(
            gen.Const("user@admin.example.com"),
            gen.Const("user@example.com"),
            gen.Const("user@other.com"),
        ),
    ))
    
    properties.TestingRun(t, gopter.ConsoleReporter(t))
}
```

#### Property Test Coverage

Each correctness property from the design document must be implemented as a property-based test:

1. Property 1: State Token Single-Use Enforcement
2. Property 2: State Token Lifecycle
3. Property 3: Session Creation and Storage
4. Property 4: Session Validation and Expiration
5. Property 5: Authentication Method Precedence
6. Property 6: ID Token Validation Completeness
7. Property 7: OIDC Authorization Flow Completeness
8. Property 8: PKCE Generation and Validation
9. Property 9: Basic Auth Credential Validation
10. Property 10: ES Credential Mapping Evaluation Order
11. Property 11: ES Credential Injection
12. Property 12: Header Normalization Consistency
13. Property 13: ForwardAuth Response Format
14. Property 14: Standalone Proxy Request Preservation
15. Property 15: Standalone Proxy Response Preservation
16. Property 16: Configuration Environment Variable Substitution
17. Property 17: Configuration Validation Completeness
18. Property 18: Discovery Document Validation
19. Property 19: Redis Session Serialization Round-Trip
20. Property 20: Redis Key TTL Consistency
21. Property 21: Logout Session Cleanup
22. Property 22: Cryptographic Randomness
23. Property 23: HTTPS Enforcement for OIDC

### Unit Testing

#### Unit Test Focus Areas

Unit tests complement property tests by focusing on:

1. **Specific Examples**: Concrete scenarios with known inputs and outputs
2. **Edge Cases**: Empty strings, nil values, boundary conditions
3. **Error Conditions**: Specific error scenarios and error messages
4. **Integration Points**: Component interactions and interfaces
5. **Mock Interactions**: Behavior with mocked dependencies

#### Unit Test Structure

```go
// Unit test example: OIDC callback with invalid state
func TestOIDCCallback_InvalidState(t *testing.T) {
    // Setup
    provider := NewOIDCProvider(config, stateStore, sessionStore, mapper, logger)
    
    // Create request with invalid state
    req := &AuthRequest{
        Path: "/auth/callback",
        Headers: map[string]string{
            "query": "code=abc123&state=invalid_state",
        },
    }
    
    // Execute
    result, err := provider.Authenticate(ctx, req)
    
    // Assert
    assert.Error(t, err)
    assert.Nil(t, result)
    assert.Contains(t, err.Error(), "Invalid or expired state token")
}

// Unit test example: Session expiration
func TestSessionValidation_ExpiredSession(t *testing.T) {
    // Setup
    store := NewInMemorySessionStore(logger)
    session := &Session{
        ID:        "session123",
        UserID:    "user123",
        ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired
    }
    store.Create(ctx, session)
    
    // Execute
    retrieved, err := store.Get(ctx, "session123")
    
    // Assert
    assert.NoError(t, err)
    assert.Nil(t, retrieved) // Should return nil for expired session
    
    // Verify session was deleted
    _, err = store.Get(ctx, "session123")
    assert.Error(t, err)
}

// Unit test example: Basic auth with bcrypt
func TestBasicAuth_ValidCredentials(t *testing.T) {
    // Setup
    hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
    config := &LocalUsersConfig{
        Users: []LocalUser{
            {
                Username:       "testuser",
                PasswordBcrypt: string(hashedPassword),
                ESUser:         "es_testuser",
            },
        },
    }
    provider := NewBasicAuthProvider(config, mapper, logger)
    
    // Create request with valid credentials
    authHeader := base64.StdEncoding.EncodeToString([]byte("testuser:password123"))
    req := &AuthRequest{
        Headers: map[string]string{
            "Authorization": "Basic " + authHeader,
        },
    }
    
    // Execute
    result, err := provider.Authenticate(ctx, req)
    
    // Assert
    assert.NoError(t, err)
    assert.True(t, result.Authenticated)
    assert.Equal(t, "testuser", result.User.Username)
    assert.Equal(t, "es_testuser", result.ESUser)
}
```

#### Test Organization

```
keyline/
├── internal/
│   ├── auth/
│   │   ├── oidc.go
│   │   ├── oidc_test.go          # Unit tests
│   │   ├── oidc_property_test.go # Property tests
│   │   ├── basic.go
│   │   ├── basic_test.go
│   │   └── basic_property_test.go
│   ├── session/
│   │   ├── manager.go
│   │   ├── manager_test.go
│   │   ├── manager_property_test.go
│   │   ├── store_redis.go
│   │   ├── store_redis_test.go
│   │   ├── store_memory.go
│   │   └── store_memory_test.go
│   ├── transport/
│   │   ├── forward_auth.go
│   │   ├── forward_auth_test.go
│   │   ├── standalone.go
│   │   └── standalone_test.go
│   └── mapper/
│       ├── credentials.go
│       ├── credentials_test.go
│       └── credentials_property_test.go
└── integration/
    ├── oidc_flow_test.go         # End-to-end OIDC flow
    ├── basic_auth_test.go        # End-to-end Basic auth
    └── proxy_test.go             # End-to-end proxy behavior
```

### Integration Testing

#### Integration Test Scenarios

1. **Complete OIDC Flow**: Mock OIDC provider, test full authorization flow
2. **Complete Basic Auth Flow**: Test credential validation and ES mapping
3. **Session Lifecycle**: Test session creation, validation, expiration, deletion
4. **ForwardAuth Mode**: Test with mock Traefik headers
5. **Standalone Proxy Mode**: Test with mock upstream service
6. **Redis Integration**: Test with real Redis instance (testcontainers)
7. **Configuration Loading**: Test with various configuration files

#### Integration Test Example

```go
func TestIntegration_OIDCFlow(t *testing.T) {
    // Setup mock OIDC provider
    mockOIDC := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        switch r.URL.Path {
        case "/.well-known/openid-configuration":
            json.NewEncoder(w).Encode(discoveryDoc)
        case "/token":
            json.NewEncoder(w).Encode(tokenResponse)
        case "/jwks":
            json.NewEncoder(w).Encode(jwks)
        }
    }))
    defer mockOIDC.Close()
    
    // Setup Keyline with mock OIDC provider
    config := &Config{
        OIDC: OIDCConfig{
            Enabled:     true,
            IssuerURL:   mockOIDC.URL,
            ClientID:    "test-client",
            ClientSecret: "test-secret",
            RedirectURL: "http://localhost:9000/auth/callback",
        },
    }
    server := NewServer(config)
    
    // Test 1: Unauthenticated request triggers OIDC redirect
    req1 := httptest.NewRequest("GET", "/", nil)
    rec1 := httptest.NewRecorder()
    server.ServeHTTP(rec1, req1)
    assert.Equal(t, http.StatusFound, rec1.Code)
    assert.Contains(t, rec1.Header().Get("Location"), mockOIDC.URL)
    
    // Test 2: Callback with valid code creates session
    req2 := httptest.NewRequest("GET", "/auth/callback?code=abc123&state=valid_state", nil)
    rec2 := httptest.NewRecorder()
    server.ServeHTTP(rec2, req2)
    assert.Equal(t, http.StatusFound, rec2.Code)
    assert.NotEmpty(t, rec2.Header().Get("Set-Cookie"))
    
    // Test 3: Subsequent request with session cookie is authenticated
    req3 := httptest.NewRequest("GET", "/", nil)
    req3.Header.Set("Cookie", rec2.Header().Get("Set-Cookie"))
    rec3 := httptest.NewRecorder()
    server.ServeHTTP(rec3, req3)
    assert.Equal(t, http.StatusOK, rec3.Code)
    assert.NotEmpty(t, rec3.Header().Get("X-Es-Authorization"))
}
```

### Test Coverage Goals

- **Unit Test Coverage**: Minimum 80% code coverage
- **Property Test Coverage**: 100% of correctness properties implemented
- **Integration Test Coverage**: All major user flows covered
- **Edge Case Coverage**: All error conditions and boundary cases tested

### Continuous Integration

```yaml
# .github/workflows/test.yml
name: Test
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    services:
      redis:
        image: redis:7
        ports:
          - 6379:6379
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.26'
      - name: Run unit tests
        run: go test -v -race -coverprofile=coverage.txt ./...
      - name: Run property tests
        run: go test -v -tags=property ./...
      - name: Run integration tests
        run: go test -v -tags=integration ./integration/...
      - name: Check coverage
        run: go tool cover -func=coverage.txt
```

### Manual Testing Checklist

After implementation, manual testing should verify:

- [ ] OIDC flow with real OIDC provider (Okta, Auth0, etc.)
- [ ] Basic auth with real credentials
- [ ] Session persistence across requests
- [ ] Session expiration after TTL
- [ ] Logout clears session
- [ ] ForwardAuth mode with Traefik
- [ ] Auth_request mode with Nginx
- [ ] Standalone proxy mode with real upstream
- [ ] Redis session store with real Redis
- [ ] Configuration loading with environment variables
- [ ] Health check endpoint returns correct status
- [ ] Metrics endpoint exposes Prometheus metrics
- [ ] Graceful shutdown waits for in-flight requests
- [ ] Error messages are clear and actionable


## Implementation Notes

### Project Structure

```
keyline/
├── cmd/
│   └── keyline/
│       └── main.go                 # Entry point
├── internal/
│   ├── auth/
│   │   ├── engine.go               # Core auth engine
│   │   ├── oidc.go                 # OIDC provider
│   │   ├── basic.go                # Basic auth provider
│   │   └── provider.go             # AuthProvider interface
│   ├── session/
│   │   ├── manager.go              # Session manager
│   │   ├── store.go                # SessionStore interface
│   │   ├── store_redis.go          # Redis implementation
│   │   └── store_memory.go         # In-memory implementation
│   ├── state/
│   │   ├── store.go                # StateTokenStore interface
│   │   ├── store_redis.go          # Redis implementation
│   │   └── store_memory.go         # In-memory implementation
│   ├── mapper/
│   │   └── credentials.go          # ES credential mapper
│   ├── transport/
│   │   ├── adapter.go              # TransportAdapter interface
│   │   ├── forward_auth.go         # ForwardAuth adapter
│   │   └── standalone.go           # Standalone proxy adapter
│   ├── config/
│   │   ├── config.go               # Configuration types
│   │   ├── loader.go               # Configuration loader
│   │   └── validator.go            # Configuration validator
│   ├── cache/
│   │   └── oidc.go                 # OIDC cache (discovery, JWKS)
│   ├── observability/
│   │   ├── logger.go               # Structured logging
│   │   ├── metrics.go              # Prometheus metrics
│   │   └── tracing.go              # OpenTelemetry tracing
│   └── server/
│       └── server.go               # HTTP server
├── pkg/
│   └── crypto/
│       └── random.go               # Cryptographic random generation
├── integration/
│   ├── oidc_flow_test.go
│   ├── basic_auth_test.go
│   └── proxy_test.go
├── config/
│   └── config.example.yaml         # Example configuration
├── docs/
│   ├── deployment.md               # Deployment guide
│   ├── configuration.md            # Configuration reference
│   └── troubleshooting.md          # Troubleshooting guide
├── go.mod
├── go.sum
├── Dockerfile
├── Makefile
└── README.md
```

### Key Dependencies

```go
// go.mod
module github.com/example/keyline

go 1.26

require (
    github.com/coreos/go-oidc/v3 v3.10.0
    github.com/go-redis/redis/v8 v8.11.5
    github.com/labstack/echo/v4 v4.12.0
    github.com/spf13/viper v1.18.2
    go.opentelemetry.io/otel v1.24.0
    go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.24.0
    go.opentelemetry.io/otel/sdk v1.24.0
    github.com/prometheus/client_golang v1.19.0
    golang.org/x/crypto v0.21.0
    golang.org/x/oauth2 v0.18.0
    github.com/leanovate/gopter v0.2.9  // Property-based testing
)
```

### Reusing elastauth Components

Keyline reuses several components from elastauth:

1. **Echo Framework**: HTTP routing and middleware
2. **Viper Configuration**: Configuration loading with environment variable support
3. **Redis Client**: Connection pooling and operations
4. **AES Encryption**: Session data encryption (if needed)
5. **Structured Logging**: slog integration
6. **OpenTelemetry**: Tracing infrastructure

### Implementation Phases

#### Phase 1: Core Infrastructure (Week 1)
- Configuration loading and validation
- Structured logging setup
- Session store interfaces and implementations
- State token store interfaces and implementations
- Basic HTTP server with Echo

#### Phase 2: OIDC Authentication (Week 2)
- OIDC provider discovery
- Authorization flow with PKCE
- Token exchange and validation
- ID token signature verification
- Session creation

#### Phase 3: Basic Authentication (Week 2)
- Basic auth credential parsing
- Bcrypt password validation
- Local user lookup
- ES credential mapping

#### Phase 4: Transport Adapters (Week 3)
- ForwardAuth adapter (Traefik/Nginx)
- Standalone proxy adapter
- Header normalization
- Request/response preservation

#### Phase 5: Observability (Week 3)
- Prometheus metrics
- OpenTelemetry tracing
- Health check endpoint
- Structured logging enhancements

#### Phase 6: Testing & Documentation (Week 4)
- Unit tests
- Property-based tests
- Integration tests
- Documentation
- Deployment guides

### Security Considerations

1. **Cryptographic Randomness**: Always use `crypto/rand` for tokens and session IDs
2. **Timing-Safe Comparison**: Use `bcrypt.CompareHashAndPassword` for password validation
3. **Cookie Security**: Always set HttpOnly, Secure, SameSite=Lax
4. **HTTPS Enforcement**: Validate TLS certificates for OIDC provider
5. **Secret Management**: Never log or expose secrets in plaintext
6. **Input Validation**: Validate all configuration and request inputs
7. **Rate Limiting**: Consider adding rate limiting for authentication attempts
8. **CSRF Protection**: State tokens provide CSRF protection for OIDC flow

### Performance Considerations

1. **Connection Pooling**: Use connection pools for Redis and OIDC provider
2. **Caching**: Cache Discovery Document and JWKS in memory
3. **Concurrency**: Limit concurrent requests to prevent resource exhaustion
4. **Timeouts**: Set appropriate timeouts for all external requests
5. **Graceful Shutdown**: Wait for in-flight requests before terminating
6. **Memory Management**: Implement cleanup for expired sessions (in-memory store)

### Deployment Considerations

1. **Container Image**: Build minimal Docker image with multi-stage build
2. **Kubernetes**: Provide Helm chart with ConfigMap and Secret management
3. **Health Checks**: Configure liveness and readiness probes
4. **Resource Limits**: Set appropriate CPU and memory limits
5. **Horizontal Scaling**: Support multiple instances with Redis session store
6. **Secret Management**: Integrate with Vault for secret injection
7. **Monitoring**: Configure Prometheus scraping and Grafana dashboards
8. **Logging**: Configure log aggregation (ELK, Loki, etc.)

### Migration from Authelia + elastauth

1. **Configuration Mapping**: Map Authelia configuration to Keyline configuration
2. **Session Migration**: No automatic migration (users will need to re-authenticate)
3. **Deployment Strategy**: Blue-green deployment or canary rollout
4. **Rollback Plan**: Keep Authelia + elastauth running during migration
5. **Testing**: Thoroughly test all authentication flows before cutover

### Future Enhancements

Potential future enhancements (not in initial scope):

1. **SAML Support**: Add SAML authentication provider
2. **LDAP Support**: Add LDAP authentication provider
3. **Multi-Factor Authentication**: Add MFA support
4. **Rate Limiting**: Add per-user and per-IP rate limiting
5. **Audit Logging**: Add detailed audit logs for compliance
6. **Session Management UI**: Add web UI for session management
7. **Dynamic Configuration**: Support configuration reload without restart
8. **Custom Claims**: Support custom claim extraction and mapping

