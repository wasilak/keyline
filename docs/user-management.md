# Dynamic Elasticsearch User Management

## Overview

Keyline's dynamic user management feature automatically creates and manages Elasticsearch users for all authenticated users, regardless of authentication method (OIDC, Basic Auth, etc.). This provides:

- **Accountability**: Each user has their own ES account with a unique username
- **Auditing**: ES audit logs show actual usernames instead of shared accounts
- **Security**: Random, short-lived passwords that are automatically rotated
- **Role-based access**: User groups/claims are mapped to Elasticsearch roles
- **Scalability**: Redis cache enables horizontal scaling across multiple Keyline instances

## How It Works

### Authentication Flow

1. User authenticates via any supported method (OIDC, Basic Auth, etc.)
2. Keyline extracts user metadata (username, groups, email, full name)
3. Keyline checks the credential cache for existing ES credentials
4. If cache miss or expired:
   - Generates a cryptographically secure random password (32+ characters)
   - Maps user groups to Elasticsearch roles using configured role mappings
   - Creates or updates the ES user via Security API
   - Encrypts and caches the credentials with configurable TTL
5. Keyline forwards requests to Elasticsearch with the user's credentials
6. ES audit logs show the actual username for all operations

### Cache Behavior

- **Cache Hit**: Credentials retrieved from cache, decrypted, and used immediately (< 10ms)
- **Cache Miss**: New password generated, ES user created/updated, credentials cached (< 500ms)
- **Cache Expiry**: After TTL expires, new password is generated on next authentication
- **Redis Backend**: Enables horizontal scaling - multiple Keyline instances share the same cache
- **Memory Backend**: Single-node deployment only - cache is not shared across instances

### Password Security

- Passwords are generated using `crypto/rand` (cryptographically secure)
- Minimum length: 32 characters with mixed case, digits, and special characters
- Passwords are **encrypted** before storing in cache using AES-256-GCM
- Passwords are **never logged** or exposed in error messages
- Passwords are automatically rotated when cache TTL expires

## Configuration Guide

### Basic Configuration

```yaml
# Enable dynamic user management
user_management:
  enabled: true
  password_length: 32        # Length of generated passwords
  credential_ttl: 1h         # How long passwords are cached

# Elasticsearch admin credentials for user management
elasticsearch:
  admin_user: admin
  admin_password: ${ES_ADMIN_PASSWORD}  # Use environment variable
  url: https://elasticsearch:9200
  timeout: 30s
  insecure_skip_verify: false  # Set to true only for development

# Cache configuration
cache:
  backend: redis              # "redis" or "memory"
  redis_url: redis://localhost:6379
  redis_password: ${REDIS_PASSWORD}
  redis_db: 0
  credential_ttl: 1h
  encryption_key: ${CACHE_ENCRYPTION_KEY}  # REQUIRED: 32 bytes base64-encoded

# Role mappings (applies to ALL authentication methods)
role_mappings:
  - claim: groups
    pattern: "admin"
    es_roles:
      - superuser
  
  - claim: groups
    pattern: "developers"
    es_roles:
      - developer
      - kibana_user
  
  - claim: email
    pattern: "*@admin.example.com"
    es_roles:
      - superuser

# Default roles when no mappings match
default_es_roles:
  - viewer
  - kibana_user
```

### Admin Credentials

The admin user must have the `manage_security` privilege in Elasticsearch:

```bash
# Create admin user in Elasticsearch
curl -X POST "https://elasticsearch:9200/_security/user/keyline_admin" \
  -H "Content-Type: application/json" \
  -d '{
    "password": "secure_password_here",
    "roles": ["superuser"],
    "full_name": "Keyline Admin",
    "email": "admin@example.com"
  }'
```

Or use an existing admin user with appropriate privileges.

### Encryption Key Setup

The encryption key **must be 32 bytes** (256 bits) for AES-256-GCM encryption:

```bash
# Generate a secure encryption key
openssl rand -base64 32

# Set as environment variable
export CACHE_ENCRYPTION_KEY="your-generated-key-here"
```

**IMPORTANT**: 
- Store the encryption key securely (use environment variables, not config files)
- All Keyline instances must use the **same encryption key** (for Redis cache)
- Rotating the encryption key will invalidate all cached credentials

### Cache Backend Selection

#### Redis (Recommended for Production)

Use Redis for:
- **Horizontal scaling**: Multiple Keyline instances share the same cache
- **High availability**: Redis persistence and replication
- **Production deployments**: Distributed cache across instances

```yaml
cache:
  backend: redis
  redis_url: redis://redis-cluster:6379
  redis_password: ${REDIS_PASSWORD}
  redis_db: 0
  credential_ttl: 1h
  encryption_key: ${CACHE_ENCRYPTION_KEY}
```

#### Memory (Development/Single-Node)

Use in-memory cache for:
- **Development**: Quick setup without Redis dependency
- **Single-node deployments**: No need for distributed cache
- **Testing**: Isolated cache per instance

```yaml
cache:
  backend: memory
  credential_ttl: 1h
  encryption_key: ${CACHE_ENCRYPTION_KEY}
```

**Note**: Memory cache is not shared across Keyline instances. Each instance maintains its own cache.

## Role Mapping Examples

### Basic Group Mapping

Map user groups to Elasticsearch roles:

```yaml
role_mappings:
  # Exact match
  - claim: groups
    pattern: "admin"
    es_roles:
      - superuser
  
  # Multiple roles for one group
  - claim: groups
    pattern: "developers"
    es_roles:
      - developer
      - kibana_user
      - monitoring_user
```

### Wildcard Patterns

Use wildcards for flexible matching:

```yaml
role_mappings:
  # Prefix wildcard: matches "admin@domain1.com", "admin@domain2.com"
  - claim: groups
    pattern: "admin@*"
    es_roles:
      - superuser
  
  # Suffix wildcard: matches "user@example.com", "admin@example.com"
  - claim: email
    pattern: "*@example.com"
    es_roles:
      - company_user
  
  # Middle wildcard: matches "admin@us.example.com", "admin@eu.example.com"
  - claim: groups
    pattern: "admin@*.example.com"
    es_roles:
      - regional_admin
```

### Multiple Groups → Multiple Roles

Users with multiple groups accumulate roles from all matching mappings:

```yaml
role_mappings:
  - claim: groups
    pattern: "developers"
    es_roles:
      - developer
  
  - claim: groups
    pattern: "team-leads"
    es_roles:
      - team_lead
      - kibana_admin

# User with groups ["developers", "team-leads"] gets roles:
# - developer
# - team_lead
# - kibana_admin
```

### Default Roles Fallback

Default roles are used **only when NO mappings match**:

```yaml
role_mappings:
  - claim: groups
    pattern: "admin"
    es_roles:
      - superuser

default_es_roles:
  - viewer
  - kibana_user

# User with groups ["admin"] → gets "superuser" role (mapping matched)
# User with groups ["unknown"] → gets "viewer" and "kibana_user" (no match, use defaults)
# User with no groups → gets "viewer" and "kibana_user" (no match, use defaults)
```

### No Default Roles (Deny Access)

If no mappings match and no default roles are configured, access is denied:

```yaml
role_mappings:
  - claim: groups
    pattern: "admin"
    es_roles:
      - superuser

# default_es_roles not defined

# User with groups ["admin"] → gets "superuser" role (mapping matched)
# User with groups ["unknown"] → access denied (no match, no defaults)
# User with no groups → access denied (no match, no defaults)
```

## Security Considerations

### Encryption Key Management

**Critical Security Requirements**:

1. **Key Generation**: Use cryptographically secure random generation
   ```bash
   openssl rand -base64 32
   ```

2. **Key Storage**: 
   - ✅ Store in environment variables
   - ✅ Use secrets management (Vault, AWS Secrets Manager, etc.)
   - ❌ Never commit to version control
   - ❌ Never store in config files

3. **Key Distribution**: All Keyline instances must use the same key (for Redis)

4. **Key Rotation**: 
   - Rotating the key invalidates all cached credentials
   - Users will need to re-authenticate after rotation
   - Plan rotation during maintenance windows

### Admin Credentials

- Use strong passwords for the admin user
- Limit admin user to `manage_security` privilege only (principle of least privilege)
- Rotate admin credentials periodically
- Store admin password in environment variables, not config files
- Monitor admin user activity in ES audit logs

### TLS/SSL

Always use TLS for production:

```yaml
elasticsearch:
  url: https://elasticsearch:9200  # Use HTTPS
  insecure_skip_verify: false      # Validate certificates
```

Only set `insecure_skip_verify: true` for development/testing with self-signed certificates.

### Password Security

- Generated passwords are 32+ characters with high entropy
- Passwords are encrypted in cache using AES-256-GCM
- Passwords are never logged or exposed in error messages
- Passwords are automatically rotated via cache TTL
- Each encryption uses a random nonce (prevents pattern analysis)

### Audit Trail

- All user creation/update operations are logged by Keyline
- ES audit logs show actual usernames for all operations
- User metadata includes source (OIDC, Basic Auth, etc.) and last authentication time
- Monitor logs for suspicious activity or failed user management operations

## Performance Tuning

### Cache TTL Optimization

Balance security and performance:

- **Short TTL (5-15 minutes)**: More secure, more ES API calls, higher latency
- **Medium TTL (1 hour)**: Balanced approach (recommended)
- **Long TTL (4-24 hours)**: Better performance, less secure, fewer ES API calls

```yaml
cache:
  credential_ttl: 1h  # Adjust based on your security requirements
```

### Cache Hit Rate

Target: > 95% cache hit rate for active users

Monitor metrics:
- `keyline_cred_cache_hits_total`
- `keyline_cred_cache_misses_total`

If cache hit rate is low:
- Increase cache TTL
- Verify Redis is accessible and performing well
- Check for cache eviction due to memory limits

### ES API Performance

User upsert performance targets:
- Cache hit: < 10ms (p95)
- Cache miss: < 500ms (p95)

If performance is poor:
- Check ES cluster health and response times
- Verify network latency between Keyline and ES
- Review ES Security API performance metrics
- Consider increasing ES cluster resources

### Horizontal Scaling

For high-traffic deployments:

1. **Use Redis cache** (required for horizontal scaling)
2. **Deploy multiple Keyline instances** behind a load balancer
3. **Use the same encryption key** across all instances
4. **Monitor cache hit rate** across all instances
5. **Scale Redis** if it becomes a bottleneck

```yaml
cache:
  backend: redis
  redis_url: redis://redis-cluster:6379
  # All instances use the same Redis and encryption key
```

### Connection Pooling

Keyline uses connection pooling for ES API calls:

```yaml
elasticsearch:
  timeout: 30s  # Request timeout
  # Connection pooling is automatic
```

## Encryption Key Rotation Procedure

Rotating the encryption key invalidates all cached credentials. Follow this procedure:

### Step 1: Plan Rotation

- Schedule during maintenance window or low-traffic period
- Notify users that they may need to re-authenticate
- Prepare new encryption key

### Step 2: Generate New Key

```bash
# Generate new 32-byte key
openssl rand -base64 32
```

### Step 3: Update Configuration

For **zero-downtime rotation** (recommended):

1. Deploy new Keyline instances with new key
2. Gradually shift traffic to new instances
3. Decommission old instances

For **maintenance window rotation**:

1. Stop all Keyline instances
2. Update encryption key in environment variables
3. Clear Redis cache (optional, will happen automatically)
4. Start Keyline instances with new key

### Step 4: Clear Cache (Optional)

```bash
# Connect to Redis
redis-cli

# Clear all Keyline credential cache entries
KEYS keyline:user:*:password
# Review keys, then delete
DEL keyline:user:alice:password keyline:user:bob:password ...

# Or flush entire database (use with caution)
FLUSHDB
```

### Step 5: Verify

- Monitor logs for successful user authentications
- Verify new credentials are being cached
- Check cache hit rate returns to normal (> 95%)
- Confirm ES audit logs show user activity

### Step 6: Document

- Record rotation date and reason
- Update key storage/secrets management
- Notify team of successful rotation

## Monitoring and Troubleshooting

### Key Metrics

Monitor these Prometheus metrics:

- `keyline_user_upserts_total{status="success|failure"}` - User creation/update count
- `keyline_user_upsert_duration_seconds{cache_status="hit|miss"}` - Upsert latency
- `keyline_cred_cache_hits_total` - Cache hit count
- `keyline_cred_cache_misses_total` - Cache miss count
- `keyline_role_mapping_matches_total{pattern="..."}` - Role mapping matches
- `keyline_es_api_calls_total{operation="...",status="..."}` - ES API call count

### Health Checks

Verify user management is working:

```bash
# Check Keyline logs for user management activity
grep "ES user created" /var/log/keyline.log
grep "ES user updated" /var/log/keyline.log

# Check cache hit rate
curl http://localhost:9000/metrics | grep keyline_cred_cache

# Check ES for created users
curl -X GET "https://elasticsearch:9200/_security/user" \
  -u admin:password
```

### Common Issues

See [Troubleshooting User Management](troubleshooting-user-management.md) for detailed troubleshooting guide.

## Best Practices

1. **Use Redis for production** - Enables horizontal scaling and high availability
2. **Secure encryption key** - Store in secrets management, never in config files
3. **Use environment variables** - For admin password and encryption key
4. **Enable TLS** - Always use HTTPS for ES API calls in production
5. **Monitor cache hit rate** - Target > 95% for optimal performance
6. **Set appropriate TTL** - Balance security (shorter) and performance (longer)
7. **Test role mappings** - Verify users get correct roles before production
8. **Monitor ES audit logs** - Verify actual usernames appear in logs
9. **Plan key rotation** - Rotate encryption key periodically
10. **Use wildcard patterns carefully** - Test patterns to avoid unintended matches

## Example Configurations

### Development Setup

```yaml
user_management:
  enabled: true
  password_length: 32
  credential_ttl: 15m  # Short TTL for testing

elasticsearch:
  admin_user: admin
  admin_password: admin_password
  url: http://localhost:9200
  insecure_skip_verify: true  # OK for development

cache:
  backend: memory  # No Redis needed
  credential_ttl: 15m
  encryption_key: "dev-key-not-for-production-use-only"

role_mappings:
  - claim: groups
    pattern: "admin"
    es_roles:
      - superuser

default_es_roles:
  - viewer
```

### Production Setup

```yaml
user_management:
  enabled: true
  password_length: 32
  credential_ttl: 1h

elasticsearch:
  admin_user: keyline_admin
  admin_password: ${ES_ADMIN_PASSWORD}
  url: https://elasticsearch.prod.example.com:9200
  timeout: 30s
  insecure_skip_verify: false

cache:
  backend: redis
  redis_url: redis://redis-cluster.prod.example.com:6379
  redis_password: ${REDIS_PASSWORD}
  redis_db: 0
  credential_ttl: 1h
  encryption_key: ${CACHE_ENCRYPTION_KEY}

role_mappings:
  - claim: groups
    pattern: "admin"
    es_roles:
      - superuser
  
  - claim: groups
    pattern: "developers"
    es_roles:
      - developer
      - kibana_user
  
  - claim: groups
    pattern: "analysts"
    es_roles:
      - analyst
      - kibana_user
  
  - claim: email
    pattern: "*@admin.example.com"
    es_roles:
      - superuser

default_es_roles:
  - viewer
  - kibana_user
```

## Migration from Static User Mapping

See [Migration Guide](migration-guide.md) for detailed migration instructions.

## Further Reading

- [Configuration Reference](configuration.md) - Complete configuration documentation
- [Migration Guide](migration-guide.md) - Migrating from static user mapping
- [Troubleshooting](troubleshooting-user-management.md) - Common issues and solutions
