# Troubleshooting User Management

This guide covers common issues and solutions for Keyline's dynamic user management feature.

## Common Issues

### 1. Encryption Key Validation Errors

#### Error: "encryption_key must be 32 bytes"

**Cause**: The encryption key is not exactly 32 bytes when base64-decoded.

**Solution**:
```bash
# Generate a correct 32-byte key
openssl rand -base64 32

# Verify the key length
echo -n "your-base64-key" | base64 -d | wc -c
# Should output: 32

# Set as environment variable
export CACHE_ENCRYPTION_KEY="your-generated-key"
```

**Configuration**:
```yaml
cache:
  encryption_key: ${CACHE_ENCRYPTION_KEY}
```

#### Error: "failed to decode encryption key"

**Cause**: The encryption key is not valid base64.

**Solution**:
```bash
# Ensure the key is properly base64-encoded
openssl rand -base64 32

# Test decoding
echo "your-key" | base64 -d
# Should not produce errors
```

### 2. Admin Credentials Issues

#### Error: "ES admin credentials invalid or insufficient privileges"

**Cause**: Admin user doesn't exist or lacks `manage_security` privilege.

**Solution**:

1. **Verify admin user exists**:
   ```bash
   curl -X GET "https://elasticsearch:9200/_security/user/keyline_admin" \
     -u elastic:elastic_password
   ```

2. **Create admin user if missing**:
   ```bash
   curl -X POST "https://elasticsearch:9200/_security/user/keyline_admin" \
     -u elastic:elastic_password \
     -H "Content-Type: application/json" \
     -d '{
       "password": "secure_password",
       "roles": ["superuser"],
       "full_name": "Keyline Admin"
     }'
   ```

3. **Verify admin user has correct privileges**:
   ```bash
   curl -X GET "https://elasticsearch:9200/_security/user/keyline_admin" \
     -u keyline_admin:password
   ```

4. **Test admin credentials**:
   ```bash
   curl -X GET "https://elasticsearch:9200/_security/user" \
     -u keyline_admin:password
   ```

#### Error: "connection refused" when connecting to Elasticsearch

**Cause**: Elasticsearch URL is incorrect or ES is not accessible.

**Solution**:

1. **Verify Elasticsearch URL**:
   ```yaml
   elasticsearch:
     url: https://elasticsearch:9200  # Check protocol (http/https) and port
   ```

2. **Test connectivity**:
   ```bash
   curl -v https://elasticsearch:9200
   ```

3. **Check network/firewall rules**:
   - Ensure Keyline can reach Elasticsearch
   - Verify DNS resolution
   - Check firewall rules

4. **For TLS issues**:
   ```yaml
   elasticsearch:
     insecure_skip_verify: true  # Only for development with self-signed certs
   ```

### 3. Role Mapping Issues

#### Error: "no role mappings matched and no default roles configured"

**Cause**: User's groups don't match any role mappings and no default roles are configured.

**Solution**:

1. **Add default roles**:
   ```yaml
   default_es_roles:
     - viewer
     - kibana_user
   ```

2. **Or add a catch-all mapping**:
   ```yaml
   role_mappings:
     - claim: groups
       pattern: "*"  # Matches any group
       es_roles:
         - viewer
   ```

3. **Debug role mapping**:
   - Check Keyline logs for role mapping evaluation
   - Verify user groups are being extracted correctly
   - Test with different patterns

#### Issue: User gets wrong ES roles

**Cause**: Role mapping patterns are incorrect or overlapping.

**Solution**:

1. **Review role mappings order**:
   ```yaml
   role_mappings:
     # More specific patterns first
     - claim: groups
       pattern: "admin"
       es_roles:
         - superuser
     
     # Less specific patterns last
     - claim: groups
       pattern: "*"
       es_roles:
         - viewer
   ```

2. **Test pattern matching**:
   - Enable debug logging: `log_level: debug`
   - Check logs for "Role mapping matched" messages
   - Verify which patterns are matching

3. **Verify user groups**:
   ```bash
   # Check what groups the user has
   # Look in Keyline logs for "User authenticated" messages
   ```

4. **Check ES user roles**:
   ```bash
   curl -X GET "https://elasticsearch:9200/_security/user/username" \
     -u keyline_admin:password
   ```

#### Issue: Multiple groups not accumulating roles

**Cause**: Expecting only first match instead of all matches.

**Expected Behavior**: ALL matching role mappings contribute roles (deduplicated).

**Example**:
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

# User with groups ["developers", "team-leads"] gets:
# - developer
# - team_lead
```

### 4. Cache Issues

#### Issue: Low cache hit rate (< 95%)

**Symptoms**:
- High latency on authenticated requests
- Many ES API calls
- `keyline_cred_cache_misses_total` metric is high

**Causes and Solutions**:

1. **Cache TTL too short**:
   ```yaml
   cache:
     credential_ttl: 1h  # Increase from 5m to 1h
   ```

2. **Redis unavailable**:
   ```bash
   # Test Redis connectivity
   redis-cli -h redis-host -p 6379 ping
   # Should return: PONG
   ```

3. **Redis memory limits**:
   ```bash
   # Check Redis memory usage
   redis-cli INFO memory
   
   # Increase maxmemory if needed
   redis-cli CONFIG SET maxmemory 256mb
   ```

4. **Multiple Keyline instances with different encryption keys**:
   - Ensure ALL instances use the SAME encryption key
   - Check environment variables on all instances

#### Error: "failed to decrypt cached password"

**Cause**: Encryption key changed or corrupted cache entry.

**Solution**:

1. **Verify encryption key is consistent**:
   ```bash
   # Check environment variable on all instances
   echo $CACHE_ENCRYPTION_KEY
   ```

2. **Clear cache and regenerate**:
   ```bash
   # Connect to Redis
   redis-cli
   
   # List credential cache keys
   KEYS keyline:user:*:password
   
   # Delete specific user's cache
   DEL keyline:user:alice:password
   
   # Or clear all credential cache (use with caution)
   KEYS keyline:user:*:password | xargs redis-cli DEL
   ```

3. **User will need to re-authenticate** to generate new cached credentials.

#### Issue: Cache not shared across Keyline instances

**Cause**: Using memory backend instead of Redis.

**Solution**:
```yaml
cache:
  backend: redis  # Change from "memory" to "redis"
  redis_url: redis://redis-cluster:6379
  redis_password: ${REDIS_PASSWORD}
  redis_db: 0
  encryption_key: ${CACHE_ENCRYPTION_KEY}
```

**Note**: All instances must use the same Redis and encryption key.

### 5. ES User Creation Failures

#### Error: "ES user upsert failed: 403 Forbidden"

**Cause**: Admin user lacks `manage_security` privilege.

**Solution**:

1. **Grant manage_security privilege**:
   ```bash
   curl -X POST "https://elasticsearch:9200/_security/role/keyline_admin_role" \
     -u elastic:elastic_password \
     -H "Content-Type: application/json" \
     -d '{
       "cluster": ["manage_security"],
       "indices": []
     }'
   
   curl -X PUT "https://elasticsearch:9200/_security/user/keyline_admin" \
     -u elastic:elastic_password \
     -H "Content-Type: application/json" \
     -d '{
       "password": "secure_password",
       "roles": ["keyline_admin_role"]
     }'
   ```

2. **Or use superuser role** (simpler but more privileges):
   ```bash
   curl -X PUT "https://elasticsearch:9200/_security/user/keyline_admin" \
     -u elastic:elastic_password \
     -H "Content-Type: application/json" \
     -d '{
       "password": "secure_password",
       "roles": ["superuser"]
     }'
   ```

#### Error: "ES user upsert failed: 429 Too Many Requests"

**Cause**: Rate limiting by Elasticsearch.

**Solution**:

1. **Increase cache TTL** to reduce ES API calls:
   ```yaml
   cache:
     credential_ttl: 4h  # Increase from 1h
   ```

2. **Check ES rate limiting settings**:
   ```bash
   curl -X GET "https://elasticsearch:9200/_cluster/settings?include_defaults=true" \
     -u admin:password | grep rate
   ```

3. **Implement exponential backoff** (already built into Keyline):
   - Keyline retries with exponential backoff
   - Check logs for retry attempts

#### Error: "ES user upsert failed: timeout"

**Cause**: Elasticsearch is slow or unresponsive.

**Solution**:

1. **Increase timeout**:
   ```yaml
   elasticsearch:
     timeout: 60s  # Increase from 30s
   ```

2. **Check ES cluster health**:
   ```bash
   curl -X GET "https://elasticsearch:9200/_cluster/health" \
     -u admin:password
   ```

3. **Check ES performance**:
   ```bash
   curl -X GET "https://elasticsearch:9200/_nodes/stats" \
     -u admin:password
   ```

4. **Scale ES cluster** if consistently slow.

### 6. Performance Issues

#### Issue: High latency on cache miss (> 500ms p95)

**Causes and Solutions**:

1. **ES cluster slow**:
   - Check ES cluster health and performance
   - Scale ES cluster resources
   - Optimize ES configuration

2. **Network latency**:
   - Measure network latency between Keyline and ES
   - Deploy Keyline closer to ES
   - Use faster network

3. **Too many retries**:
   - Check logs for retry attempts
   - Improve ES stability to reduce retries

#### Issue: High latency on cache hit (> 10ms p95)

**Causes and Solutions**:

1. **Redis slow**:
   ```bash
   # Check Redis latency
   redis-cli --latency
   
   # Check Redis slow log
   redis-cli SLOWLOG GET 10
   ```

2. **Network latency to Redis**:
   - Deploy Keyline closer to Redis
   - Use faster network

3. **Redis overloaded**:
   - Scale Redis (more memory, CPU)
   - Use Redis cluster for high traffic

### 7. Horizontal Scaling Issues

#### Issue: Different Keyline instances creating different passwords

**Cause**: Instances using different encryption keys or not sharing Redis cache.

**Solution**:

1. **Verify all instances use same encryption key**:
   ```bash
   # On each instance
   echo $CACHE_ENCRYPTION_KEY
   # Should be identical
   ```

2. **Verify all instances use same Redis**:
   ```yaml
   cache:
     backend: redis  # Not "memory"
     redis_url: redis://same-redis-cluster:6379
   ```

3. **Test cache sharing**:
   - Authenticate on instance A
   - Verify cached credentials work on instance B

#### Issue: Cache invalidation not working across instances

**Cause**: Using memory backend or different Redis databases.

**Solution**:
```yaml
cache:
  backend: redis
  redis_url: redis://redis-cluster:6379
  redis_db: 0  # Same DB on all instances
```

## Debugging Tips

### Enable Debug Logging

```yaml
observability:
  log_level: debug
```

**What to look for**:
- "User authenticated" - Shows extracted user metadata
- "Role mapping matched" - Shows which mappings matched
- "ES user created" - Confirms user creation
- "ES user updated" - Confirms user update
- "Cache hit" / "Cache miss" - Shows cache behavior
- "Encrypted password" / "Decrypted password" - Shows encryption operations

### Check Metrics

```bash
# Get all metrics
curl http://localhost:9000/metrics

# Filter user management metrics
curl http://localhost:9000/metrics | grep keyline_user
curl http://localhost:9000/metrics | grep keyline_cred_cache
```

**Key metrics**:
- `keyline_user_upserts_total{status="success"}` - Successful user creations
- `keyline_user_upserts_total{status="failure"}` - Failed user creations
- `keyline_user_upsert_duration_seconds` - Upsert latency
- `keyline_cred_cache_hits_total` - Cache hits
- `keyline_cred_cache_misses_total` - Cache misses
- `keyline_role_mapping_matches_total` - Role mapping matches

### Verify ES Users

```bash
# List all ES users
curl -X GET "https://elasticsearch:9200/_security/user" \
  -u keyline_admin:password

# Get specific user
curl -X GET "https://elasticsearch:9200/_security/user/alice" \
  -u keyline_admin:password

# Check user's roles
curl -X GET "https://elasticsearch:9200/_security/user/alice" \
  -u keyline_admin:password | jq '.alice.roles'

# Check user metadata
curl -X GET "https://elasticsearch:9200/_security/user/alice" \
  -u keyline_admin:password | jq '.alice.metadata'
```

### Check ES Audit Logs

```bash
# List audit log indices
curl -X GET "https://elasticsearch:9200/_cat/indices/.security-audit*" \
  -u admin:password

# Search audit logs for user activity
curl -X GET "https://elasticsearch:9200/.security-audit-*/_search" \
  -u admin:password \
  -H "Content-Type: application/json" \
  -d '{
    "query": {
      "match": {
        "user.name": "alice"
      }
    },
    "sort": [{"@timestamp": "desc"}],
    "size": 10
  }'
```

### Test Role Mappings

```bash
# Enable debug logging
export LOG_LEVEL=debug

# Authenticate as test user
curl -u testuser:password https://auth.example.com/

# Check logs for role mapping evaluation
grep "Role mapping" /var/log/keyline.log
```

### Verify Cache Operations

```bash
# Connect to Redis
redis-cli -h redis-host -p 6379

# List credential cache keys
KEYS keyline:user:*:password

# Get cached value (encrypted)
GET keyline:user:alice:password

# Check TTL
TTL keyline:user:alice:password

# Monitor cache operations in real-time
MONITOR
```

## Log Analysis Guide

### Successful User Creation

```
INFO User authenticated username=alice source=oidc groups=[developers,users]
DEBUG Role mapping matched group=developers pattern=developers roles=[developer,kibana_user]
DEBUG Role mapping matched group=users pattern=users roles=[kibana_user]
INFO ES user created username=alice roles=[developer,kibana_user] duration=245ms
DEBUG Encrypted password for cache username=alice
DEBUG Cached credentials username=alice ttl=1h
```

### Cache Hit

```
INFO User authenticated username=alice source=oidc
DEBUG Cache hit username=alice
DEBUG Decrypted password from cache username=alice
INFO Using cached credentials username=alice duration=5ms
```

### Role Mapping Failure

```
INFO User authenticated username=bob source=oidc groups=[unknown-group]
DEBUG Role mapping not matched group=unknown-group pattern=admin
DEBUG Role mapping not matched group=unknown-group pattern=developers
WARN No role mappings matched username=bob groups=[unknown-group]
ERROR User upsert failed username=bob error="no role mappings matched and no default roles configured"
```

### ES API Failure

```
INFO User authenticated username=alice source=oidc
DEBUG Role mapping matched group=admin pattern=admin roles=[superuser]
ERROR ES API call failed operation=create_user username=alice status=403 error="Forbidden"
WARN Retrying ES API call attempt=1 backoff=1s
ERROR ES API call failed operation=create_user username=alice status=403 error="Forbidden"
WARN Retrying ES API call attempt=2 backoff=2s
ERROR ES user upsert failed username=alice error="ES API call failed after 3 attempts"
```

## Getting Help

If you're still experiencing issues:

1. **Check logs** with debug level enabled
2. **Collect metrics** from `/_metrics` endpoint
3. **Verify configuration** with `--validate-config`
4. **Test components individually**:
   - ES connectivity
   - Redis connectivity
   - Admin credentials
   - Role mappings
5. **Review documentation**:
   - [User Management Guide](user-management.md)
   - [Configuration Reference](configuration.md)
   - [Migration Guide](migration-guide.md)

## Common Configuration Mistakes

### 1. Wrong encryption key format

❌ **Wrong**:
```yaml
cache:
  encryption_key: "my-secret-key"  # Not 32 bytes
```

✅ **Correct**:
```yaml
cache:
  encryption_key: ${CACHE_ENCRYPTION_KEY}  # Generated with: openssl rand -base64 32
```

### 2. Missing admin credentials

❌ **Wrong**:
```yaml
user_management:
  enabled: true

# Missing elasticsearch.admin_user and admin_password
```

✅ **Correct**:
```yaml
user_management:
  enabled: true

elasticsearch:
  admin_user: keyline_admin
  admin_password: ${ES_ADMIN_PASSWORD}
  url: https://elasticsearch:9200
```

### 3. No role mappings or defaults

❌ **Wrong**:
```yaml
user_management:
  enabled: true

# No role_mappings or default_es_roles
```

✅ **Correct**:
```yaml
user_management:
  enabled: true

role_mappings:
  - claim: groups
    pattern: "admin"
    es_roles:
      - superuser

default_es_roles:
  - viewer
  - kibana_user
```

### 4. Memory cache with multiple instances

❌ **Wrong**:
```yaml
cache:
  backend: memory  # Not shared across instances
```

✅ **Correct**:
```yaml
cache:
  backend: redis
  redis_url: redis://redis-cluster:6379
  encryption_key: ${CACHE_ENCRYPTION_KEY}
```

### 5. Inconsistent encryption keys

❌ **Wrong**:
```bash
# Instance A
export CACHE_ENCRYPTION_KEY="key1"

# Instance B
export CACHE_ENCRYPTION_KEY="key2"  # Different key!
```

✅ **Correct**:
```bash
# All instances
export CACHE_ENCRYPTION_KEY="same-key-for-all-instances"
```
