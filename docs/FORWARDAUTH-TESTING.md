# Forward Auth Testing with Traefik

This guide explains how to test Keyline in forward auth mode with Traefik as the reverse proxy.

## Architecture

```
Client → Traefik (port 8080) → Forward Auth Request → Keyline (port 9000)
                              ↓ (if authenticated)
                              → Elasticsearch (port 9200)
```

## Setup

1. **Set environment variables:**
   ```bash
   export ELASTIC_PASSWORD=changeme
   ```

2. **Start the stack:**
   ```bash
   docker-compose -f docker-compose-forwardauth.yml up -d
   ```

3. **Wait for Elasticsearch to be ready:**
   ```bash
   docker-compose -f docker-compose-forwardauth.yml logs -f keyline-es01
   # Wait for "Cluster health status changed from [YELLOW] to [GREEN]"
   ```

## Testing

### Test 1: Access without credentials (should fail)
```bash
curl -v http://es.localhost:8080/_cluster/health?pretty
# Expected: 401 Unauthorized
```

### Test 2: Access with invalid credentials (should fail)
```bash
curl -v -u testuser:wrongpassword http://es.localhost:8080/_cluster/health?pretty
# Expected: 401 Unauthorized with "Authentication failed: invalid credentials"
```

### Test 3: Access with valid credentials (should succeed)
```bash
curl -v -u testuser:password http://es.localhost:8080/_cluster/health?pretty
# Expected: 200 OK with cluster health JSON
```

### Test 4: Session persistence (should work without re-auth)
```bash
# First request creates session
curl -v -u testuser:password -c cookies.txt http://es.localhost:8080/_cluster/health?pretty

# Second request uses session cookie (no credentials needed)
curl -v -b cookies.txt http://es.localhost:8080/_cluster/health?pretty
# Expected: 200 OK (authenticated via session)
```

## How Forward Auth Works

1. **Client sends request** to Traefik at `http://es.localhost:8080/_cluster/health`

2. **Traefik intercepts** and sends forward auth request to Keyline:
   - URL: `http://keyline:9000/auth/verify`
   - Headers: All original request headers (including `Authorization`)

3. **Keyline authenticates**:
   - Validates credentials (Basic Auth or session cookie)
   - Maps user to ES user
   - Returns response:
     - **Success (200)**: Includes `X-Es-Authorization` header with ES credentials
     - **Failure (401)**: Returns error message

4. **Traefik forwards request** to Elasticsearch:
   - If Keyline returned 200: Adds `X-Es-Authorization` header and proxies to ES
   - If Keyline returned 401: Returns 401 to client

5. **Elasticsearch processes** request with ES credentials from `X-Es-Authorization` header

## Traefik Dashboard

Access the Traefik dashboard at: http://localhost:8081

You can see:
- Active routers and middlewares
- Request metrics
- Service health

## Cleanup

```bash
docker-compose -f docker-compose-forwardauth.yml down -v
```

## Configuration Files

- **docker-compose-forwardauth.yml**: Docker Compose setup with Traefik
- **config/test-config-forwardauth.yaml**: Keyline configuration for forward auth mode

## Differences from Standalone Mode

| Feature | Standalone Mode | Forward Auth Mode |
|---------|----------------|-------------------|
| Proxy | Keyline proxies requests | Traefik proxies requests |
| Auth endpoint | All requests go through Keyline | Only `/auth/verify` endpoint used |
| Headers | Keyline replaces Authorization header | Traefik adds X-Es-Authorization header |
| Use case | Simple deployments | Integration with existing reverse proxy |

## Troubleshooting

### Check Keyline logs
```bash
docker-compose -f docker-compose-forwardauth.yml logs -f keyline
```

### Check Traefik logs
```bash
docker-compose -f docker-compose-forwardauth.yml logs -f traefik
```

### Check Elasticsearch logs
```bash
docker-compose -f docker-compose-forwardauth.yml logs -f keyline-es01
```

### Verify Traefik configuration
Visit http://localhost:8081 and check:
- Router `elasticsearch` is active
- Middleware `keyline-auth` is attached
- Service `elasticsearch` is healthy
