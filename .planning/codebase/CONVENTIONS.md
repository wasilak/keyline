# Coding Conventions
_Generated: 2026-05-16 | Focus: quality_

## Summary
Keyline is a Go project following idiomatic Go style enforced by `gofmt` and `go vet` (no golangci-yml — only the standard toolchain is configured in `Taskfile.yml`). Error handling uses sentinel `fmt.Errorf` without wrapping for user-facing messages and `%w` wrapping for internal propagation. Structured logging via `log/slog` with context propagation is the universal logging pattern.

---

## Code Style and Formatting

**Formatter:** `gofmt` + `goimports` (run via `task format`).

**Linter:** `go vet ./...` only. No `.golangci.yml` exists in the repo. Linter directives (`//nolint:errcheck`, `//nolint:gosec`) appear in exactly 3 places in `internal/auth/ldap.go` with inline justification comments.

**Line length / style:** No explicit limit; files follow gofmt defaults. All source files in `internal/` are gofmt-clean.

---

## Naming Conventions

**Packages:** Short, lowercase, single-word. Examples: `auth`, `config`, `session`, `usermgmt`, `observability`, `elasticsearch`.

**Types (exported):** PascalCase structs with a noun or noun-phrase name.
```go
type LDAPProvider struct { ... }        // internal/auth/ldap.go
type AuthResult struct { ... }          // internal/auth/basic.go
type EngineResult struct { ... }        // internal/auth/engine.go
type RoleMapping struct { ... }         // internal/config/config.go
```

**Constructor functions:** `New<Type>` returning `(*Type, error)`.
```go
func NewLDAPProvider(cfg *config.LDAPConfig) (*LDAPProvider, error)
func NewBasicAuthProvider(cfg *config.LocalUsersConfig) (*BasicAuthProvider, error)
func NewEngine(cfg *config.Config, ...) (*Engine, error)
```

**Methods:** camelCase verbs. Receiver name is a short abbreviation of the type (`p` for providers, `e` for Engine, `rm` for RoleMapper).
```go
func (p *LDAPProvider) Authenticate(ctx context.Context, req *AuthRequest) *AuthResult
func (e *Engine) authenticateWithBasicAuth(ctx context.Context, req *EngineRequest) *EngineResult
func (rm *RoleMapper) matchPattern(value, pattern string) bool
```

**Constants:** camelCase with descriptive prefix for grouping.
```go
const (
    ldapDefaultConnectionTimeout    = 10 * time.Second
    ldapDefaultUsernameAttribute    = "sAMAccountName"
    ldapDefaultEmailAttribute       = "mail"
)
```

**Variables:** camelCase. Short names only in tight local scopes (`g`, `b`, `r`). Descriptive names for anything wider in scope.

**Config struct fields:** PascalCase with `mapstructure` tag using `snake_case` for YAML keys.
```go
type LDAPConfig struct {
    TLSSkipVerify bool   `mapstructure:"tls_skip_verify"`
    SearchBase    string `mapstructure:"search_base"`
}
```

---

## Configuration Pattern

Config is a single flat struct tree in `internal/config/config.go`. All subsections are embedded by value (not pointer) in the top-level `Config` struct:

```go
type Config struct {
    Server         ServerConfig        `mapstructure:"server"`
    LDAP           LDAPConfig          `mapstructure:"ldap"`
    Elasticsearch  ElasticsearchConfig `mapstructure:"elasticsearch"`
    // ...
}
```

**Loading:** `config.Load(configFile string) (*Config, error)` in `internal/config/loader.go`. Uses `github.com/spf13/viper` to read YAML, then `v.Unmarshal(&cfg)`. After unmarshal, `substituteEnvVars(&cfg)` does `${VAR_NAME}` expansion field-by-field (explicit, not reflection-based).

**Secret injection pattern:** Sensitive fields in YAML use `${ENV_VAR}` placeholders, never literal values. `ldap.bind_password` enforces this at provider construction time:
```go
if !strings.HasPrefix(cfg.BindPassword, "${") {
    return nil, fmt.Errorf("ldap.bind_password must be an environment variable reference in the form ${ENV_VAR}")
}
```

**Defaults for optional fields:** Applied in `New*` constructors, not in the config loader:
```go
if cfg.UsernameAttribute == "" {
    cfg.UsernameAttribute = ldapDefaultUsernameAttribute
}
```

---

## Error Handling Patterns

Two distinct error-handling modes exist side by side:

**1. Wrapping with `%w` (internal propagation):**
Used when the error crosses layer boundaries — caller can inspect or unwrap.
```go
// internal/config/loader.go
if err := v.ReadInConfig(); err != nil {
    return nil, fmt.Errorf("failed to read config file: %w", err)
}

// internal/auth/ldap.go
return "", "", "", "", fmt.Errorf("LDAP user search error: %w", err)
```

**2. Plain `fmt.Errorf` (user-facing / sanitized):**
Used when the raw error would leak internal details. The original error is logged, a clean message is returned.
```go
// internal/auth/ldap.go
slog.ErrorContext(ctx, "LDAP connection failed", slog.String("error", err.Error()))
return &AuthResult{Authenticated: false, Error: fmt.Errorf("LDAP connection failed")}
```

**3. Result structs carry errors (not error returns):**
`Authenticate()` returns `*AuthResult` with an embedded `Error` field — never a second return value. This is consistent across all auth providers.
```go
type AuthResult struct {
    Authenticated bool
    Username      string
    Error         error   // non-nil on failure
}
```

**4. Constructor errors use standard `(T, error)` return:**
```go
func NewLDAPProvider(cfg *config.LDAPConfig) (*LDAPProvider, error)
```

**Ignored errors:** Explicitly suppressed with `//nolint:errcheck` and a comment explaining why (e.g., `defer conn.Close()` in cleanup paths).

---

## Logging

**Library:** Standard library `log/slog` (Go 1.21+). Imported as `"log/slog"`. No wrapper or custom logger type.

**Context propagation:** All log calls use `slog.InfoContext`, `slog.WarnContext`, `slog.ErrorContext`, `slog.DebugContext`. Bare `slog.Info` only in top-level server startup (`server.go`).

**Structured fields:** Key-value pairs passed as `slog.String(key, val)`, `slog.Any(key, val)`, `slog.Int(key, val)` — not as bare `interface{}` pairs.

```go
slog.InfoContext(ctx, "LDAP authentication successful",
    slog.String("username", username),
    slog.Any("groups", groups),
)

slog.WarnContext(ctx, "LDAP user search failed",
    slog.String("username", username),
    slog.String("error", err.Error()),
)
```

**Log levels and semantics:**
- `Info`: Normal auth flow events (attempt started, success)
- `Warn`: User-caused failures (wrong password, user not found, not in required groups, group search failure)
- `Error`: Infrastructure/system failures (LDAP connection down, service account bind fail, bcrypt error)
- `Debug`: Cache misses, absent cookies, routine negative checks

**Security in logs:** `observability.HashSessionID()` in `internal/observability/logging.go` hashes session IDs before logging. Passwords are never logged.

**HTTP request logging:** Provided by `slogecho` middleware (`github.com/samber/slog-echo`) wired in `internal/server/server.go`.

---

## Comment Style

**Package-level and exported types:** Short doc comments on every exported symbol.
```go
// LDAPProvider implements Basic Auth via an LDAP/Active Directory server.
type LDAPProvider struct { ... }

// NewLDAPProvider creates a new LDAPProvider from the given config.
// Returns an error if the config is disabled or missing required fields.
func NewLDAPProvider(cfg *config.LDAPConfig) (*LDAPProvider, error) {
```

**Inline section separators:** Multi-step functions use numbered section banners for readability:
```go
// --- 1. Extract credentials from the Basic Auth header ---
// --- 2. Connect to LDAP server ---
// --- 3. Service account bind ---
```

**Unexported helpers:** Short single-line comments explain _why_, not _what_:
```go
// ldapConn abstracts *ldap.Conn for testability.
type ldapConn interface { ... }

// dialFn is swapped in tests to inject a mock connection.
dialFn func(cfg *config.LDAPConfig) (ldapConn, error)
```

**Inline comments:** Used for non-obvious decisions or suppressed linter warnings with justification:
```go
conn.Close() //nolint:errcheck
InsecureSkipVerify: cfg.TLSSkipVerify, //nolint:gosec // controlled by operator config
```

---

## Import Organization

Three groups separated by blank lines:
1. Standard library
2. Third-party packages
3. Internal packages (`github.com/yourusername/keyline/internal/...`)

```go
import (
    "context"
    "fmt"
    "log/slog"

    "github.com/go-ldap/ldap/v3"

    "github.com/yourusername/keyline/internal/config"
)
```

---

## Interface Design for Testability

Dependencies that need mocking are expressed as interfaces defined in the consuming package, not the providing package:

```go
// internal/auth/ldap.go — consumer defines the interface
type ldapConn interface {
    Bind(username, password string) error
    Search(searchRequest *ldap.SearchRequest) (*ldap.SearchResult, error)
    SetTimeout(timeout time.Duration)
    Close() error
}
```

The concrete implementation (`*ldap.Conn`) satisfies the interface implicitly. The provider stores a `dialFn` function field so tests can inject a mock connection without any global state.

---

*Convention analysis: 2026-05-16*
