# Audit Logging

Keyline emits a structured audit event for every authentication decision — regardless of outcome. Audit events are written via `log/slog` and appear in the same JSON log stream as operational logs. They are distinguishable by the `event` field value `auth.decision`.

## Event Format

Every audit event includes:

| Field | Type | Description |
|---|---|---|
| `event` | string | Always `auth.decision` |
| `result` | string | `success` or `failure` |
| `auth_method` | string | Method used: `session`, `basic`, `ldap`, `oidc`, `unknown` |
| `username` | string | Authenticated username (empty on failure before username extraction) |
| `source_ip` | string | Client IP from the incoming request |
| `http_method` | string | HTTP method of the proxied request |
| `path` | string | Request path |
| `time` | string | RFC3339 timestamp (added by slog handler) |
| `trace_id` | string | OTel trace ID — **only present when OTel is enabled and a span is active** |
| `span_id` | string | OTel span ID — **only present when OTel is enabled and a span is active** |

### Example — successful LDAP login

```json
{
  "time": "2026-05-17T10:23:41Z",
  "level": "INFO",
  "msg": "audit",
  "event": "auth.decision",
  "result": "success",
  "auth_method": "ldap",
  "username": "alice",
  "source_ip": "10.0.0.5",
  "http_method": "GET",
  "path": "/app/dashboard",
  "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736",
  "span_id": "00f067aa0ba902b7"
}
```

### Example — failed basic auth

```json
{
  "time": "2026-05-17T10:24:01Z",
  "level": "INFO",
  "msg": "audit",
  "event": "auth.decision",
  "result": "failure",
  "auth_method": "basic",
  "username": "bob",
  "source_ip": "10.0.0.7",
  "http_method": "POST",
  "path": "/api/index/_search"
}
```

## Filtering Audit Events

Because every audit event has `"msg": "audit"` and `"event": "auth.decision"`, they are easy to isolate with standard tools.

**jq:**
```bash
journalctl -u keyline | jq 'select(.event == "auth.decision")'
```

**Loki LogQL:**
```logql
{app="keyline"} | json | event = "auth.decision"
```

**Loki — failures only:**
```logql
{app="keyline"} | json | event = "auth.decision" | result = "failure"
```

## OTel Trace Correlation (AUDIT-02)

When OTel tracing is enabled (`otel_enabled: true`), `trace_id` and `span_id` are injected into every audit event. This lets you jump from an audit log line directly to the corresponding distributed trace in Tempo or Jaeger.

If OTel is disabled, or no span is active in the request context, `trace_id` and `span_id` are omitted.

See [tracing.md](./tracing.md) for OTel configuration and span inventory.

## Security Notes

- **No credentials or secrets appear in audit events.** Only the username is logged; passwords, tokens, and session IDs are never included.
- Audit events are emitted at `INFO` level. To suppress them, filter by `event != "auth.decision"` at your log aggregator rather than raising the log level, which would suppress other useful operational logs.
