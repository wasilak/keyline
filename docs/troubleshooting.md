# Troubleshooting Guide

This guide covers common issues and their solutions when deploying and operating Keyline.

## Table of Contents

- [Configuration Issues](#configuration-issues)
- [OIDC Provider Issues](#oidc-provider-issues)
- [Redis Connection Issues](#redis-connection-issues)
- [Session Issues](#session-issues)
- [Authentication Issues](#authentication-issues)
- [Proxy Issues](#proxy-issues)
- [Logging and Debugging](#logging-and-debugging)

## Configuration Issues

### Configuration File Not Found

**Symptom**: Error message "failed to load configuration: config file not found"

**Solution**:
- Verify the config file path is correct
- Use `--config` flag to specify the path explicitly
- Check file permissions (must be readable)
- Ensure the file exists at the specified location

```bash
# Specify config file explicitly
./keyline --config /path/to/config.yaml

# Validate configuration
./keyline --validate-config --config /path/to/config.yaml
```

### Invalid Configuration Format

**Symptom**: Error message "failed to parse configuration: yaml: ..."

**Solution**:
- Validate YAML syntax using a YAML validator
- Check for proper indentation (use spaces, not tabs)
- Ensure all required fields are present
- Use `--validate-config` flag to check configuration

```bash
# Validate YAML syntax
yamllint config.yaml

# Validate Keyline configuration
./keyline --validate-config --config config.yaml
```

### Environment Variable Substitution Not Working

**Symptom**: Configuration values show `${VAR_NAME}` instead of actual values

**Solution**:
- Ensure environment variables are exported before starting Keyline
- Check variable names match exactly (case-sensitive)
- Verify variables are set in the environment

```bash
# Check if variable is set
echo $OIDC_CLIENT_SECRET

# Export variable if not set
export OIDC_CLIENT_SECRET="your-secret-here"

# Verify substitution
./keyline --validate-config --config config.yaml
```

### Missing Required Configuration

**Symptom**: Error message "validation failed: field X is required"

**Solution**:
- Review the [configuration reference](configuration.md) for required fields
- Ensure all required fields for your deployment mode are set
- Check that OIDC provider configuration is complete if using OIDC

**Required fields by mode**:

ForwardAuth mode:
- `server.mode: forwardAuth`
- `server.port`
- `auth.oidc` or `auth.localUsers`

Standalone mode:
- `server.mode: standalone`
- `server.port`
- `server.upstreamURL`
- `auth.oidc` or `auth.localUsers`

## OIDC Provider Issues

### Discovery Endpoint Unreachable

**Symptom**: Error message "failed to load OIDC discovery: Get ... dial tcp: ..."

**Solution**:
- Verify the issuer URL is correct and accessible
- Check network connectivity to the OIDC provider
- Ensure firewall rules allow outbound HTTPS connections
- Verify DNS resolution works for the provider domain

```bash
# Test connectivity
curl -v https://your-oidc-provider.com/.well-known/openid-configuration

# Check DNS resolution
nslookup your-oidc-provider.com

# Test from container (if using Docker)
docker run --rm curlimages/curl:latest curl -v https://your-oidc-provider.com/.well-known/openid-configuration
```

### Invalid Client Credentials

**Symptom**: Error message "token exchange failed: invalid_client"

**Solution**:
- Verify client ID and client secret are correct
- Check that the client is enabled in the OIDC provider
- Ensure the redirect URI is registered in the OIDC provider
- Verify the client secret hasn't expired

**Redirect URI format**:
- ForwardAuth: `https://your-domain.com/_oauth/callback`
- Standalone: `https://your-domain.com/_oauth/callback`

### Token Validation Fails

**Symptom**: Error message "token validation failed: ..."

**Solution**:
- Verify the OIDC provider's signing keys are accessible
- Check that the token hasn't expired
- Ensure the audience claim matches your client ID
- Verify the issuer claim matches your configured issuer

```bash
# Decode JWT token to inspect claims
echo "YOUR_TOKEN" | cut -d. -f2 | base64 -d | jq .

# Check JWKS endpoint
curl https://your-oidc-provider.com/.well-known/jwks.json
```

### Redirect Loop

**Symptom**: Browser keeps redirecting between Keyline and OIDC provider

**Solution**:
- Verify the redirect URI is correctly configured
- Check that cookies are being set (not blocked by browser)
- Ensure the callback endpoint is accessible
- Verify session store is working correctly
- Check that the state parameter is being preserved

**Debug steps**:
1. Open browser developer tools (Network tab)
2. Clear cookies and cache
3. Attempt authentication
4. Check for `keyline_session` cookie being set
5. Verify redirect URLs in the network log

### Scope Not Granted

**Symptom**: Error message "required scope not granted: ..."

**Solution**:
- Verify the requested scopes are supported by the provider
- Check that the client is authorized for the requested scopes
- Ensure the user has consented to the requested scopes
- Review the OIDC provider's scope configuration

## Redis Connection Issues

### Connection Refused

**Symptom**: Error message "failed to connect to Redis: dial tcp ... connection refused"

**Solution**:
- Verify Redis is running and accessible
- Check the Redis host and port configuration
- Ensure firewall rules allow connections to Redis
- Verify network connectivity

```bash
# Test Redis connectivity
redis-cli -h redis-host -p 6379 ping

# Check Redis is listening
netstat -an | grep 6379

# Test from container (if using Docker)
docker run --rm redis:7-alpine redis-cli -h redis-host -p 6379 ping
```

### Authentication Failed

**Symptom**: Error message "failed to authenticate to Redis: NOAUTH ..."

**Solution**:
- Verify the Redis password is correct
- Check that Redis is configured to require authentication
- Ensure the password is properly set in configuration

```bash
# Test authentication
redis-cli -h redis-host -p 6379 -a your-password ping

# Check Redis AUTH requirement
redis-cli -h redis-host -p 6379 CONFIG GET requirepass
```

### TLS Connection Failed

**Symptom**: Error message "failed to connect to Redis: tls: ..."

**Solution**:
- Verify Redis is configured for TLS
- Check that the TLS certificate is valid
- Ensure `session.redis.tls` is set to `true`
- Verify the certificate chain is trusted

```bash
# Test TLS connection
openssl s_client -connect redis-host:6379 -starttls redis

# Check certificate
echo | openssl s_client -connect redis-host:6379 -starttls redis 2>/dev/null | openssl x509 -noout -text
```

### Session Data Not Persisting

**Symptom**: Sessions are lost after Keyline restart

**Solution**:
- Verify Redis is configured as the session store
- Check that `session.store` is set to `redis`
- Ensure Redis is not being cleared between restarts
- Verify the session TTL is appropriate

```bash
# Check session keys in Redis
redis-cli -h redis-host -p 6379 KEYS "keyline:session:*"

# Check TTL on a session key
redis-cli -h redis-host -p 6379 TTL "keyline:session:abc123"
```

## Session Issues

### Session Expires Too Quickly

**Symptom**: Users are logged out frequently

**Solution**:
- Increase `session.maxAge` in configuration
- Verify the session store TTL matches `maxAge`
- Check that the session cookie is being sent with requests
- Ensure the cookie domain and path are correct

```yaml
session:
  maxAge: 3600  # 1 hour (increase as needed)
```

### Session Cookie Not Set

**Symptom**: Authentication succeeds but subsequent requests are unauthenticated

**Solution**:
- Verify the cookie domain is correct for your deployment
- Check that the cookie is not being blocked by browser settings
- Ensure `session.cookie.secure` matches your HTTPS configuration
- Verify `session.cookie.sameSite` is appropriate

**Cookie settings**:
- `secure: true` requires HTTPS
- `sameSite: lax` is recommended for most deployments
- `domain` should match your application domain

### Session Not Found

**Symptom**: Error message "session not found" for valid session cookie

**Solution**:
- Check that the session store is accessible
- Verify the session hasn't expired
- Ensure the session key prefix matches configuration
- Check Redis connectivity if using Redis store

```bash
# Check if session exists in Redis
redis-cli -h redis-host -p 6379 GET "keyline:session:YOUR_SESSION_ID"
```

## Authentication Issues

### Basic Auth Always Returns 401

**Symptom**: Valid credentials return 401 Unauthorized

**Solution**:
- Verify the username exists in `auth.localUsers`
- Check that the password hash is correct
- Ensure the Authorization header is being sent
- Verify the password was hashed with bcrypt

```bash
# Generate bcrypt hash for password
htpasswd -bnBC 10 "" your-password | tr -d ':\n'

# Test Basic Auth
curl -v -u username:password https://your-domain.com/
```

### OIDC Auth Fails with "invalid_grant"

**Symptom**: Token exchange fails with "invalid_grant" error

**Solution**:
- Verify the authorization code hasn't expired
- Check that the code verifier matches the code challenge (PKCE)
- Ensure the redirect URI matches exactly
- Verify the client credentials are correct

### User Has No Roles

**Symptom**: User authenticates but has no roles assigned

**Solution**:
- Verify the OIDC claim mapping is correct
- Check that the OIDC provider includes the expected claims
- Ensure `auth.oidc.claimMappings.roles` is configured
- Review the ID token claims

```yaml
auth:
  oidc:
    claimMappings:
      roles: groups  # or roles, depending on your provider
```

### Elasticsearch Credentials Not Mapped

**Symptom**: Requests to Elasticsearch fail with authentication error

**Solution**:
- Verify `auth.credentialMapping` is configured
- Check that the role mapping matches user roles
- Ensure the Elasticsearch credentials are valid
- Review the credential mapping rules

```yaml
auth:
  credentialMapping:
    - roles: ["admin"]
      esUsername: "elastic_admin"
      esPassword: "${ES_ADMIN_PASSWORD}"
```

## Proxy Issues

### Upstream Connection Failed

**Symptom**: Error message "failed to proxy request: dial tcp ... connection refused"

**Solution**:
- Verify the upstream URL is correct
- Check that the upstream service is running
- Ensure network connectivity to upstream
- Verify firewall rules allow connections

```bash
# Test upstream connectivity
curl -v http://upstream-host:port/

# Check from Keyline container
docker exec keyline-container curl -v http://upstream-host:port/
```

### Request Headers Not Forwarded

**Symptom**: Upstream service doesn't receive expected headers

**Solution**:
- Verify `server.forwardHeaders` is configured correctly
- Check that the headers are being set by the reverse proxy
- Ensure header names match exactly (case-sensitive)
- Review the reverse proxy configuration

**Traefik example**:
```yaml
http:
  middlewares:
    keyline-auth:
      forwardAuth:
        address: "http://keyline:9000/_auth"
        authResponseHeaders:
          - "X-Forwarded-User"
          - "X-Forwarded-Groups"
```

### WebSocket Upgrade Fails

**Symptom**: WebSocket connections fail to establish

**Solution**:
- Verify `server.enableWebSocket` is set to `true`
- Check that the reverse proxy supports WebSocket upgrades
- Ensure the Upgrade and Connection headers are preserved
- Verify the upstream service supports WebSockets

```yaml
server:
  enableWebSocket: true
```

## Logging and Debugging

### Enable Debug Logging

To enable detailed debug logging:

```yaml
logging:
  level: debug
  format: json
```

Or via environment variable:

```bash
export LOG_LEVEL=debug
./keyline
```

### Structured Logging Fields

Keyline uses structured logging with the following fields:

- `level`: Log level (debug, info, warn, error)
- `msg`: Log message
- `time`: Timestamp
- `component`: Component name (auth, session, proxy, etc.)
- `request_id`: Unique request ID
- `user`: Username (if authenticated)
- `method`: HTTP method
- `path`: Request path
- `status`: HTTP status code
- `duration`: Request duration

### Common Log Messages

**"OIDC discovery loaded successfully"**
- OIDC provider configuration loaded
- Provider is ready for authentication

**"session created"**
- New session created for user
- Check `user` and `session_id` fields

**"session not found"**
- Session cookie is invalid or expired
- User needs to re-authenticate

**"token validation failed"**
- OIDC token validation failed
- Check token expiration and signing keys

**"upstream request failed"**
- Proxy request to upstream failed
- Check upstream connectivity and logs

### Health Check Endpoint

Use the health check endpoint to verify Keyline status:

```bash
# Check health
curl http://localhost:9000/_health

# Expected response (healthy)
{
  "status": "healthy",
  "checks": {
    "session_store": "ok",
    "oidc_provider": "ok"
  }
}

# Expected response (unhealthy)
{
  "status": "unhealthy",
  "checks": {
    "session_store": "ok",
    "oidc_provider": "failed: discovery not loaded"
  }
}
```

### Metrics Endpoint

Use the metrics endpoint to monitor Keyline:

```bash
# Check metrics
curl http://localhost:9000/_metrics

# Key metrics to monitor
# - keyline_auth_attempts_total: Total authentication attempts
# - keyline_auth_successes_total: Successful authentications
# - keyline_auth_failures_total: Failed authentications
# - keyline_session_creates_total: Sessions created
# - keyline_session_lookups_total: Session lookups
# - keyline_proxy_requests_total: Proxied requests
# - keyline_proxy_request_duration_seconds: Request duration
```

### Request Tracing

Enable OpenTelemetry tracing for detailed request traces:

```yaml
tracing:
  enabled: true
  endpoint: "http://jaeger:4318"
  serviceName: "keyline"
```

View traces in Jaeger UI to see:
- Request flow through Keyline
- Authentication steps
- Session lookups
- Proxy requests
- Error details

### Common Debug Scenarios

**Debug OIDC flow**:
1. Enable debug logging
2. Clear browser cookies
3. Attempt authentication
4. Review logs for:
   - "redirecting to OIDC provider"
   - "callback received"
   - "token exchange successful"
   - "session created"

**Debug session issues**:
1. Enable debug logging
2. Check session cookie in browser
3. Review logs for:
   - "session created"
   - "session found"
   - "session not found"
   - "session expired"

**Debug proxy issues**:
1. Enable debug logging
2. Make request to proxied endpoint
3. Review logs for:
   - "proxying request"
   - "upstream request successful"
   - "upstream request failed"

### Getting Help

If you're still experiencing issues:

1. Check the [configuration reference](configuration.md)
2. Review the [deployment guide](deployment.md)
3. Enable debug logging and review logs
4. Check health and metrics endpoints
5. Review OpenTelemetry traces (if enabled)
6. Open an issue on GitHub with:
   - Keyline version
   - Configuration (redact secrets)
   - Relevant log messages
   - Steps to reproduce

## Performance Tuning

### High Latency

**Symptom**: Requests take longer than expected

**Solution**:
- Enable connection pooling for upstream
- Increase `server.readTimeout` and `server.writeTimeout`
- Use Redis for session store (faster than in-memory for distributed deployments)
- Enable OIDC provider caching
- Review upstream service performance

### High Memory Usage

**Symptom**: Keyline consumes excessive memory

**Solution**:
- Use Redis for session store instead of in-memory
- Reduce `session.maxAge` to expire sessions sooner
- Limit the number of concurrent connections
- Review session cleanup interval

### High CPU Usage

**Symptom**: Keyline consumes excessive CPU

**Solution**:
- Reduce bcrypt cost for password hashing (not recommended for production)
- Enable connection pooling
- Review logging level (debug logging is expensive)
- Check for excessive authentication attempts (possible attack)

## Security Considerations

### Suspicious Activity

**Symptom**: High number of failed authentication attempts

**Solution**:
- Review authentication failure metrics
- Check logs for suspicious patterns
- Consider implementing rate limiting
- Review firewall rules
- Enable IP-based access controls

### Token Leakage

**Symptom**: Tokens appear in logs or error messages

**Solution**:
- Verify sensitive values are not logged
- Check error messages for token exposure
- Review logging configuration
- Ensure `logging.redactSensitive` is enabled (if available)

### Session Hijacking

**Symptom**: Unauthorized access to user sessions

**Solution**:
- Ensure `session.cookie.secure` is `true` (HTTPS only)
- Set `session.cookie.httpOnly` to `true`
- Use `session.cookie.sameSite: strict` or `lax`
- Reduce `session.maxAge` for sensitive applications
- Enable session rotation on privilege escalation
