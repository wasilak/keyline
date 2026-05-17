# Distributed Tracing

Keyline emits OpenTelemetry traces for all authentication and user management operations. Traces are exported via OTLP and can be visualised in Jaeger, Tempo, or any OTLP-compatible backend.

## Configuration

Tracing is enabled alongside the full OTel integration by setting the following values in `keyline.yaml`:

```yaml
otel_enabled: true
otel_endpoint: "http://otel-collector:4318"  # OTLP HTTP endpoint
otel_service_name: "keyline"                 # service.name resource attribute
otel_tracing_enabled: true                   # enables span export
```

When `otel_enabled` is `false` the tracer is a no-op and no spans are exported.

## Span Inventory

### Authentication — `internal/auth`

| Span name | Parent | Key attributes | Emitted by |
|:---|:---|:---|:---|
| `auth.authenticate` | — (root) | `auth.provider` | `engine.go` |
| `oidc.token_exchange` | `auth.authenticate` | `oidc.issuer` | `oidc.go` |
| `ldap.bind` | `auth.authenticate` | `ldap.bind_type` (`service_account` \| `user`) | `ldap.go` |
| `ldap.search` | `auth.authenticate` | `ldap.username` | `ldap.go` |

The `ldap.bind` span appears up to three times per authentication:

1. **service_account** — initial bind to perform the user search
2. **user** — bind as the authenticating user to verify the password
3. **service_account** — re-bind to perform the group-membership search

### User Management — `internal/usermgmt`

| Span name | Parent | Notes |
|:---|:---|:---|
| `cache.get` | `auth.authenticate` | Checks credential cache before hitting ES |
| `es.create_credentials` | `auth.authenticate` | Password generation for new/rotated credentials |
| `es.upsert_user` | `auth.authenticate` | Creates or updates the ES native user |
| `cache.set` | `auth.authenticate` | Stores the new password in cache |

### Elasticsearch — `internal/elasticsearch`

| Span name | Parent | Key attributes |
|:---|:---|:---|
| `es.create_or_update_user` | `es.upsert_user` | `es.username`, `http.status_code` |
| `es.get_user` | caller | `es.username` |
| `es.delete_user` | caller | `es.username` |

## Example Trace

A full successful LDAP authentication produces the following span tree:

```
auth.authenticate
├── ldap.bind            (bind_type=service_account)
├── ldap.search          (ldap.username=alice)
├── ldap.bind            (bind_type=user)
├── ldap.bind            (bind_type=service_account)
├── cache.get
├── es.create_credentials
├── es.upsert_user
│   └── es.create_or_update_user
└── cache.set
```

On a cache hit the tree is shorter — only `cache.get` is emitted after the LDAP spans.

## Error Handling

All spans call `span.RecordError(err)` and `span.SetStatus(codes.Error, ...)` on failure, so error details are surfaced directly in the trace without needing to correlate with logs.

## Instrumentation Pattern

Inline sub-operation spans follow this pattern throughout the codebase:

```go
_, s := otel.Tracer("keyline").Start(ctx, "operation.name")
result, err := doSomething()
if err != nil {
    s.RecordError(err)
    s.SetStatus(codes.Error, "short description")
    s.End()
    return nil, err
}
s.End()
```

Whole-function spans use `defer span.End()` instead.
