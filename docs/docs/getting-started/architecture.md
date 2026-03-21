---
sidebar_label: Architecture
sidebar_position: 2
---

# Architecture

Keyline is a unified authentication proxy service that replaces the existing Authelia + elastauth stack. It provides dual authentication modes (OIDC and Basic Auth) simultaneously, supports three deployment modes (forwardAuth, auth_request, standalone proxy), and automatically injects Elasticsearch credentials into authenticated requests.

## Design Goals

- **Unified Service**: Single binary replacing two-service architecture
- **Dual Authentication**: Support both interactive (OIDC) and programmatic (Basic Auth) access simultaneously
- **Deployment Flexibility**: Work with Traefik, Nginx, or as standalone proxy
- **Security First**: Implement PKCE, secure session management, and cryptographic best practices
- **Production Ready**: Built-in observability, health checks, and graceful shutdown
- **Full Observability**: OpenTelemetry tracing and structured logging from day one
- **Unified Caching**: Single cache interface for sessions, state tokens, and OIDC data

## High-Level Architecture

```mermaid
flowchart TB
    subgraph "Keyline Service"
        subgraph "Observability Layer"
            Otelecho[otelecho<br/>tracing MW]
            SlogEcho[slog-echo<br/>logging MW]
            Loggergo[loggergo<br/>global slog]
        end
        
        subgraph "Transport Adapter Layer"
            ForwardAuth[ForwardAuth<br/>Adapter]
            AuthRequest[Auth_Request<br/>Adapter]
            Standalone[Standalone<br/>Proxy]
        end
        
        subgraph "Core Authentication Engine"
            OIDC[OIDC Handler]
            Basic[Basic Auth<br/>Validator]
            Session[Session<br/>Manager]
            Mapper[ES Credential<br/>Mapper & Injector]
        end
        
        subgraph "Cache Layer (cachego)"
            Cache[Unified Cache Interface<br/>Redis or Memory backend]
        end
    end
    
    OIDCProvider[OIDC Provider<br/>Okta, etc.]
    CacheBackend[Cache Backend<br/>Redis/Memory]
    ProtectedService[Protected Service<br/>Kibana, ES, etc.]
    
    ForwardAuth --> OIDC
    AuthRequest --> OIDC
    Standalone --> OIDC
    
    OIDC --> Mapper
    Basic --> Mapper
    Session --> Mapper
    
    Mapper --> Cache
    Cache --> CacheBackend
    
    OIDC --> OIDCProvider
    Standalone --> ProtectedService
```

## Component Responsibilities

### Observability Layer

| Component | Purpose |
|-----------|---------|
| **otelecho** | Automatic OpenTelemetry tracing for all HTTP requests |
| **slog-echo** | Automatic structured logging for all HTTP requests with trace correlation |
| **loggergo** | Global slog configuration (JSON/text format, log levels) |

### Transport Adapter Layer

| Component | Purpose |
|-----------|---------|
| **ForwardAuth Adapter** | Handles Traefik X-Forwarded-* headers, returns auth decisions |
| **Auth_Request Adapter** | Handles Nginx X-Original-* headers, returns auth decisions |
| **Standalone Proxy** | Proxies authenticated requests to upstream, handles WebSocket upgrades |

### Core Authentication Engine

| Component | Purpose |
|-----------|---------|
| **OIDC Handler** | Manages authorization flow, token exchange, ID token validation |
| **Basic Auth Validator** | Validates local user credentials using bcrypt |
| **Session Manager** | Creates, validates, extends, and deletes user sessions |
| **ES Credential Mapper** | Maps authenticated users to Elasticsearch credentials |

### Cache Layer (cachego)

| Feature | Description |
|---------|-------------|
| **Unified Interface** | Single cache interface for all storage needs |
| **Sessions** | Stores user sessions with TTL (key: `session:{id}`) |
| **State Tokens** | Stores OIDC CSRF tokens with 5-minute TTL (key: `state:{id}`) |
| **OIDC Discovery** | Caches discovery documents (key: `oidc:discovery:{issuer}`) |
| **JWKS** | Caches JSON Web Key Sets (key: `oidc:jwks:{issuer}`) |
| **Backend Agnostic** | Supports Redis or in-memory backends via configuration |

## Technology Stack

| Layer | Technology |
|-------|------------|
| **Language** | Go 1.22+ |
| **Web Framework** | Echo v4 |
| **Configuration** | Viper |
| **Cache Layer** | cachego (unified interface for Redis/in-memory) |
| **Logging** | loggergo (global slog setup) |
| **Echo Logging** | slog-echo (request logging middleware) |
| **Tracing** | otelgo (OpenTelemetry setup) |
| **Echo Tracing** | otelecho (request tracing middleware) |
| **OIDC** | coreos/go-oidc v3 + golang.org/x/oauth2 |
| **Proxy** | net/http/httputil.ReverseProxy |
| **Crypto** | crypto/rand, bcrypt |

## Authentication Flow

### OIDC Authentication Flow

```mermaid
sequenceDiagram
    participant User
    participant Keyline
    participant OIDC
    participant Session
    
    User->>Keyline: Access protected resource
    Keyline->>Keyline: Generate state + PKCE
    Keyline->>Session: Store state token
    Keyline->>User: Redirect to OIDC
    User->>OIDC: Authenticate
    OIDC->>User: Redirect with code
    User->>Keyline: Callback with code
    Keyline->>Session: Validate state
    Keyline->>OIDC: Exchange code for tokens
    OIDC->>Keyline: Return ID token
    Keyline->>Keyline: Validate token
    Keyline->>Session: Create session
    Keyline->>User: Redirect with cookie
```

### Dynamic User Management Flow

```mermaid
sequenceDiagram
    participant User
    participant Keyline
    participant Cache
    participant ES
    
    User->>Keyline: Authenticate
    Keyline->>Cache: Check cached credentials
    alt Cache hit
        Cache-->>Keyline: Return credentials
    else Cache miss
        Keyline->>Keyline: Generate password
        Keyline->>Keyline: Map groups to roles
        Keyline->>ES: Create/update user
        ES-->>Keyline: User created
        Keyline->>Cache: Cache credentials
    end
    Keyline->>User: Access granted
```

## Deployment Modes

### ForwardAuth Mode (Traefik/Nginx)

```mermaid
sequenceDiagram
    participant User as Browser
    participant Traefik as Traefik
    participant Keyline as Keyline
    participant Kibana as Kibana
    
    User->>Traefik: Request
    Traefik->>Keyline: 1. Forward Auth Check
    Keyline-->>Traefik: 2. 200 + Header
    Traefik->>Kibana: 3. Proxy with ES Auth
```

### Standalone Mode

```mermaid
sequenceDiagram
    participant User as Browser
    participant Keyline as Keyline
    participant Kibana as Kibana
    
    User->>Keyline: 1. Request
    Keyline->>Keyline: 2. Auth Check (internal)
    Keyline->>Kibana: 3. Proxy with ES Auth
```

## Next Steps

- [**Quick Start**](./quick-start.md) - Get Keyline running in 5 minutes
- [**Configuration**](../configuration.md) - Learn about configuration options
- [**Deployment Modes**](../deployment-modes/forwardauth-traefik.md) - Choose your deployment mode
