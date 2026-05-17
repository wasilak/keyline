---
sidebar_label: Overview
sidebar_position: 1
---

# Authentication Overview

Keyline supports multiple authentication methods: **OIDC (OpenID Connect)** for interactive browser users, **Basic Auth** (local users) for programmatic access, and **LDAP / Active Directory** for corporate directory authentication. This guide provides an overview of all authentication methods and how Keyline handles them.

## Supported Authentication Methods

| Method | Use Case | Session | Best For |
|--------|----------|---------|----------|
| **OIDC** | Interactive browser authentication | Yes (cookie-based) | Human users, SSO |
| **Basic Auth** | Programmatic/API access | No (stateless) | CI/CD, monitoring, scripts |
| **LDAP / AD** | Corporate directory authentication | No (stateless) | Existing AD environments |

## Dual Authentication Architecture

Keyline automatically selects the appropriate authentication method based on the incoming request:

```mermaid
flowchart TD
    Request[Incoming Request] --> CheckSession{Has Valid<br/>Session Cookie?}
    
    CheckSession -->|Yes| UseSession[Use Existing Session]
    
    CheckSession -->|No| CheckBasic{Has Basic<br/>Auth Header?}
    
    CheckBasic -->|Yes| BasicAuth[Basic Authentication]
    CheckBasic -->|No| CheckCallback{Is Callback<br/>Path?}
    
    CheckCallback -->|Yes| OIDCCallback[Process OIDC Callback]
    CheckCallback -->|No| OIDCAuth[Initiate OIDC Flow]
    
    UseSession --> Access[Grant Access]
    BasicAuth --> Access
    OIDCCallback --> Access
    OIDCAuth --> Access
```

## Authentication Flow Comparison

### OIDC Flow (Interactive Users)

```mermaid
sequenceDiagram
    participant User
    participant Keyline
    participant OIDC
    participant Session
    participant ES
    
    User->>Keyline: Access protected resource
    Keyline->>Keyline: No session found
    Keyline->>Keyline: Generate state + PKCE
    Keyline->>Session: Store state token
    Keyline->>User: 302 Redirect to OIDC
    User->>OIDC: Authenticate
    OIDC->>User: 302 Redirect with code
    User->>Keyline: Callback with code
    Keyline->>Session: Validate state
    Keyline->>OIDC: Exchange code for tokens
    OIDC->>Keyline: Return ID token
    Keyline->>Keyline: Validate token signature & claims
    Keyline->>Keyline: Extract user identity
    Keyline->>Keyline: Map to ES roles
    Keyline->>Session: Create session
    Keyline->>User: 302 Redirect with cookie
    User->>Keyline: Request with cookie
    Keyline->>Keyline: Validate session
    Keyline->>ES: Forward with credentials
    ES->>Keyline: Response
    Keyline->>User: Access granted
```

### Basic Auth Flow (Local Users and LDAP)

Both local users and LDAP authentication use the same `Authorization: Basic` header. The auth engine selects the method automatically based on the username: local users are checked first, and if no local match is found the request falls through to LDAP.

Actual priority order (from `internal/auth/engine.go`): **Session → Basic Auth header (local users checked first → LDAP fallback) → OIDC redirect**.

```mermaid
sequenceDiagram
    participant Client
    participant Keyline
    participant ES
    
    Client->>Keyline: Request with Authorization header
    Keyline->>Keyline: Decode Basic auth credentials
    Keyline->>Keyline: Find user in local_users
    alt Local user found
        Keyline->>Keyline: Validate bcrypt password
    else No local match — LDAP fallback
        Keyline->>Keyline: Service-account bind to LDAP
        Keyline->>Keyline: Search and bind user against LDAP
    end
    Keyline->>Keyline: Map to ES roles
    Keyline->>ES: Forward with credentials
    ES->>Keyline: Response
    Keyline->>Client: Response
```

## Key Security Features

### OIDC Security

| Feature | Purpose |
|---------|---------|
| **PKCE** | Prevents authorization code interception attacks |
| **State Token** | CSRF protection, single-use, 5-minute TTL |
| **ID Token Validation** | Signature, issuer, audience, expiration checks |
| **JWKS Rotation** | Automatic key refresh every 24 hours |
| **Secure Cookies** | HttpOnly, Secure, SameSite=Lax attributes |

### Basic Auth Security

| Feature | Purpose |
|---------|---------|
| **Bcrypt Hashing** | Timing-safe password comparison |
| **No Session Storage** | Stateless authentication |
| **WWW-Authenticate Header** | Proper 401 response for failed auth |
| **No Plaintext Logging** | Credentials never logged |

### LDAP Security

| Feature | Purpose |
|---------|---------|
| **Service-Account Bind** | Lookup uses a dedicated service account; user credentials are only used to validate the user bind |
| **env-var Credential** | `bind_password` must be a `${ENV_VAR}` reference; plaintext values are rejected at startup |
| **TLS Modes** | `ldaps` (recommended) or `starttls`; plaintext `none` is dev-only |
| **LDAP Injection Prevention** | Username is escaped before use in search filters |
| **Required Groups** | Optional access gate — only members of `required_groups` are admitted |

## Session Management

### Cookie-Based Sessions (OIDC)

| Attribute | Value | Purpose |
|-----------|-------|---------|
| `HttpOnly` | `true` | Prevents JavaScript access (XSS protection) |
| `Secure` | `true` | Requires HTTPS transmission |
| `SameSite` | `Lax` | Prevents CSRF attacks |
| `Max-Age` | Configurable (default: 24h) | Session TTL |

### Session Storage Backends

| Backend | Use Case | Pros | Cons |
|---------|----------|------|------|
| **Memory** | Development, single-node | Simple, no dependencies | Lost on restart, no scaling |
| **Redis** | Production, multi-node | Persistent, scalable | Requires Redis infrastructure |

## Configuration Summary

### OIDC Configuration

```yaml
oidc:
  enabled: true
  issuer_url: https://accounts.google.com
  client_id: ${OIDC_CLIENT_ID}
  client_secret: ${OIDC_CLIENT_SECRET}
  redirect_url: https://auth.example.com/auth/callback
  scopes:
    - openid
    - email
    - profile
```

### Basic Auth Configuration

```yaml
local_users:
  enabled: true
  users:
    - username: ci-pipeline
      password_bcrypt: ${CI_PASSWORD_BCRYPT}
      groups:
        - ci
      email: ci@example.com
      full_name: CI Pipeline
```

### Session Configuration

```yaml
session:
  ttl: 24h
  cookie_name: keyline_session
  cookie_domain: .example.com
  session_secret: ${SESSION_SECRET}  # Min 32 bytes
```

## Authentication Endpoints

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/_auth` | GET | ForwardAuth validation endpoint |
| `/auth/callback` | GET | OIDC callback handler |
| `/auth/logout` | GET/POST | Session logout |
| `/*` | ANY | Protected resources |

> **LDAP has no dedicated endpoint.** It reuses the `Authorization: Basic` header path alongside local users. When a request arrives with a Basic Auth header, the engine checks local users first; if no local user matches the username, it falls back to LDAP automatically.

## Next Steps

- **[OIDC Authentication](./oidc-authentication.md)** - Detailed OIDC setup and configuration
- **[Local Users (Basic Auth)](./local-users-basic-auth.md)** - Configure Basic Authentication
- **[LDAP Authentication](./ldap-authentication.md)** - Configure LDAP / Active Directory authentication
- **[Session Management](./session-management.md)** - Session storage and configuration
- **[Logout](./logout.md)** - Session termination and OIDC logout
