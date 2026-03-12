# Dynamic Elasticsearch User Management - Implementation Tasks

## Task Breakdown

### Phase 1: Configuration and Foundation (2-3 days)

- [ ] 1. Update configuration structures
  - [x] 1.1 Add `RoleMapping` struct to `internal/config/config.go`
  - [x] 1.2 Add `DefaultESRoles []string` to `Config` struct
  - [x] 1.3 Add `UserMgmtConfig` struct with `Enabled`, `PasswordLength`, `CredentialTTL`
  - [x] 1.4 Add `CacheConfig` struct with `Backend`, `RedisURL`, `RedisPassword`, `RedisDB`, `CredentialTTL`, `EncryptionKey`
  - [x] 1.5 Update `ElasticsearchConfig` with `AdminUser`, `AdminPassword`, `URL`, `Timeout`
  - [x] 1.6 Add `Groups []string`, `Email string`, `FullName string` to `LocalUser` struct
  - [-] 1.7 Remove `ESUser string` field from `LocalUser` (breaking change)
  - [~] 1.8 Update config validation in `internal/config/validator.go`
  - [~] 1.9 Add validation for role mappings (pattern syntax, non-empty roles)
  - [~] 1.10 Add validation for admin credentials (required if user_management.enabled)
  - [~] 1.11 Add validation for encryption key (must be 32 bytes when decoded)
  - [~] 1.12 Update example config files with new structure including encryption_key

- [ ] 2. Create password generator
  - [ ] 2.1 Create `internal/usermgmt/password.go`
  - [ ] 2.2 Implement `PasswordGenerator` struct with configurable length
  - [ ] 2.3 Implement `Generate()` method using `crypto/rand`
  - [ ] 2.4 Use charset with uppercase, lowercase, digits, special characters
  - [ ] 2.5 Add unit tests for password generation
    - [ ] 2.5.1 Test password length
    - [ ] 2.5.2 Test character set inclusion
    - [ ] 2.5.3 Test randomness (no duplicates in 1000 generations)
    - [ ] 2.5.4 Test error handling

- [ ] 2.6 Create credential encryptor
  - [ ] 2.6.1 Create `internal/usermgmt/encryptor.go`
  - [ ] 2.6.2 Implement `Encryptor` interface with `Encrypt()` and `Decrypt()` methods
  - [ ] 2.6.3 Implement `NewEncryptor(key []byte)` constructor with key validation (must be 32 bytes)
  - [ ] 2.6.4 Implement `Encrypt()` using AES-256-GCM with random nonce
  - [ ] 2.6.5 Implement `Decrypt()` to reverse encryption
  - [ ] 2.6.6 Use base64 encoding for cache storage
  - [ ] 2.6.7 Add unit tests for encryption
    - [ ] 2.6.7.1 Test encryption/decryption round-trip
    - [ ] 2.6.7.2 Test invalid key length (not 32 bytes)
    - [ ] 2.6.7.3 Test decryption with wrong key
    - [ ] 2.6.7.4 Test decryption with corrupted ciphertext
    - [ ] 2.6.7.5 Test that same plaintext produces different ciphertexts (random nonce)

### Phase 2: Elasticsearch API Client (3-4 days)

- [ ] 3. Implement ES API client
  - [ ] 3.1 Create `internal/elasticsearch/client.go`
  - [ ] 3.2 Define `Client` interface with methods:
    - [ ] 3.2.1 `CreateOrUpdateUser(ctx, req) error`
    - [ ] 3.2.2 `GetUser(ctx, username) (*User, error)`
    - [ ] 3.2.3 `DeleteUser(ctx, username) error`
    - [ ] 3.2.4 `ValidateConnection(ctx) error`
  - [ ] 3.3 Define `UserRequest` and `User` structs
  - [ ] 3.4 Implement `client` struct with HTTP client, admin credentials, config
  - [ ] 3.5 Implement `NewClient(config, logger)` constructor
  - [ ] 3.6 Implement `CreateOrUpdateUser` method
    - [ ] 3.6.1 Build PUT request to `/_security/user/{username}`
    - [ ] 3.6.2 Set admin credentials in Authorization header
    - [ ] 3.6.3 Marshal request body (password, roles, full_name, email, metadata)
    - [ ] 3.6.4 Add OpenTelemetry tracing with `otelgo`
    - [ ] 3.6.5 Add retry logic (3 attempts, exponential backoff)
    - [ ] 3.6.6 Handle HTTP status codes (200, 401, 403, 429, 5xx)
    - [ ] 3.6.7 Add request timeout (30s)
  - [ ] 3.7 Implement `GetUser` method
  - [ ] 3.8 Implement `DeleteUser` method
  - [ ] 3.9 Implement `ValidateConnection` method (call on startup)
  - [ ] 3.10 Add circuit breaker pattern for ES unavailability

- [ ] 4. Add ES API client tests
  - [ ] 4.1 Create `internal/elasticsearch/client_test.go`
  - [ ] 4.2 Add unit tests with mocked HTTP responses
    - [ ] 4.2.1 Test successful user creation (200 OK)
    - [ ] 4.2.2 Test user update (200 OK)
    - [ ] 4.2.3 Test invalid admin credentials (401/403)
    - [ ] 4.2.4 Test rate limiting (429)
    - [ ] 4.2.5 Test ES unavailable (5xx)
    - [ ] 4.2.6 Test network timeout
    - [ ] 4.2.7 Test retry logic
    - [ ] 4.2.8 Test circuit breaker
  - [ ] 4.3 Add integration tests with real ES cluster (optional)

### Phase 3: Role Mapper (2-3 days)

- [ ] 5. Implement role mapper
  - [ ] 5.1 Create `internal/usermgmt/rolemapper.go`
  - [ ] 5.2 Define `RoleMapper` struct with config and logger
  - [ ] 5.3 Implement `NewRoleMapper(config, logger)` constructor
  - [ ] 5.4 Implement `MapGroupsToRoles(ctx, groups) ([]string, error)` method
    - [ ] 5.4.1 Iterate through all role mappings
    - [ ] 5.4.2 Match each group against each mapping pattern
    - [ ] 5.4.3 Collect ALL matching roles (use map to deduplicate)
    - [ ] 5.4.4 If no matches and default_es_roles defined, use defaults
    - [ ] 5.4.5 If no matches and no defaults, return error
    - [ ] 5.4.6 Log matched mappings with `loggergo`
  - [ ] 5.5 Implement `matchPattern(value, pattern)` method
    - [ ] 5.5.1 Exact match
    - [ ] 5.5.2 Wildcard prefix (`admin@*`)
    - [ ] 5.5.3 Wildcard suffix (`*@example.com`)
    - [ ] 5.5.4 Wildcard middle (`admin@*.com`)
  - [ ] 5.6 Add validation for role mapping patterns

- [ ] 6. Add role mapper tests
  - [ ] 6.1 Create `internal/usermgmt/rolemapper_test.go`
  - [ ] 6.2 Add unit tests for pattern matching
    - [ ] 6.2.1 Test exact match
    - [ ] 6.2.2 Test wildcard prefix
    - [ ] 6.2.3 Test wildcard suffix
    - [ ] 6.2.4 Test wildcard middle
    - [ ] 6.2.5 Test no match
  - [ ] 6.3 Add unit tests for role mapping
    - [ ] 6.3.1 Test single group, single role
    - [ ] 6.3.2 Test single group, multiple roles
    - [ ] 6.3.3 Test multiple groups, multiple roles (accumulation)
    - [ ] 6.3.4 Test no matches, use default roles
    - [ ] 6.3.5 Test no matches, no defaults (error)
    - [ ] 6.3.6 Test empty groups array
    - [ ] 6.3.7 Test role deduplication
  - [ ] 6.4 Add property-based tests
    - [ ] 6.4.1 Property: Mapping is deterministic (same groups → same roles)
    - [ ] 6.4.2 Property: Multiple groups accumulate roles (no duplicates)

### Phase 4: User Manager (3-4 days)

- [ ] 7. Implement user manager
  - [ ] 7.1 Create `internal/usermgmt/manager.go`
  - [ ] 7.2 Define `Manager` interface with methods:
    - [ ] 7.2.1 `UpsertUser(ctx, authUser) (*Credentials, error)`
    - [ ] 7.2.2 `InvalidateCache(ctx, username) error`
  - [ ] 7.3 Define `AuthenticatedUser` and `Credentials` structs
  - [ ] 7.4 Implement `manager` struct with:
    - [ ] 7.4.1 `esClient elasticsearch.Client`
    - [ ] 7.4.2 `roleMapper *RoleMapper`
    - [ ] 7.4.3 `cache cachego.CacheInterface`
    - [ ] 7.4.4 `pwdGen *PasswordGenerator`
    - [ ] 7.4.5 `encryptor Encryptor`
    - [ ] 7.4.6 `cacheTTL time.Duration`
    - [ ] 7.4.7 `config *config.Config`
    - [ ] 7.4.8 `logger *loggergo.Logger`
  - [ ] 7.5 Implement `NewManager(esClient, roleMapper, cache, pwdGen, encryptor, config, logger)` constructor
  - [ ] 7.6 Implement `UpsertUser` method
    - [ ] 7.6.1 Check cache for existing credentials
    - [ ] 7.6.2 If cache hit, decrypt password and return credentials
    - [ ] 7.6.3 If cache miss or decryption fails, generate new password
    - [ ] 7.6.4 Map groups to roles using RoleMapper
    - [ ] 7.6.5 Create UserRequest with username, password, roles, metadata
    - [ ] 7.6.6 Call ES API client to create/update user
    - [ ] 7.6.7 Encrypt password before caching
    - [ ] 7.6.8 Cache encrypted credentials with TTL
    - [ ] 7.6.9 Return plaintext credentials
    - [ ] 7.6.10 Add OpenTelemetry tracing
    - [ ] 7.6.11 Add Prometheus metrics (upserts, duration, cache hits/misses)
  - [ ] 7.7 Implement `InvalidateCache` method

- [ ] 8. Add user manager tests
  - [ ] 8.1 Create `internal/usermgmt/manager_test.go`
  - [ ] 8.2 Add unit tests with mocked dependencies
    - [ ] 8.2.1 Test cache hit path (decrypt and return, no ES call)
    - [ ] 8.2.2 Test cache miss path (ES call, encrypt, cache update)
    - [ ] 8.2.3 Test decryption failure (regenerate password)
    - [ ] 8.2.4 Test password generation failure
    - [ ] 8.2.5 Test role mapping failure
    - [ ] 8.2.6 Test ES API failure
    - [ ] 8.2.7 Test encryption failure (should not fail request, just skip caching)
    - [ ] 8.2.8 Test cache write failure (should not fail request)
    - [ ] 8.2.9 Test cache invalidation
  - [ ] 8.3 Add integration tests
    - [ ] 8.3.1 Test end-to-end user upsert with real cache and encryption
    - [ ] 8.3.2 Test cache expiration and refresh
    - [ ] 8.3.3 Test that cached passwords are encrypted (not plaintext)

### Phase 5: Auth Integration (3-4 days)

- [ ] 9. Update auth engine
  - [ ] 9.1 Update `internal/auth/engine.go`
  - [ ] 9.2 Add `userManager usermgmt.Manager` field to `Engine` struct
  - [ ] 9.3 Update `NewEngine` constructor to accept user manager
  - [ ] 9.4 Update `Authenticate` method
    - [ ] 9.4.1 After successful authentication, extract user metadata
    - [ ] 9.4.2 Create `AuthenticatedUser` struct with username, groups, email, full_name, source
    - [ ] 9.4.3 Call `userManager.UpsertUser(ctx, authUser)`
    - [ ] 9.4.4 Use returned credentials for ES Authorization header
    - [ ] 9.4.5 Update `AuthResult` to include ES credentials
  - [ ] 9.5 Add error handling for user management failures
  - [ ] 9.6 Add logging for user management operations

- [ ] 10. Update OIDC provider
  - [ ] 10.1 Update `internal/auth/oidc.go`
  - [ ] 10.2 Implement `extractGroups(claims)` method
    - [ ] 10.2.1 Handle `[]interface{}` (array of interfaces)
    - [ ] 10.2.2 Handle `[]string` (string array)
    - [ ] 10.2.3 Handle `string` (single group)
    - [ ] 10.2.4 Return empty array if no groups claim
  - [ ] 10.3 Update `CreateSessionFromClaims` to extract groups
  - [ ] 10.4 Update `AuthResult` to include groups
  - [ ] 10.5 Add tests for group extraction

- [ ] 11. Update Basic Auth provider
  - [ ] 11.1 Update `internal/auth/basic.go`
  - [ ] 11.2 Update `Authenticate` method to return groups from local user config
  - [ ] 11.3 Update `AuthResult` to include email, full_name, groups
  - [ ] 11.4 Add tests for group extraction from local users

- [ ] 12. Update credential mapper
  - [ ] 12.1 Update `internal/mapper/credentials.go`
  - [ ] 12.2 Remove `MapOIDCUser` method (replaced by role mapper)
  - [ ] 12.3 Remove `MapLocalUser` method (replaced by role mapper)
  - [ ] 12.4 Keep `GetESCredentials` for backward compatibility (if needed)
  - [ ] 12.5 Update tests to reflect changes

### Phase 6: Transport Integration (2-3 days)

- [ ] 13. Update transport adapters
  - [ ] 13.1 Update `internal/transport/forward_auth.go`
    - [ ] 13.1.1 Use ES credentials from auth result
    - [ ] 13.1.2 Set Authorization header with generated credentials
    - [ ] 13.1.3 Remove old credential mapping logic
  - [ ] 13.2 Update `internal/transport/standalone.go`
    - [ ] 13.2.1 Use ES credentials from auth result
    - [ ] 13.2.2 Set Authorization header with generated credentials
    - [ ] 13.2.3 Remove old credential mapping logic
  - [ ] 13.3 Add tests for transport adapters with new credential flow

### Phase 7: Main Application Integration (1-2 days)

- [ ] 14. Update main application
  - [ ] 14.1 Update `cmd/keyline/main.go`
  - [ ] 14.2 Initialize ES API client
    - [ ] 14.2.1 Create client with admin credentials
    - [ ] 14.2.2 Validate connection on startup
    - [ ] 14.2.3 Handle validation errors gracefully
  - [ ] 14.3 Initialize password generator
  - [ ] 14.4 Initialize credential encryptor
    - [ ] 14.4.1 Load encryption key from config
    - [ ] 14.4.2 Validate key is 32 bytes
    - [ ] 14.4.3 Create encryptor instance
  - [ ] 14.5 Initialize role mapper
  - [ ] 14.6 Initialize user manager with encryptor
  - [ ] 14.7 Pass user manager to auth engine
  - [ ] 14.8 Add feature flag check (`user_management.enabled`)
  - [ ] 14.9 Add startup logging for user management status

### Phase 8: Testing and Validation (3-4 days)

- [ ] 15. Integration tests
  - [ ] 15.1 Create `integration/user_management_test.go`
  - [ ] 15.2 Test OIDC authentication with user creation
    - [ ] 15.2.1 Authenticate OIDC user
    - [ ] 15.2.2 Verify ES user created
    - [ ] 15.2.3 Verify roles assigned correctly
    - [ ] 15.2.4 Verify credentials cached
    - [ ] 15.2.5 Verify subsequent requests use cached credentials
  - [ ] 15.3 Test Basic Auth with user creation
    - [ ] 15.3.1 Authenticate local user with groups
    - [ ] 15.3.2 Verify ES user created
    - [ ] 15.3.3 Verify roles assigned from groups
  - [ ] 15.4 Test cache expiration
    - [ ] 15.4.1 Authenticate user
    - [ ] 15.4.2 Wait for cache TTL
    - [ ] 15.4.3 Authenticate again
    - [ ] 15.4.4 Verify new password generated
  - [ ] 15.5 Test role mapping scenarios
    - [ ] 15.5.1 Single group → single role
    - [ ] 15.5.2 Multiple groups → multiple roles
    - [ ] 15.5.3 No groups → default roles
    - [ ] 15.5.4 No groups, no defaults → access denied
  - [ ] 15.6 Test ES unavailability
    - [ ] 15.6.1 Stop ES
    - [ ] 15.6.2 Attempt authentication
    - [ ] 15.6.3 Verify graceful error handling
    - [ ] 15.6.4 Verify circuit breaker activates

- [ ] 16. Property-based tests
  - [ ] 16.1 Create `internal/usermgmt/properties_test.go`
  - [ ] 16.2 Add property tests for password generation
    - [ ] 16.2.1 Property: All passwords are valid (length, charset)
    - [ ] 16.2.2 Property: No duplicates in 10,000 generations
  - [ ] 16.3 Add property tests for role mapping
    - [ ] 16.3.1 Property: Mapping is deterministic
    - [ ] 16.3.2 Property: Multiple groups accumulate roles
  - [ ] 16.4 Add property tests for cache operations
    - [ ] 16.4.1 Property: Set then Get returns same value
    - [ ] 16.4.2 Property: Expired entries return cache miss

### Phase 9: Documentation and Migration (2-3 days)

- [ ] 17. Update documentation
  - [ ] 17.1 Create `docs/user-management.md`
    - [ ] 17.1.1 Overview of dynamic user management
    - [ ] 17.1.2 Configuration guide
    - [ ] 17.1.3 Role mapping examples
    - [ ] 17.1.4 Security considerations (encryption key management)
    - [ ] 17.1.5 Performance tuning
    - [ ] 17.1.6 Encryption key rotation procedure
  - [ ] 17.2 Update `docs/configuration.md`
    - [ ] 17.2.1 Document new config sections
    - [ ] 17.2.2 Document role_mappings syntax
    - [ ] 17.2.3 Document default_es_roles
    - [ ] 17.2.4 Document user_management config
  - [ ] 17.3 Create `docs/migration-guide.md`
    - [ ] 17.3.1 Migration from static user mapping
    - [ ] 17.3.2 Breaking changes (LocalUser.ESUser removed)
    - [ ] 17.3.3 Configuration examples
    - [ ] 17.3.4 Rollback procedure
  - [ ] 17.4 Update `README.md`
    - [ ] 17.4.1 Add user management feature description
    - [ ] 17.4.2 Add quick start example
  - [ ] 17.5 Create `docs/troubleshooting-user-management.md`
    - [ ] 17.5.1 Common issues and solutions
    - [ ] 17.5.2 Debugging tips
    - [ ] 17.5.3 Log analysis guide

- [ ] 18. Update example configurations
  - [ ] 18.1 Update `config/config.example.yaml`
  - [ ] 18.2 Create `config/user-management-example.yaml`
  - [ ] 18.3 Update test configs with user management enabled
  - [ ] 18.4 Add comments explaining each config option

### Phase 10: Monitoring and Observability (1-2 days)

- [ ] 19. Add metrics
  - [ ] 19.1 Create `internal/usermgmt/metrics.go`
  - [ ] 19.2 Define Prometheus metrics
    - [ ] 19.2.1 `keyline_user_upserts_total` (counter, status label)
    - [ ] 19.2.2 `keyline_user_upsert_duration_seconds` (histogram, cache_status label)
    - [ ] 19.2.3 `keyline_cred_cache_hits_total` (counter)
    - [ ] 19.2.4 `keyline_cred_cache_misses_total` (counter)
    - [ ] 19.2.5 `keyline_role_mapping_matches_total` (counter, pattern label)
    - [ ] 19.2.6 `keyline_es_api_calls_total` (counter, operation, status labels)
  - [ ] 19.3 Register metrics with Prometheus
  - [ ] 19.4 Instrument user manager with metrics
  - [ ] 19.5 Instrument ES API client with metrics
  - [ ] 19.6 Instrument role mapper with metrics

- [ ] 20. Add observability
  - [ ] 20.1 Add OpenTelemetry spans to all operations
    - [ ] 20.1.1 User upsert span
    - [ ] 20.1.2 ES API call spans
    - [ ] 20.1.3 Role mapping span
    - [ ] 20.1.4 Cache operation spans
  - [ ] 20.2 Add structured logging with `loggergo`
    - [ ] 20.2.1 Log user creation/update
    - [ ] 20.2.2 Log role mapping results
    - [ ] 20.2.3 Log cache hits/misses
    - [ ] 20.2.4 Log ES API errors
  - [ ] 20.3 Create Grafana dashboard (optional)
    - [ ] 20.3.1 User upsert rate and duration
    - [ ] 20.3.2 Cache hit rate
    - [ ] 20.3.3 Role mapping distribution
    - [ ] 20.3.4 ES API error rate

### Phase 11: Final Testing and Deployment (2-3 days)

- [ ] 21. End-to-end testing
  - [ ] 21.1 Test with real Elasticsearch cluster
  - [ ] 21.2 Test with Redis cache
  - [ ] 21.3 Test with in-memory cache
  - [ ] 21.4 Test horizontal scaling (multiple Keyline instances with Redis)
  - [ ] 21.5 Test all authentication methods (OIDC, Basic Auth)
  - [ ] 21.6 Test all role mapping scenarios
  - [ ] 21.7 Performance testing
    - [ ] 21.7.1 Measure user upsert latency (cache hit/miss)
    - [ ] 21.7.2 Measure cache hit rate
    - [ ] 21.7.3 Load testing with concurrent requests
  - [ ] 21.8 Security testing
    - [ ] 21.8.1 Verify passwords are never logged
    - [ ] 21.8.2 Verify admin credentials are never exposed
    - [ ] 21.8.3 Verify TLS is used for ES API calls
    - [ ] 21.8.4 Verify passwords are encrypted in cache (not plaintext)
    - [ ] 21.8.5 Verify encryption key is not logged or exposed
    - [ ] 21.8.6 Test encryption key rotation (invalidates cache)

- [ ] 22. Deployment preparation
  - [ ] 22.1 Update deployment documentation
  - [ ] 22.2 Create migration checklist
  - [ ] 22.3 Prepare rollback plan
  - [ ] 22.4 Update CI/CD pipeline
    - [ ] 22.4.1 Add integration tests to CI
    - [ ] 22.4.2 Add property-based tests to CI
  - [ ] 22.5 Create release notes
  - [ ] 22.6 Tag release version

## Task Dependencies

```
Phase 1 (Config) → Phase 2 (ES Client) → Phase 4 (User Manager)
                → Phase 3 (Role Mapper) → Phase 4 (User Manager)
                
Phase 4 (User Manager) → Phase 5 (Auth Integration) → Phase 6 (Transport)

Phase 6 (Transport) → Phase 7 (Main App) → Phase 8 (Testing)

Phase 8 (Testing) → Phase 9 (Docs) → Phase 10 (Monitoring) → Phase 11 (Deployment)
```

## Estimated Timeline

- **Phase 1**: 2-3 days
- **Phase 2**: 3-4 days
- **Phase 3**: 2-3 days
- **Phase 4**: 3-4 days
- **Phase 5**: 3-4 days
- **Phase 6**: 2-3 days
- **Phase 7**: 1-2 days
- **Phase 8**: 3-4 days
- **Phase 9**: 2-3 days
- **Phase 10**: 1-2 days
- **Phase 11**: 2-3 days

**Total**: 24-35 days (approximately 5-7 weeks)

## Success Criteria

- [ ] All unit tests pass
- [ ] All integration tests pass
- [ ] All property-based tests pass
- [ ] Code coverage > 80%
- [ ] No build warnings or errors
- [ ] Documentation complete
- [ ] Performance targets met (< 500ms p95 for cache miss, < 10ms for cache hit)
- [ ] Cache hit rate > 95% in production
- [ ] All authentication methods work with dynamic user management
- [ ] Horizontal scaling works with Redis cache
- [ ] ES audit logs show actual usernames
- [ ] Role mappings work correctly for all scenarios
