# Naming Conventions and Brand Agnosticism

## Brand-Agnostic Code

Whatever project name we settle on, code MUST be written in a way that is **name-agnostic**. All projects (Keyline, Cerebro, etc.) should follow this principle.

### ✅ Allowed Uses of Project Name

- **Documentation**: README, specs, comments, user-facing docs
- **Package/Crate/Module name**: `keyline` in go.mod, `cerebro` in Cargo.toml, `@cerebro/frontend` in package.json
- **Binary name**: `keyline` or `cerebro` executable
- **Repository name**: GitHub repo name
- **User-facing messages**: CLI help text, startup messages, log messages

### ❌ NOT Allowed in Code

- **Function names**: ❌ `func keylineStart()` / `fn cerebro_start()` → ✅ `func start()` / `fn start()`
- **Struct names**: ❌ `type KeylineServer struct` / `struct CerebroServer` → ✅ `type Server struct` / `struct Server`
- **Variable names**: ❌ `keylineConfig` / `cerebro_config` → ✅ `config`
- **Trait/Interface names**: ❌ `KeylineProvider` / `CerebroProvider` → ✅ `AuthProvider`
- **Method names**: ❌ `initKeyline()` / `init_cerebro()` → ✅ `init()`
- **Constants**: ❌ `KEYLINE_VERSION` / `CEREBRO_VERSION` → ✅ `VERSION`
- **Component names**: ❌ `KeylineDashboard` / `CerebroDashboard` → ✅ `Dashboard`

## Go Naming Conventions (Keyline, etc.)

Follow standard Go naming conventions:

### Package Names
- Short, lowercase, no underscores
- No unnecessary prefixes
- Examples: `auth`, `session`, `config`, `transport`
- Current structure: `internal/auth/`, `internal/session/`, `pkg/crypto/`

### Type Names (Structs, Interfaces)
- Use PascalCase for exported types
- Use camelCase for unexported types
- Examples: `SessionManager`, `AuthProvider`, `OIDCConfig`
- Interface names: `AuthProvider`, `SessionStore`, `StateTokenStore`

### Function and Method Names
- Use camelCase for exported functions
- Use camelCase for unexported functions
- Examples: `NewServer`, `validateConfig`, `generateState`

### Constants
- Use PascalCase for exported constants
- Use camelCase for unexported constants
- Examples: `DefaultPort`, `MaxConnections`, `SessionTimeout`

### Avoid Stuttering
- ❌ `auth.AuthProvider` → ✅ `auth.Provider`
- ❌ `session.SessionManager` → ✅ `session.Manager`
- ❌ `config.ConfigLoader` → ✅ `config.Loader`

## Rust Naming Conventions for Cerebro Backend

Follow standard Rust naming conventions:

### Module Names
- Short, lowercase, snake_case
- No unnecessary prefixes
- Examples: `cluster`, `auth`, `session`, `config`
- Current structure: `backend/src/cluster/`, `backend/src/auth/`

### Type Names (Structs, Enums, Traits)
- Use PascalCase for type names
- Examples: `ClusterManager`, `AuthProvider`, `SessionManager`
- Trait names: `AuthProvider`, `CacheInterface`, `ElasticsearchClient`

### Function and Method Names
- Use snake_case for functions and methods
- Examples: `parse_groups`, `validate_config`, `init_provider`

### Constants and Statics
- Use SCREAMING_SNAKE_CASE
- Examples: `DEFAULT_PORT`, `MAX_CONNECTIONS`, `SESSION_TIMEOUT`

### Avoid Stuttering
- ❌ `cluster::ClusterManager` → ✅ `cluster::Manager`
- ❌ `auth::AuthProvider` → ✅ `auth::Provider`
- ❌ `session::SessionManager` → ✅ `session::Manager`

## TypeScript/React Naming Conventions for Cerebro Frontend

Follow standard TypeScript and React conventions:

### Component Names
- Use PascalCase for React components
- Examples: `Dashboard`, `ClusterView`, `RestConsole`
- Not: `CerebroDashboard`, `CerebroClusterView`

### Hook Names
- Start with `use` prefix
- Use camelCase
- Examples: `useTheme`, `usePreferences`, `useClusterHealth`
- Not: `useCerebroTheme`, `useCerebroPreferences`

### Function Names
- Use camelCase for functions
- Examples: `parseRequest`, `validateJson`, `formatBytes`

### Interface and Type Names
- Use PascalCase for interfaces and types
- Examples: `ClusterInfo`, `UserPreferences`, `ApiClient`
- Not: `CerebroClusterInfo`, `CerebroUserPreferences`

### File Names
- Use kebab-case for file names
- Examples: `cluster-view.tsx`, `rest-console.tsx`, `api-client.ts`
- Not: `cerebro-cluster-view.tsx`, `CerebroRestConsole.tsx`

## Examples

### ✅ Good - Brand Agnostic (Go - Keyline)

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
```

### ❌ Bad - Brand Specific (Go - Keyline)

```go
type KeylineAuthProvider interface {
    AuthenticateKeyline(ctx context.Context, req *KeylineRequest) (*KeylineResult, error)
    GetKeylineType() string
}

type KeylineProviderFactory struct {
    keylineProviders map[string]KeylineAuthProvider
}
```

### ✅ Good - Brand Agnostic (Rust Backend)

```rust
// backend/src/auth/mod.rs
pub trait AuthProvider {
    async fn get_user(&self, ctx: Context, req: &AuthRequest) -> Result<UserInfo>;
    fn provider_type(&self) -> &str;
}

pub struct Factory {
    providers: HashMap<String, Box<dyn AuthProvider>>,
}

impl Factory {
    pub fn create(&self, provider_type: &str, config: Config) -> Result<Box<dyn AuthProvider>> {
        // ...
    }
}
```

### ❌ Bad - Brand Specific (Rust Backend)

```rust
pub trait CerebroAuthProvider {
    async fn get_cerebro_user(&self, ctx: Context, req: &CerebroRequest) -> Result<CerebroUserInfo>;
    fn get_cerebro_type(&self) -> &str;
}

pub struct CerebroProviderFactory {
    cerebro_providers: HashMap<String, Box<dyn CerebroAuthProvider>>,
}
```

### ✅ Good - Brand Agnostic (TypeScript Frontend)

```typescript
// frontend/src/components/Dashboard.tsx
interface ClusterSummary {
  id: string;
  name: string;
  health: 'green' | 'yellow' | 'red';
}

export function Dashboard(): JSX.Element {
  const clusters = useClusters();
  return <ClusterTable clusters={clusters} />;
}

// frontend/src/hooks/useTheme.ts
export function useTheme(): ThemeContextValue {
  const context = useContext(ThemeContext);
  return context;
}
```

### ❌ Bad - Brand Specific (TypeScript Frontend)

```typescript
interface CerebroClusterSummary {
  cerebroId: string;
  cerebroName: string;
  cerebroHealth: 'green' | 'yellow' | 'red';
}

export function CerebroDashboard(): JSX.Element {
  const cerebroClusters = useCerebroClusters();
  return <CerebroClusterTable clusters={cerebroClusters} />;
}
```

## Project-Specific Naming Guidelines

### Keyline (Go Authentication Proxy)
- Auth types: `Provider`, `SessionManager`, `User`
- Config types: `ServerConfig`, `OIDCConfig`, `SessionConfig`
- Transport types: `Adapter`, `ForwardAuthAdapter`, `StandaloneProxyAdapter`
- Not: `KeylineProvider`, `KeylineSessionManager`, `KeylineAdapter`

### Cerebro (Rust/TypeScript)
#### Backend (Rust)
- Cluster types: `ClusterManager`, `ClusterConnection`, `ClusterHealth`
- Auth types: `AuthProvider`, `SessionManager`, `AuthUser`
- Config types: `ServerConfig`, `AuthConfig`, `ClusterConfig`
- Not: `CerebroClusterManager`, `CerebroAuthProvider`

#### Frontend (TypeScript/React)
- Components: `Dashboard`, `ClusterView`, `RestConsole`, `ThemeProvider`
- Hooks: `useTheme`, `usePreferences`, `useClusters`, `useApiClient`
- Types: `ClusterInfo`, `UserPreferences`, `ApiClient`
- Not: `CerebroDashboard`, `useCerebroTheme`, `CerebroClusterInfo`

## Why This Matters

1. **Reusability**: Code can be forked/reused without renaming everything
2. **Clarity**: Shorter names are easier to read and understand
3. **Language Idioms**: Follows standard Go, Rust, and TypeScript conventions
4. **Maintainability**: Less coupling to brand name
5. **Professionalism**: Shows understanding of proper design patterns
6. **Community**: Makes projects more approachable for contributors

## Documentation is Different

In documentation, user-facing messages, and comments, using the project name is fine:

```go
// Provider implements the Keyline authentication provider interface
type Provider interface {
    // ...
}

func main() {
    fmt.Println("Keyline - Authentication Proxy for Elasticsearch")
    // ...
}
```

```rust
/// AuthProvider implements the Cerebro authentication provider interface
pub trait AuthProvider {
    // ...
}

fn main() {
    println!("Cerebro - Elasticsearch Web Admin Tool");
    // ...
}
```

```typescript
/**
 * Dashboard component displays all configured Cerebro clusters
 */
export function Dashboard(): JSX.Element {
  // ...
}
```

The key is: **code structure and naming should be generic, documentation can be branded**.