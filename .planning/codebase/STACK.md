# Technology Stack
_Generated: 2026-05-16 | Focus: tech_

## Summary

Keyline is a Go-based authentication proxy that sits in front of Elasticsearch/Kibana. It is built as a single statically-linked binary with no CGO dependencies, packaged in a two-stage Docker image. The documentation site is a separate Docusaurus project under `docs/`.

---

## Languages

**Primary:**
- Go 1.26 — all application code under `internal/`, `cmd/`, `pkg/`, `integration/`

**Secondary:**
- JavaScript/TypeScript — documentation site only (`docs/`)

---

## Runtime

**Environment:**
- Go 1.26 (declared in `go.mod`, pinned in `Dockerfile` as `golang:1.26-alpine`)
- CGO disabled (`CGO_ENABLED=0`) — fully static binary

**Package Manager:**
- Go modules (`go.mod` / `go.sum`)
- npm (docs only, Node.js ≥24 required per `docs/package.json`)

**Lockfile:**
- `go.sum` — present and committed

---

## Frameworks

**Core HTTP:**
- `github.com/labstack/echo/v4` v4.15.1 — HTTP server framework for routing and middleware

**Configuration:**
- `github.com/spf13/viper` v1.21.0 — config loading from YAML + env var substitution (`${VAR}` syntax)

**Testing:**
- `github.com/stretchr/testify` v1.11.1 — assertions and test helpers
- Standard `testing` package — unit and integration tests

**Build/Dev:**
- `task` (Taskfile v3) — primary build runner; see `Taskfile.yml`
- `gofmt` + `goimports` — formatting (enforced in CI)
- `go vet` — static analysis (enforced in CI)

**Documentation:**
- Docusaurus v3.7.0 with `@docusaurus/theme-mermaid` — static site at `docs/`

---

## Key Dependencies

**HTTP server:**
- `github.com/labstack/echo/v4` v4.15.1 — routing, middleware, request/response handling

**LDAP:**
- `github.com/go-ldap/ldap/v3` v3.4.13 — LDAP/Active Directory authentication (`internal/auth/ldap.go`)

**OIDC / JWT:**
- `gopkg.in/square/go-jose.v2` v2.6.0 — JWT signature verification for OIDC ID tokens (`internal/auth/oidc.go`)

**Cache abstraction:**
- `github.com/wasilak/cachego` v0.0.11 — unified Redis + in-memory cache interface (`internal/cache/cache.go`)
  - Backed by `github.com/redis/go-redis/v9` v9.18.0 (Redis) and `github.com/dgraph-io/badger/v4` v4.9.1 + `github.com/patrickmn/go-cache` (memory)

**Observability:**
- `github.com/prometheus/client_golang` v1.23.2 — Prometheus metrics exposition (`internal/observability/metrics.go`)
- `go.opentelemetry.io/otel` v1.42.0 — OpenTelemetry tracing core
- `go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho` v0.67.0 — Echo tracing middleware
- `github.com/wasilak/otelgo` v1.3.0 — OTLP exporter wrapper (`internal/observability/tracing.go`)
- `github.com/wasilak/loggergo` v1.8.2 — structured `slog` logger setup
- `github.com/samber/slog-echo` v1.21.0 — Echo access log middleware using `slog`

**Password hashing:**
- `golang.org/x/crypto` v0.49.0 — bcrypt for local user passwords

**Concurrency / utilities:**
- `dario.cat/mergo` v1.0.2 — config struct merging
- `github.com/google/uuid` v1.6.0 — UUID generation (session IDs, state tokens)

---

## Build System

**Binary build (development):**
```bash
task build          # go build -o bin/keyline ./cmd/keyline
task dev            # build + run
task test           # go test ./...
task format         # gofmt -w . && goimports -w .
task lint           # go vet + gofmt check
```

**Release build:**
```bash
task release:build:target   # CGO_ENABLED=0, outputs to ./dist/keyline-<GOOS>-<GOARCH>
```

**Docker:**
- Two-stage build: `golang:1.26-alpine` builder → `alpine:3.23` runtime
- Non-root user (`keyline:1000`)
- Default port: `9000`
- Health check: `wget http://localhost:9000/_health`
- Entry point: `/app/keyline --config /etc/keyline/config.yaml`

---

## CI/CD

**Platform:** GitHub Actions (`.github/workflows/ci-cd.yml`)

**Stages:**
1. **test** — `go fmt`, `go vet`, `go test ./...` on `ubuntu-latest`
2. **build-binaries** — cross-compile for `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`
3. **build-docker** — parallel per-platform Docker builds pushed to `ghcr.io/<repo>` (native `ubuntu-24.04-arm` runner for ARM64, no QEMU)
4. **create-manifest** — fan-in multi-arch manifest
5. **create-release** — GitHub release with binaries on `v*.*.*` tags

**Go version in CI:** `1.26` (read from `go.mod` via `go-version-file`)

---

## Configuration Pattern

- Config file: YAML, loaded by Viper from path given via `--config` flag
- Environment variable substitution: `${VAR_NAME}` syntax in YAML values (resolved by Viper at load time)
- Validation: `keyline --validate-config` flag
- Config struct root: `internal/config/config.go` — `Config` struct with typed sub-structs per subsystem

---

## Platform Requirements

**Development:**
- Go 1.26+
- `task` CLI (Taskfile runner)
- `goimports` (for `task format`)
- Docker (for integration tests)

**Production:**
- Docker image: `alpine:3.23` base, port `9000`
- Or bare binary: static, no runtime dependencies
- Redis (recommended) or in-memory cache for sessions/state
