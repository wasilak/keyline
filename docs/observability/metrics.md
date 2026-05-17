# Keyline Metrics Reference

All metrics are exposed at the `/metrics` endpoint in the standard Prometheus text exposition format.

---

## Core Auth & Request Metrics

These metrics have existed since v1.0 and use no prefix (legacy naming).

| Metric | Type | Labels | Description |
|---|---|---|---|
| `auth_attempts_total` | Counter | `method`, `result` | Total authentication attempts |
| `auth_duration_seconds` | Histogram | `method` | Duration of authentication operations |
| `active_sessions_total` | Gauge | — | Number of currently active sessions |
| `session_operations_total` | Counter | `operation`, `result` | Session create/delete/validate counts |
| `oidc_provider_requests_total` | Counter | `provider`, `result` | Outbound OIDC provider requests |
| `upstream_proxy_duration_seconds` | Histogram | `result` | Duration of upstream proxy calls |
| `concurrent_requests` | Gauge | — | Number of requests being processed concurrently |
| `errors_total` | Counter | `kind` | Internal error counts by kind |

---

## User Management Metrics

Added in v2.1. Track Elasticsearch user upsert flows and credential cache.

### `keyline_user_upserts_total`
**Type:** Counter | **Labels:** `status`

Total number of Elasticsearch user upsert operations.

| `status` | Meaning |
|---|---|
| `success` | User created or updated without error |
| `failure` | ES API call failed after retries |

```promql
# Upsert failure rate over 5 minutes
rate(keyline_user_upserts_total{status="failure"}[5m])
  / rate(keyline_user_upserts_total[5m])
```

### `keyline_user_upsert_duration_seconds`
**Type:** Histogram | **Labels:** `cache_status`

Duration of user upsert operations, split by credential cache outcome.

| `cache_status` | Meaning |
|---|---|
| `hit` | Credentials were served from cache (duration ≈ 0) |
| `miss` | Full ES call was made |

```promql
# p99 upsert latency for cache misses
histogram_quantile(0.99,
  rate(keyline_user_upsert_duration_seconds_bucket{cache_status="miss"}[5m])
)
```

### `keyline_cred_cache_hits_total`
**Type:** Counter

Credential cache hits. High values indicate the cache is effective.

### `keyline_cred_cache_misses_total`
**Type:** Counter

Credential cache misses (ES API was called to validate/create the user).

```promql
# Cache hit ratio
rate(keyline_cred_cache_hits_total[5m])
  / (rate(keyline_cred_cache_hits_total[5m]) + rate(keyline_cred_cache_misses_total[5m]))
```

### `keyline_role_mapping_matches_total`
**Type:** Counter | **Labels:** `pattern`

Number of times each role-mapping rule matched an incoming auth request. The `pattern` label is the raw pattern string from the config.

```promql
# Which patterns are matching most often
topk(5, sum by (pattern) (rate(keyline_role_mapping_matches_total[5m])))
```

---

## Elasticsearch API Metrics

Added in v2.1. Track individual HTTP calls to the Elasticsearch security API.

### `keyline_es_api_calls_total`
**Type:** Counter | **Labels:** `operation`, `status`

Total number of ES security API calls.

| `operation` | `status` | Meaning |
|---|---|---|
| `create_user` | `success` / `failure` | PUT `/_security/user/{username}` |
| `get_user` | `success` / `failure` | GET `/_security/user/{username}` |
| `delete_user` | `success` / `failure` | DELETE `/_security/user/{username}` |

Note: `create_user` may be called up to 3 times per upsert due to the retry loop. Each attempt is counted individually.

```promql
# ES API error rate by operation
sum by (operation) (rate(keyline_es_api_calls_total{status="failure"}[5m]))
  / sum by (operation) (rate(keyline_es_api_calls_total[5m]))
```

### `keyline_circuit_breaker_state`
**Type:** Gauge

Current state of the Elasticsearch circuit breaker.

| Value | State | Meaning |
|---|---|---|
| `0` | Closed | Normal operation; requests are allowed |
| `1` | Open | ES is degraded; requests are blocked for 30s |
| `2` | Half-Open | Testing recovery; limited requests allowed |

```promql
# Alert when circuit breaker opens
keyline_circuit_breaker_state == 1
```

Circuit breaker config: opens after **5 consecutive failures**, stays open for **30s**, requires **2 successes** in half-open to close, resets failure count after **60s** of no failures.

---

## LDAP Metrics

Added in v2.1. Track LDAP bind, search, and connection health.

### `keyline_ldap_bind_attempts_total`
**Type:** Counter | **Labels:** `result`

Total LDAP bind calls across all three bind types (service account initial bind, user credential bind, service account re-bind after user verification).

| `result` | Meaning |
|---|---|
| `success` | Bind returned no error |
| `failure` | Bind was rejected or connection error |

Note: a single successful authentication produces 3 `success` increments (initial service bind + user bind + re-bind for group search).

```promql
# LDAP bind failure rate
rate(keyline_ldap_bind_attempts_total{result="failure"}[5m])
  / rate(keyline_ldap_bind_attempts_total[5m])
```

### `keyline_ldap_search_duration_seconds`
**Type:** Histogram | **Buckets:** 10ms, 50ms, 100ms, 500ms, 1s, 5s

Duration of LDAP user search operations (`conn.Search` in `searchUser`). Does not include group searches (those are non-fatal and lower priority).

```promql
# p95 LDAP search latency
histogram_quantile(0.95,
  rate(keyline_ldap_search_duration_seconds_bucket[5m])
)
```

### `keyline_ldap_connection_errors_total`
**Type:** Counter

Number of times `dialFn` (the TCP/TLS dial to the LDAP server) failed. A sustained increase here indicates network or LDAP server availability problems.

```promql
# Connection error spike alert
increase(keyline_ldap_connection_errors_total[5m]) > 3
```

---

## Suggested Alerting Rules

```yaml
groups:
  - name: keyline
    rules:
      - alert: KeylineCircuitBreakerOpen
        expr: keyline_circuit_breaker_state == 1
        for: 0m
        labels:
          severity: warning
        annotations:
          summary: "Keyline ES circuit breaker is open"

      - alert: KeylineESHighErrorRate
        expr: |
          sum(rate(keyline_es_api_calls_total{status="failure"}[5m]))
            / sum(rate(keyline_es_api_calls_total[5m])) > 0.1
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "ES API error rate > 10%"

      - alert: KeylineLDAPConnectionErrors
        expr: increase(keyline_ldap_connection_errors_total[5m]) > 3
        for: 0m
        labels:
          severity: critical
        annotations:
          summary: "LDAP connection errors spiking"

      - alert: KeylineLowCacheHitRatio
        expr: |
          rate(keyline_cred_cache_hits_total[5m])
            / (rate(keyline_cred_cache_hits_total[5m]) + rate(keyline_cred_cache_misses_total[5m]))
            < 0.5
        for: 10m
        labels:
          severity: info
        annotations:
          summary: "Credential cache hit ratio below 50%"
```
