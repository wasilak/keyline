# Migration Guide: Static to Dynamic User Management

This guide helps you migrate from Keyline's static user mapping to dynamic user management.

## Overview

**Static User Mapping (Old)**:
- Pre-configured ES users in `elasticsearch.users`
- OIDC/local users mapped to specific ES users via configuration
- Shared ES accounts across multiple users
- Limited auditing (ES logs show shared usernames)

**Dynamic User Management (New)**:
- ES users created automatically for each authenticated user
- User groups/claims mapped to ES roles
- Unique ES account per user
- Full auditing (ES logs show actual usernames)
- Encrypted credential caching for performance
- Horizontal scaling support with Redis

## Breaking Changes

### 1. LocalUser.ESUser Field Removed

**Old Configuration**:
```yaml
local_users:
  users:
    - username: alice
      password_bcrypt: $2a$10$...
      es_user: admin  # ❌ Removed
```

**New Configuration**:
```yaml
local_users:
  users:
    - username: alice
      password_bcrypt: $2a$10$...
      groups:  # ✅ Use groups instead
        - admin
      email: alice@example.com
      full_name: Alice Admin
```

### 2. OIDC Mappings Changed

**Old Configuration**:
```yaml
oidc:
  mappings:
    - claim: email
      pattern: "*@admin.example.com"
      es_user: admin  # ❌ Maps to ES user
  default_es_user: readonly
```

**New Configuration**:
```yaml
role_mappings:  # ✅ Top-level, applies to all auth methods
  - claim: email
    pattern: "*@admin.example.com"
    es_roles:  # ✅ Maps to ES roles
      - superuser

default_es_roles:  # ✅ Top-level default
  - viewer
  - kibana_user
```

### 3. Elasticsearch Configuration Changed

**Old Configuration**:
```yaml
elasticsearch:
  users:  # ❌ Static user list
    - username: admin
      password: ${ES_ADMIN_PASSWORD}
    - username: readonly
      password: ${ES_READONLY_PASSWORD}
```

**New Configuration**:
```yaml
elasticsearch:
  admin_user: keyline_admin  # ✅ Admin for user management
  admin_password: ${ES_ADMIN_PASSWORD}
  url: https://elasticsearch:9200
  timeout: 30s
```

### 4. New Required Configuration

**Cache Encryption Key** (required):
```yaml
cache:
  encryption_key: ${CACHE_ENCRYPTION_KEY}  # ✅ New required field
```

**User Management** (optional, but recommended):
```yaml
user_management:
  enabled: true  # ✅ Enable dynamic user management
  password_length: 32
  credential_ttl: 1h
```

## Migration Steps

### Step 1: Prepare Elasticsearch

Create an admin user with `manage_security` privilege:

```bash
# Create Keyline admin user in Elasticsearch
curl -X POST "https://elasticsearch:9200/_security/user/keyline_admin" \
  -u elastic:elastic_password \
  -H "Content-Type: application/json" \
  -d '{
    "password": "secure_admin_password",
    "roles": ["superuser"],
    "full_name": "Keyline Admin",
    "email": "keyline@example.com",
    "metadata": {
      "managed_by": "keyline"
    }
  }'
```

Or use an existing admin user with appropriate privileges.

### Step 2: Create Encryption Key

Generate a 32-byte encryption key:

```bash
# Generate encryption key
openssl rand -base64 32

# Set as environment variable
export CACHE_ENCRYPTION_KEY="your-generated-key-here"
```

**Important**: Store this key securely. All Keyline instances must use the same key.

### Step 3: Map Users to Groups

Identify your current user-to-ES-user mappings and convert them to group-based mappings.

**Example Conversion**:

Old static mapping:
- alice@example.com → admin ES user
- bob@example.com → developer ES user
- charlie@example.com → readonly ES user

New group-based mapping:
- alice@example.com → groups: ["admin"] → ES roles: ["superuser"]
- bob@example.com → groups: ["developers"] → ES roles: ["developer", "kibana_user"]
- charlie@example.com → no groups → ES roles: ["viewer", "kibana_user"] (default)

### Step 4: Update Configuration

#### 4.1 Update Local Users

**Before**:
```yaml
local_users:
  enabled: true
  users:
    - username: alice
      password_bcrypt: $2a$10$...
      es_user: admin
    
    - username: bob
      password_bcrypt: $2a$10$...
      es_user: developer
```

**After**:
```yaml
local_users:
  enabled: true
  users:
    - username: alice
      password_bcrypt: $2a$10$...
      groups:
        - admin
      email: alice@example.com
      full_name: Alice Admin
    
    - username: bob
      password_bcrypt: $2a$10$...
      groups:
        - developers
      email: bob@example.com
      full_name: Bob Developer
```

#### 4.2 Move OIDC Mappings to Role Mappings

**Before**:
```yaml
oidc:
  enabled: true
  # ... other OIDC config ...
  mappings:
    - claim: email
      pattern: "*@admin.example.com"
      es_user: admin
    - claim: email
      pattern: "*@example.com"
      es_user: readonly
  default_es_user: readonly
```

**After**:
```yaml
oidc:
  enabled: true
  # ... other OIDC config ...
  # Remove mappings and default_es_user

# Add top-level role_mappings
role_mappings:
  - claim: email
    pattern: "*@admin.example.com"
    es_roles:
      - superuser
  
  - claim: email
    pattern: "*@example.com"
    es_roles:
      - viewer
      - kibana_user

default_es_roles:
  - viewer
  - kibana_user
```

#### 4.3 Update Elasticsearch Configuration

**Before**:
```yaml
elasticsearch:
  users:
    - username: admin
      password: ${ES_ADMIN_PASSWORD}
    - username: developer
      password: ${ES_DEVELOPER_PASSWORD}
    - username: readonly
      password: ${ES_READONLY_PASSWORD}
```

**After**:
```yaml
elasticsearch:
  admin_user: keyline_admin
  admin_password: ${ES_ADMIN_PASSWORD}
  url: https://elasticsearch:9200
  timeout: 30s
  insecure_skip_verify: false
```

#### 4.4 Add User Management Configuration

```yaml
user_management:
  enabled: true
  password_length: 32
  credential_ttl: 1h
```

#### 4.5 Update Cache Configuration

**Before**:
```yaml
cache:
  backend: redis
  redis_url: redis://localhost:6379
  redis_db: 0
```

**After**:
```yaml
cache:
  backend: redis
  redis_url: redis://localhost:6379
  redis_db: 0
  credential_ttl: 1h
  encryption_key: ${CACHE_ENCRYPTION_KEY}
```

### Step 5: Update Environment Variables

**Before**:
```bash
export SESSION_SECRET=$(openssl rand -base64 32)
export OIDC_CLIENT_SECRET=your-oidc-secret
export ES_ADMIN_PASSWORD=admin-password
export ES_DEVELOPER_PASSWORD=developer-password
export ES_READONLY_PASSWORD=readonly-password
```

**After**:
```bash
export SESSION_SECRET=$(openssl rand -base64 32)
export CACHE_ENCRYPTION_KEY=$(openssl rand -base64 32)
export OIDC_CLIENT_SECRET=your-oidc-secret
export ES_ADMIN_PASSWORD=keyline-admin-password
```

### Step 6: Test Configuration

Validate the new configuration:

```bash
keyline --validate-config --config config.yaml
```

Fix any validation errors before proceeding.

### Step 7: Deploy

#### Option A: Blue-Green Deployment (Zero Downtime)

1. Deploy new Keyline instances with new configuration
2. Verify new instances are working correctly
3. Gradually shift traffic to new instances
4. Monitor ES audit logs for actual usernames
5. Decommission old instances

#### Option B: Maintenance Window Deployment

1. Schedule maintenance window
2. Stop old Keyline instances
3. Deploy new Keyline instances with new configuration
4. Verify functionality
5. Resume traffic

### Step 8: Verify Migration

1. **Test Authentication**:
   ```bash
   # Test OIDC login
   curl -v https://auth.example.com/
   
   # Test Basic Auth
   curl -u alice:password https://auth.example.com/
   ```

2. **Check ES Users Created**:
   ```bash
   # List ES users
   curl -X GET "https://elasticsearch:9200/_security/user" \
     -u keyline_admin:password
   
   # Should see users like: alice, bob, charlie (not admin, developer, readonly)
   ```

3. **Verify ES Audit Logs**:
   ```bash
   # Check ES audit logs
   curl -X GET "https://elasticsearch:9200/_cat/indices/.security-audit*"
   
   # Logs should show actual usernames (alice, bob) not shared accounts
   ```

4. **Monitor Metrics**:
   ```bash
   # Check cache hit rate
   curl http://localhost:9000/metrics | grep keyline_cred_cache
   
   # Target: > 95% cache hit rate
   ```

5. **Test Role Mappings**:
   - Authenticate as different users
   - Verify they have correct ES roles
   - Test access to ES resources based on roles

## Configuration Examples

### Example 1: Simple Migration

**Before**:
```yaml
oidc:
  enabled: true
  issuer_url: https://accounts.google.com
  client_id: your-client-id
  client_secret: ${OIDC_CLIENT_SECRET}
  redirect_url: https://auth.example.com/auth/callback
  mappings:
    - claim: email
      pattern: "*@admin.example.com"
      es_user: admin
  default_es_user: readonly

elasticsearch:
  users:
    - username: admin
      password: ${ES_ADMIN_PASSWORD}
    - username: readonly
      password: ${ES_READONLY_PASSWORD}
```

**After**:
```yaml
oidc:
  enabled: true
  issuer_url: https://accounts.google.com
  client_id: your-client-id
  client_secret: ${OIDC_CLIENT_SECRET}
  redirect_url: https://auth.example.com/auth/callback
  scopes:
    - openid
    - email
    - profile

user_management:
  enabled: true
  password_length: 32
  credential_ttl: 1h

elasticsearch:
  admin_user: keyline_admin
  admin_password: ${ES_ADMIN_PASSWORD}
  url: https://elasticsearch:9200
  timeout: 30s

cache:
  backend: redis
  redis_url: redis://localhost:6379
  redis_db: 0
  credential_ttl: 1h
  encryption_key: ${CACHE_ENCRYPTION_KEY}

role_mappings:
  - claim: email
    pattern: "*@admin.example.com"
    es_roles:
      - superuser

default_es_roles:
  - viewer
  - kibana_user
```

### Example 2: Complex Migration with Multiple Auth Methods

**Before**:
```yaml
oidc:
  enabled: true
  # ... OIDC config ...
  mappings:
    - claim: email
      pattern: "*@admin.example.com"
      es_user: admin
    - claim: email
      pattern: "*@dev.example.com"
      es_user: developer
  default_es_user: readonly

local_users:
  enabled: true
  users:
    - username: monitoring
      password_bcrypt: $2a$10$...
      es_user: monitoring_user
    
    - username: ci-pipeline
      password_bcrypt: $2a$10$...
      es_user: ci_user

elasticsearch:
  users:
    - username: admin
      password: ${ES_ADMIN_PASSWORD}
    - username: developer
      password: ${ES_DEVELOPER_PASSWORD}
    - username: readonly
      password: ${ES_READONLY_PASSWORD}
    - username: monitoring_user
      password: ${ES_MONITORING_PASSWORD}
    - username: ci_user
      password: ${ES_CI_PASSWORD}
```

**After**:
```yaml
oidc:
  enabled: true
  # ... OIDC config ...
  scopes:
    - openid
    - email
    - profile
    - groups

local_users:
  enabled: true
  users:
    - username: monitoring
      password_bcrypt: $2a$10$...
      groups:
        - monitoring
      email: monitoring@example.com
      full_name: Monitoring User
    
    - username: ci-pipeline
      password_bcrypt: $2a$10$...
      groups:
        - ci
      email: ci@example.com
      full_name: CI Pipeline

user_management:
  enabled: true
  password_length: 32
  credential_ttl: 1h

elasticsearch:
  admin_user: keyline_admin
  admin_password: ${ES_ADMIN_PASSWORD}
  url: https://elasticsearch:9200
  timeout: 30s

cache:
  backend: redis
  redis_url: redis://localhost:6379
  redis_db: 0
  credential_ttl: 1h
  encryption_key: ${CACHE_ENCRYPTION_KEY}

role_mappings:
  # OIDC email-based mappings
  - claim: email
    pattern: "*@admin.example.com"
    es_roles:
      - superuser
  
  - claim: email
    pattern: "*@dev.example.com"
    es_roles:
      - developer
      - kibana_user
  
  # Local user group mappings
  - claim: groups
    pattern: "monitoring"
    es_roles:
      - monitoring_user
  
  - claim: groups
    pattern: "ci"
    es_roles:
      - ci_user

default_es_roles:
  - viewer
  - kibana_user
```

## Rollback Procedure

If you need to rollback to static user mapping:

### Step 1: Keep Old Configuration

Before migration, backup your old configuration:

```bash
cp config.yaml config.yaml.backup
```

### Step 2: Rollback Steps

1. Stop new Keyline instances
2. Restore old configuration:
   ```bash
   cp config.yaml.backup config.yaml
   ```
3. Remove new environment variables:
   ```bash
   unset CACHE_ENCRYPTION_KEY
   ```
4. Restore old environment variables:
   ```bash
   export ES_ADMIN_PASSWORD=old-admin-password
   export ES_DEVELOPER_PASSWORD=old-developer-password
   export ES_READONLY_PASSWORD=old-readonly-password
   ```
5. Start old Keyline instances
6. Verify functionality

### Step 3: Clean Up (Optional)

Remove dynamically created ES users:

```bash
# List users created by Keyline
curl -X GET "https://elasticsearch:9200/_security/user" \
  -u admin:password | jq 'to_entries[] | select(.value.metadata.managed_by == "keyline")'

# Delete specific user
curl -X DELETE "https://elasticsearch:9200/_security/user/alice" \
  -u admin:password
```

## Troubleshooting

### Issue: Configuration Validation Fails

**Error**: `encryption_key must be 32 bytes`

**Solution**:
```bash
# Generate correct key
openssl rand -base64 32

# Verify key length
echo -n "your-key" | base64 -d | wc -c
# Should output: 32
```

### Issue: Admin Credentials Invalid

**Error**: `ES admin credentials invalid or insufficient privileges`

**Solution**:
1. Verify admin user exists in ES
2. Verify admin user has `manage_security` privilege
3. Test credentials:
   ```bash
   curl -X GET "https://elasticsearch:9200/_security/user" \
     -u keyline_admin:password
   ```

### Issue: No Role Mappings Match

**Error**: `no role mappings matched and no default roles configured`

**Solution**:
1. Add `default_es_roles` to configuration
2. Or ensure at least one role mapping matches all users
3. Test role mappings with different user groups

### Issue: Cache Hit Rate Low

**Symptom**: High latency, many ES API calls

**Solution**:
1. Increase `credential_ttl`
2. Verify Redis is accessible
3. Check Redis memory limits
4. Monitor metrics:
   ```bash
   curl http://localhost:9000/metrics | grep keyline_cred_cache
   ```

### Issue: Users Can't Access ES Resources

**Symptom**: 403 Forbidden errors from ES

**Solution**:
1. Verify user has correct ES roles:
   ```bash
   curl -X GET "https://elasticsearch:9200/_security/user/alice" \
     -u keyline_admin:password
   ```
2. Verify ES roles exist and have correct privileges
3. Check role mapping configuration
4. Test with different user groups

## Best Practices

1. **Test in staging first** - Validate migration in non-production environment
2. **Backup configuration** - Keep old configuration for rollback
3. **Monitor metrics** - Watch cache hit rate and ES API call rate
4. **Gradual rollout** - Use blue-green deployment for zero downtime
5. **Verify audit logs** - Confirm actual usernames appear in ES logs
6. **Document mappings** - Keep record of group-to-role mappings
7. **Secure encryption key** - Store in secrets management, not config files
8. **Plan key rotation** - Schedule periodic encryption key rotation
9. **Test role mappings** - Verify users get correct roles before production
10. **Monitor ES user creation** - Watch for failed user creation attempts

## Further Reading

- [User Management Guide](user-management.md) - Complete user management documentation
- [Configuration Reference](configuration.md) - Full configuration options
- [Troubleshooting](troubleshooting-user-management.md) - Common issues and solutions
