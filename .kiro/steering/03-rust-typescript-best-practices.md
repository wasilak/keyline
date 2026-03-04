# Rust and TypeScript Best Practices for Cerebro Rewrite

## Rust Backend Best Practices

### Code Style and Formatting

#### Formatting
- Always run `cargo fmt` before committing
- Use `cargo clippy` to catch common mistakes and improve code quality
- Follow standard Rust formatting conventions (handled by rustfmt)

#### Error Handling
- Always handle errors explicitly, never use `.unwrap()` in production code
- Use `?` operator for error propagation
- Wrap errors with context using `anyhow` or custom error types
- Return `Result<T, E>` for fallible operations

```rust
// ✅ Good
pub async fn start(&self) -> Result<()> {
    self.init().await
        .context("failed to initialize server")?;
    Ok(())
}

// ❌ Bad
pub async fn start(&self) {
    self.init().await.unwrap(); // panics on error
}
```

#### Traits and Interfaces
- Keep traits small and focused
- Use trait bounds to constrain generic types
- Implement traits for types, not the other way around

```rust
// ✅ Good - small, focused trait
#[async_trait]
pub trait AuthProvider: Send + Sync {
    async fn get_user(&self, ctx: Context, req: &AuthRequest) -> Result<UserInfo>;
    fn provider_type(&self) -> &str;
}

// ❌ Bad - too many methods
#[async_trait]
pub trait AuthProvider {
    async fn get_user(&self, ctx: Context, req: &AuthRequest) -> Result<UserInfo>;
    fn provider_type(&self) -> &str;
    fn validate(&self) -> Result<()>;
    fn configure(&mut self, config: Config) -> Result<()>;
    async fn start(&self) -> Result<()>;
    async fn stop(&self) -> Result<()>;
    // ... 10 more methods
}
```

### Async/Await and Tokio

#### Async Functions
- Use `async fn` for asynchronous operations
- Use `tokio::spawn` for concurrent tasks
- Always handle task cancellation with `tokio::select!` or channels

```rust
// ✅ Good
pub async fn fetch_cluster_health(&self, cluster_id: &str) -> Result<ClusterHealth> {
    let client = self.get_client(cluster_id)?;
    
    tokio::time::timeout(
        Duration::from_secs(30),
        client.health()
    )
    .await
    .context("health check timeout")?
}

// ❌ Bad
pub async fn fetch_cluster_health(&self, cluster_id: &str) -> Result<ClusterHealth> {
    let client = self.get_client(cluster_id)?;
    client.health().await // no timeout, could hang forever
}
```

#### Concurrency
- Use `Arc` for shared ownership across threads
- Use `RwLock` or `Mutex` for interior mutability
- Prefer message passing (channels) over shared state when possible

```rust
// ✅ Good
pub struct SessionManager {
    sessions: Arc<RwLock<HashMap<String, Session>>>,
    config: SessionConfig,
}

impl SessionManager {
    pub async fn get_session(&self, token: &str) -> Option<Session> {
        let sessions = self.sessions.read().await;
        sessions.get(token).cloned()
    }
}

// ❌ Bad
pub struct SessionManager {
    sessions: HashMap<String, Session>, // not thread-safe
    config: SessionConfig,
}
```

### Project Structure for Cerebro Backend

```
backend/
├── src/
│   ├── main.rs              # Entry point, server setup
│   ├── config/              # Configuration management
│   │   ├── mod.rs
│   │   └── types.rs
│   ├── auth/                # Authentication
│   │   ├── mod.rs
│   │   ├── session.rs
│   │   ├── local.rs
│   │   └── oidc.rs
│   ├── cluster/             # Cluster management
│   │   ├── mod.rs
│   │   ├── manager.rs
│   │   └── client.rs
│   ├── routes/              # API routes
│   │   ├── mod.rs
│   │   ├── auth.rs
│   │   └── clusters.rs
│   └── lib.rs               # Library exports
├── tests/                   # Integration tests
└── Cargo.toml
```

### Dependencies Management

#### Minimal Dependencies
- Carefully evaluate third-party crates
- Pin dependency versions in `Cargo.toml`
- Use `cargo audit` to check for security vulnerabilities

#### Common Cerebro Dependencies
```toml
[dependencies]
# Web framework
axum = "0.7"
tokio = { version = "1", features = ["full"] }

# Serialization
serde = { version = "1", features = ["derive"] }
serde_json = "1"

# Configuration
config = "0.14"

# Error handling
anyhow = "1"
thiserror = "1"

# Logging
tracing = "0.1"
tracing-subscriber = "0.3"

# Elasticsearch client
elasticsearch = "8"

# Embedded assets
rust-embed = "8"
```

### Testing

#### Unit Tests
```rust
#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_session_creation() {
        let manager = SessionManager::new(SessionConfig::default());
        let user = AuthUser {
            id: "test".to_string(),
            username: "testuser".to_string(),
            roles: vec!["admin".to_string()],
        };
        
        let token = manager.create_session(user).await.unwrap();
        assert!(!token.is_empty());
    }
}
```

#### Property-Based Tests
```rust
#[cfg(test)]
mod property_tests {
    use proptest::prelude::*;

    proptest! {
        #[test]
        fn test_config_override_precedence(
            file_value in any::<String>(),
            env_value in any::<String>()
        ) {
            // Property: env vars always override file config
            let config = load_config_with_overrides(file_value, env_value);
            assert_eq!(config.value, env_value);
        }
    }
}
```

## TypeScript/React Frontend Best Practices

### Code Style and Formatting

#### TypeScript Configuration
- Use strict mode in `tsconfig.json`
- Enable all strict type checking options
- Use ESLint and Prettier for code quality

```json
{
  "compilerOptions": {
    "strict": true,
    "noImplicitAny": true,
    "strictNullChecks": true,
    "noUnusedLocals": true,
    "noUnusedParameters": true
  }
}
```

#### Type Safety
- Always define types for props, state, and API responses
- Use interfaces for object shapes
- Use type aliases for unions and complex types
- Avoid `any` type

```typescript
// ✅ Good
interface ClusterInfo {
  id: string;
  name: string;
  health: 'green' | 'yellow' | 'red';
  nodes: number;
}

function ClusterCard({ cluster }: { cluster: ClusterInfo }): JSX.Element {
  return <div>{cluster.name}</div>;
}

// ❌ Bad
function ClusterCard({ cluster }: { cluster: any }): JSX.Element {
  return <div>{cluster.name}</div>;
}
```

### React Best Practices

#### Component Structure
- Use functional components with hooks
- Keep components small and focused
- Extract reusable logic into custom hooks
- Use TypeScript for prop types

```typescript
// ✅ Good
interface DashboardProps {
  refreshInterval?: number;
}

export function Dashboard({ refreshInterval = 30000 }: DashboardProps): JSX.Element {
  const { clusters, loading, error } = useClusters(refreshInterval);
  
  if (loading) return <Spinner />;
  if (error) return <ErrorMessage error={error} />;
  
  return <ClusterTable clusters={clusters} />;
}

// ❌ Bad
export function Dashboard(props: any): JSX.Element {
  const [clusters, setClusters] = useState([]);
  const [loading, setLoading] = useState(true);
  
  useEffect(() => {
    // Inline data fetching logic (should be in a hook)
    fetch('/api/clusters')
      .then(res => res.json())
      .then(data => {
        setClusters(data);
        setLoading(false);
      });
  }, []);
  
  return <div>{/* ... */}</div>;
}
```

#### Custom Hooks
- Extract reusable logic into custom hooks
- Name hooks with `use` prefix
- Return objects for multiple values

```typescript
// ✅ Good
export function useClusters(refreshInterval: number) {
  const [clusters, setClusters] = useState<ClusterInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);
  
  useEffect(() => {
    const fetchClusters = async () => {
      try {
        const data = await apiClient.getClusters();
        setClusters(data);
      } catch (err) {
        setError(err as Error);
      } finally {
        setLoading(false);
      }
    };
    
    fetchClusters();
    const interval = setInterval(fetchClusters, refreshInterval);
    return () => clearInterval(interval);
  }, [refreshInterval]);
  
  return { clusters, loading, error };
}
```

### State Management

#### Zustand for Global State
```typescript
// ✅ Good
interface ThemeStore {
  theme: Theme;
  setTheme: (theme: Theme) => void;
}

export const useThemeStore = create<ThemeStore>((set) => ({
  theme: 'system',
  setTheme: (theme) => set({ theme }),
}));
```

#### TanStack Query for Server State
```typescript
// ✅ Good
export function useClusterHealth(clusterId: string) {
  return useQuery({
    queryKey: ['cluster', clusterId, 'health'],
    queryFn: () => apiClient.getClusterHealth(clusterId),
    refetchInterval: 30000,
  });
}
```

### Project Structure for Cerebro Frontend

```
frontend/
├── src/
│   ├── main.tsx             # Entry point
│   ├── App.tsx              # Root component
│   ├── components/          # React components
│   │   ├── Dashboard.tsx
│   │   ├── ClusterView.tsx
│   │   └── RestConsole.tsx
│   ├── hooks/               # Custom hooks
│   │   ├── useTheme.ts
│   │   ├── usePreferences.ts
│   │   └── useClusters.ts
│   ├── api/                 # API client
│   │   └── client.ts
│   ├── types/               # TypeScript types
│   │   └── index.ts
│   └── styles/              # Global styles
├── public/                  # Static assets
└── package.json
```

### Testing

#### Component Tests (Vitest + React Testing Library)
```typescript
import { render, screen } from '@testing-library/react';
import { Dashboard } from './Dashboard';

describe('Dashboard', () => {
  it('renders cluster list', () => {
    render(<Dashboard />);
    expect(screen.getByText('Clusters')).toBeInTheDocument();
  });
  
  it('displays loading state', () => {
    render(<Dashboard />);
    expect(screen.getByRole('progressbar')).toBeInTheDocument();
  });
});
```

#### Property-Based Tests (fast-check)
```typescript
import fc from 'fast-check';

describe('REST Console Parser', () => {
  it('should parse any valid HTTP method and path', () => {
    fc.assert(
      fc.property(
        fc.constantFrom('GET', 'POST', 'PUT', 'DELETE'),
        fc.string(),
        (method, path) => {
          const input = `${method} ${path}`;
          const result = parseRequest(input);
          expect(result.method).toBe(method);
          expect(result.path).toBe(path);
        }
      )
    );
  });
});
```

## Common Pitfalls to Avoid

### Rust
1. **Using `.unwrap()` in production**: Always handle errors properly
2. **Blocking in async code**: Use async versions of blocking operations
3. **Not using `Arc` for shared state**: Required for thread-safe sharing
4. **Ignoring clippy warnings**: Clippy catches many common mistakes
5. **Not handling cancellation**: Always provide a way to stop async tasks

### TypeScript/React
1. **Using `any` type**: Defeats the purpose of TypeScript
2. **Not memoizing expensive computations**: Use `useMemo` and `useCallback`
3. **Prop drilling**: Use context or state management instead
4. **Not cleaning up effects**: Always return cleanup functions from `useEffect`
5. **Mutating state directly**: Always use setState or state management setters

## Cerebro-Specific Guidelines

### Backend
- Use `tracing` for structured logging with context
- Implement health check endpoints for Kubernetes
- Use `rust-embed` for embedding frontend assets
- Support graceful shutdown with signal handling
- Validate all configuration at startup

### Frontend
- Use Mantine UI components consistently
- Implement dark/light theme support
- Store preferences in localStorage
- Handle API errors gracefully with user-friendly messages
- Implement loading states for all async operations
