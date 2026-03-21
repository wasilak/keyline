---
sidebar_label: Observability
sidebar_position: 1
---

# Observability

Keyline provides comprehensive observability through health checks, logging, metrics, and distributed tracing.

## Health Checks

### `/healthz` - Basic Health Check

```bash
curl http://localhost:9000/healthz
```

**Response:**
```json
{
  "status": "healthy",
  "version": "1.0.0"
}
```

### `/_health` - Detailed Health Check

```bash
curl http://localhost:9000/_health
```

**Response:**
```json
{
  "status": "healthy",
  "checks": {
    "session_store": "ok",
    "oidc_provider": "ok",
    "elasticsearch": "ok"
  }
}
```

### Kubernetes Probes

```yaml
livenessProbe:
  httpGet:
    path: /healthz
    port: 9000
  initialDelaySeconds: 30
  periodSeconds: 30

readinessProbe:
  httpGet:
    path: /healthz
    port: 9000
  initialDelaySeconds: 10
  periodSeconds: 10
```

## Logging

### Configuration

```yaml
observability:
  log_level: info
  log_format: json  # or 'text'
```

### Log Levels

| Level | Use Case |
|-------|----------|
| `debug` | Development, troubleshooting |
| `info` | Production default |
| `warn` | Warning conditions |
| `error` | Critical failures |

### Example Output

**JSON Format (Production):**
```json
{
  "timestamp": "2024-01-01T00:00:00Z",
  "level": "info",
  "message": "User authenticated successfully",
  "username": "user@example.com",
  "method": "oidc",
  "source_ip": "192.168.1.1"
}
```

**Text Format (Development):**
```
2024-01-01T00:00:00Z INFO User authenticated successfully username=user@example.com method=oidc
```

## Metrics

### Configuration

```yaml
observability:
  metrics_enabled: true
```

### Endpoint

```bash
curl http://localhost:9000/_metrics
```

### Available Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `keyline_auth_attempts_total` | Counter | Total authentication attempts |
| `keyline_auth_successes_total` | Counter | Successful authentications |
| `keyline_auth_failures_total` | Counter | Failed authentications |
| `keyline_session_creates_total` | Counter | Sessions created |
| `keyline_user_upserts_total` | Counter | ES user upserts |
| `keyline_cred_cache_hits_total` | Counter | Credential cache hits |
| `keyline_cred_cache_misses_total` | Counter | Credential cache misses |

### Prometheus Integration

```yaml
# Prometheus scrape config
- job_name: 'keyline'
  static_configs:
    - targets: ['keyline:9000']
  metrics_path: '/_metrics'
```

## Distributed Tracing

### Configuration

```yaml
observability:
  otel_enabled: true
  otel_endpoint: http://otel-collector:4318
  otel_service_name: keyline
  otel_trace_ratio: 1.0  # 0.0 to 1.0
```

### Trace Propagation

Keyline supports:
- W3C Trace Context headers
- B3 headers (Zipkin compatibility)

### Compatible Backends

- Jaeger
- Zipkin
- AWS X-Ray
- Google Cloud Trace
- Any OTLP-compatible collector
