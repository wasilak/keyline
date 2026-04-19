# Dynamic Elasticsearch User Management - Requirements

## Overview

Implement dynamic Elasticsearch user management in Keyline to automatically create and manage ES users for ALL authenticated users (OIDC, Basic Auth, etc.). This provides accountability, auditing, and proper role-based access control without requiring pre-configured ES users.

**Key principle**: Regardless of authentication method, once a user is authenticated, their metadata (username, groups/roles) is used to upsert a local ES user with appropriate role mappings.

## Goals

- Automatically create ES user accounts for all authenticated users
- Map user groups/claims to Elasticsearch roles dynamically
- Cache credentials securely with configurable TTL
- Enable horizontal scaling with Redis cache backend
- Provide accountability and auditing via individual ES accounts

## Non-Goals

- Automatic ES role creation (roles must exist in ES)
- User deletion/cleanup (manual process)
- Password rotation policies (handled by cache TTL)
- Multi-cluster support (single ES cluster only)
- Custom password policies (fixed secure policy)

## Functional Requirements

### FR1: Elasticsearch User Management API Client
- Implement ES Security API client for user operations
- Support `PUT /_security/user/{username}` (create/update user)
- Support `GET /_security/user/{username}` (check if user exists)
- Support `DELETE /_security/user/{username}` (optional, for cleanup)
- Handle ES API errors gracefully

### FR2: Password Generation
- Generate cryptographically secure random passwords
- Minimum password length: 32 characters
- Include uppercase, lowercase, digits, and special characters
- Passwords are never logged or exposed

### FR3: Credential Caching (using cachego)
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

### FR4: Role Mapping Configuration
- New config section: `role_mappings` (top-level, applies to all auth methods)
- Each mapping has: claim/group, pattern, es_roles (array)
- Support for `default_es_roles` (array of role names)
- **Mapping evaluation logic**:
  1. Evaluate ALL role_mappings in order
  2. Collect ALL matching ES roles from matching mappings
  3. If at least one mapping matched, use collected roles
  4. If NO mappings matched AND `default_es_roles` is defined, use default roles
  5. If NO mappings matched AND `default_es_roles` is NOT defined, deny access
- **Multiple group matches → multiple ES roles** (ES handles this naturally)
- Works for OIDC groups, local user groups, and future auth methods

### FR5: Admin Credentials Configuration
- New config section: `elasticsearch.admin_user` and `elasticsearch.admin_password`
- Used exclusively for Security API calls
- Separate from user credentials
- Validated on startup

### FR6: User Upsert Logic (applies to ALL authentication methods)
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

### NFR1: Performance
- User upsert operation completes in < 500ms (p95)
- Cache hit rate > 95% for active users
- No blocking operations on hot path

### NFR2: Security
- Passwords are cryptographically random (crypto/rand)
- Passwords are never logged
- **Passwords are encrypted in cache using AES-256-GCM**
- **Encryption key stored securely (environment variable, not in config file)**
- Admin credentials are never exposed in logs or errors
- TLS required for ES API calls in production
- Encryption key must be rotated periodically (invalidates cache)

### NFR3: Reliability
- Retry transient ES API failures (3 attempts with exponential backoff)
- Graceful degradation if ES is unavailable
- Circuit breaker for ES API calls

### NFR4: Observability
- Log all user creation/update operations
- Metrics for:
  - User upserts (count, duration)
  - Cache hits/misses
  - ES API call success/failure rates
  - Role mapping matches
- Structured logging with context

## User Stories

### US-1: As a system administrator, I want ALL authenticated users to automatically get ES accounts
- [ ] **US-1.1**: When ANY user authenticates (OIDC, Basic Auth, etc.), Keyline creates an ES user account
- [ ] **US-1.2**: The ES username matches the authenticated user identifier
- [ ] **US-1.3**: A random, secure password is generated
- [ ] **US-1.4**: The user is created via ES Security API (`PUT /_security/user/{username}`)
- [ ] **US-1.5**: This applies to OIDC users, local users, and any future authentication methods

### US-2: As a system administrator, I want user groups to map to ES roles
- [ ] **US-2.1**: Configuration supports mapping user groups/claims to ES role names
- [ ] **US-2.2**: Multiple groups can map to multiple roles (ES handles users with multiple roles)
- [ ] **US-2.3**: Wildcard patterns are supported (e.g., `*-admins` → `superuser`)
- [ ] **US-2.4**: **Default roles are ONLY used if NO mappings matched**
- [ ] **US-2.5**: If no mappings match and no default roles configured, access is denied
- [ ] **US-2.6**: Works for OIDC groups, local user groups, and any future authentication methods
- [ ] **US-2.7**: Local users can have 0 or more groups defined in configuration

### US-3: As a system administrator, I want credentials to be cached for performance and scalability
- [ ] **US-3.1**: Generated passwords are cached using cachego (Redis or in-memory)
- [ ] **US-3.2**: **Redis backend**: Enables horizontal scaling (multiple Keyline instances share cache)
- [ ] **US-3.3**: **Memory backend**: Single-node deployment only (cache not shared)
- [ ] **US-3.4**: Cache TTL is configurable (default: 1 hour)
- [ ] **US-3.5**: Cached credentials are used for subsequent requests within TTL
- [ ] **US-3.6**: When cache expires, a new password is generated and the ES user is updated
- [ ] **US-3.7**: Cache key format allows multiple Keyline instances to coordinate

### US-4: As a system administrator, I want to configure admin credentials for user management
- [ ] **US-4.1**: Configuration includes admin ES credentials for API calls
- [ ] **US-4.2**: Admin user must have `manage_security` privilege
- [ ] **US-4.3**: Keyline validates admin credentials on startup
- [ ] **US-4.4**: Clear error messages if admin credentials are invalid or lack permissions

### US-5: As a developer, I want ES user operations to be resilient
- [ ] **US-5.1**: Retry logic for transient ES API failures
- [ ] **US-5.2**: Graceful handling of ES unavailability
- [ ] **US-5.3**: Logging of all user management operations
- [ ] **US-5.4**: Metrics for user creation, updates, and cache hits/misses

### US-6: As a system administrator, I want to control user metadata
- [ ] **US-6.1**: User full name is set from authentication metadata (OIDC claims, local user config, etc.)
- [ ] **US-6.2**: User email is set from authentication metadata
- [ ] **US-6.3**: User metadata includes source (e.g., "oidc:google", "basic_auth", "ldap")
- [ ] **US-6.4**: User metadata includes last authentication timestamp
- [ ] **US-6.5**: Metadata is updated on each authentication (not just first time)

## Glossary

| Term | Definition |
|------|------------|
| ES | Elasticsearch |
| OIDC | OpenID Connect |
| cachego | Credential caching library used by Keyline |
| Role Mapping | Process of mapping user groups/claims to ES roles |
| Upsert | Create or update operation |
| TTL | Time To Live - cache expiration duration |

## Traceability Matrix

| Requirement | Design Section | Tasks |
|-------------|----------------|-------|
| FR1 | ES API Client | 3, 4 |
| FR2 | Password Generator | 2 |
| FR3 | Credential Encryptor, User Manager | 2.6, 7, 8 |
| FR4 | Role Mapper | 5, 6 |
| FR5 | Config, Main App | 1, 14 |
| FR6 | User Manager, Auth Integration | 7, 8, 9, 10, 11 |
| NFR1 | Performance section | 21.7 |
| NFR2 | Security section | 21.8 |
| NFR3 | Error Handling section | 3.10, 15.6 |
| NFR4 | Monitoring section | 19, 20 |
| US-1 | User Upsert Logic | 9, 10, 11 |
| US-2 | Role Mapper | 5, 6 |
| US-3 | Credential Caching | 7, 8 |
| US-4 | Admin Credentials | 1, 14 |
| US-5 | Error Handling | 3.10, 15.6 |
| US-6 | User Metadata | 7, 9 |

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
  - claim: groups
    pattern: "admin"
    es_roles:
      - superuser

  - claim: groups
    pattern: "*-developers"
    es_roles:
      - developer
      - kibana_user

# Default roles if NO mappings matched
default_es_roles:
  - viewer
  - kibana_user

cache:
  backend: redis  # or "memory"
  redis_url: redis://localhost:6379
  redis_password: ${REDIS_PASSWORD}
  redis_db: 0
  credential_ttl: 1h
  encryption_key: ${CACHE_ENCRYPTION_KEY}
```

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
