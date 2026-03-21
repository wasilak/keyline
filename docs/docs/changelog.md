# Keyline Changelog

All notable changes to Keyline will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial Docusaurus documentation site setup

### Changed
- N/A

### Deprecated
- N/A

### Removed
- N/A

### Fixed
- N/A

### Security
- N/A

---

## [1.0.0] - 2026-03-21

### Added
- Initial release of Keyline authentication proxy
- OIDC authentication with PKCE support
- Local users with Basic Auth
- Dynamic Elasticsearch user management
- Role-based access control with group mappings
- Redis and in-memory session storage
- ForwardAuth mode for Traefik/Nginx
- Standalone proxy mode
- OpenTelemetry tracing integration
- Prometheus metrics
- Structured logging with loggergo
- Docker Compose examples for all deployment scenarios
- Comprehensive test suite (unit, integration, property-based)

### Security
- Cryptographically secure session IDs and state tokens
- Secure cookies with HttpOnly, Secure, SameSite attributes
- Bcrypt password hashing
- AES-256-GCM encryption for cached credentials
- PKCE for OIDC authorization code flow
- TLS enforcement for OIDC provider connections

---

## Version History

- [1.0.0] - 2026-03-21 - Initial release

[Unreleased]: https://github.com/wasilak/keyline/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/wasilak/keyline/releases/tag/v1.0.0
