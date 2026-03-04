---
inclusion: always
---

# Context7 MCP Tools for Research and Development

## Mandatory Context7 Usage

During research and development phases of Cerebro rewrite, you MUST use Context7 MCP tools to gather up-to-date information about libraries, frameworks, and technologies.

## When to Use Context7 Tools

### Research Phase Requirements
- **ALWAYS** use Context7 when researching new libraries or frameworks
- **ALWAYS** use Context7 when investigating integration approaches
- **ALWAYS** use Context7 when looking up current best practices
- **ALWAYS** use Context7 when checking for updated APIs or configuration patterns

### Specific Use Cases for Cerebro Rewrite

1. **Rust Backend Research**
   - Research Axum web framework patterns and best practices
   - Investigate Elasticsearch Rust client (elasticsearch-rs)
   - Look up async/await patterns with Tokio
   - Research rust-embed for asset embedding
   - Investigate authentication libraries (OIDC, JWT)

2. **Frontend Framework Research**
   - Research Mantine UI components and patterns
   - Investigate React 18+ features and hooks
   - Look up Vite build configuration
   - Research TanStack Query for server state
   - Investigate Zustand for client state management

3. **Configuration and Deployment Research**
   - Research config-rs for Rust configuration
   - Investigate Docker multi-stage builds for Rust
   - Look up GitHub Actions for Rust + Node.js projects
   - Research cross-compilation for multiple platforms

## Context7 Tool Usage Pattern

### Step 1: Resolve Library ID
```
Use mcp_context7_resolve_library_id to find the correct library
- Provide clear library name (e.g., "axum", "mantine", "elasticsearch-rs")
- Include context about your use case
- Select the most relevant result based on reputation and snippets
```

### Step 2: Query Documentation
```
Use mcp_context7_query_docs with the resolved library ID
- Ask specific questions about implementation
- Request configuration examples
- Look for best practices and common patterns
```

### Research Quality Standards

**Comprehensive Research**:
- Don't rely on outdated knowledge - always check current documentation
- Use Context7 to verify API changes and new features
- Research multiple approaches before making implementation decisions

**Documentation Integration**:
- Include Context7 findings in design decisions
- Reference current documentation in implementation comments
- Update Cerebro documentation based on latest best practices

## Cerebro-Specific Research Areas

### Priority Research Topics

1. **Rust Web Framework (Axum)**
   - Current Axum patterns and middleware
   - Router configuration and state management
   - Error handling and response types
   - WebSocket support (if needed for real-time updates)

2. **Elasticsearch Client**
   - elasticsearch-rs API and usage patterns
   - Connection pooling and failover
   - Request/response handling
   - Version compatibility (ES 7, 8, 9, OpenSearch)

3. **Frontend UI Framework (Mantine)**
   - Mantine v7 components and theming
   - Dark/light mode implementation
   - Form handling and validation
   - Table components for cluster/index display

4. **Authentication & Security**
   - OIDC implementation in Rust
   - JWT token validation
   - Session management patterns
   - Secure cookie handling

5. **Build and Deployment**
   - rust-embed for asset embedding
   - Cross-compilation for multiple platforms
   - Docker multi-stage builds
   - GitHub Actions for Rust projects

### Research Documentation

**Document Findings**:
- Include Context7 research results in design documents
- Reference specific library versions and features
- Note any breaking changes or migration requirements

**Share Knowledge**:
- Update steering rules based on research findings
- Include research-based recommendations in code comments
- Document decision rationale based on current best practices

## Integration with Development Workflow

### Before Implementation
1. **Research Phase**: Use Context7 to understand current best practices
2. **Design Phase**: Incorporate research findings into design decisions
3. **Implementation Phase**: Reference current documentation during coding

### During Implementation
- Use Context7 to resolve specific implementation questions
- Look up current API patterns and examples
- Verify configuration options and parameters

### After Implementation
- Use Context7 to research testing best practices
- Look up deployment and monitoring patterns
- Research performance optimization techniques

## Quality Assurance

### Research Validation
- Cross-reference multiple sources when possible
- Verify information with official documentation
- Check for recent updates or breaking changes

### Implementation Validation
- Test implementations against current library versions
- Validate configuration against current documentation
- Ensure compatibility with latest best practices

## Example Research Queries

### Rust Backend
```
"How to implement middleware in Axum 0.7"
"Elasticsearch Rust client connection pooling"
"Tokio async runtime best practices"
"rust-embed serve static files"
"OIDC authentication in Rust"
```

### TypeScript Frontend
```
"Mantine UI v7 dark mode implementation"
"React Router v6 authentication guards"
"TanStack Query error handling"
"Vite environment variables"
"Monaco Editor React integration"
```

### Build & Deploy
```
"Docker multi-stage build for Rust and Node.js"
"GitHub Actions cross-compile Rust"
"Rust binary size optimization"
"Embedding frontend assets in Rust binary"
```

## Remember

**Stay Current**: Technology moves fast. Context7 helps ensure Cerebro uses current best practices and avoids deprecated patterns.

**Research First**: Before implementing any new feature or integration, research the current state of the technology using Context7.

**Document Decisions**: Include research findings in design documents and code comments to help future maintainers understand the rationale.

The goal is to build Cerebro rewrite using the most current and appropriate technologies, patterns, and best practices available.