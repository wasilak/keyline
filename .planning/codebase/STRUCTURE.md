# Directory Structure
_Generated: 2026-05-16 | Focus: arch_

## Summary
Keyline follows a clean Go project layout with all application code under `internal/` and a thin `cmd/keyline/` entry point. The top-level structure separates concerns clearly: config loading, auth providers, HTTP server, transport, and user management each live in their own package with co-located tests.

---

## Annotated Directory Tree

```
/Users/piotrek/git/keyline/
├── cmd/
│   └── keyline/
│       └── main.go              # Entry point — wires all components together
├── internal/
│   ├── auth/                    # Authentication providers
│   │   ├── ldap.go              # LDAP provider (in progress)
│   │   ├── ldap_test.go         # LDAP unit tests
│   │   ├── local.go             # Local user auth
│   │   └── oidc.go              # OIDC provider
│   ├── config/                  # Configuration loading & validation
│   │   ├── config.go            # Config structs and Viper loading
│   │   └── config_test.go       # Config parsing tests (table-driven)
│   ├── engine/                  # Core authentication dispatch engine
│   │   └── engine.go            # Main auth flow (session → local → LDAP → OIDC)
│   ├── esclient/                # Elasticsearch HTTP client
│   │   ├── client.go            # ES Security API client with retry/circuit-breaker
│   │   └── client_test.go       # Tests using httptest.NewServer
│   ├── roles/                   # Role mapping pipeline
│   │   ├── mapper.go            # Maps LDAP groups / OIDC claims → ES roles
│   │   └── mapper_test.go       # Property-invariant tests
│   ├── server/                  # HTTP server setup
│   │   └── server.go            # Echo router, middleware chain, CORS config
│   ├── transport/               # HTTP transport with OTel instrumentation
│   │   └── transport.go         # Outbound client with tracing
│   └── usermgmt/                # User upsert pipeline
│       ├── upsert.go            # UpsertUser: creates/updates ES user with roles
│       └── upsert_test.go
├── integration/                 # Integration tests (build tag: integration)
│   └── *_test.go                # End-to-end flows against real services
├── docs/                        # Docusaurus documentation site
├── .taskmaster/                 # Task Master AI project config & tasks
├── .planning/                   # GSD planning artifacts
│   └── codebase/                # This codebase map
├── Taskfile.yml                 # Build tasks (test, build, lint, docker)
├── Dockerfile                   # Two-stage Alpine image
├── config.example.yaml          # Reference config with comments
├── go.mod                       # Go 1.26 module (Echo v4, go-ldap/v3, Viper, ...)
└── go.sum
```

---

## Package Responsibilities

| Package | Responsibility |
|---|---|
| `cmd/keyline` | Wire dependencies, start HTTP server, graceful shutdown |
| `internal/config` | Load `config.yaml` via Viper, substitute `${ENV_VAR}` references, validate |
| `internal/auth` | Auth provider implementations (local, OIDC, LDAP) with a common interface |
| `internal/engine` | Dispatch auth requests: session cookie → Basic Auth (local then LDAP) → OIDC redirect |
| `internal/esclient` | Talk to Elasticsearch Security API to create/update users and roles |
| `internal/roles` | Map external group memberships (LDAP groups, OIDC claims) to ES role names |
| `internal/usermgmt` | `UpsertUser` pipeline: takes `AuthenticatedUser`, calls role mapper, calls ES client |
| `internal/server` | Echo HTTP server, middleware (CORS, OTel, logging), route registration |
| `internal/transport` | Instrumented outbound HTTP client for ES and OTel Collector |

---

## Entry Point — Startup Order (`cmd/keyline/main.go`)

1. Load config (`internal/config`)
2. Init OTel tracing (`internal/transport`)
3. Create ES client (`internal/esclient`)
4. Init auth providers: local users, OIDC, LDAP if enabled (`internal/auth`)
5. Create auth engine with providers (`internal/engine`)
6. Create user upsert pipeline (`internal/usermgmt`)
7. Start Echo HTTP server (`internal/server`)
8. Block on OS signal → graceful shutdown

---

## Config Loading Pattern

Config is loaded from `config.yaml` using Viper with environment variable substitution:
- Struct tags: `mapstructure:"field_name"`
- Secrets use `${ENV_VAR_NAME}` syntax in yaml — substituted at load time
- Only `ldap.bind_password` currently enforces the env-var pattern; other secrets can be inline
- Config validation happens immediately after loading (missing required fields → fatal)

---

## Test File Locations

| Layer | Location | Pattern |
|---|---|---|
| Unit | `internal/**/*_test.go` | Co-located with source |
| Integration | `integration/*_test.go` | Separate dir, `//go:build integration` |

---

## Where to Add New Code

| Extension | Location |
|---|---|
| New auth provider | `internal/auth/<provider>.go` + register in `cmd/keyline/main.go` |
| New config section | `internal/config/config.go` struct + `config.example.yaml` |
| New HTTP route | `internal/server/server.go` Echo route registration |
| New role mapping source | `internal/roles/mapper.go` |
| New ES API call | `internal/esclient/client.go` |
