# Dynamic Elasticsearch User Management - Requirements

## Overview

Implement dynamic Elasticsearch user management in Keyline to automatically create and manage ES users for ALL authenticated users (OIDC, Basic Auth, etc.). This provides accountability, auditing, and proper role-based access control without requiring pre-configured ES users.

**Key principle**: Regardless of authentication method, once a user is authenticated, their metadata (username, groups/roles) is used to upsert a local ES user with appropriate role mappings.

## Background

Keyline's predecessor (elastauth) implemented this feature for LDAP authentication. When ANY user authenticates (OIDC, Basic Auth, etc.), Keyline should:

1. Generate a random password for the user
2. Create or update the user in Elasticsearch via the Security API
3. Map user groups/claims to Elasticsearch roles
4. Cache the credentials with a configurable TTL (using cachego)
5. Use the cached credentials for subsequent requests

**Cache backend significance**:
- **Redis**: Enables horizontal scaling of Keyline (multiple instances share cache)
- **Memory**: Single-node deployment only (cache not shared across instances)

This approach provides:
- **Accountability**: Each user has their own ES account
- **Auditing**: ES audit logs show actual usernames
- **Security**: Random, short-lived passwords
- **Role-based access**: User groups map to ES roles
- **Scalability**: Redis cache enables multi-instance deployments

## User Stories

### US-1: As a system administrator, I want ALL authenticated users to automatically get ES accounts
**Acceptance Criteria:**
- When ANY user authenticates (OIDC, Basic Auth, etc.), Keyline creates an ES user account
- The ES username matches the authenticated user identifier
- A random, secure password is generated
- The user is created via ES Security API (`PUT /_security/user/{username}`)
- This applies to OIDC users, local users, and any future authentication methods

### US-2: As a system administrator, I want user groups to map to ES roles
**Acceptance Criteria:**
- Configuration supports mapping user groups/claims to ES role names
- Multiple groups can map to multiple roles (ES handles users with multiple roles)
- Wildcard patterns are supported (e.g., `*-admins` → `superuser`)
- **Default roles are ONLY used if NO mappings matched**
- If no mappings match and no default roles configured, access is denied
- Works for OIDC groups, local user groups, and any future authentication methods
- Local users can have 0 or more groups defined in configuration

### US-3: As a system administrator, I want credentials to be cached for performance and scalability
**Acceptance Criteria:**
- Generated passwords are cached using cachego (Redis or in-memory)
- **Redis backend**: Enables horizontal scaling (multiple Keyline instances share cache)
- **Memory backend**: Single-node deployment only (cache not shared)
- Cache TTL is configurable (default: 1 hour)
- Cached credentials are used for subsequent requests within TTL
- When cache expires, a new password is generated and the ES user is updated
- Cache key format allows multiple Keyline instances to coordinate

### US-4: As a system administrator, I want to configure admin credentials for user management
**Acceptance Criteria:**
- Configuration includes admin ES credentials for API calls
- Admin user must have `manage_security` privilege
- Keyline validates admin credentials on startup
- Clear error messages if admin credentials are invalid or lack permissions

### US-5: As a developer, I want ES user operations to be resilient
**Acceptance Criteria:**
- Retry logic for transient ES API failures
- Graceful handling of ES unavailability
- Logging of all user management operations
- Metrics for user creation, updates, and cache hits/misses

### US-6: As a system administrator, I want to control user metadata
**Acceptance Criteria:**
- User full name is set from authentication metadata (OIDC claims, local user config, etc.)
- User email is set from authentication metadata
- User metadata includes source (e.g., "oidc:google", "basic_auth", "ldap")
- User metadata includes last authentication timestamp
- Metadata is updated on each authentication (not just first time)

## Functional Requirements

### FR-1: Elasticsearch User Management API Client
- Implement ES Security API client for user operations
- Support `PUT /_security/user/{username}` (create/update user)
- Support `GET /_security/user/{username}` (check if user exists)
- Support `DELETE /_security/user/{username}` (optional, for cleanup)
- Handle ES API errors gracefully

### FR-2: Password Generation
- Generate cryptographically secure random passwords
- Minimum password length: 32 characters
- Include uppercase, lowercase, digits, and special characters
- Passwords are never logged or exposed

### FR-3: Credential Caching (using cachego)
- Use existing cachego library for cache abstraction
- Cache key format: `keyline:user:{username}:password`
- **Encrypt passwords before storing in cache** using AES-256-GCM
- Encryption key provided via configuration (environment variable recommended)
- Decrypt passwords when retrieving from cache before passing to ES
- Configurable TTL (default: 1 hour, range: 5 minutes to 24 hours)
- **Redis backend**: Shared cache across multiple Keyline instances (horizontal scaling)
- **Memory backend**: Local cache per instance (single-node only)
- Cache invalidation on user update
- Cache operations are atomic to prevent race conditions
- Encryption key must be 32 bytes (256 bits) for AES-256
- Same encryption key must be used across all Keyline instances (for Redis)

### FR-4: Role Mapping Configuration
- New config section: `role_mappings` (top-level, applies to all auth methods)
- Each mapping has: claim/group, pattern, es_roles (array)
- Support for `default_es_roles` (array of role names)
- **Mapping evaluation logic**:
  1. Evaluate ALL role_mappings in order
  2. Collect ALL matching ES roles from matching mappings
  3. If at least one mapping matched, use collected roles
  4. If NO mappings matched AND `default_es_roles` is defined, use default roles
  5. If NO mappings matched AND `default_es_roles` is NOT defined, deny access
- **Multiple group matches → multiple ES roles** (ES handles this natively)
- Works for OIDC groups, local user groups, and future auth methods

### FR-5: Admin Credentials Configuration
- New config section: `elasticsearch.admin_user` and `elasticsearch.admin_password`
- Used exclusively for Security API calls
- Separate from user credentials
- Validated on startup

### FR-6: User Upsert Logic (applies to ALL authentication methods)
1. User authenticates via ANY method (OIDC, Basic Auth, etc.)
2. Extract user metadata:
   - Username (required)
   - Groups (0 or more - from OIDC claims or local user config)
   - Email (optional)
   - Full name (optional)
3. Check cache for existing credentials
4. If cache hit and not expired, use cached credentials
5. If cache miss or expired:
   a. Generate new random password
   b. Map user groups to ES roles:
      - Evaluate ALL role_mappings against user's groups
      - Collect ALL matching ES roles
      - If at least one mapping matched, use collected roles
      - If NO mappings matched AND default_es_roles defined, use default roles
      - If NO mappings matched AND default_es_roles NOT defined, deny access
   c. Create or update ES user with:
      - Username (from authentication)
      - Password (generated)
      - Roles (from role mappings or defaults)
      - Metadata (name, email, source, last_auth, groups)
   d. Cache the new credentials with TTL
6. Return credentials for Authorization header
7. Forward request to ES with generated credentials

**Examples**:
- User with groups ["developers", "users"] → matches "developers" mapping → gets developer + kibana_user roles
- User with groups ["unknown-group"] → no matches → gets default_es_roles (if defined)
- User with no groups → no matches → gets default_es_roles (if defined)
- User with no groups and no default_es_roles → access denied

## Non-Functional Requirements

### NFR-1: Performance
- User upsert operation completes in < 500ms (p95)
- Cache hit rate > 95% for active users
- No blocking operations on hot path

### NFR-2: Security
- Passwords are cryptographically random (crypto/rand)
- Passwords are never logged
- **Passwords are encrypted in cache using AES-256-GCM**
- **Encryption key stored securely (environment variable, not in config file)**
- Admin credentials are never exposed in logs or errors
- TLS required for ES API calls in production
- Encryption key must be rotated periodically (invalidates cache)

### NFR-3: Reliability
- Retry transient ES API failures (3 attempts with exponential backoff)
- Graceful degradation if ES is unavailable
- Circuit breaker for ES API calls

### NFR-4: Observability
- Log all user creation/update operations
- Metrics for:
  - User upserts (count, duration)
  - Cache hits/misses
  - ES API call success/failure rates
  - Role mapping matches
- Structured logging with context

## Configuration Schema

```yaml
# Local users configuration
local_users:
  enabled: true
  users:
    - username: testuser
      password_bcrypt: $2a$10$...
      groups:
        - developers
        - users
      email: testuser@example.com
      full_name: Test User
    
    - username: admin
      password_bcrypt: $2a$10$...
      groups:
        - admin
        - superusers
      email: admin@example.com
      full_name: Admin User
    
    - username: viewer
      password_bcrypt: $2a$10$...
      # No groups - will use default_es_roles
      email: viewer@example.com
      full_name: Viewer User

elasticsearch:
  # Admin credentials for user management API
  admin_user: admin
  admin_password: ${ES_ADMIN_PASSWORD}
  
  # ES cluster URL for API calls
  url: https://elasticsearch:9200
  timeout: 30s
  insecure_skip_verify: false

# Role mappings: User groups/claims → ES roles
# Applies to ALL authentication methods (OIDC, Basic Auth with groups, etc.)
# Multiple matching groups result in multiple ES roles
role_mappings:
  # Group-based mappings (works for OIDC groups and local user groups)
  - claim: groups
    pattern: "admin"
    es_roles:
      - superuser
  
  - claim: groups
    pattern: "superusers"
    es_roles:
      - superuser
  
  - claim: groups
    pattern: "*-developers"
    es_roles:
      - developer
      - kibana_user
  
  - claim: groups
    pattern: "developers"
    es_roles:
      - developer
      - kibana_user
  
  - claim: groups
    pattern: "viewers"
    es_roles:
      - viewer
  
  # Email-based mappings (works for OIDC and local users with email)
  - claim: email
    pattern: "*@admin.example.com"
    es_roles:
      - superuser

# Default roles if NO mappings matched
# Only applied when user has no groups OR no groups matched any pattern
# If empty and no mapping matches, access is denied
default_es_roles:
  - viewer
  - kibana_user

oidc:
  enabled: true
  # ... existing OIDC config ...
  
  # User identity claim (default: "email")
  # This claim is used as the ES username
  user_identity_claim: email

cache:
  backend: redis  # or "memory"
  # Redis: Enables horizontal scaling (shared cache across instances)
  # Memory: Single-node only (cache not shared)
  redis_url: redis://localhost:6379
  redis_password: ${REDIS_PASSWORD}
  redis_db: 0
  
  # Credential cache TTL (how long passwords are cached)
  # After TTL expires, new password is generated and ES user is updated
  credential_ttl: 1h
  
  # Encryption key for cached credentials (REQUIRED)
  # Must be 32 bytes (256 bits) for AES-256-GCM
  # Use environment variable for security
  # All Keyline instances must use the same key (for Redis)
  encryption_key: ${CACHE_ENCRYPTION_KEY}
```

## Out of Scope

- Automatic ES role creation (roles must exist in ES)
- User deletion/cleanup (manual process)
- Password rotation policies (handled by cache TTL)
- Multi-cluster support (single ES cluster only)
- Custom password policies (fixed secure policy)

## Dependencies

- Elasticsearch Security API (requires security features enabled)
- Admin ES user with `manage_security` privilege
- cachego library (already integrated)
- Redis (optional, for distributed caching and horizontal scaling)
- Existing authentication flows (OIDC, Basic Auth, etc.)

## Success Criteria

1. ALL authenticated users (OIDC, Basic Auth, etc.) automatically get ES accounts
2. ES audit logs show actual usernames, not shared accounts
3. User groups correctly map to ES roles (multiple groups → multiple roles)
4. Credentials are cached and reused within TTL
5. Redis cache enables horizontal scaling (multiple Keyline instances)
6. Memory cache works for single-node deployments
7. No performance degradation compared to static user mapping
8. All tests pass (unit, integration, property-based)
9. Documentation includes setup guide and troubleshooting

## Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| ES API unavailable | Users can't authenticate | Implement circuit breaker, fallback to cached credentials |
| Admin credentials invalid | User management fails | Validate on startup, clear error messages |
| Cache unavailable | Performance degradation | Fallback to in-memory cache, log warnings |
| Role mapping misconfiguration | Users get wrong permissions | Validation on startup, dry-run mode |
| Password generation collision | Security issue | Use crypto/rand, 32+ character length |

## Testing Strategy

### Unit Tests
- Password generation (randomness, length, character sets)
- Role mapping logic (pattern matching, priority)
- Cache operations (set, get, expiry)
- ES API client (mocked responses)

### Integration Tests
- End-to-end OIDC flow with user creation
- Role mapping with real ES cluster
- Cache TTL expiration and refresh
- ES API error handling

### Property-Based Tests
- Password generation always produces valid passwords
- Role mappings are deterministic
- Cache operations are idempotent
- User upsert is idempotent

## Documentation Requirements

1. Configuration guide for dynamic user management
2. Role mapping examples and best practices
3. Troubleshooting guide for common issues
4. Migration guide from static to dynamic user management
5. Security considerations and recommendations
