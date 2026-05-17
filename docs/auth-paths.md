# Authentication Paths

Keyline supports five authentication paths. They are evaluated in a fixed precedence order on every request. This document describes each path: when it fires, what the request must look like, what Keyline returns, and what the audit log records.

## Precedence Order

| Priority | Method | Trigger |
|---|---|---|
| 1 (highest) | **session** | Session cookie present and valid |
| 2 | **basic** | `Authorization: Basic` header + username found in `local_users` |
| 3 | **ldap** | `Authorization: Basic` header + no matching local user (LDAP enabled) |
| 4 | **oidc** | No credential header and no session; OIDC enabled |
| 5 (fallback) | **unknown** | No method enabled or no credential of any kind |

The engine short-circuits: the first path that produces a result (success or failure) terminates evaluation. Subsequent paths are not attempted.

## Deployment Modes

Keyline runs in one of two modes, configured via `server.mode`:

| Mode | Value | Use case |
|---|---|---|
| **ForwardAuth** | `forward_auth` | Traefik / Nginx `auth_request` — Keyline responds 200 or 401; the reverse proxy handles the actual request |
| **Standalone** | `standalone` | Keyline is the reverse proxy; it forwards authenticated requests to `upstream.url` with `Authorization: Basic` injected |

The auth logic is identical in both modes. The difference is in what Keyline does *after* a decision is made.

---

## Path 1 — Session (Cookie)

### When it fires

Every request. The engine checks for a cookie whose name matches `session.cookie_name` (default: `_keyline_session`). If the cookie is absent or the session is expired/invalid, evaluation falls through to the next path.

### How it works

1. Cookie value is used as a session ID.
2. Session record is looked up in the cache (Redis or in-memory).
3. If found, `userManager.UpsertUser` is called to provision or refresh the Elasticsearch user.
4. The `X-Es-Authorization` header is set to `Basic <base64(esUser:esPassword)>`.

### ForwardAuth

```
Client → Traefik → Keyline (ForwardAuth)
           ↑         |
           |    200 OK + X-Es-Authorization
           |         ↓
       Traefik → Upstream (with injected header)
```

Keyline responds `200 OK`. Traefik forwards the original request to the upstream, including any headers Keyline added.

### Standalone

```
Client → Keyline (Standalone) → Upstream
              (injects Authorization: Basic esUser:esPassword)
```

Keyline proxies the request to `upstream.url`, replacing the `Authorization` header with the ES credentials.

### curl example (ForwardAuth mode)

```bash
# Assuming Traefik is at https://proxy.example.com
# The session cookie was set during a previous OIDC or basic login
curl -v \
  --cookie "_keyline_session=<session-id>" \
  https://proxy.example.com/app/dashboard
# Expected: 200 OK from upstream (Keyline returned 200 to Traefik)
```

### curl example (Standalone mode)

```bash
# Keyline is the proxy at https://keyline.example.com
curl -v \
  --cookie "_keyline_session=<session-id>" \
  https://keyline.example.com/app/dashboard
# Expected: upstream response proxied through Keyline
```

### Audit log

```json
{
  "time": "2026-05-17T10:00:00Z",
  "level": "INFO",
  "msg": "audit",
  "event": "auth.decision",
  "result": "success",
  "auth_method": "session",
  "username": "alice",
  "source_ip": "10.0.0.5",
  "http_method": "GET",
  "path": "/app/dashboard"
}
```

Failed session (cookie present but expired):

```json
{
  "event": "auth.decision",
  "result": "failure",
  "auth_method": "session",
  "username": "",
  "source_ip": "10.0.0.5",
  "http_method": "GET",
  "path": "/app/dashboard"
}
```

> When the session cookie is absent the engine does not emit a session failure — it simply falls through. A session audit event only appears when the cookie is present but the session lookup fails (e.g. user management error).

---

## Path 2 — Basic Auth (Local Users)

### When it fires

An `Authorization: Basic` header is present, `local_users.enabled: true`, and the decoded username matches an entry in `local_users.users`. If the username is not in the local list and LDAP is enabled, the request falls through to the LDAP path.

### How it works

1. Header is decoded: `base64(username:password)`.
2. `BasicAuthProvider.Authenticate` checks the bcrypt hash from config.
3. On success, `userManager.UpsertUser` provisions the ES user.

### curl example

```bash
# Basic auth with a local user
curl -v \
  -u "alice:mysecretpassword" \
  https://proxy.example.com/api/index/_search

# Equivalent explicit header
curl -v \
  -H "Authorization: Basic $(echo -n 'alice:mysecretpassword' | base64)" \
  https://proxy.example.com/api/index/_search
```

### Audit log — success

```json
{
  "event": "auth.decision",
  "result": "success",
  "auth_method": "basic",
  "username": "alice",
  "source_ip": "10.0.0.5",
  "http_method": "GET",
  "path": "/api/index/_search"
}
```

### Audit log — failure (wrong password)

```json
{
  "event": "auth.decision",
  "result": "failure",
  "auth_method": "basic",
  "username": "alice",
  "source_ip": "10.0.0.5",
  "http_method": "GET",
  "path": "/api/index/_search"
}
```

### Response on failure

ForwardAuth: `401 Unauthorized`  
Standalone: `401 Unauthorized`

---

## Path 3 — LDAP

### When it fires

An `Authorization: Basic` header is present, `ldap.enabled: true`, and the username does **not** match any local user. (If both local users and LDAP are enabled, LDAP is the fallback for unknown usernames.)

### How it works

1. Header is decoded: `base64(username:password)`.
2. `LDAPProvider.Authenticate` performs a bind against the configured LDAP server using the service account (`bind_dn` / `bind_password`).
3. A search is run with `search_filter` (e.g. `(sAMAccountName={username})`).
4. Groups are fetched if `group_search_base` / `group_search_filter` are configured.
5. On success, `userManager.UpsertUser` provisions the ES user with the LDAP groups.

### curl example

```bash
# LDAP user (not in local_users)
curl -v \
  -u "jsmith:correcthorsebatterystaple" \
  https://proxy.example.com/api/index/_search
```

### Audit log — success

```json
{
  "event": "auth.decision",
  "result": "success",
  "auth_method": "ldap",
  "username": "jsmith",
  "source_ip": "10.0.0.5",
  "http_method": "GET",
  "path": "/api/index/_search",
  "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736",
  "span_id": "00f067aa0ba902b7"
}
```

(`trace_id` / `span_id` appear only when `otel_enabled: true`.)

### Audit log — failure (bad credentials)

```json
{
  "event": "auth.decision",
  "result": "failure",
  "auth_method": "ldap",
  "username": "jsmith",
  "source_ip": "10.0.0.5",
  "http_method": "GET",
  "path": "/api/index/_search"
}
```

### Response on failure

ForwardAuth: `401 Unauthorized`  
Standalone: `401 Unauthorized`

---

## Path 4 — OIDC

### When it fires

No `Authorization` header is present, no valid session cookie, and `oidc.enabled: true`. This path **always** initiates a redirect — it never authenticates in a single round-trip.

### Full flow

```
1. Client → Keyline   (no creds)
2. Keyline → Client   302 Redirect to OIDC provider

3. Client → OIDC provider   (login at provider UI)
4. OIDC provider → Client   302 Redirect to oidc.redirect_url (/auth/callback?code=...&state=...)

5. Client → Keyline /auth/callback
6. Keyline exchanges code for tokens with OIDC provider
7. Keyline creates session, sets session cookie
8. Keyline → Client   302 Redirect to original URL

9. Client → Keyline   (session cookie now present → Path 1 handles it)
```

### ForwardAuth note

In ForwardAuth mode, the initial redirect (`302`) is returned to Traefik/Nginx as the response to the `auth_request` sub-request. The reverse proxy must be configured to honour `401`/`302` responses from the auth service and redirect the browser accordingly. For Traefik this is the default behaviour with `forwardAuth.authResponseHeaders`.

### curl — step 1: trigger OIDC redirect

```bash
# No credentials — expect 302 to OIDC provider
curl -v https://proxy.example.com/app/dashboard
# < HTTP/1.1 302 Found
# < Location: https://sso.example.com/auth?client_id=...&redirect_uri=...&state=...
```

### curl — step 5: simulate callback (for testing only)

In practice the browser follows the redirect automatically. For manual testing:

```bash
# After the provider redirects back to /auth/callback:
curl -v "https://keyline.example.com/auth/callback?code=<auth_code>&state=<state_value>"
# < HTTP/1.1 302 Found
# < Set-Cookie: _keyline_session=<session-id>; HttpOnly; Path=/
# < Location: /app/dashboard
```

### Audit log — redirect (not yet authenticated)

The OIDC redirect itself counts as a failed auth decision because the user is not yet authenticated at that point:

```json
{
  "event": "auth.decision",
  "result": "failure",
  "auth_method": "oidc",
  "username": "",
  "source_ip": "10.0.0.5",
  "http_method": "GET",
  "path": "/app/dashboard"
}
```

After the callback completes and the user is redirected back with a session cookie, the **next** request hits Path 1 (session) and emits a success event with `auth_method: session`.

---

## Path 5 — Unknown (No Method Available)

### When it fires

All of the following are true:
- No valid session cookie
- No `Authorization: Basic` header (or basic/LDAP both disabled)
- `oidc.enabled: false`

This is the "nothing works" fallback. The engine returns `401 Unauthorized` immediately.

### Audit log

```json
{
  "event": "auth.decision",
  "result": "failure",
  "auth_method": "unknown",
  "username": "",
  "source_ip": "10.0.0.5",
  "http_method": "GET",
  "path": "/app/dashboard"
}
```

### Typical cause

Misconfiguration — all auth methods are disabled, or the request came in with no credentials and OIDC is off. Check `local_users.enabled`, `ldap.enabled`, and `oidc.enabled` in your config.

---

## Summary Table

| `auth_method` | Trigger | Success response | Failure response |
|---|---|---|---|
| `session` | Session cookie valid | 200 + `X-Es-Authorization` | 500 (usermgmt error) |
| `basic` | `Authorization: Basic` + local user | 200 + `X-Es-Authorization` | 401 |
| `ldap` | `Authorization: Basic` + LDAP user | 200 + `X-Es-Authorization` | 401 |
| `oidc` | No creds, OIDC enabled | 302 → provider (then session) | 302 or 500 |
| `unknown` | No method matched | — | 401 |

## Related Docs

- [Audit Logging](./observability/audit.md) — full audit event reference
- [Tracing](./observability/tracing.md) — OTel span inventory per auth path
- [Metrics](./observability/metrics.md) — counters and histograms for auth outcomes
