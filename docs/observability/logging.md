# Structured Logging

Keyline uses [`loggergo`](https://github.com/wasilak/loggergo) (v1.8.2) for structured, levelled logging backed by Go's `slog` standard library.

## Configuration

```yaml
log_level: "info"       # debug | info | warn | error
log_format: "json"      # json | text
log_pretty: false       # pretty-print JSON (development only)
```

All log lines are written to stdout.

## OTel Log Bridge

When `otel_enabled: true` and the OTel SDK initialises successfully, Keyline activates the OpenTelemetry log bridge. This routes every `slog` log record through the OTel Logs SDK, exporting it to your OTLP collector alongside traces and metrics.

### How it works

1. During startup, `cmd/keyline/main.go` initialises the OTel SDK (tracing + logging).
2. If initialisation succeeds, `loggergo` is configured with:
   - `Output: fanout` — writes to both stdout and the OTel log exporter
   - `OtelServiceName` — set to the value of `otel_service_name`
   - `OtelTracingEnabled: true` — correlates log records with the active trace/span IDs
   - `OtelLoggerName: "keyline/logger"` — identifies the instrumentation scope
3. If OTel initialisation fails, Keyline falls back to plain stdout JSON logging and continues running.

### Trace correlation

When the bridge is active, every log record emitted inside a traced handler automatically includes `trace_id` and `span_id` fields. This allows you to jump from a log line directly to the corresponding trace in Jaeger or Tempo.

## Log Fields

All request-scoped log lines include the following fields:

| Field | Type | Description |
|:---|:---|:---|
| `time` | RFC3339 | Timestamp |
| `level` | string | `DEBUG` \| `INFO` \| `WARN` \| `ERROR` |
| `msg` | string | Human-readable message |
| `trace_id` | string | OTel trace ID (when bridge active) |
| `span_id` | string | OTel span ID (when bridge active) |
| `username` | string | Sanitised username (where applicable) |
| `error` | string | Error message (error/warn lines only) |

## Log Levels

| Level | Used for |
|:---|:---|
| `DEBUG` | Connection pool details, cache internals, role-mapping decisions |
| `INFO` | Successful authentications, user upserts, server startup/shutdown |
| `WARN` | Failed login attempts, circuit-breaker state changes |
| `ERROR` | Infrastructure failures (LDAP, ES), configuration errors |
