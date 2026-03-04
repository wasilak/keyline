# Implementation Plan: Keyline Authentication Proxy

## Overview

This implementation plan converts the Keyline design into actionable coding tasks following a 6-phase approach. The implementation uses modern Go observability libraries (loggergo, otelgo, cachego) and builds a unified authentication proxy supporting dual authentication modes (OIDC + Basic Auth) and three deployment modes (forwardAuth, auth_request, standalone proxy).

**Technology Stack**: Go 1.22+, Echo v4, Viper, cachego, loggergo, otelgo, slog-echo, otelecho, coreos/go-oidc, gopter (property testing)

**Implementation Timeline**: 4 weeks (6 phases)

## Tasks

- [-] 1. Phase 1: Core Infrastructure Setup (Week 1)
  - [x] 1.1 Initialize Go project structure and dependencies
    - Create project directory structure (cmd/, internal/, pkg/, integration/, docs/)
    - Initialize go.mod with Go 1.22+ and required dependencies
    - Add dependencies: Echo v4, Viper, cachego, loggergo, otelgo, slog-echo, otelecho, coreos/go-oidc, gopter
    - Create Makefile with build, test, lint, and run targets
    - _Requirements: All requirements (foundation)_

  - [x] 1.2 Implement configuration types and schema
    - Create internal/config/config.go with all configuration structs
    - Define ServerConfig, OIDCConfig, LocalUsersConfig, SessionConfig, CacheConfig, ElasticsearchConfig, UpstreamConfig, ObservabilityConfig
    - Add struct tags for Viper mapstructure binding
    - Add otelgo-specific fields (service_name, service_version, environment, trace_ratio)
    - _Requirements: 12.1, 12.5, 20.1_

  - [x] 1.3 Implement configuration loader with environment variable substitution
    - Create internal/config/loader.go with LoadConfig function
    - Implement ${VAR_NAME} environment variable substitution using Viper
    - Handle missing environment variables with descriptive errors
    - Support --config flag and CONFIG_FILE environment variable
    - _Requirements: 12.1, 12.2, 12.3, 12.4_

  - [x] 1.4 Implement configuration validator
    - Create internal/config/validator.go with ValidateConfig function
    - Validate required fields based on enabled features (OIDC, local users, mode)
    - Validate session_secret is at least 32 bytes
    - Validate password_bcrypt values are valid bcrypt hashes
    - Validate redirect_url is valid HTTPS URL
    - Validate at least one authentication method is enabled
    - Validate cache backend configuration (redis_url if backend=redis)
    - Return descriptive errors for each validation failure
    - _Requirements: 12.5, 12.6, 12.7, 12.8, 12.9, 12.10, 12.11, 17.1, 20.2, 20.3, 20.4, 20.5, 20.6, 20.7, 20.8, 20.9_

  - [ ]* 1.5 Write property test for configuration validation
    - **Property 17: Configuration Validation Completeness**
    - **Validates: Requirements 12.5, 12.6, 12.7, 12.8, 12.9, 12.10, 12.11, 20.1-20.9**
    - Generate random configurations with missing/invalid fields
    - Verify validation catches all errors
    - Use gopter with minimum 100 iterations

  - [ ]* 1.6 Write property test for environment variable substitution
    - **Property 16: Configuration Environment Variable Substitution**
    - **Validates: Requirements 12.2, 12.3, 12.4**
    - Generate random config values with ${VAR_NAME} syntax
    - Verify substitution works correctly
    - Verify missing variables cause startup failure

  - [x] 1.7 Initialize observability with loggergo and otelgo
    - Initialize loggergo in main.go with config values (log_level, log_format)
    - Initialize otelgo if otel_enabled=true with OTLP exporter
    - Store shutdown function for graceful cleanup
    - Set global slog logger for use throughout application
    - Configure trace provider with service name, version, environment
    - _Requirements: 14.1, 14.7, 14.8, 14.9, 19.1, 19.8, 19.9_

  - [x] 1.8 Initialize cachego backend
    - Create internal/cache/cache.go with InitCache function
    - Initialize Redis backend if cache.backend=redis
    - Initialize memory backend if cache.backend=memory
    - Return cachego.Cache interface
    - Test connection and return error if initialization fails
    - _Requirements: 16.1, 16.2, 16.9, 17.1_

  - [ ]* 1.9 Write property test for cache serialization
    - **Property 19: Cache Serialization Round-Trip**
    - **Validates: Requirements 16.3, 16.5**
    - Generate random sessions and state tokens
    - Verify serialize-deserialize produces equivalent objects
    - Test with both memory and Redis backends
    - Use gopter minimum 100 iterations

  - [ ]* 1.10 Write property test for cache TTL consistency
    - **Property 20: Cache Key TTL Consistency**
    - **Validates: Requirements 16.4, 16.8**
    - Generate random sessions with various TTLs
    - Verify cache key TTL matches session expiration
    - Verify state tokens have 5-minute TTL
    - Test with Redis backend

  - [x] 1.11 Implement session operations using cachego
    - Create internal/session/session.go with session helper functions
    - Implement CreateSession(ctx, cache, session) with manual span
    - Implement GetSession(ctx, cache, sessionID) with manual span
    - Implement DeleteSession(ctx, cache, sessionID) with manual span
    - Use slog.InfoContext(ctx, ...) for all logging
    - Serialize sessions to JSON with key prefix "session:"
    - _Requirements: 4.1, 4.2, 4.3, 4.4, 4.5, 4.6, 4.7, 4.8_

  - [x] 1.12 Implement state token operations using cachego
    - Create internal/state/state.go with state token helper functions
    - Implement StoreStateToken(ctx, cache, token) with manual span
    - Implement GetStateToken(ctx, cache, tokenID) with manual span (marks as used)
    - Implement DeleteStateToken(ctx, cache, tokenID) with manual span
    - Use slog.InfoContext(ctx, ...) for all logging
    - Serialize tokens to JSON with key prefix "state:"
    - Set 5-minute TTL for all state tokens
    - _Requirements: 3.2, 3.5, 3.7, 3.8, 16.7, 16.8_

  - [ ]* 1.13 Write property test for state token single-use enforcement
    - **Property 1: State Token Single-Use Enforcement**
    - **Validates: Requirements 3.5, 3.6, 3.7, 3.8, 13.14, 13.15**
    - Generate random state tokens
    - Verify first use succeeds, second use fails
    - Test with both memory and Redis backends

  - [ ]* 1.14 Write property test for state token lifecycle
    - **Property 2: State Token Lifecycle**
    - **Validates: Requirements 3.1, 3.2, 13.1, 13.15**
    - Generate random state tokens with original URLs
    - Verify 5-minute TTL, cryptographic randomness (32 bytes)
    - Verify deletion after use or expiration

  - [x] 1.15 Implement basic HTTP server with Echo and middleware
    - Create internal/server/server.go with Server struct
    - Initialize Echo instance
    - Add otelecho middleware for automatic request tracing
    - Add slog-echo middleware for automatic request logging
    - Add middleware: RequestID, Recover, CORS
    - Configure read/write timeouts from config
    - Implement graceful shutdown with 30-second timeout
    - Add signal handling for SIGTERM and SIGINT
    - _Requirements: 15.9, 15.10, 19.2, 19.3, 19.7_

  - [x] 1.16 Implement health check endpoint
    - Create /healthz endpoint handler
    - Return 200 with JSON {status: "healthy", version: "x.y.z"}
    - Check cache accessibility using cache.Exists(ctx, "healthcheck")
    - Return 503 if cache unavailable
    - Endpoint requires no authentication
    - _Requirements: 11.1, 11.2, 11.3, 11.4_

- [ ] 2. Checkpoint - Phase 1 Complete
  - Ensure all tests pass, ask the user if questions arise.


- [ ] 3. Phase 2: OIDC Authentication (Week 2)
  - [x] 3.1 Implement OIDC cache for Discovery Document and JWKS
    - Create internal/cache/oidc.go with OIDCCache struct
    - Implement caching for Discovery Document
    - Implement caching for JWKS with expiry tracking
    - Add RefreshJWKS method with 24-hour refresh interval
    - Use sync.RWMutex for thread-safe access
    - _Requirements: 1.5, 1.6, 17.1, 17.2, 17.3_

  - [x] 3.2 Implement OIDC provider discovery
    - Create internal/auth/oidc.go with OIDCProvider struct
    - Implement discovery document fetch from {issuer}/.well-known/openid-configuration
    - Extract authorization_endpoint, token_endpoint, userinfo_endpoint, jwks_uri, issuer
    - Validate issuer matches configured issuer_url
    - Retry up to 3 times with exponential backoff on failure
    - Refuse to start if discovery fails after retries
    - _Requirements: 1.1, 1.2, 1.3, 1.4, 15.6_

  - [ ]* 3.3 Write property test for discovery document validation
    - **Property 18: Discovery Document Validation**
    - **Validates: Requirements 1.4**
    - Generate discovery documents with various issuer values
    - Verify mismatch causes startup failure

  - [x] 3.4 Implement JWKS fetching and caching
    - Fetch JWKS from jwks_uri during initialization
    - Parse and cache public keys for signature verification
    - Implement background refresh every 24 hours
    - Log warning and continue with cached JWKS on refresh failure
    - Retry up to 3 times with exponential backoff on initial fetch failure
    - _Requirements: 1.6, 1.7, 15.4, 15.5_

  - [ ] 3.5 Implement cryptographic random generation utilities
    - Create pkg/crypto/random.go with GenerateRandomBytes function
    - Use crypto/rand for cryptographically secure randomness
    - Implement GenerateStateToken (32 bytes)
    - Implement GenerateSessionID (32 bytes)
    - Return base64-encoded strings
    - _Requirements: 13.1, 13.2_

  - [ ]* 3.6 Write property test for cryptographic randomness
    - **Property 22: Cryptographic Randomness**
    - **Validates: Requirements 13.1, 13.2**
    - Generate multiple state tokens and session IDs
    - Verify sufficient entropy (32 bytes minimum)
    - Verify uniqueness across generations
    - Verify no predictable patterns

  - [ ] 3.7 Implement PKCE generation
    - Add GeneratePKCE function to pkg/crypto/random.go
    - Generate code_verifier (43-128 characters, URL-safe)
    - Derive code_challenge using S256 method (SHA256 hash, base64url-encoded)
    - Return both verifier and challenge
    - _Requirements: 3.3, 13.16_

  - [ ]* 3.8 Write property test for PKCE generation
    - **Property 8: PKCE Generation and Validation**
    - **Validates: Requirements 3.3, 13.16**
    - Generate random PKCE pairs
    - Verify code_challenge is correctly derived from code_verifier using S256
    - Verify verifier is cryptographically random

  - [ ] 3.9 Implement OIDC authorization flow initiation
    - Implement Authenticate method in OIDCProvider
    - Generate state token and store with original URL
    - Generate PKCE code_verifier and code_challenge
    - Store code_verifier with state token
    - Build authorization URL with all required parameters
    - Return redirect response with authorization URL
    - _Requirements: 3.1, 3.2, 3.3, 3.4_

  - [ ] 3.10 Implement OIDC callback handler
    - Implement HandleCallback method in OIDCProvider
    - Extract state and code from query parameters
    - Validate state token exists and is unused
    - Mark state token as used and delete from store
    - Return 400 for invalid/expired/used state tokens
    - _Requirements: 3.5, 3.6, 3.7, 3.8_

  - [ ] 3.11 Implement token exchange
    - Implement exchangeToken method in OIDCProvider
    - Make POST request to token_endpoint with code, client_id, client_secret, redirect_uri, grant_type, code_verifier
    - Use 30-second timeout for request
    - Parse token response to extract ID token, access token, refresh token
    - Return 401 on token exchange failure
    - _Requirements: 3.9, 3.10, 17.7_

  - [ ] 3.12 Implement ID token validation
    - Implement validateIDToken method in OIDCProvider
    - Verify ID token signature using JWKS public keys
    - Validate iss claim matches configured issuer_url
    - Validate aud claim matches configured client_id
    - Validate exp claim is in the future
    - Return 401 for any validation failure
    - _Requirements: 3.11, 3.12, 3.13, 3.14, 13.9, 13.10, 13.11, 13.12, 13.13_

  - [ ]* 3.13 Write property test for ID token validation
    - **Property 6: ID Token Validation Completeness**
    - **Validates: Requirements 3.11, 3.13, 13.9-13.13**
    - Generate ID tokens with various claim combinations
    - Verify signature validation, iss/aud/exp checks
    - Verify any validation failure returns 401

  - [ ] 3.14 Implement session creation from OIDC user
    - Extract user claims from validated ID token
    - Generate cryptographically random session ID
    - Create Session with user identity, mapped ES user, expiration
    - Store session in SessionStore
    - Return session cookie with HttpOnly, Secure, SameSite=Lax
    - _Requirements: 3.15, 4.1, 4.2, 4.3, 4.4, 13.5, 13.6, 13.7, 13.8_

  - [ ]* 3.15 Write property test for session creation
    - **Property 3: Session Creation and Storage**
    - **Validates: Requirements 4.1-4.4, 13.2, 13.5-13.8**
    - Generate random OIDC users
    - Verify session ID is cryptographically random
    - Verify cookie has correct security attributes
    - Verify session stored with all required fields

  - [ ] 3.16 Implement redirect to original URL after authentication
    - After session creation, retrieve original URL from state token
    - Return 302 redirect to original URL
    - Include Set-Cookie header with session cookie
    - _Requirements: 3.16_

  - [ ]* 3.17 Write property test for complete OIDC flow
    - **Property 7: OIDC Authorization Flow Completeness**
    - **Validates: Requirements 3.1-3.16, 13.16**
    - Simulate complete OIDC flow with mock provider
    - Verify all steps execute in correct order
    - Verify state token, PKCE, token exchange, validation, session creation

  - [ ] 3.18 Add OIDC health check to /healthz endpoint
    - When OIDC is enabled, verify Discovery Document was loaded
    - Return 503 if Discovery Document not loaded
    - _Requirements: 11.5, 11.6_

- [ ] 4. Checkpoint - Phase 2 Complete
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 5. Phase 3: Basic Authentication (Week 2)
  - [ ] 5.1 Implement Basic Auth credential parsing
    - Create internal/auth/basic.go with BasicAuthProvider struct
    - Implement Authenticate method
    - Extract Authorization header from request
    - Decode base64-encoded credentials
    - Return 401 with WWW-Authenticate header if decoding fails
    - _Requirements: 5.1, 5.2_

  - [ ] 5.2 Implement username and password extraction
    - Split decoded credentials on ":" separator
    - Extract username and password
    - Handle edge cases (missing separator, empty values)
    - _Requirements: 5.3_

  - [ ] 5.3 Implement local user lookup
    - Search configured LocalUser list for matching username
    - Return 401 with WWW-Authenticate header if username not found
    - _Requirements: 5.4, 5.5_

  - [ ] 5.4 Implement bcrypt password validation
    - Use bcrypt.CompareHashAndPassword for timing-safe comparison
    - Compare provided password against stored password_bcrypt
    - Return 401 with WWW-Authenticate header if validation fails
    - _Requirements: 5.6, 5.7, 13.3_

  - [ ]* 5.5 Write property test for Basic Auth validation
    - **Property 9: Basic Auth Credential Validation**
    - **Validates: Requirements 5.1-5.7, 13.3**
    - Generate random usernames and passwords
    - Verify base64 decoding, username lookup, bcrypt validation
    - Verify timing-safe comparison

  - [ ] 5.6 Implement ES user mapping for local users
    - Retrieve es_user value from matched LocalUser
    - Return AuthResult with authenticated user and mapped ES user
    - Do not create session (Basic Auth is stateless)
    - _Requirements: 5.8, 5.9_

  - [ ] 5.7 Implement ES credential mapper
    - Create internal/mapper/credentials.go with CredentialMapper struct
    - Implement MapOIDCUser method with claim extraction and pattern matching
    - Implement MapLocalUser method (simple lookup)
    - Implement GetESCredentials method to retrieve ES username/password
    - Return error if ES user not found in configuration
    - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.5, 6.6, 6.7, 6.8, 6.9_

  - [ ] 5.8 Implement wildcard pattern matching for OIDC mappings
    - Add matchPattern function to CredentialMapper
    - Support * wildcard matching (e.g., "*@admin.example.com")
    - Evaluate mappings in configuration order
    - Return first matching es_user or default_es_user
    - _Requirements: 6.3, 6.4, 6.5, 6.6_

  - [ ]* 5.9 Write property test for ES credential mapping
    - **Property 10: ES Credential Mapping Evaluation Order**
    - **Validates: Requirements 6.1-6.6**
    - Generate random OIDC users with various claim values
    - Verify mappings evaluated in order
    - Verify first match wins, default used if no match

  - [ ] 5.10 Implement ES credential encoding and injection
    - Encode ES credentials as Basic auth (base64 of username:password)
    - Add X-Es-Authorization header with "Basic {encoded_credentials}"
    - Never log ES credentials in plaintext
    - _Requirements: 6.10, 6.11, 6.12_

  - [ ]* 5.11 Write property test for ES credential injection
    - **Property 11: ES Credential Injection**
    - **Validates: Requirements 6.7-6.11**
    - Generate random authenticated users (OIDC and Basic)
    - Verify ES user mapping, credential retrieval, encoding, header injection

- [ ] 6. Checkpoint - Phase 3 Complete
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 7. Phase 4: Transport Adapters (Week 3)
  - [ ] 7.1 Implement TransportAdapter interface
    - Create internal/transport/adapter.go with TransportAdapter interface
    - Define RequestContext struct for normalized request information
    - _Requirements: 7.1, 7.2, 7.3, 8.1, 8.2, 8.3_

  - [ ] 7.2 Implement header normalization for ForwardAuth mode
    - Create internal/transport/forward_auth.go with ForwardAuthAdapter
    - Implement normalizeHeaders method
    - Support X-Forwarded-* headers (Traefik)
    - Support X-Original-* headers (Nginx)
    - Normalize to RequestContext with method, path, host, originalURL
    - _Requirements: 7.1, 7.2, 7.3, 8.1, 8.2, 8.3, 8.4_

  - [ ]* 7.3 Write property test for header normalization
    - **Property 12: Header Normalization Consistency**
    - **Validates: Requirements 7.1-7.3, 8.1-8.4**
    - Generate requests with Traefik and Nginx headers
    - Verify both produce same RequestContext
    - Test with various method, path, host combinations

  - [ ] 7.4 Implement authentication engine
    - Create internal/auth/engine.go with AuthEngine struct
    - Implement Authenticate method with authentication precedence logic
    - Check session cookie first, then Basic Auth header, then initiate OIDC flow
    - Return AuthResult with authentication decision
    - _Requirements: 2.1, 2.2, 2.3_

  - [ ]* 7.5 Write property test for authentication precedence
    - **Property 5: Authentication Method Precedence**
    - **Validates: Requirements 2.1-2.3**
    - Generate requests with various credential combinations
    - Verify session cookie takes precedence
    - Verify Basic Auth attempted when no session
    - Verify OIDC flow initiated when no credentials

  - [ ] 7.6 Implement session validation in authentication engine
    - Extract session cookie from request
    - Retrieve session from SessionStore
    - Check if session exists and is not expired
    - Delete expired sessions from store
    - Treat non-existent or expired sessions as unauthenticated
    - _Requirements: 4.5, 4.6, 4.7, 4.8_

  - [ ]* 7.7 Write property test for session validation
    - **Property 4: Session Validation and Expiration**
    - **Validates: Requirements 4.5-4.8**
    - Generate sessions with various expiration times
    - Verify valid sessions are used
    - Verify expired sessions are deleted and treated as unauthenticated

  - [ ] 7.8 Implement ForwardAuth response handler
    - Implement HandleRequest method in ForwardAuthAdapter
    - Normalize headers to RequestContext
    - Delegate to AuthEngine for authentication
    - Return 200 with X-Es-Authorization header on success
    - Return 401 for Basic Auth failures
    - Return 302 for OIDC redirects
    - Never proxy requests to upstream
    - Preserve Cookie headers from original request
    - _Requirements: 7.4, 7.5, 7.7, 7.8_

  - [ ]* 7.9 Write property test for ForwardAuth response format
    - **Property 13: ForwardAuth Response Format**
    - **Validates: Requirements 7.4, 7.5, 7.7**
    - Generate authenticated and unauthenticated requests
    - Verify 200 with header for authenticated
    - Verify 401/302 for unauthenticated
    - Verify no proxying occurs

  - [ ] 7.10 Implement callback handling in ForwardAuth mode
    - Detect /auth/callback path from X-Forwarded-Uri or X-Original-URI
    - Process OIDC callback
    - Return 302 with Set-Cookie header
    - _Requirements: 7.6_

  - [ ] 7.11 Implement standalone proxy adapter
    - Create internal/transport/standalone.go with StandaloneProxyAdapter
    - Initialize httputil.ReverseProxy with configured upstream URL
    - Implement HandleRequest method
    - Authenticate request using AuthEngine
    - Proxy authenticated requests to upstream
    - Do not proxy /auth/callback, /auth/logout, /healthz
    - _Requirements: 9.1, 9.2, 9.3, 9.4_

  - [ ] 7.12 Implement request preservation in standalone proxy
    - Preserve HTTP method, path, query parameters
    - Preserve request headers (except hop-by-hop headers)
    - Preserve request body
    - Add X-Es-Authorization header before forwarding
    - Use configured upstream timeout
    - _Requirements: 9.5, 9.6, 9.7, 9.14_

  - [ ]* 7.13 Write property test for proxy request preservation
    - **Property 14: Standalone Proxy Request Preservation**
    - **Validates: Requirements 9.1-9.7, 9.14**
    - Generate random HTTP requests
    - Verify method, path, query, headers, body preserved
    - Verify X-Es-Authorization header added
    - Verify internal endpoints not proxied

  - [ ] 7.14 Implement response preservation in standalone proxy
    - Stream response body from upstream to client
    - Preserve response status code
    - Preserve response headers (except hop-by-hop headers)
    - _Requirements: 9.8, 9.9, 9.10_

  - [ ]* 7.15 Write property test for proxy response preservation
    - **Property 15: Standalone Proxy Response Preservation**
    - **Validates: Requirements 9.8-9.10**
    - Generate random upstream responses
    - Verify status code, headers, body preserved

  - [ ] 7.16 Implement upstream error handling
    - Return 502 Bad Gateway if upstream connection fails
    - Return 504 Gateway Timeout if upstream times out
    - Log errors with upstream URL and error details
    - _Requirements: 9.11, 9.12, 15.7_

  - [ ] 7.17 Implement WebSocket upgrade support
    - Detect WebSocket upgrade requests (Upgrade: websocket header)
    - Forward upgrade headers to upstream
    - Establish bidirectional connection
    - Stream data in both directions
    - _Requirements: 9.13_

  - [ ] 7.18 Implement logout endpoint
    - Create /auth/logout handler
    - Extract session ID from session cookie
    - Delete session from SessionStore
    - Return Set-Cookie with Max-Age=0 to clear cookie
    - Redirect to OIDC provider end_session_endpoint if available
    - Otherwise redirect to configured logout_redirect_url or return 200
    - Handle requests without session gracefully (return 200)
    - _Requirements: 10.1, 10.2, 10.3, 10.4, 10.5, 10.6_

  - [ ]* 7.19 Write property test for logout session cleanup
    - **Property 21: Logout Session Cleanup**
    - **Validates: Requirements 10.1-10.3**
    - Generate sessions and logout requests
    - Verify session deleted from store
    - Verify cookie cleared with Max-Age=0
    - Verify subsequent requests treated as unauthenticated


- [ ] 8. Checkpoint - Phase 4 Complete
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 9. Phase 5: Observability (Week 3)
  - [ ] 9.1 Implement Prometheus metrics
    - Create internal/observability/metrics.go with metrics definitions
    - Add counter for authentication attempts (labels: method, result)
    - Add histogram for authentication request duration (labels: method)
    - Add gauge for active sessions
    - Add counter for session operations (labels: operation)
    - Add counter for OIDC provider requests (labels: endpoint, result)
    - Add histogram for upstream proxy request duration
    - Add gauge for current concurrent requests
    - Add counter for errors (labels: error_type)
    - _Requirements: 18.2, 18.3, 18.4, 18.5, 18.6, 18.7, 18.8, 18.9_

  - [ ] 9.2 Implement /metrics endpoint
    - Create /metrics endpoint handler
    - Expose Prometheus metrics in text format
    - Endpoint requires no authentication
    - _Requirements: 18.1, 18.10_

  - [ ] 9.3 Implement OpenTelemetry tracing initialization
    - Create internal/observability/tracing.go with InitTracer function
    - Initialize OTLP exporter with configured endpoint
    - Configure trace provider with W3C Trace Context propagation
    - Return no-op tracer if initialization fails (log warning)
    - _Requirements: 19.1, 19.8, 19.9_

  - [ ] 9.4 Implement request tracing spans
    - Create span for each incoming request (name: "keyline.request")
    - Add span attributes: http.method, http.url, http.status_code, auth.method, auth.result
    - Propagate trace context using W3C Trace Context headers
    - _Requirements: 19.2, 19.3, 19.7_

  - [ ] 9.5 Implement child spans for operations
    - Create child span for OIDC provider requests (name: "keyline.oidc.{endpoint}")
    - Create child span for SessionStore operations (name: "keyline.session.{operation}")
    - Create child span for upstream proxy requests (name: "keyline.proxy.request")
    - _Requirements: 19.4, 19.5, 19.6_

  - [ ] 9.6 Enhance structured logging with context fields
    - Add authentication event logging with username, method, source_ip, result
    - Add OIDC flow event logging with state_token_id, callback_result, error_details
    - Add session event logging with hashed session_id, action, username
    - Add configuration loading logging with config_file, oidc_enabled, local_users_count, mode
    - Add upstream proxy error logging with upstream_url, error_type, response_time
    - Never log sensitive values (passwords, tokens, credentials, full session IDs)
    - _Requirements: 14.2, 14.3, 14.4, 14.5, 14.6, 14.10_

  - [ ] 9.7 Implement error handling with structured logging
    - Log session store failures at ERROR level with service unavailable response
    - Log OIDC provider failures at ERROR level with appropriate response
    - Log configuration errors at ERROR level during startup
    - Log unexpected errors at ERROR level with stack trace
    - _Requirements: 15.1, 15.2, 15.3, 15.8_

  - [ ] 9.8 Implement concurrent request limiting
    - Add middleware to track concurrent requests
    - Limit to configured max_concurrent (default 1000)
    - Return 503 "Server overloaded" when limit reached
    - Update concurrent requests gauge metric
    - _Requirements: 17.5, 17.6_

  - [ ] 9.9 Implement request body size limiting
    - Add middleware to limit request body size to 1MB
    - Return 413 "Request too large" if exceeded
    - _Requirements: 17.9, 17.10_

  - [ ] 9.10 Implement HTTPS enforcement for OIDC provider
    - Configure HTTP client for OIDC provider with TLS certificate validation
    - Use HTTPS for all OIDC provider requests (token_endpoint, userinfo_endpoint, jwks_uri)
    - Fail requests if TLS validation fails
    - _Requirements: 13.17, 13.18_

  - [ ]* 9.11 Write property test for HTTPS enforcement
    - **Property 23: HTTPS Enforcement for OIDC**
    - **Validates: Requirements 13.17, 13.18**
    - Generate OIDC provider requests
    - Verify all use HTTPS
    - Verify TLS certificate validation enabled

- [ ] 10. Checkpoint - Phase 5 Complete
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 11. Phase 6: Testing & Documentation (Week 4)
  - [ ] 11.1 Write unit tests for configuration loading
    - Test configuration file loading
    - Test environment variable substitution
    - Test missing environment variable handling
    - Test configuration validation for all error cases
    - _Requirements: 12.1-12.11, 20.1-20.9_

  - [ ] 11.2 Write unit tests for OIDC provider
    - Test discovery document fetch and validation
    - Test JWKS fetch and caching
    - Test authorization flow initiation
    - Test callback handling with invalid state
    - Test token exchange failure
    - Test ID token validation failure
    - _Requirements: 1.1-1.7, 3.1-3.16_

  - [ ] 11.3 Write unit tests for Basic Auth provider
    - Test credential decoding
    - Test username lookup
    - Test password validation with bcrypt
    - Test invalid credentials handling
    - _Requirements: 5.1-5.9_

  - [ ] 11.4 Write unit tests for session management
    - Test session creation with cryptographic random ID
    - Test session validation with valid session
    - Test session validation with expired session
    - Test session validation with non-existent session
    - Test session deletion
    - Test in-memory store cleanup
    - _Requirements: 4.1-4.10_

  - [ ] 11.5 Write unit tests for ES credential mapper
    - Test OIDC claim extraction
    - Test wildcard pattern matching
    - Test mapping evaluation order
    - Test default ES user fallback
    - Test local user ES mapping
    - Test ES credentials retrieval
    - Test missing credentials error
    - _Requirements: 6.1-6.12_

  - [ ] 11.6 Write unit tests for transport adapters
    - Test header normalization (Traefik and Nginx)
    - Test ForwardAuth response format
    - Test standalone proxy request preservation
    - Test standalone proxy response preservation
    - Test upstream error handling
    - _Requirements: 7.1-7.8, 8.1-8.6, 9.1-9.14_

  - [ ] 11.7 Write unit tests for logout functionality
    - Test logout with valid session
    - Test logout without session
    - Test cookie clearing
    - Test OIDC provider logout redirect
    - _Requirements: 10.1-10.6_

  - [ ] 11.8 Write unit tests for health check endpoint
    - Test healthy response
    - Test unhealthy session store
    - Test unhealthy OIDC (discovery not loaded)
    - _Requirements: 11.1-11.6_

  - [ ] 11.9 Write integration test for complete OIDC flow
    - Setup mock OIDC provider
    - Test unauthenticated request triggers redirect
    - Test callback with valid code creates session
    - Test subsequent request with session cookie is authenticated
    - Test session expiration
    - Test logout clears session
    - _Requirements: 1.1-1.7, 3.1-3.16, 4.1-4.10, 10.1-10.6_

  - [ ] 11.10 Write integration test for Basic Auth flow
    - Setup Keyline with local users
    - Test valid credentials authenticate successfully
    - Test invalid username returns 401
    - Test invalid password returns 401
    - Test ES credential mapping
    - _Requirements: 5.1-5.9, 6.7-6.11_

  - [ ] 11.11 Write integration test for ForwardAuth mode
    - Setup Keyline in forwardAuth mode
    - Test with Traefik headers
    - Test with Nginx headers
    - Test authenticated request returns 200 with header
    - Test unauthenticated request returns 401 or 302
    - Test callback handling
    - _Requirements: 7.1-7.8, 8.1-8.6_

  - [ ] 11.12 Write integration test for standalone proxy mode
    - Setup Keyline in standalone mode with mock upstream
    - Test authenticated request is proxied
    - Test unauthenticated request triggers auth flow
    - Test request/response preservation
    - Test upstream error handling
    - Test WebSocket upgrade
    - _Requirements: 9.1-9.14_

  - [ ] 11.13 Write integration test for Redis session store
    - Setup Keyline with Redis (testcontainers)
    - Test session creation and retrieval
    - Test session expiration with TTL
    - Test state token storage with prefix
    - Test connection failure handling
    - _Requirements: 16.1-16.10_

  - [ ] 11.14 Write integration test for observability
    - Test Prometheus metrics are exposed
    - Test metrics are updated correctly
    - Test OpenTelemetry spans are created
    - Test structured logging includes context fields
    - _Requirements: 14.1-14.10, 18.1-18.10, 19.1-19.9_

  - [ ] 11.15 Create example configuration file
    - Create config/config.example.yaml with all configuration options
    - Include comments explaining each option
    - Show examples for OIDC, local users, Redis, standalone mode
    - Include environment variable placeholders
    - _Requirements: All configuration requirements_

  - [ ] 11.16 Write deployment documentation
    - Create docs/deployment.md
    - Document Kubernetes deployment with Helm
    - Document Docker deployment
    - Document Traefik integration
    - Document Nginx integration
    - Document Vault secret management
    - Include health check and readiness probe configuration
    - _Requirements: All deployment-related requirements_

  - [ ] 11.17 Write configuration reference documentation
    - Create docs/configuration.md
    - Document all configuration options with types and defaults
    - Document environment variable substitution
    - Document OIDC mapping patterns
    - Document session store options
    - Document deployment modes
    - _Requirements: 12.1-12.11, 20.1-20.12_

  - [ ] 11.18 Write troubleshooting guide
    - Create docs/troubleshooting.md
    - Document common configuration errors
    - Document OIDC provider connection issues
    - Document Redis connection issues
    - Document session expiration issues
    - Document logging and debugging tips
    - _Requirements: 14.1-14.10, 15.1-15.10_

  - [ ] 11.19 Write README with quick start guide
    - Create README.md with project overview
    - Document features and architecture
    - Provide quick start guide with Docker
    - Link to detailed documentation
    - Include example configuration
    - Document building from source
    - _Requirements: All requirements (overview)_

  - [ ] 11.20 Create Dockerfile with multi-stage build
    - Create Dockerfile with Go build stage
    - Use minimal base image (alpine or distroless)
    - Copy binary and example config
    - Set appropriate user and permissions
    - Expose port 9000
    - Document build and run commands
    - _Requirements: Deployment requirements_

  - [ ] 11.21 Create Makefile for common tasks
    - Add targets: build, test, lint, run, docker-build, docker-run
    - Add target for running property tests
    - Add target for running integration tests
    - Add target for generating test coverage report
    - _Requirements: Development workflow_

  - [ ] 11.22 Implement --validate-config flag
    - Add command-line flag to validate configuration without starting server
    - Load and validate configuration
    - Print validation results
    - Exit with code 0 on success, 1 on failure
    - _Requirements: 20.10, 20.11, 20.12_

- [ ] 12. Checkpoint - Phase 6 Complete
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 13. Final Integration and Verification
  - [ ] 13.1 Run complete test suite
    - Run all unit tests: `go test -v -race ./...`
    - Run all property tests: `go test -v -tags=property ./...`
    - Run all integration tests: `go test -v -tags=integration ./integration/...`
    - Verify all tests pass
    - Generate coverage report and verify >80% coverage

  - [ ] 13.2 Build and test Docker image
    - Build Docker image: `docker build -t keyline:latest .`
    - Run container with example config
    - Verify health check endpoint responds
    - Verify metrics endpoint responds
    - Test with real OIDC provider (optional)

  - [ ] 13.3 Verify all 23 correctness properties are tested
    - Review property test coverage
    - Ensure all 23 properties from design document are implemented
    - Verify minimum 100 iterations per property test
    - Document any properties that cannot be tested and why

  - [ ] 13.4 Manual testing with real services
    - Test OIDC flow with real OIDC provider (Okta, Auth0, etc.)
    - Test Basic Auth with real credentials
    - Test ForwardAuth mode with Traefik
    - Test standalone proxy mode with real upstream
    - Test Redis session store with real Redis
    - Test session persistence across restarts (with Redis)
    - Test logout functionality
    - Test graceful shutdown

  - [ ] 13.5 Security review
    - Verify no plaintext logging of sensitive values
    - Verify cryptographic randomness for tokens and session IDs
    - Verify bcrypt timing-safe password comparison
    - Verify cookie security attributes (HttpOnly, Secure, SameSite)
    - Verify HTTPS enforcement for OIDC provider
    - Verify TLS certificate validation
    - Verify PKCE implementation
    - Review error messages for information disclosure

  - [ ] 13.6 Performance testing
    - Test concurrent request handling
    - Test session store performance (memory and Redis)
    - Test OIDC provider request caching
    - Verify connection pooling works correctly
    - Test graceful degradation under load

- [ ] 14. Final Checkpoint - Implementation Complete
  - Ensure all tests pass, ask the user if questions arise.


## Notes

- Tasks marked with `*` are optional property-based tests that can be skipped for faster MVP, but are strongly recommended for production readiness
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation at phase boundaries
- Property tests validate universal correctness properties with minimum 100 iterations
- Unit tests validate specific examples, edge cases, and error conditions
- Integration tests validate end-to-end flows with real or mocked dependencies
- All 23 correctness properties from the design document must be implemented as property tests
- Reuse elastauth components where possible: Echo, Viper, Redis client, OpenTelemetry, structured logging
- Follow Go best practices: error handling, context propagation, graceful shutdown
- Security is paramount: crypto/rand, bcrypt, secure cookies, HTTPS, TLS validation
- Observability is built-in: structured logging, Prometheus metrics, OpenTelemetry tracing
- Configuration validation happens at startup with clear error messages
- Manual testing with real services is required before production deployment

## Implementation Timeline

- **Week 1**: Phase 1 (Core Infrastructure) + Phase 2 start (OIDC Discovery)
- **Week 2**: Phase 2 complete (OIDC Authentication) + Phase 3 (Basic Authentication)
- **Week 3**: Phase 4 (Transport Adapters) + Phase 5 (Observability)
- **Week 4**: Phase 6 (Testing & Documentation) + Final Integration

## Success Criteria

- All 23 correctness properties implemented as property tests
- Unit test coverage >80%
- All integration tests pass
- Docker image builds and runs successfully
- Manual testing with real OIDC provider succeeds
- Documentation complete and accurate
- Security review passes
- Performance testing shows acceptable results
- Ready for production deployment
