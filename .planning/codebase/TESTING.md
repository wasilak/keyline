# Testing Patterns
_Generated: 2026-05-16 | Focus: quality_

## Summary
Keyline uses `go test` + `testify` for all testing, with two distinct layers: unit tests co-located with source and integration tests in a separate `integration/` directory gated by a build tag. Mocking is done via hand-written interface implementations with injectable function fields — no third-party mock generators.

## Framework & Tooling
- **Test runner:** `go test ./...` or `task test`
- **Assertions:** `testify/assert` (non-fatal) and `testify/require` (fatal on failure)
- **Integration gate:** `//go:build integration` build tag on files in `integration/`
- **Coverage:** no enforcement thresholds configured

## Test Layers

### Unit Tests
- Located in `internal/**/*_test.go` alongside source files
- ~193 test functions across 13 files
- Run without any build tags: `go test ./...`

### Integration Tests
- Located in `integration/` directory
- Gated by `//go:build integration`
- Run separately: `go test -tags integration ./integration/...`

## Test Styles

### Flat Per-Scenario Functions (auth layer)
Used in `internal/auth/` — each scenario gets its own `TestXxx_scenarioName` function:
```go
func TestLDAPProvider_Authenticate_success(t *testing.T) { ... }
func TestLDAPProvider_Authenticate_wrongPassword(t *testing.T) { ... }
func TestLDAPProvider_Authenticate_userNotFound(t *testing.T) { ... }
```

### Table-Driven Tests (config, ES client)
Used in `internal/config/` and `internal/esclient/`:
```go
func TestParseConfig(t *testing.T) {
    cases := []struct {
        name  string
        input string
        want  Config
    }{ ... }
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) { ... })
    }
}
```

### Property-Invariant Tests (role mapper)
Used in `internal/roles/` to verify mapping invariants hold across arbitrary inputs.

## Mocking Approach
Hand-written interface implementations with injectable function fields — no mock generator used.

```go
type mockLDAPConn struct {
    bindFn   func(username, password string) error
    searchFn func(req *ldap.SearchRequest) (*ldap.SearchResult, error)
}

func (m *mockLDAPConn) Bind(u, p string) error { return m.bindFn(u, p) }
func (m *mockLDAPConn) Search(r *ldap.SearchRequest) (*ldap.SearchResult, error) { return m.searchFn(r) }
```

- **LDAP provider:** injectable `dialFn` on the provider struct; tests swap it for one returning a `mockLDAPConn`
- **Elasticsearch client:** `httptest.NewServer` stands in for the ES cluster
- **Session cache:** real in-memory cache instance (no mock needed)

## Fixture Helpers
Defined in the same `_test.go` file as the tests that use them:
```go
func validLDAPConfig() LDAPConfig { ... }
func basicAuthHeader(user, pass string) string { ... }
func userSearchResult(dn string, attrs map[string][]string) *ldap.SearchResult { ... }
```

- `t.TempDir()` for ephemeral config files
- `defer os.Unsetenv("KEY")` pattern for env var cleanup

## Coverage Gaps
- No `TestMain` for shared setup/teardown
- No shared `testutil` package — helpers are local to each package
- No fuzz tests
- No coverage threshold enforcement
- Group search and TLS mode variants may have incomplete coverage
