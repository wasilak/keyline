# Go Naming Conventions (Keyline)

## Package Names
- Short, lowercase, no underscores
- No unnecessary prefixes
- Examples: `auth`, `session`, `config`, `transport`, `elasticsearch`, `usermgmt`, `cache`, `server`, `observability`, `state`

## Type Names (Structs, Interfaces)
- **Exported types**: PascalCase
  - Examples: `SessionManager`, `AuthProvider`, `OIDCConfig`, `Server`, `Client`, `Provider`, `Manager`
- **Unexported types**: camelCase
  - Examples: `sessionManager`, `authProvider`, `oidcConfig`
- **Interface names**: Describe capability (no `Interface` suffix)
  - Examples: `AuthProvider`, `SessionStore`, `StateTokenStore`, `CacheBackend`, `ElasticsearchClient`

## Function and Method Names
- **Exported functions**: camelCase
  - Examples: `NewServer`, `ValidateConfig`, `Start`, `Shutdown`
- **Unexported functions**: camelCase
  - Examples: `validateConfig`, `generateState`, `parseGroups`
- **Constructor naming**: `New[Typename]` for exported constructors
  - Examples: `NewServer`, `NewOIDCProvider`, `NewSessionManager`
- **Method naming**: Descriptive verbs
  - Examples: `Authenticate()`, `CreateSession()`, `ValidateToken()`, `GetUser()`

## Constants
- **Exported constants**: PascalCase
  - Examples: `DefaultPort`, `MaxConnections`, `SessionTimeout`, `DefaultCookieName`
- **Unexported constants**: camelCase
  - Examples: `defaultPort`, `maxConnections`, `sessionTimeout`

## Avoid Stuttering
❌ Bad | ✅ Good
-------|-------
`auth.AuthProvider` | `auth.Provider`
`session.SessionManager` | `session.Manager`
`config.ConfigLoader` | `config.Loader`
`usermgmt.UserManager` | `usermgmt.Manager`
`elasticsearch.ElasticsearchClient` | `elasticsearch.Client`

## Project-Specific Types (Keyline)

### Auth Types
- `Provider` (interface for auth providers)
- `SessionManager` (session state management)
- `User` (authenticated user information)

### Config Types
- `ServerConfig` (server settings)
- `OIDCConfig` (OIDC provider configuration)
- `SessionConfig` (session settings)
- `CacheConfig` (cache backend settings)

### Transport Types
- `Adapter` (HTTP adapter interface)
- `ForwardAuthAdapter` (Traefik forwardAuth mode)
- `StandaloneProxyAdapter` (standalone proxy mode)

### User Management
- `Manager` (user management interface)
- `PasswordGenerator` (password generation utilities)
- `RoleMapper` (role claim to ES role mapping)
- `Encryptor` (credential encryption/decryption)

## Examples

### ✅ Good - Proper Go Naming

```go
// internal/auth/provider.go
type Provider interface {
    Authenticate(ctx context.Context, req *Request) (*Result, error)
    Type() string
}

type Factory struct {
    providers map[string]Provider
}

func (f *Factory) Create(providerType string, cfg Config) (Provider, error) {
    // ...
}

// internal/cache/backend.go
type Backend interface {
    Get(key string) ([]byte, error)
    Set(key string, value []byte, ttl time.Duration) error
    Delete(key string) error
}

// internal/usermgmt/types.go
type Manager interface {
    CreateUser(ctx context.Context, user *User) error
    GetUser(ctx context.Context, username string) (*User, error)
    UpdateUser(ctx context.Context, user *User) error
    DeleteUser(ctx context.Context, username string) error
}
```

### ❌ Bad - Stuttering/Incorrect Naming

```go
// ❌ Stuttering: auth.AuthProvider
type AuthProvider interface { ... }

// ❌ Unnecessary suffix: ConfigInterface
type ConfigInterface interface { ... }

// ❌ Non-descriptive: Handler
type Handler struct { ... }

// ❌ Bad method name: Do
func (s *Server) Do() error { ... }
```

## Why This Matters

1. **Language Idioms**: Follows standard Go conventions
2. **Readability**: Shorter, clearer names
3. **Maintainability**: Less coupling to specific implementations
4. **Community**: Makes projects more approachable for contributors
5. **Professionalism**: Shows understanding of Go best practices
