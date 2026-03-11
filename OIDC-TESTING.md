# OIDC Testing Guide for Keyline

This guide covers testing Keyline with OIDC authentication using a mock OAuth2 server.

## Overview

We use `ghcr.io/navikt/mock-oauth2-server` as a lightweight OIDC provider for testing. This allows us to test:
- OIDC discovery and JWKS fetching
- Authorization flow with PKCE
- Token exchange and ID token validation
- Session creation and management
- Logout functionality

## Test Scenarios

### 1. Standalone Proxy Mode with OIDC
### 2. Forward Auth Mode (Traefik) with OIDC

---

## Scenario 1: Standalone Proxy Mode with OIDC

### Architecture
```
Browser → Keyline (localhost:9000) → Elasticsearch (localhost:9201)
                ↓
         Mock OIDC Provider (localhost:8090)
```

### Setup

1. **Start the services:**
```bash
export ELASTIC_PASSWORD=changeme
docker-compose -f docker-compose-oidc.yml up -d
```

2. **Verify services are running:**
```bash
# Check Elasticsearch
curl -k -u elastic:changeme https://localhost:9201

# Check Mock OIDC Provider
curl http://localhost:8090/default/.well-known/openid-configuration
```

3. **Start Keyline:**
```bash
ES_PASSWORD=changeme go run ./cmd/keyline --config config/test-config-oidc.yaml
```

### Testing OIDC Flow

#### Test 1: Unauthenticated Request Triggers OIDC Redirect

```bash
curl -v http://localhost:9000/_cluster/health
```

**Expected:**
- 302 redirect to mock OIDC provider
- Location header contains authorization URL with:
  - `client_id=keyline-test`
  - `response_type=code`
  - `scope=openid profile email`
  - `state=<random_token>`
  - `code_challenge=<pkce_challenge>`
  - `code_challenge_method=S256`

#### Test 2: Complete OIDC Flow in Browser

1. **Open browser and navigate to:**
```
http://localhost:9000/_cluster/health
```

2. **You'll be redirected to mock OIDC login page:**
   - The mock server shows an interactive login form
   - Enter any username (e.g., `testuser@example.com`)
   - Click "Sign in"

3. **After successful authentication:**
   - You'll be redirected back to Keyline callback
   - Keyline exchanges the code for tokens
   - Session cookie is set
   - You're redirected to original URL (`/_cluster/health`)
   - Elasticsearch response is displayed

#### Test 3: Subsequent Requests Use Session Cookie

```bash
# First, get the session cookie from browser dev tools or use curl with cookie jar
curl -v -c cookies.txt http://localhost:9000/_cluster/health

# Follow the redirects and complete OIDC flow in browser
# Then use the session cookie for subsequent requests
curl -v -b cookies.txt http://localhost:9000/_cluster/health
```

**Expected:**
- No redirect (session cookie is valid)
- Direct response from Elasticsearch
- Keyline logs show session validation

#### Test 4: Basic Auth Still Works (Dual Authentication)

```bash
curl -u testuser:password http://localhost:9000/_cluster/health
```

**Expected:**
- 200 OK with cluster health
- No OIDC redirect (Basic Auth takes precedence)

#### Test 5: Logout

```bash
# With session cookie
curl -v -b cookies.txt http://localhost:9000/auth/logout
```

**Expected:**
- Session deleted from store
- Cookie cleared (Max-Age=0)
- 200 OK or redirect to logout URL

### Verification

Check Keyline logs for:
```
level=INFO msg="OIDC discovery successful"
level=INFO msg="JWKS fetched successfully"
level=INFO msg="OIDC authorization flow initiated" state_token=...
level=INFO msg="OIDC callback received" state=... code=...
level=INFO msg="Token exchange successful"
level=INFO msg="ID token validated successfully"
level=INFO msg="Session created" username=testuser@example.com es_user=elastic
```

---

## Scenario 2: Forward Auth Mode (Traefik) with OIDC

### Architecture
```
Browser → Traefik (localhost:8082) → Elasticsearch (localhost:9202)
              ↓
         Keyline (localhost:9001) ← Forward Auth
              ↓
         Mock OIDC Provider (localhost:8091)
```

### Setup

1. **Start the services:**
```bash
export ELASTIC_PASSWORD=changeme
docker-compose -f docker-compose-oidc-forwardauth.yml up -d
```

2. **Verify services are running:**
```bash
# Check Elasticsearch
curl -k -u elastic:changeme https://localhost:9202

# Check Mock OIDC Provider
curl http://localhost:8091/default/.well-known/openid-configuration

# Check Traefik dashboard
open http://localhost:8083
```

3. **Start Keyline:**
```bash
ES_PASSWORD=changeme go run ./cmd/keyline --config config/test-config-oidc-forwardauth.yaml
```

### Testing OIDC Flow with Traefik

#### Test 1: Unauthenticated Request Triggers OIDC Redirect

```bash
curl -v http://es-oidc.localhost:8082/_cluster/health
```

**Expected:**
- Traefik calls Keyline at `/auth/verify`
- Keyline returns 302 redirect to OIDC provider
- Browser is redirected to mock OIDC login

#### Test 2: Complete OIDC Flow in Browser

1. **Open browser and navigate to:**
```
http://es-oidc.localhost:8082/_cluster/health
```

2. **You'll be redirected to mock OIDC login page:**
   - Enter any username (e.g., `admin@example.com`)
   - Click "Sign in"

3. **After successful authentication:**
   - Redirected to Keyline callback
   - Session cookie set
   - Redirected to original URL
   - Traefik forwards request to Elasticsearch with Authorization header
   - Cluster health displayed

#### Test 3: Subsequent Requests Use Session Cookie

```bash
# Use browser or curl with cookie jar
curl -v -c cookies.txt -L http://es-oidc.localhost:8082/_cluster/health

# Complete OIDC flow in browser, then:
curl -v -b cookies.txt http://es-oidc.localhost:8082/_cluster/health
```

**Expected:**
- Traefik calls Keyline `/auth/verify`
- Keyline validates session cookie
- Returns 200 with `Authorization` header
- Traefik forwards request to Elasticsearch with that header
- Elasticsearch responds with cluster health

#### Test 4: Basic Auth Still Works

```bash
curl -u testuser:password http://es-oidc.localhost:8082/_cluster/health
```

**Expected:**
- 200 OK with cluster health
- Basic Auth validated by Keyline
- No OIDC redirect

#### Test 5: Logout

```bash
curl -v -b cookies.txt http://localhost:9001/auth/logout
```

**Expected:**
- Session deleted
- Cookie cleared
- 200 OK

### Verification

Check Keyline logs for:
```
level=INFO msg="ForwardAuth request received" method=GET path=/_cluster/health
level=INFO msg="OIDC authorization flow initiated"
level=INFO msg="OIDC callback received"
level=INFO msg="Session created" username=admin@example.com es_user=elastic
level=INFO msg="ForwardAuth authentication successful" username=admin@example.com
```

Check Traefik logs for:
```
level=debug msg="Calling forwardAuth" url=http://host.docker.internal:9001/auth/verify
level=debug msg="Received forwardAuth response" status=200
```

---

## Mock OIDC Provider Details

### Discovery Endpoint
```bash
curl http://localhost:8090/default/.well-known/openid-configuration
```

### Interactive Login
The mock server provides an interactive login page at the authorization endpoint. You can:
- Enter any username (it will be used as the `sub` claim)
- Optionally add custom claims via query parameters

### Custom Claims
You can add custom claims to the ID token by adding them to the authorization URL:
```
http://localhost:8090/default/authorize?...&claims={"email":"user@example.com","name":"Test User"}
```

### Token Introspection
The mock server doesn't validate tokens strictly, making it perfect for testing the OIDC flow without complex setup.

---

## Troubleshooting

### OIDC Discovery Fails
```bash
# Check if mock server is accessible
curl http://localhost:8090/default/.well-known/openid-configuration

# Check Keyline logs for discovery errors
```

### Redirect Loop
- Check that `redirect_url` in config matches Keyline's callback URL
- Verify session cookie is being set (check browser dev tools)
- Check that cookie domain matches the request domain

### Token Exchange Fails
- Check Keyline logs for token exchange errors
- Verify `client_id` and `client_secret` match in config
- Check that mock OIDC server is accessible from Keyline

### Session Not Persisting
- Verify `session_secret` is at least 32 bytes
- Check cookie attributes (HttpOnly, Secure, SameSite)
- Verify cache backend is working (memory or Redis)

### Forward Auth Not Working
- Check Traefik dashboard (http://localhost:8083) for middleware status
- Verify `host.docker.internal` resolves correctly
- Check Traefik logs for forward auth calls
- Verify Keyline is listening on the correct port (9001)

---

## Cleanup

```bash
# Stop standalone proxy mode
docker-compose -f docker-compose-oidc.yml down -v

# Stop forward auth mode
docker-compose -f docker-compose-oidc-forwardauth.yml down -v
```

---

## Summary

You now have two complete OIDC testing environments:

1. **Standalone Proxy Mode** (port 9000)
   - Direct proxy to Elasticsearch
   - OIDC authentication with session management
   - Basic Auth fallback

2. **Forward Auth Mode** (port 9001)
   - Traefik reverse proxy
   - Forward auth to Keyline
   - OIDC authentication with session management
   - Basic Auth fallback

Both modes support:
- ✅ OIDC discovery and JWKS fetching
- ✅ Authorization flow with PKCE
- ✅ Token exchange and validation
- ✅ Session creation and management
- ✅ Dual authentication (OIDC + Basic Auth)
- ✅ Logout functionality
