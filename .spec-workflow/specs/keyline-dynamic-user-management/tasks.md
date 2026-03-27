# Dynamic Elasticsearch User Management - Implementation Tasks

## Phase 1: Backend - Configuration and Foundation

- [x] 1.1 Update configuration structures
  - File: internal/config/config.go, internal/config/validator.go
  - Add RoleMapping, CacheConfig, UserMgmtConfig structs
  - Update LocalUser with Groups, Email, FullName fields
  - Remove ESUser field from LocalUser (breaking change)
  - Add config validation for new fields
  - _Leverage: existing config package structures, mapstructure tags_
  - _Requirements: FR3, FR4, FR5_
  - _Prompt: Role: Go Backend Developer | Task: Add new configuration structures for dynamic user management including RoleMapping, CacheConfig, UserMgmtConfig, and update LocalUser with groups/email/fullName fields. Remove ESUser field from LocalUser | Restrictions: Do not change existing config loading logic, only add new structures. Ensure backward compatibility where possible | Success: All new config structures compile, validator accepts new fields, example configs updated_

- [x] 1.2 Create password generator
  - File: internal/usermgmt/password.go, internal/usermgmt/password_test.go
  - Implement PasswordGenerator struct with configurable length
  - Use crypto/rand for cryptographically secure generation
  - Include uppercase, lowercase, digits, special characters
  - Add unit tests for randomness and character set
  - _Leverage: crypto/rand, math/big_
  - _Requirements: FR2, NFR2_
  - _Prompt: Role: Go Security Developer | Task: Implement PasswordGenerator using crypto/rand for cryptographically secure password generation with configurable length (default 32) and charset including uppercase, lowercase, digits, and special characters | Restrictions: Do NOT use math/rand. Never log passwords. Must use crypto/rand for security | Success: Generates 32+ char passwords, includes all character types, passes randomness tests (no duplicates in 1000 generations)_

- [x] 1.3 Create credential encryptor
  - File: internal/usermgmt/encryptor.go, internal/usermgmt/encryptor_test.go
  - Implement Encryptor interface with Encrypt() and Decrypt()
  - Use AES-256-GCM for authenticated encryption
  - Generate random nonce for each encryption
  - Base64 encode for cache storage
  - _Leverage: crypto/aes, crypto/cipher, encoding/base64, crypto/rand_
  - _Requirements: FR3, NFR2_
  - _Prompt: Role: Go Cryptography Developer | Task: Implement Encryptor interface with AES-256-GCM encryption for password caching. Include Encrypt() and Decrypt() methods with random nonce generation and base64 encoding for storage | Restrictions: Key must be exactly 32 bytes. Never log encryption keys or plaintext passwords. Use crypto/rand for nonce | Success: Round-trip encryption/decryption works, rejects invalid key lengths, same plaintext produces different ciphertexts (random nonce)_

## Phase 2: Backend - Elasticsearch API Client

- [x] 2.1 Implement ES API client
  - File: internal/elasticsearch/client.go
  - Create Client interface with CreateOrUpdateUser, GetUser, DeleteUser, ValidateConnection
  - Implement HTTP client with TLS configuration
  - Add retry logic (3 attempts, exponential backoff)
  - Implement circuit breaker pattern
  - Add OpenTelemetry tracing with otelgo
  - _Leverage: github.com/wasilak/otelgo, net/http_
  - _Requirements: FR1, FR5, NFR3_
  - _Prompt: Role: Go Elasticsearch Developer | Task: Implement ES Security API client with CreateOrUpdateUser, GetUser, DeleteUser, and ValidateConnection methods. Include retry logic (3 attempts, exponential backoff), circuit breaker, and OpenTelemetry tracing | Restrictions: Do not hardcode admin credentials - use config. Must include otelgo tracing. Handle all HTTP error codes appropriately | Success: All methods implemented, retry logic works, circuit breaker activates on ES unavailability, tracing spans created_

- [ ] 2.2 Add ES API client tests
  - File: internal/elasticsearch/client_test.go
  - Mock HTTP responses for all error codes
  - Test success paths (200 OK)
  - Test error handling (401, 403, 404, 429, 5xx)
  - Test retry logic and circuit breaker
  - _Leverage: net/http/httptest_
  - _Requirements: FR1, NFR3_
  - _Prompt: Role: Go Test Developer | Task: Create comprehensive unit tests for ES API client with mocked HTTP responses covering success paths, error handling (401, 403, 404, 429, 5xx), retry logic, and circuit breaker | Restrictions: Use httptest.Server for mocking. Do not require real ES cluster for unit tests | Success: All error codes tested, retry logic verified, circuit breaker tested, >80% coverage_

## Phase 3: Backend - Role Mapper

- [x] 3.1 Implement role mapper
  - File: internal/usermgmt/rolemapper.go
  - Create RoleMapper struct with config and logger
  - Implement MapGroupsToRoles method
  - Evaluate ALL role mappings (not stop at first)
  - Collect ALL matching ES roles (deduplicate)
  - Fall back to default_es_roles if no matches
  - Return error if no matches AND no defaults
  - _Leverage: existing mapper.matchPattern logic, loggergo_
  - _Requirements: FR4, US-2_
  - _Prompt: Role: Go Authorization Developer | Task: Implement RoleMapper with MapGroupsToRoles method that evaluates ALL role mappings, collects matching ES roles, and falls back to default_es_roles if no matches. Include pattern matching (exact, wildcard prefix/suffix/middle) | Restrictions: Must evaluate ALL mappings (not stop at first match). Deduplicate roles. Return error only if no matches AND no defaults | Success: Multiple groups accumulate roles, wildcards work, defaults applied correctly, access denied when appropriate_

- [x] 3.2 Add role mapper tests
  - File: internal/usermgmt/rolemapper_test.go
  - Test pattern matching (exact, wildcards)
  - Test multiple group accumulation
  - Test default roles fallback
  - Test access denial (no matches, no defaults)
  - _Leverage: testify/assert_
  - _Requirements: FR4, US-2_
  - _Prompt: Role: Go Test Developer | Task: Create unit tests for role mapper covering pattern matching (exact, wildcards), multiple group accumulation, default roles fallback, and access denial when no matches and no defaults | Restrictions: Test all pattern types. Include edge cases (empty groups, duplicate roles) | Success: All pattern types tested, accumulation verified, defaults tested, error cases covered_

## Phase 4: Backend - User Manager

- [x] 4.1 Implement user manager
  - File: internal/usermgmt/manager.go
  - Create Manager interface with UpsertUser, InvalidateCache
  - Check cache for existing credentials
  - Generate password on cache miss
  - Map groups to roles
  - Create/update ES user
  - Encrypt and cache credentials
  - Add Prometheus metrics
  - Add OpenTelemetry tracing
  - _Leverage: github.com/wasilak/cachego, otelgo, Prometheus client_
  - _Requirements: FR3, FR4, FR6, US-3, NFR4_
  - _Prompt: Role: Go Backend Developer | Task: Implement UserManager with UpsertUser method that orchestrates cache lookup, password generation, role mapping, ES user creation, and credential caching with encryption. Include InvalidateCache method | Restrictions: Must encrypt passwords before caching. Handle cache failures gracefully (don't fail request). Include otelgo tracing and Prometheus metrics | Success: Cache hit path works, cache miss path creates ES user, encryption/decryption works, metrics recorded, tracing spans created_

- [x] 4.2 Add user manager tests
  - File: internal/usermgmt/manager_test.go
  - Test cache hit path (decrypt and return)
  - Test cache miss path (generate, map, create, cache)
  - Test decryption failure (regenerate)
  - Test error paths (ES failure, cache failure)
  - Mock all dependencies
  - _Leverage: gomock or testify/mock_
  - _Requirements: FR3, FR6, US-3_
  - _Prompt: Role: Go Test Developer | Task: Create unit tests for user manager with mocked dependencies covering cache hit (decrypt and return), cache miss (generate, map, create, cache), decryption failure (regenerate), and error paths | Restrictions: Mock all dependencies (ES client, cache, pwdGen, encryptor). Test that cache failures don't fail requests | Success: All paths tested, error handling verified, encryption tested in cache context_

## Phase 5: Backend - Auth Integration

- [x] 5.1 Update auth engine
  - File: internal/auth/engine.go
  - Add userManager field to Engine struct
  - Update Authenticate to call UpsertUser after auth
  - Use returned credentials for Authorization header
  - Handle user management failures gracefully
  - _Leverage: existing auth engine patterns_
  - _Requirements: FR6, US-1_
  - _Prompt: Role: Go Auth Developer | Task: Integrate UserManager into Auth Engine by adding userManager field, updating Authenticate method to call UpsertUser after successful authentication, and using returned credentials for ES Authorization header | Restrictions: Maintain backward compatibility. Don't break existing auth flows. Handle user management failures gracefully | Success: All auth methods create ES users, credentials used for Authorization header, errors handled gracefully_

- [x] 5.2 Update OIDC provider
  - File: internal/auth/oidc.go
  - Extract groups from OIDC claims
  - Handle []interface{}, []string, string formats
  - Include groups, email, full_name in AuthResult
  - Handle missing groups claim (return empty array)
  - _Leverage: existing claims extraction logic_
  - _Requirements: FR6, US-2, US-6_
  - _Prompt: Role: Go OIDC Developer | Task: Update OIDC provider to extract groups from claims (handle []interface{}, []string, string formats) and include groups, email, full_name in AuthResult | Restrictions: Handle missing groups claim gracefully (return empty array). Don't break existing OIDC flow | Success: Groups extracted from all formats, email/full_name included, missing claims handled_

- [x] 5.3 Update Basic Auth provider
  - File: internal/auth/basic.go
  - Return groups from local user config
  - Include email, full_name in AuthResult
  - Handle users with no groups
  - _Leverage: LocalUser config structure_
  - _Requirements: FR6, US-2, US-6_
  - _Prompt: Role: Go Auth Developer | Task: Update Basic Auth provider to return groups, email, full_name from local user config in AuthResult | Restrictions: Maintain backward compatibility with existing local user configs. Handle users with no groups | Success: Local user groups returned, email/full_name included, users with no groups work_

## Phase 6: Backend - Transport Integration

- [x] 6.1 Update transport adapters
  - File: internal/transport/forward_auth.go, internal/transport/standalone.go
  - Use ES credentials from auth result
  - Set Authorization header with generated credentials
  - Remove old credential mapping logic
  - Test both forward_auth and standalone modes
  - _Leverage: existing transport patterns_
  - _Requirements: FR6_
  - _Prompt: Role: Go Transport Developer | Task: Update transport adapters to use ES credentials from auth result (ESUser, ESPassword) and set Authorization header with generated credentials. Remove old credential mapping logic | Restrictions: Don't break existing transport flows. Ensure both forward_auth and standalone modes work | Success: Authorization header set with generated credentials, old logic removed, both modes work_

## Phase 7: Backend - Main Application Integration

- [x] 7.1 Update main application
  - File: cmd/keyline/main.go
  - Initialize ES API client with admin credentials
  - Validate ES connection on startup
  - Initialize password generator
  - Initialize encryptor (validate 32-byte key)
  - Initialize role mapper
  - Initialize user manager
  - Pass user manager to auth engine
  - Add feature flag check (user_management.enabled)
  - Add startup logging
  - _Leverage: existing component initialization patterns_
  - _Requirements: FR1, FR2, FR3, FR4, FR5_
  - _Prompt: Role: Go Application Developer | Task: Initialize all new components in main.go: ES API client (with admin credentials, validate on startup), password generator, encryptor (validate 32-byte key), role mapper, user manager. Pass user manager to auth engine. Add feature flag check | Restrictions: Validate all components on startup. Fail fast if critical config missing. Log user management status | Success: All components initialized, validation passes, feature flag works, startup logging added_

## Phase 8: Testing

- [ ] 8.1 Integration tests
  - File: integration/user_management_test.go
  - Test OIDC auth with user creation
  - Test Basic Auth with groups
  - Test cache expiration
  - Test role mapping scenarios
  - Test ES unavailability handling
  - _Leverage: testcontainers-go_
  - _Requirements: FR6, US-1, US-2, US-3, NFR3_
  - _Prompt: Role: Go Test Developer (Integration) | Task: Create integration tests covering end-to-end OIDC auth with user creation, Basic Auth with groups, cache expiration, role mapping scenarios, and ES unavailability handling | Restrictions: Requires real ES cluster and Redis. Use testcontainers if available. Clean up after tests | Success: All auth methods tested, cache expiration verified, role mappings work, ES unavailability handled_

- [ ] 8.2 Property-based tests
  - File: internal/usermgmt/properties_test.go
  - Test password generation (always valid, no duplicates)
  - Test role mapping (deterministic, accumulative)
  - Test cache operations (idempotent)
  - Run 10,000+ iterations for password uniqueness
  - _Leverage: testing/quick_
  - _Requirements: FR2, FR3, FR4, NFR2_
  - _Prompt: Role: Go Test Developer (Property-Based) | Task: Create property-based tests for password generation (always valid, no duplicates), role mapping (deterministic, accumulative), and cache operations (idempotent) | Restrictions: Use testing/quick or gobwas/gotest. Run 10,000+ iterations for password uniqueness | Success: Properties verified, no duplicates in 10k password generations, determinism proven_

## Phase 9: Documentation

- [ ] 9.1 Update documentation
  - File: docs/user-management.md, docs/configuration.md, docs/migration-guide.md, docs/troubleshooting-user-management.md, README.md
  - Create user management overview
  - Document configuration with role mapping examples
  - Create migration guide from static mapping
  - Create troubleshooting guide
  - Update README with feature description
  - Include security considerations
  - _Leverage: existing docs structure_
  - _Requirements: All_
  - _Prompt: Role: Technical Writer | Task: Create comprehensive documentation including user management overview, configuration guide with role mapping examples, migration guide from static mapping, troubleshooting guide, and update README with new feature description | Restrictions: Include security considerations (encryption key management). Provide working config examples | Success: All docs created, examples work, migration path clear, troubleshooting covers common issues_

- [ ] 9.2 Update example configurations
  - File: config/config.example.yaml, config/user-management-example.yaml
  - Add role_mappings section
  - Add default_es_roles section
  - Add cache config with encryption_key
  - Add elasticsearch admin credentials
  - Use environment variable placeholders for secrets
  - Include working examples for all pattern types
  - _Leverage: configuration schema from requirements_
  - _Requirements: FR3, FR4, FR5_
  - _Prompt: Role: Configuration Developer | Task: Update example configs with new sections (role_mappings, default_es_roles, cache with encryption_key, elasticsearch admin credentials). Add comments explaining each option | Restrictions: Use environment variable placeholders for secrets. Include working examples for all pattern types | Success: All new config sections documented, examples work, secrets properly referenced_

## Phase 10: Monitoring and Observability

- [x] 10.1 Add metrics
  - File: internal/usermgmt/metrics.go
  - Define keyline_user_upserts_total (counter)
  - Define keyline_user_upsert_duration_seconds (histogram)
  - Define keyline_cred_cache_hits/misses_total (counters)
  - Define keyline_role_mapping_matches_total (counter)
  - Define keyline_es_api_calls_total (counter)
  - Register metrics with Prometheus
  - Instrument all components
  - _Leverage: Prometheus client_
  - _Requirements: NFR4_
  - _Prompt: Role: Go Observability Developer | Task: Create Prometheus metrics for user upserts (count, duration), cache hits/misses, role mapping matches, and ES API calls. Register metrics and instrument all components | Restrictions: Use appropriate metric types (counter, histogram). Include relevant labels. Don't expose sensitive data in labels | Success: All metrics defined, registered, instrumented, visible in Prometheus_

- [x] 10.2 Add observability
  - File: All component files
  - Add OpenTelemetry spans to all operations
  - Add structured logging with loggergo
  - Include context in logs
  - Don't log sensitive data (passwords, keys)
  - _Leverage: github.com/wasilak/otelgo, github.com/wasilak/loggergo_
  - _Requirements: NFR4_
  - _Prompt: Role: Go Observability Developer | Task: Add OpenTelemetry spans to all operations (user upsert, ES API calls, role mapping, cache ops) and structured logging with loggergo for all user management operations | Restrictions: Use otelgo for tracing. Include context in logs. Don't log sensitive data (passwords, keys) | Success: All operations traced, structured logging in place, traces visible in Jaeger/Tempo_

## Phase 11: Final Testing and Deployment

- [ ] 11.1 End-to-end testing
  - File: Manual testing, performance tests
  - Test with real ES cluster and Redis
  - Test horizontal scaling (multiple Keyline instances)
  - Verify all auth methods
  - Test all role mapping scenarios
  - Run performance tests (`<500ms` p95 cache miss)
  - Verify security (passwords encrypted, not logged)
  - _Leverage: wrk, hey for performance testing_
  - _Requirements: All_
  - _Prompt: Role: QA Engineer | Task: Perform comprehensive end-to-end testing with real ES cluster and Redis, test horizontal scaling with multiple Keyline instances, verify all auth methods, test all role mapping scenarios, run performance tests, and conduct security testing | Restrictions: Test in production-like environment. Verify performance targets (`<500ms` p95 cache miss, `>95%` hit rate). Verify security (passwords encrypted, not logged) | Success: All scenarios pass, performance targets met, security verified, horizontal scaling works_

- [ ] 11.2 Deployment preparation
  - File: RELEASE-NOTES.md, CI/CD pipeline, deployment docs
  - Update deployment documentation
  - Create migration checklist
  - Prepare rollback plan
  - Update CI/CD with integration tests
  - Create release notes
  - Tag release version
  - Document encryption key rotation procedure
  - _Leverage: existing deployment docs, CI/CD pipeline_
  - _Requirements: All_
  - _Prompt: Role: DevOps Engineer | Task: Update deployment documentation, create migration checklist, prepare rollback plan, update CI/CD pipeline with integration tests, create release notes, and tag release version | Restrictions: Include encryption key rotation procedure. Document breaking changes (LocalUser.ESUser removed) | Success: Deployment docs complete, CI/CD updated, release notes published, version tagged_

## Task Dependencies

```
Phase 1 (Config) → Phase 2 (ES Client) → Phase 4 (User Manager)
                → Phase 3 (Role Mapper) → Phase 4 (User Manager)

Phase 4 (User Manager) → Phase 5 (Auth Integration) → Phase 6 (Transport)

Phase 6 (Transport) → Phase 7 (Main App) → Phase 8 (Testing)

Phase 8 (Testing) → Phase 9 (Docs) → Phase 10 (Monitoring) → Phase 11 (Deployment)
```

## Estimated Timeline

| Phase | Description | Estimated Time |
|-------|-------------|----------------|
| Phase 1 | Backend - Configuration and Foundation | 2-3 days |
| Phase 2 | Backend - Elasticsearch API Client | 3-4 days |
| Phase 3 | Backend - Role Mapper | 2-3 days |
| Phase 4 | Backend - User Manager | 3-4 days |
| Phase 5 | Backend - Auth Integration | 3-4 days |
| Phase 6 | Backend - Transport Integration | 2-3 days |
| Phase 7 | Backend - Main Application Integration | 1-2 days |
| Phase 8 | Testing | 3-4 days |
| Phase 9 | Documentation | 2-3 days |
| Phase 10 | Monitoring and Observability | 1-2 days |
| Phase 11 | Final Testing and Deployment | 2-3 days |

**Total**: 24-35 days (approximately 5-7 weeks)
