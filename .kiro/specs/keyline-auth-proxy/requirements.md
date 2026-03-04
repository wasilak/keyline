# Requirements Document

## Introduction

Keyline is an authentication proxy service that replaces the existing Authelia + elastauth stack with a single unified service. It provides dual authentication modes (OIDC and Basic Auth) simultaneously, supports multiple deployment modes (forwardAuth, auth_request, standalone proxy), and automatically injects Elasticsearch credentials into authenticated requests. The service is designed for Kubernetes environments with Vault-managed secrets and supports both browser-based interactive authentication and programmatic API access.

## Glossary

- **Keyline**: The authentication proxy service being specified
- **OIDC_Provider**: The OpenID Connect identity provider (e.g., Okta)
- **Discovery_Document**: The JSON document at `{issuer}/.well-known/openid-configuration` containing OIDC endpoint URLs
- **State_Token**: A cryptographically random 32-byte token used for CSRF protection in OIDC flows
- **Session_Store**: The storage backend for user sessions (in-memory or Redis)
- **Reverse_Proxy**: The upstream HTTP proxy (Traefik or Nginx) in forwardAuth/auth_request modes
- **Protected_Service**: The upstream service being protected (e.g., Kibana, Elasticsearch)
- **ES_Credentials**: Elasticsearch username and password encoded as Basic authentication
- **Local_User**: A user defined in Keyline configuration for programmatic access
- **OIDC_User**: A user authenticated via the OpenID Connect flow
- **Session_Cookie**: An HttpOnly, Secure cookie containing only the session identifier
- **Callback_Endpoint**: The `/auth/callback` endpoint that receives OIDC authorization codes
- **JWKS**: JSON Web Key Set containing public keys for ID token signature validation
- **ID_Token**: A JWT token returned by the OIDC provider containing user identity claims
- **Authorization_Code**: A single-use code returned by OIDC provider after user authentication
- **PKCE**: Proof Key for Code Exchange, an OAuth 2.0 security extension
- **Claim**: A key-value pair in an ID token (e.g., email, sub, name)


## Requirements

### Requirement 1: OIDC Auto-Discovery

**User Story:** As a system administrator, I want Keyline to automatically discover OIDC endpoints from the provider, so that I don't need to manually configure authorization, token, and JWKS URLs.

#### Acceptance Criteria

1. WHEN Keyline starts with OIDC enabled, THE Keyline SHALL fetch the Discovery_Document from `{issuer_url}/.well-known/openid-configuration`
2. IF the Discovery_Document fetch fails, THEN THE Keyline SHALL log an error and refuse to start
3. WHEN the Discovery_Document is successfully fetched, THE Keyline SHALL extract and store the authorization_endpoint, token_endpoint, userinfo_endpoint, jwks_uri, and issuer values
4. IF the issuer value in the Discovery_Document does not match the configured issuer_url, THEN THE Keyline SHALL log an error and refuse to start
5. THE Keyline SHALL cache the Discovery_Document in memory after successful fetch
6. THE Keyline SHALL refresh the JWKS from jwks_uri every 24 hours to handle key rotation
7. IF the JWKS refresh fails, THEN THE Keyline SHALL log a warning and continue using the cached JWKS

### Requirement 2: Dual Authentication Mode Selection

**User Story:** As a user, I want Keyline to automatically select the appropriate authentication method based on my request, so that both browser users and programmatic clients can access protected services without configuration changes.

#### Acceptance Criteria

1. WHEN a request contains a valid Session_Cookie, THE Keyline SHALL authenticate using the session and skip other authentication methods
2. WHEN a request contains no valid Session_Cookie and includes an Authorization header with Basic scheme, THE Keyline SHALL attempt Local_User authentication
3. WHEN a request contains no valid Session_Cookie and no Authorization header, THE Keyline SHALL initiate the OIDC authentication flow
4. WHEN a request path is `/auth/callback`, THE Keyline SHALL process it as an OIDC callback regardless of other headers
5. THE Keyline SHALL support both authentication modes simultaneously without requiring mode selection configuration


### Requirement 3: OIDC Authorization Flow with PKCE

**User Story:** As a browser user, I want to authenticate using my OIDC provider credentials, so that I can access protected services with single sign-on.

#### Acceptance Criteria

1. WHEN an unauthenticated request arrives, THE Keyline SHALL generate a cryptographically random 32-byte State_Token
2. WHEN generating the State_Token, THE Keyline SHALL store it in the Session_Store with the original request URL, a 5-minute TTL, and mark it as unused
3. WHEN redirecting to the OIDC_Provider, THE Keyline SHALL include the State_Token in the state parameter and generate PKCE code_challenge and code_verifier values
4. THE Keyline SHALL redirect the user to the authorization_endpoint with parameters: client_id, redirect_uri, response_type=code, scope, state, code_challenge, and code_challenge_method=S256
5. WHEN the Callback_Endpoint receives a request, THE Keyline SHALL validate that the state parameter matches an unused State_Token in the Session_Store
6. IF the state parameter is invalid, expired, or already used, THEN THE Keyline SHALL return HTTP 400 with error message "Invalid or expired state token"
7. WHEN the state is valid, THE Keyline SHALL mark the State_Token as used and delete it from the Session_Store
8. WHEN the state is valid, THE Keyline SHALL exchange the Authorization_Code for tokens by making a POST request to the token_endpoint with client_id, client_secret, code, redirect_uri, grant_type=authorization_code, and code_verifier
9. IF the token exchange fails, THEN THE Keyline SHALL return HTTP 401 with error message "Token exchange failed"
10. WHEN tokens are successfully received, THE Keyline SHALL validate the ID_Token signature using the JWKS public keys
11. IF the ID_Token signature is invalid, THEN THE Keyline SHALL return HTTP 401 with error message "Invalid token signature"
12. WHEN the signature is valid, THE Keyline SHALL validate that the iss claim matches the configured issuer_url and the aud claim matches the configured client_id
13. IF the iss or aud claims are invalid, THEN THE Keyline SHALL return HTTP 401 with error message "Invalid token claims"
14. WHEN the ID_Token is fully validated, THE Keyline SHALL extract user identity claims and create a session
15. WHEN the session is created, THE Keyline SHALL redirect the user to the original request URL stored with the State_Token


### Requirement 4: Session Management

**User Story:** As a browser user, I want my authentication to persist across requests, so that I don't need to re-authenticate for every page load.

#### Acceptance Criteria

1. WHEN an OIDC_User is successfully authenticated, THE Keyline SHALL generate a cryptographically random session identifier
2. WHEN creating a session, THE Keyline SHALL store the session identifier, user identity, mapped ES user, creation timestamp, and expiration timestamp in the Session_Store
3. THE Keyline SHALL set the session expiration time based on the configured session_ttl value
4. WHEN creating a session, THE Keyline SHALL return a Session_Cookie with attributes: HttpOnly=true, Secure=true, SameSite=Lax, and value=session_identifier
5. WHEN a request contains a Session_Cookie, THE Keyline SHALL retrieve the session from the Session_Store using the cookie value
6. IF the session does not exist in the Session_Store, THEN THE Keyline SHALL treat the request as unauthenticated
7. IF the session exists but is expired, THEN THE Keyline SHALL delete the session from the Session_Store and treat the request as unauthenticated
8. WHEN a valid non-expired session is found, THE Keyline SHALL use the stored user identity and ES user for the request
9. WHERE the Session_Store is configured as Redis, THE Keyline SHALL connect to Redis at startup and use it for all session operations
10. WHERE the Session_Store is configured as in-memory, THE Keyline SHALL use an in-memory map with mutex protection for all session operations

### Requirement 5: Local User Authentication

**User Story:** As a CI/CD pipeline or monitoring tool, I want to authenticate using Basic Auth credentials, so that I can access protected services programmatically without browser interaction.

#### Acceptance Criteria

1. WHEN a request contains an Authorization header with Basic scheme, THE Keyline SHALL decode the base64-encoded credentials
2. IF the credentials cannot be decoded, THEN THE Keyline SHALL return HTTP 401 with WWW-Authenticate header
3. WHEN credentials are decoded, THE Keyline SHALL extract the username and password
4. THE Keyline SHALL search the configured Local_User list for a matching username
5. IF no matching username is found, THEN THE Keyline SHALL return HTTP 401 with WWW-Authenticate header
6. WHEN a matching username is found, THE Keyline SHALL validate the password using bcrypt comparison against the stored password_bcrypt value
7. IF the password validation fails, THEN THE Keyline SHALL return HTTP 401 with WWW-Authenticate header
8. WHEN the password is valid, THE Keyline SHALL retrieve the mapped es_user value for the Local_User
9. WHEN authentication succeeds, THE Keyline SHALL proceed with the request using the mapped ES user without creating a session


### Requirement 6: Elasticsearch Credential Mapping and Injection

**User Story:** As a system administrator, I want Keyline to automatically map authenticated users to Elasticsearch credentials, so that each user accesses Elasticsearch with appropriate permissions.

#### Acceptance Criteria

1. WHEN an OIDC_User is authenticated, THE Keyline SHALL extract claims from the ID_Token according to configured oidc_mappings
2. THE Keyline SHALL evaluate each oidc_mapping in order until a match is found
3. WHEN evaluating an oidc_mapping, THE Keyline SHALL extract the claim value specified by the claim field
4. THE Keyline SHALL compare the claim value against the pattern using wildcard matching
5. WHEN a claim value matches a pattern, THE Keyline SHALL use the corresponding es_user value for that mapping
6. IF no oidc_mapping matches, THEN THE Keyline SHALL use the configured default_es_user value
7. WHEN a Local_User is authenticated, THE Keyline SHALL use the es_user value configured for that Local_User
8. WHEN an ES user is determined, THE Keyline SHALL retrieve the corresponding ES_Credentials from the configured elasticsearch.users list
9. IF no matching ES_Credentials are found, THEN THE Keyline SHALL log an error and return HTTP 500
10. WHEN ES_Credentials are found, THE Keyline SHALL encode them as Basic authentication (base64 of username:password)
11. THE Keyline SHALL add the X-Es-Authorization header with value "Basic {encoded_credentials}" to the response or proxied request
12. THE Keyline SHALL never log or expose ES_Credentials in plaintext

### Requirement 7: ForwardAuth Mode (Traefik)

**User Story:** As a system administrator using Traefik, I want Keyline to work as a forwardAuth middleware, so that I can protect multiple services with a single authentication layer.

#### Acceptance Criteria

1. WHERE Keyline is configured with mode=forward_auth, THE Keyline SHALL read the X-Forwarded-Uri header to determine the original request path
2. WHERE Keyline is configured with mode=forward_auth, THE Keyline SHALL read the X-Forwarded-Method header to determine the original request method
3. WHERE Keyline is configured with mode=forward_auth, THE Keyline SHALL read the X-Forwarded-Host header to determine the original request host
4. WHEN authentication succeeds in forwardAuth mode, THE Keyline SHALL return HTTP 200 with the X-Es-Authorization header
5. WHEN authentication fails in forwardAuth mode, THE Keyline SHALL return HTTP 401 for Basic Auth failures or HTTP 302 for OIDC redirects
6. WHEN the X-Forwarded-Uri indicates the Callback_Endpoint path, THE Keyline SHALL process the OIDC callback and return HTTP 302 with Set-Cookie header
7. THE Keyline SHALL not proxy requests to upstream services in forwardAuth mode
8. THE Keyline SHALL preserve all Cookie headers from the original request when validating sessions


### Requirement 8: Auth_Request Mode (Nginx)

**User Story:** As a system administrator using Nginx, I want Keyline to work as an auth_request backend, so that I can protect services with Nginx's authentication mechanism.

#### Acceptance Criteria

1. WHERE Keyline is configured with mode=forward_auth, THE Keyline SHALL accept X-Original-URI as an alternative to X-Forwarded-Uri
2. WHERE Keyline is configured with mode=forward_auth, THE Keyline SHALL accept X-Original-Method as an alternative to X-Forwarded-Method
3. WHERE Keyline is configured with mode=forward_auth, THE Keyline SHALL accept X-Original-Host as an alternative to X-Forwarded-Host
4. THE Keyline SHALL normalize Traefik and Nginx header conventions into a single internal representation
5. WHEN processing requests with Nginx headers, THE Keyline SHALL apply the same authentication logic as Traefik requests
6. WHEN processing requests with Nginx headers, THE Keyline SHALL return the same response codes and headers as Traefik requests

### Requirement 9: Standalone Proxy Mode

**User Story:** As a system administrator without a reverse proxy, I want Keyline to act as a standalone proxy, so that I can protect services without deploying Traefik or Nginx.

#### Acceptance Criteria

1. WHERE Keyline is configured with mode=standalone, THE Keyline SHALL proxy all authenticated requests to the configured upstream.url
2. WHERE Keyline is configured with mode=standalone, THE Keyline SHALL handle the Callback_Endpoint internally without proxying
3. WHERE Keyline is configured with mode=standalone, THE Keyline SHALL handle the /auth/logout endpoint internally without proxying
4. WHERE Keyline is configured with mode=standalone, THE Keyline SHALL handle the /healthz endpoint internally without proxying
5. WHEN proxying a request in standalone mode, THE Keyline SHALL add the X-Es-Authorization header before forwarding
6. WHEN proxying a request in standalone mode, THE Keyline SHALL preserve all request headers except hop-by-hop headers
7. WHEN proxying a request in standalone mode, THE Keyline SHALL preserve the request method, path, query parameters, and body
8. WHEN proxying a request in standalone mode, THE Keyline SHALL stream the response body from the Protected_Service to the client
9. WHEN proxying a request in standalone mode, THE Keyline SHALL preserve all response headers except hop-by-hop headers
10. WHEN proxying a request in standalone mode, THE Keyline SHALL preserve the response status code
11. IF the upstream connection fails, THEN THE Keyline SHALL return HTTP 502 with error message "Bad Gateway"
12. IF the upstream connection times out, THEN THE Keyline SHALL return HTTP 504 with error message "Gateway Timeout"
13. WHEN a WebSocket upgrade is requested in standalone mode, THE Keyline SHALL forward the upgrade headers and establish a bidirectional connection
14. THE Keyline SHALL use the configured upstream.timeout value for all upstream requests


### Requirement 10: Logout Functionality

**User Story:** As a browser user, I want to log out of my session, so that I can end my authenticated session and clear my credentials.

#### Acceptance Criteria

1. WHEN a request is made to /auth/logout, THE Keyline SHALL extract the session identifier from the Session_Cookie
2. WHEN a session identifier is found, THE Keyline SHALL delete the session from the Session_Store
3. WHEN deleting a session, THE Keyline SHALL return a Set-Cookie header with the Session_Cookie name, empty value, and Max-Age=0
4. WHEN a session is deleted, THE Keyline SHALL redirect to the OIDC_Provider end_session_endpoint if available in the Discovery_Document
5. IF the end_session_endpoint is not available, THEN THE Keyline SHALL redirect to a configured logout_redirect_url or return HTTP 200 with message "Logged out"
6. WHEN no session identifier is found in the logout request, THE Keyline SHALL return HTTP 200 with message "No active session"

### Requirement 11: Health Check Endpoint

**User Story:** As a Kubernetes operator, I want a health check endpoint, so that I can monitor Keyline availability and readiness.

#### Acceptance Criteria

1. THE Keyline SHALL expose a /healthz endpoint that requires no authentication
2. WHEN /healthz is requested, THE Keyline SHALL return HTTP 200 with JSON body containing status and version fields
3. WHEN /healthz is requested, THE Keyline SHALL verify that the Session_Store is accessible
4. IF the Session_Store is not accessible, THEN THE Keyline SHALL return HTTP 503 with JSON body containing status="unhealthy" and an error message
5. WHERE OIDC is enabled, WHEN /healthz is requested, THE Keyline SHALL verify that the Discovery_Document was successfully loaded
6. WHERE OIDC is enabled, IF the Discovery_Document was not loaded, THEN THE Keyline SHALL return HTTP 503 with status="unhealthy"

### Requirement 12: Configuration Loading

**User Story:** As a system administrator, I want Keyline to load configuration from files and environment variables, so that I can manage secrets securely with Vault.

#### Acceptance Criteria

1. WHEN Keyline starts, THE Keyline SHALL load configuration from a YAML file specified by the --config flag or CONFIG_FILE environment variable
2. THE Keyline SHALL support environment variable substitution in configuration values using ${VAR_NAME} syntax
3. WHEN a configuration value contains ${VAR_NAME}, THE Keyline SHALL replace it with the value of the VAR_NAME environment variable
4. IF an environment variable referenced in configuration is not set, THEN THE Keyline SHALL log an error and refuse to start
5. THE Keyline SHALL validate all required configuration fields at startup
6. IF any required configuration field is missing, THEN THE Keyline SHALL log an error specifying the missing field and refuse to start
7. THE Keyline SHALL validate that session_secret is at least 32 bytes when decoded
8. THE Keyline SHALL validate that all password_bcrypt values are valid bcrypt hashes
9. THE Keyline SHALL validate that all ES_Credentials are valid base64-encoded strings
10. THE Keyline SHALL validate that redirect_url is a valid HTTPS URL
11. IF any configuration validation fails, THEN THE Keyline SHALL log a descriptive error and refuse to start


### Requirement 13: Security Controls

**User Story:** As a security engineer, I want Keyline to implement security best practices, so that authentication is protected against common attacks.

#### Acceptance Criteria

1. THE Keyline SHALL generate all State_Token values using a cryptographically secure random number generator
2. THE Keyline SHALL generate all session identifiers using a cryptographically secure random number generator
3. THE Keyline SHALL validate all Local_User passwords using bcrypt comparison with timing-safe equality
4. THE Keyline SHALL never log passwords, ES_Credentials, session identifiers, or State_Token values in plaintext
5. THE Keyline SHALL set Session_Cookie with HttpOnly=true to prevent JavaScript access
6. THE Keyline SHALL set Session_Cookie with Secure=true to require HTTPS transmission
7. THE Keyline SHALL set Session_Cookie with SameSite=Lax to prevent CSRF attacks
8. THE Keyline SHALL store only the session identifier in the Session_Cookie, never user credentials or ES_Credentials
9. THE Keyline SHALL validate ID_Token signatures using public keys from JWKS before trusting any claims
10. THE Keyline SHALL validate that the iss claim in the ID_Token matches the configured issuer_url
11. THE Keyline SHALL validate that the aud claim in the ID_Token matches the configured client_id
12. THE Keyline SHALL validate that the exp claim in the ID_Token is in the future
13. IF any ID_Token validation fails, THEN THE Keyline SHALL reject the token and return HTTP 401
14. THE Keyline SHALL mark State_Token values as single-use and reject any reuse attempts
15. THE Keyline SHALL delete State_Token values after successful use or after TTL expiration
16. THE Keyline SHALL use PKCE (code_challenge and code_verifier) in the OIDC authorization flow
17. THE Keyline SHALL make all requests to the OIDC_Provider token_endpoint using HTTPS
18. THE Keyline SHALL validate TLS certificates when connecting to the OIDC_Provider

### Requirement 14: Logging and Observability

**User Story:** As a system operator, I want structured logging with context, so that I can troubleshoot authentication issues and monitor system behavior.

#### Acceptance Criteria

1. THE Keyline SHALL use structured logging with fields for timestamp, level, message, and context
2. WHEN logging authentication events, THE Keyline SHALL include fields for username, authentication_method, source_ip, and result
3. WHEN logging OIDC flow events, THE Keyline SHALL include fields for state_token_id (not value), callback_result, and error_details
4. WHEN logging session events, THE Keyline SHALL include fields for session_id (hashed), action (created/validated/expired/deleted), and username
5. WHEN logging configuration loading, THE Keyline SHALL include fields for config_file, oidc_enabled, local_users_count, and mode
6. WHEN logging upstream proxy errors, THE Keyline SHALL include fields for upstream_url, error_type, and response_time
7. THE Keyline SHALL log at INFO level for successful authentication, session creation, and startup events
8. THE Keyline SHALL log at WARN level for failed authentication attempts, expired sessions, and JWKS refresh failures
9. THE Keyline SHALL log at ERROR level for configuration errors, OIDC provider connection failures, and upstream proxy failures
10. THE Keyline SHALL never log sensitive values including passwords, tokens, credentials, or full session identifiers


### Requirement 15: Error Handling and Resilience

**User Story:** As a system operator, I want Keyline to handle errors gracefully, so that temporary failures don't cause service outages.

#### Acceptance Criteria

1. WHEN the Session_Store connection fails during a request, THE Keyline SHALL log an error and return HTTP 503 with message "Service temporarily unavailable"
2. WHEN the OIDC_Provider token_endpoint is unreachable, THE Keyline SHALL log an error and return HTTP 502 with message "Authentication provider unavailable"
3. WHEN the OIDC_Provider token_endpoint returns an error response, THE Keyline SHALL log the error details and return HTTP 401 with message "Authentication failed"
4. WHEN the JWKS fetch fails during startup, THE Keyline SHALL retry up to 3 times with exponential backoff before refusing to start
5. WHEN the JWKS refresh fails during operation, THE Keyline SHALL log a warning and continue using the cached JWKS
6. WHEN the Discovery_Document fetch fails during startup, THE Keyline SHALL retry up to 3 times with exponential backoff before refusing to start
7. IF the upstream Protected_Service is unreachable in standalone mode, THEN THE Keyline SHALL return HTTP 502 after the configured timeout
8. WHEN an unexpected error occurs during request processing, THE Keyline SHALL log the full error with stack trace and return HTTP 500 with message "Internal server error"
9. THE Keyline SHALL implement graceful shutdown that waits for in-flight requests to complete before terminating
10. WHEN receiving SIGTERM or SIGINT, THE Keyline SHALL stop accepting new requests and wait up to 30 seconds for existing requests to complete

### Requirement 16: Redis Session Store Integration

**User Story:** As a system administrator deploying multiple Keyline instances, I want to use Redis for session storage, so that sessions work across all instances.

#### Acceptance Criteria

1. WHERE session.store is configured as "redis", THE Keyline SHALL connect to Redis using the configured session.redis_url at startup
2. IF the Redis connection fails at startup, THEN THE Keyline SHALL log an error and refuse to start
3. WHEN storing a session in Redis, THE Keyline SHALL use the session identifier as the key and serialize the session data as JSON
4. WHEN storing a session in Redis, THE Keyline SHALL set the Redis key TTL to match the session expiration time
5. WHEN retrieving a session from Redis, THE Keyline SHALL deserialize the JSON data into a session object
6. IF a Redis operation fails during request processing, THEN THE Keyline SHALL log an error and return HTTP 503
7. WHEN storing a State_Token in Redis, THE Keyline SHALL use a key prefix "state:" to distinguish from session keys
8. WHEN storing a State_Token in Redis, THE Keyline SHALL set the Redis key TTL to 5 minutes
9. THE Keyline SHALL implement connection pooling for Redis with a minimum of 5 and maximum of 20 connections
10. THE Keyline SHALL implement automatic Redis reconnection with exponential backoff if the connection is lost


### Requirement 17: Performance and Resource Management

**User Story:** As a system operator, I want Keyline to perform efficiently under load, so that authentication doesn't become a bottleneck.

#### Acceptance Criteria

1. THE Keyline SHALL cache the Discovery_Document in memory after the initial fetch
2. THE Keyline SHALL cache the JWKS in memory after the initial fetch
3. THE Keyline SHALL cache parsed JWKS public keys to avoid repeated parsing
4. WHERE the Session_Store is in-memory, THE Keyline SHALL implement automatic cleanup of expired sessions every 5 minutes
5. THE Keyline SHALL limit the maximum number of concurrent requests to 1000
6. IF the concurrent request limit is reached, THEN THE Keyline SHALL return HTTP 503 with message "Server overloaded"
7. THE Keyline SHALL implement request timeouts of 30 seconds for all OIDC_Provider requests
8. THE Keyline SHALL implement connection pooling for OIDC_Provider requests with keep-alive enabled
9. THE Keyline SHALL limit the maximum size of request bodies to 1MB
10. IF a request body exceeds 1MB, THEN THE Keyline SHALL return HTTP 413 with message "Request too large"

### Requirement 18: Metrics and Monitoring

**User Story:** As a system operator, I want Keyline to expose metrics, so that I can monitor authentication performance and detect issues.

#### Acceptance Criteria

1. THE Keyline SHALL expose a /metrics endpoint in Prometheus format
2. THE Keyline SHALL track a counter metric for total authentication attempts with labels: method (oidc/basic), result (success/failure)
3. THE Keyline SHALL track a histogram metric for authentication request duration with labels: method (oidc/basic)
4. THE Keyline SHALL track a gauge metric for active sessions
5. THE Keyline SHALL track a counter metric for session operations with labels: operation (created/validated/expired/deleted)
6. THE Keyline SHALL track a counter metric for OIDC provider requests with labels: endpoint (token/userinfo/jwks), result (success/failure)
7. THE Keyline SHALL track a histogram metric for upstream proxy request duration in standalone mode
8. THE Keyline SHALL track a gauge metric for current concurrent requests
9. THE Keyline SHALL track a counter metric for errors with labels: error_type
10. THE /metrics endpoint SHALL require no authentication


### Requirement 19: OpenTelemetry Integration

**User Story:** As a system operator using distributed tracing, I want Keyline to emit OpenTelemetry traces, so that I can trace requests across services.

#### Acceptance Criteria

1. WHERE opentelemetry is enabled in configuration, THE Keyline SHALL initialize an OpenTelemetry tracer at startup
2. WHERE opentelemetry is enabled, THE Keyline SHALL create a span for each incoming request with name "keyline.request"
3. WHERE opentelemetry is enabled, THE Keyline SHALL add span attributes for: http.method, http.url, http.status_code, auth.method, auth.result
4. WHERE opentelemetry is enabled, THE Keyline SHALL create child spans for OIDC provider requests with name "keyline.oidc.{endpoint}"
5. WHERE opentelemetry is enabled, THE Keyline SHALL create child spans for Session_Store operations with name "keyline.session.{operation}"
6. WHERE opentelemetry is enabled, THE Keyline SHALL create child spans for upstream proxy requests with name "keyline.proxy.request"
7. WHERE opentelemetry is enabled, THE Keyline SHALL propagate trace context using W3C Trace Context headers
8. WHERE opentelemetry is enabled, THE Keyline SHALL export traces to the configured OTLP endpoint
9. IF OpenTelemetry initialization fails, THEN THE Keyline SHALL log a warning and continue without tracing

### Requirement 20: Configuration Validation and Documentation

**User Story:** As a system administrator, I want clear configuration validation errors, so that I can quickly fix configuration issues.

#### Acceptance Criteria

1. WHEN Keyline starts with invalid configuration, THE Keyline SHALL print a clear error message identifying the specific configuration problem
2. IF oidc.enabled is true and oidc.issuer_url is missing, THEN THE Keyline SHALL print "Configuration error: oidc.issuer_url is required when OIDC is enabled"
3. IF oidc.enabled is true and oidc.client_id is missing, THEN THE Keyline SHALL print "Configuration error: oidc.client_id is required when OIDC is enabled"
4. IF oidc.enabled is true and oidc.client_secret is missing, THEN THE Keyline SHALL print "Configuration error: oidc.client_secret is required when OIDC is enabled"
5. IF oidc.enabled is true and oidc.redirect_url is missing, THEN THE Keyline SHALL print "Configuration error: oidc.redirect_url is required when OIDC is enabled"
6. IF local_users.enabled is true and local_users.users is empty, THEN THE Keyline SHALL print "Configuration error: at least one local user must be configured when local users are enabled"
7. IF mode is "standalone" and upstream.url is missing, THEN THE Keyline SHALL print "Configuration error: upstream.url is required in standalone mode"
8. IF both oidc.enabled and local_users.enabled are false, THEN THE Keyline SHALL print "Configuration error: at least one authentication method must be enabled"
9. IF elasticsearch.users is empty, THEN THE Keyline SHALL print "Configuration error: at least one Elasticsearch user must be configured"
10. THE Keyline SHALL provide a --validate-config flag that validates configuration and exits without starting the server
11. WHEN --validate-config is used, THE Keyline SHALL print "Configuration valid" and exit with code 0 if validation succeeds
12. WHEN --validate-config is used, THE Keyline SHALL print validation errors and exit with code 1 if validation fails

