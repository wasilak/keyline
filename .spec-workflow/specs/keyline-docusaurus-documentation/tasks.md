# Keyline Docusaurus Documentation - Tasks

## Overview

Break down the documentation implementation into atomic, actionable tasks. Each task includes a _Prompt field with specific guidance for implementation.

**Spec**: keyline-docusaurus-documentation  
**Total Tasks**: 16  
**Estimated Effort**: 3-4 days

---

## Task List

- [x] 1. Initialize Docusaurus Project

**Description**: Set up Docusaurus v3 project structure with all dependencies

**Files**:
- `docs/package.json`
- `docs/.gitignore`
- `docs/README.md`

**Requirements Reference**: FR1, NFR2

**_Prompt**:
```
Implement the task for spec keyline-docusaurus-documentation, first run spec-workflow-guide to get the workflow guide then implement the task:

**Role**: Frontend Developer specializing in Docusaurus

**Task**: Initialize Docusaurus v3 project in docs/ directory

**Context**:
- Reference: Secan docs project structure
- Use Docusaurus v3 latest stable version
- Follow GitHub Pages deployment pattern

**Restrictions**:
- Do NOT configure docusaurus.config.js yet (Task 2)
- Do NOT create content directories yet (Task 3)
- Only initialize project skeleton

**Leverage**:
- Secan docs/package.json as reference
- Docusaurus v3 documentation

**Success Criteria**:
1. docs/package.json created with Docusaurus v3 dependencies
2. docs/.gitignore created (node_modules, build, .docusaurus)
3. docs/README.md with basic setup instructions
4. Run `npm install` in docs/ succeeds without errors

**Instructions**:
1. Mark task as in-progress [-] in tasks.md
2. Create docs/package.json with dependencies:
   - @docusaurus/core@^3.0.0
   - @docusaurus/preset-classic@^3.0.0
   - @docusaurus/theme-mermaid@^3.0.0
   - react@^18.0.0
   - react-dom@^18.0.0
3. Create docs/.gitignore
4. Create docs/README.md with setup steps
5. Run `cd docs && npm install` to verify
6. Mark task as complete [x] in tasks.md
7. Log implementation with log-implementation tool
```

---

- [x] 2. Configure Docusaurus Site

**Description**: Create docusaurus.config.js with GitHub Pages configuration

**Files**:
- `docs/docusaurus.config.js`

**Requirements Reference**: FR1, NFR4

**_Prompt**:
```
Implement the task for spec keyline-docusaurus-documentation, first run spec-workflow-guide to get the workflow guide then implement the task:

**Role**: Frontend Developer specializing in Docusaurus configuration

**Task**: Configure Docusaurus site for Keyline

**Context**:
- Follow Secan docusaurus.config.js pattern
- Configure for GitHub Pages deployment at wasilak.github.io/keyline/
- Enable dark mode, Mermaid diagrams, versioning

**Restrictions**:
- Do NOT create sidebars.js yet (Task 3)
- Do NOT create content yet (Task 4+)
- Focus only on site configuration

**Leverage**:
- Secan docusaurus.config.js as reference
- Design document theme configuration section

**Success Criteria**:
1. docusaurus.config.js created with all required sections
2. baseUrl set to '/keyline/'
3. GitHub Pages organization/project configured
4. Dark mode enabled with Catppuccin-inspired theme
5. Mermaid theme configured
6. Versioning configured for current (1.0.x Latest)
7. onBrokenLinks set to 'warn'

**Instructions**:
1. Mark task as in-progress [-] in tasks.md
2. Create docusaurus.config.js with:
   - Site metadata (title, tagline, favicon)
   - URL configuration (GitHub Pages)
   - Preset configuration (classic)
   - Theme configuration (dark mode, prism, mermaid)
   - Custom webpack plugin for warnings
3. Test config syntax: `cd docs && node -c docusaurus.config.js`
4. Mark task as complete [x] in tasks.md
5. Log implementation with log-implementation tool
```

---

- [x] 3. Create Navigation Structure

**Description**: Create sidebars.js and category metadata files

**Files**:
- `docs/sidebars.js`
- `docs/docs/01-getting-started/_category_.json`
- `docs/docs/02-authentication/_category_.json`
- `docs/docs/03-user-management/_category_.json`
- `docs/docs/04-deployment-modes/_category_.json`
- `docs/docs/05-deployment/_category_.json`
- `docs/docs/06-configuration/_category_.json`
- `docs/docs/07-observability/_category_.json`
- `docs/docs/08-integrations/_category_.json`
- `docs/docs/09-troubleshooting/_category_.json`
- `docs/docs/10-contributing/_category_.json`

**Requirements Reference**: FR2, FR8

**_Prompt**:
```
Implement the task for spec keyline-docusaurus-documentation, first run spec-workflow-guide to get the workflow guide then implement the task:

**Role**: Information Architect specializing in documentation navigation

**Task**: Create sidebar navigation and category structure

**Context**:
- 10-section navigation structure from design document
- Numbered prefixes for consistent ordering
- Category metadata for enhanced sidebar behavior

**Restrictions**:
- Do NOT create content files yet (Task 4+)
- Only create sidebar config and category metadata
- Use exact section names from design document

**Leverage**:
- Secan sidebars.js as reference
- Design document Information Architecture section

**Success Criteria**:
1. sidebars.js created with all 10 categories
2. Each category has correct items array (matching planned files)
3. All _category_.json files created with label and position
4. Navigation structure matches design document exactly

**Instructions**:
1. Mark task as in-progress [-] in tasks.md
2. Create sidebars.js with 10 categories
3. Create docs/docs/[section]/ directories
4. Create _category_.json in each section directory
5. Verify syntax: `cd docs && node -c sidebars.js`
6. Mark task as complete [x] in tasks.md
7. Log implementation with log-implementation tool
```

---

- [x] 4. Create Directory Structure and Static Assets

**Description**: Create all content directories and static asset placeholders

**Files**:
- `docs/docs/index.mdx`
- `docs/docs/changelog.md`
- `docs/static/img/logo.svg`
- `docs/static/img/favicon.svg`
- `docs/static/img/diagrams/.gitkeep`
- `docs/static/img/screenshots/.gitkeep`
- `docs/src/css/custom.css`
- `docs/src/components/.gitkeep`

**Requirements Reference**: FR2, FR4

**_Prompt**:
```
Implement the task for spec keyline-docusaurus-documentation, first run spec-workflow-guide to get the workflow guide then implement the task:

**Role**: Frontend Developer specializing in Docusaurus structure

**Task**: Create complete directory structure and asset placeholders

**Context**:
- Follow design document Content Architecture section
- Create all directories from structure diagram
- Add placeholder files for future assets

**Restrictions**:
- Do NOT create content pages yet (Task 5-14)
- Create only directories and placeholder files
- Use .gitkeep for empty directories

**Leverage**:
- Design document directory structure
- Secan docs structure as reference

**Success Criteria**:
1. All 10 section directories created under docs/docs/
2. static/img/ directory structure created
3. src/css/ and src/components/ directories created
4. docs/docs/index.mdx created with basic landing page
5. docs/docs/changelog.md created (placeholder)
6. custom.css created with basic theme overrides

**Instructions**:
1. Mark task as in-progress [-] in tasks.md
2. Create all directories from design document
3. Create index.mdx with basic landing page content
4. Create changelog.md with placeholder
5. Create custom.css with Catppuccin dark theme colors
6. Create placeholder SVG files for logo and favicon
7. Mark task as complete [x] in tasks.md
8. Log implementation with log-implementation tool
```

---

- [x] 5. Migrate Getting Started Content

**Description**: Migrate and transform content to Getting Started section

**Files**:
- `docs/docs/01-getting-started/01-about.md`
- `docs/docs/01-getting-started/02-architecture.md`
- `docs/docs/01-getting-started/03-quick-start.md`
- `docs/docs/01-getting-started/04-configuration-basics.md`
- `docs/docs/01-getting-started/05-migration-from-elastauth.md`

**Source Content**:
- README.md (sections: Overview, Features, Architecture, Quick Start, Use Cases)
- ELASTAUTH-TO-KEYLINE-EVOLUTION.md (full document)
- TESTING-QUICK.md (quick start content)

**Requirements Reference**: FR3, US-1, US-2

**_Prompt**:
```
Implement the task for spec keyline-docusaurus-documentation, first run spec-workflow-guide to get the workflow guide then implement the task:

**Role**: Technical Writer specializing in developer documentation

**Task**: Migrate existing content to Getting Started section

**Context**:
- Transform README.md overview into about.md
- Create architecture.md with Mermaid diagrams from design
- Combine quick start guides into 03-quick-start.md
- Migrate elastauth evolution doc to migration guide
- Add configuration basics from config.example.yaml

**Restrictions**:
- Do NOT copy-paste: transform content for Docusaurus format
- Add frontmatter to all files (id, title, description, sidebar_label)
- Use Mermaid for diagrams (not images)
- Keep technical accuracy intact

**Leverage**:
- README.md for features and overview
- ELASTAUTH-TO-KEYLINE-EVOLUTION.md for migration content
- Design document for Mermaid diagram syntax
- config.example.yaml for configuration examples

**Success Criteria**:
1. All 5 files created with proper frontmatter
2. about.md explains Keyline purpose clearly
3. architecture.md includes 2+ Mermaid diagrams
4. quick-start.md has working quick start commands
5. migration-from-elastauth.md has comparison table and checklist
6. All internal links work correctly
7. No broken references to old file paths

**Instructions**:
1. Mark task as in-progress [-] in tasks.md
2. Read source files (README.md, ELASTAUTH-TO-KEYLINE-EVOLUTION.md, TESTING-QUICK.md)
3. Create 01-about.md from README overview sections
4. Create 02-architecture.md with Mermaid diagrams
5. Create 03-quick-start.md combining quick start guides
6. Create 04-configuration-basics.md with key config concepts
7. Create 05-migration-from-elastauth.md from evolution doc
8. Test all internal links
9. Mark task as complete [x] in tasks.md
10. Log implementation with log-implementation tool
```

---

- [x] 6. Migrate Authentication Content

**Description**: Create Authentication section from existing specs and README

**Files**:
- `docs/docs/02-authentication/01-overview.md`
- `docs/docs/02-authentication/02-oidc-authentication.md`
- `docs/docs/02-authentication/03-local-users-basic-auth.md`
- `docs/docs/02-authentication/04-session-management.md`
- `docs/docs/02-authentication/05-logout.md`

**Source Content**:
- .kiro/specs/keyline-auth-proxy/requirements.md (OIDC flow details)
- .kiro/specs/keyline-auth-proxy/design.md (architecture diagrams)
- README.md (authentication sections)
- config/test-config-oidc.yaml (OIDC examples)
- config/test-config.yaml (local users examples)

**Requirements Reference**: FR3, US-3

**_Prompt**:
```
Implement the task for spec keyline-docusaurus-documentation, first run spec-workflow-guide to get the workflow guide then implement the task:

**Role**: Technical Writer specializing in authentication documentation

**Task**: Create comprehensive Authentication section

**Context**:
- Extract OIDC flow details from .kiro specs
- Document local users (Basic Auth) from config examples
- Explain session management from README and specs
- Add logout functionality documentation

**Restrictions**:
- Transform technical specs into user-friendly guides
- Include working configuration examples
- Add Mermaid sequence diagram for OIDC flow
- Do NOT include dynamic user management (Task 7)

**Leverage**:
- .kiro/specs/keyline-auth-proxy/*.md for OIDC details
- config/test-config-oidc.yaml for examples
- Design document for OIDC flow diagram
- README.md for session management overview

**Success Criteria**:
1. All 5 files created with proper frontmatter
2. overview.md explains dual authentication (OIDC + Basic)
3. oidc-authentication.md has complete OIDC flow diagram
4. local-users-basic-auth.md has bcrypt password examples
5. session-management.md explains Redis vs memory backends
6. logout.md explains session cleanup and OIDC end_session
7. All configuration examples are tested and working

**Instructions**:
1. Mark task as in-progress [-] in tasks.md
2. Read .kiro/specs/keyline-auth-proxy/requirements.md and design.md
3. Create 01-overview.md explaining authentication methods
4. Create 02-oidc-authentication.md with sequence diagram
5. Create 03-local-users-basic-auth.md with examples
6. Create 04-session-management.md with backend options
7. Create 05-logout.md with session cleanup steps
8. Add configuration examples to each page
9. Test all configuration examples
10. Mark task as complete [x] in tasks.md
11. Log implementation with log-implementation tool
```

---

- [x] 7. Migrate User Management Content

**Description**: Create User Management section from dynamic user mgmt specs

**Files**:
- `docs/docs/03-user-management/01-dynamic-user-management.md`
- `docs/docs/03-user-management/02-role-mappings.md`
- `docs/docs/03-user-management/03-credential-caching.md`
- `docs/docs/03-user-management/04-password-encryption.md`
- `docs/docs/03-user-management/05-admin-credentials.md`

**Source Content**:
- .spec-workflow/specs/keyline-dynamic-user-management/requirements.md
- .spec-workflow/specs/keyline-dynamic-user-management/design.md
- .kiro/specs/keyline-dynamic-user-management/requirements.md
- docs/user-management.md
- docs/troubleshooting-user-management.md
- config/user-management-example.yaml

**Requirements Reference**: FR3, US-3

**_Prompt**:
```
Implement the task for spec keyline-docusaurus-documentation, first run spec-workflow-guide to get the workflow guide then implement the task:

**Role**: Technical Writer specializing in Elasticsearch security documentation

**Task**: Create comprehensive User Management section

**Context**:
- Dynamic ES user management is Keyline's key differentiator
- Migrate from technical specs to user guides
- Include role mapping patterns and examples
- Explain credential caching and encryption
- Document admin credentials requirements

**Restrictions**:
- Transform specs into practical how-to guides
- Include troubleshooting tips from existing docs
- Add Mermaid flow diagram for user upsert
- Use examples from user-management-example.yaml

**Leverage**:
- .spec-workflow/specs/keyline-dynamic-user-management/*.md
- docs/user-management.md for existing content
- docs/troubleshooting-user-management.md for troubleshooting
- config/user-management-example.yaml for examples
- Design document for user upsert diagram

**Success Criteria**:
1. All 5 files created with proper frontmatter
2. dynamic-user-management.md explains benefits and flow
3. role-mappings.md has pattern matching examples (wildcards, email, groups)
4. credential-caching.md explains Redis vs memory, TTL, cache invalidation
5. password-encryption.md explains AES-256-GCM, key management
6. admin-credentials.md explains manage_security privilege requirements
7. User upsert flow diagram included
8. Troubleshooting section addresses common issues

**Instructions**:
1. Mark task as in-progress [-] in tasks.md
2. Read .spec-workflow/specs/keyline-dynamic-user-management/*.md
3. Read docs/user-management.md and troubleshooting-user-management.md
4. Create 01-dynamic-user-management.md with overview and flow diagram
5. Create 02-role-mappings.md with pattern examples
6. Create 03-credential-caching.md with backend comparison
7. Create 04-password-encryption.md with security details
8. Create 05-admin-credentials.md with ES setup instructions
9. Add troubleshooting tips to each page
10. Test all configuration examples
11. Mark task as complete [x] in tasks.md
12. Log implementation with log-implementation tool
```

---

- [x] 8. Migrate Deployment Modes Content

**Description**: Create Deployment Modes section from deployment docs and configs

**Files**:
- `docs/docs/04-deployment-modes/01-forwardauth-traefik.md`
- `docs/docs/04-deployment-modes/02-auth-request-nginx.md`
- `docs/docs/04-deployment-modes/03-standalone-proxy.md`

**Source Content**:
- docs/deployment.md (mode-specific sections)
- docs/FORWARDAUTH-TESTING.md
- docker-compose-forwardauth.yml
- docker-compose-oidc-forwardauth.yml
- config/test-config-forwardauth.yaml
- config/test-config-oidc-forwardauth.yaml

**Requirements Reference**: FR3, US-4

**_Prompt**:
```
Implement the task for spec keyline-docusaurus-documentation, first run spec-workflow-guide to get the workflow guide then implement the task:

**Role**: DevOps Engineer specializing in proxy deployments

**Task**: Create Deployment Modes section

**Context**:
- Three deployment modes: forwardAuth, auth_request, standalone
- Each mode has different integration patterns
- Include Traefik and Nginx examples
- Show header forwarding and response handling

**Restrictions**:
- Separate from Task 9 (Deployment platforms)
- Focus on mode configuration, not infrastructure
- Include working docker-compose examples
- Add architecture diagrams for each mode

**Leverage**:
- docs/deployment.md for mode explanations
- docker-compose-forwardauth.yml for Traefik example
- docker-compose-oidc-forwardauth.yml for OIDC+forwardAuth
- config/test-config-forwardauth.yaml for examples
- Design document for mode diagrams

**Success Criteria**:
1. All 3 files created with proper frontmatter
2. forwardauth-traefik.md has complete Traefik middleware config
3. auth-request-nginx.md has Nginx auth_request configuration
4. standalone-proxy.md has upstream proxy configuration
5. Each mode has architecture diagram (Mermaid)
6. Each mode has working docker-compose example
7. Header forwarding explained (X-Forwarded-*, X-Es-Authorization)

**Instructions**:
1. Mark task as in-progress [-] in tasks.md
2. Read docs/deployment.md and FORWARDAUTH-TESTING.md
3. Create 01-forwardauth-traefik.md with Traefik examples
4. Create 02-auth-request-nginx.md with Nginx examples
5. Create 03-standalone-proxy.md with upstream config
6. Add Mermaid flow diagram for each mode
7. Include docker-compose examples from existing files
8. Test configurations where possible
9. Mark task as complete [x] in tasks.md
10. Log implementation with log-implementation tool
```

---

- [x] 9. Migrate Deployment Platform Content

**Description**: Create Deployment section with platform-specific guides

**Files**:
- `docs/docs/05-deployment/01-docker.md`
- `docs/docs/05-deployment/02-kubernetes.md`
- `docs/docs/05-deployment/03-binary.md`
- `docs/docs/05-deployment/04-high-availability.md`
- `docs/docs/05-deployment/05-security-best-practices.md`

**Source Content**:
- docs/deployment.md (platform-specific sections)
- docs/docker-compose-README.md
- All docker-compose*.yml files
- docs/ROLLBACK-PLAN.md
- README.md (deployment sections)

**Requirements Reference**: FR3, US-4

**_Prompt**:
```
Implement the task for spec keyline-docusaurus-documentation, first run spec-workflow-guide to get the workflow guide then implement the task:

**Role**: DevOps Engineer specializing in containerized deployments

**Task**: Create Deployment section with platform guides

**Context**:
- Docker deployment with compose examples
- Kubernetes manifests and Helm chart guidance
- Binary installation for bare-metal
- High-availability with Redis and multiple instances
- Security best practices for production

**Restrictions**:
- Separate from Task 8 (deployment modes)
- Focus on infrastructure, not Keyline configuration
- Include production-ready examples
- Add security checklist

**Leverage**:
- docs/docker-compose-README.md for Docker content
- docker-compose.yml, docker-compose-oidc.yml for examples
- docs/deployment.md for K8s and HA content
- docs/ROLLBACK-PLAN.md for rollback procedures
- README.md for deployment overview

**Success Criteria**:
1. All 5 files created with proper frontmatter
2. docker.md has compose examples for all scenarios
3. kubernetes.md has Deployment, Service, Ingress manifests
4. binary.md has download and installation steps
5. high-availability.md has Redis clustering and load balancing
6. security-best-practices.md has TLS, secrets, network policies
7. Each platform has troubleshooting section

**Instructions**:
1. Mark task as in-progress [-] in tasks.md
2. Read docs/docker-compose-README.md and deployment.md
3. Create 01-docker.md with compose examples
4. Create 02-kubernetes.md with K8s manifests
5. Create 03-binary.md with installation steps
6. Create 04-high-availability.md with Redis clustering
7. Create 05-security-best-practices.md with checklist
8. Include rollback procedures from ROLLBACK-PLAN.md
9. Mark task as complete [x] in tasks.md
10. Log implementation with log-implementation tool
```

---

- [x] 10. Create Configuration Reference

**Description**: Create Configuration section with full reference and examples

**Files**:
- `docs/docs/06-configuration/01-reference.md`
- `docs/docs/06-configuration/02-environment-variables.md`
- `docs/docs/06-configuration/03-examples.md`
- `docs/docs/06-configuration/04-validation.md`

**Source Content**:
- config/config.example.yaml (full reference)
- All config/*.yaml files (examples)
- README.md (configuration sections)
- docs/configuration.md (existing reference)

**Requirements Reference**: FR3, US-3

**_Prompt**:
```
Implement the task for spec keyline-docusaurus-documentation, first run spec-workflow-guide to get the workflow guide then implement the task:

**Role**: Technical Writer specializing in configuration documentation

**Task**: Create comprehensive Configuration section

**Context**:
- Full configuration reference with all options
- Environment variable substitution explained
- Scenario-based examples (7 from config.example.yaml)
- Configuration validation process

**Restrictions**:
- Use config.example.yaml as primary source
- Include all configuration sections from example
- Organize by functional area (server, oidc, cache, etc.)
- Add YAML syntax highlighting to all examples

**Leverage**:
- config/config.example.yaml for complete reference
- config/*.yaml files for scenario examples
- docs/configuration.md for existing content
- README.md for configuration overview

**Success Criteria**:
1. All 4 files created with proper frontmatter
2. reference.md covers all config sections with descriptions
3. environment-variables.md explains ${VAR} syntax and validation
4. examples.md has 7+ scenario-based configurations
5. validation.md explains --validate-config flag and startup validation
6. All examples use correct YAML syntax
7. Cross-references to other sections work correctly

**Instructions**:
1. Mark task as in-progress [-] in tasks.md
2. Read config/config.example.yaml completely
3. Create 01-reference.md with all config sections
4. Create 02-environment-variables.md with substitution examples
5. Create 03-examples.md with scenario-based configs
6. Create 04-validation.md with validation process
7. Add cross-references to deployment and auth sections
8. Verify all YAML examples are valid
9. Mark task as complete [x] in tasks.md
10. Log implementation with log-implementation tool
```

---

- [x] 11. Create Observability Content

**Description**: Create Observability section from README and config examples

**Files**:
- `docs/docs/07-observability/01-logging.md`
- `docs/docs/07-observability/02-metrics.md`
- `docs/docs/07-observability/03-tracing.md`
- `docs/docs/07-observability/04-health-checks.md`

**Source Content**:
- README.md (Monitoring, Observability sections)
- config/config.example.yaml (observability config)
- .kiro/specs/keyline-auth-proxy/design.md (observability design)

**Requirements Reference**: FR3, NFR4

**_Prompt**:
```
Implement the task for spec keyline-docusaurus-documentation, first run spec-workflow-guide to get the workflow guide then implement the task:

**Role**: SRE specializing in observability and monitoring

**Task**: Create comprehensive Observability section

**Context**:
- Logging with loggergo (structured, JSON/text)
- Metrics with Prometheus (keyline_* metrics)
- Tracing with OpenTelemetry (otelgo)
- Health checks for Kubernetes readiness

**Restrictions**:
- Include actual metric names and descriptions
- Add example Prometheus scrape config
- Explain trace sampling configuration
- Show health check response format

**Leverage**:
- README.md Monitoring section for overview
- config/config.example.yaml observability section
- .kiro/specs/keyline-auth-proxy/design.md for details
- Design document for observability diagrams

**Success Criteria**:
1. All 4 files created with proper frontmatter
2. logging.md explains log levels, formats, structured fields
3. metrics.md lists all Prometheus metrics with descriptions
4. tracing.md explains OpenTelemetry setup and sampling
5. health-checks.md shows response format and K8s probes
6. Each page has configuration examples
7. Integration with Grafana mentioned (future enhancement)

**Instructions**:
1. Mark task as in-progress [-] in tasks.md
2. Read README.md Monitoring and Observability sections
3. Create 01-logging.md with loggergo configuration
4. Create 02-metrics.md with Prometheus metrics list
5. Create 03-tracing.md with OpenTelemetry setup
6. Create 04-health-checks.md with K8s probe examples
7. Add example configurations to each page
8. Mark task as complete [x] in tasks.md
9. Log implementation with log-implementation tool
```

---

- [x] 12. Create Integrations Content

**Description**: Create Integrations section for external systems

**Files**:
- `docs/docs/08-integrations/01-elasticsearch.md`
- `docs/docs/08-integrations/02-kibana.md`
- `docs/docs/08-integrations/03-oidc-providers.md`
- `docs/docs/08-integrations/04-redis.md`

**Source Content**:
- README.md (Use Cases, Architecture)
- docker-compose*.yml files (integration examples)
- config/*.yaml files (integration configs)

**Requirements Reference**: FR3, US-4

**_Prompt**:
```
Implement the task for spec keyline-docusaurus-documentation, first run spec-workflow-guide to get the workflow guide then implement the task:

**Role**: Integration Specialist specializing in Elasticsearch ecosystem

**Task**: Create Integrations section for external systems

**Context**:
- Elasticsearch integration (versions 7, 8, 9, OpenSearch)
- Kibana integration (proxy patterns)
- OIDC providers (Google, Azure AD, Okta, generic)
- Redis integration (caching backends)

**Restrictions**:
- Focus on integration patterns, not basic setup
- Include version compatibility notes
- Add provider-specific OIDC configurations
- Explain Redis connection options

**Leverage**:
- README.md Use Cases for integration scenarios
- docker-compose*.yml for ES and Redis examples
- config/*.yaml for provider configurations
- Design document for integration diagrams

**Success Criteria**:
1. All 4 files created with proper frontmatter
2. elasticsearch.md covers ES 7/8/9 and OpenSearch compatibility
3. kibana.md explains proxy patterns and headers
4. oidc-providers.md has configs for Google, Azure AD, Okta
5. redis.md explains connection, TLS, authentication
6. Each integration has troubleshooting section
7. Version compatibility matrix included

**Instructions**:
1. Mark task as in-progress [-] in tasks.md
2. Read README.md Use Cases and Architecture
3. Create 01-elasticsearch.md with version matrix
4. Create 02-kibana.md with proxy patterns
5. Create 03-oidc-providers.md with provider configs
6. Create 04-redis.md with connection options
7. Add troubleshooting tips to each page
8. Mark task as complete [x] in tasks.md
9. Log implementation with log-implementation tool
```

---

- [x] 13. Create Troubleshooting Content

**Description**: Create Troubleshooting section from existing troubleshooting docs

**Files**:
- `docs/docs/09-troubleshooting/01-general.md`
- `docs/docs/09-troubleshooting/02-oidc-issues.md`
- `docs/docs/09-troubleshooting/03-user-management.md`
- `docs/docs/09-troubleshooting/04-deployment.md`
- `docs/docs/09-troubleshooting/05-faq.md`

**Source Content**:
- docs/troubleshooting.md
- docs/troubleshooting-user-management.md
- docs/TESTING.md (common issues)
- docs/TESTING-GUIDE.md (debugging tips)

**Requirements Reference**: FR3, US-5

**_Prompt**:
```
Implement the task for spec keyline-docusaurus-documentation, first run spec-workflow-guide to get the workflow guide then implement the task:

**Role**: Support Engineer specializing in troubleshooting guides

**Task**: Create comprehensive Troubleshooting section

**Context**:
- Organize by symptom/category for easy navigation
- Include general troubleshooting methodology
- Specific guides for OIDC, user management, deployment
- FAQ for common questions

**Restrictions**:
- Use existing troubleshooting content as primary source
- Organize by symptom → cause → solution pattern
- Include log examples and debugging commands
- Add FAQ with quick answers

**Leverage**:
- docs/troubleshooting.md for general issues
- docs/troubleshooting-user-management.md for user mgmt issues
- docs/TESTING.md and TESTING-GUIDE.md for debugging tips
- Config examples for validation troubleshooting

**Success Criteria**:
1. All 5 files created with proper frontmatter
2. general.md has troubleshooting methodology and common commands
3. oidc-issues.md covers callback, token, JWKS issues
4. user-management.md covers ES API, role mapping, cache issues
5. deployment.md covers Docker, K8s, binary deployment issues
6. faq.md has 10+ common questions with answers
7. Each issue has: symptom, cause, solution, prevention

**Instructions**:
1. Mark task as in-progress [-] in tasks.md
2. Read docs/troubleshooting.md and troubleshooting-user-management.md
3. Create 01-general.md with methodology and tools
4. Create 02-oidc-issues.md organized by OIDC flow stage
5. Create 03-user-management.md organized by component
6. Create 04-deployment.md organized by platform
7. Create 05-faq.md with common questions
8. Add log examples and debug commands to each page
9. Mark task as complete [x] in tasks.md
10. Log implementation with log-implementation tool
```

---

- [x] 14. Create Contributing Content

**Description**: Create Contributing section for developers

**Files**:
- `docs/docs/10-contributing/01-development.md`
- `docs/docs/10-contributing/02-testing.md`
- `docs/docs/10-contributing/03-release-process.md`
- `docs/docs/10-contributing/04-security-reports.md`

**Source Content**:
- docs/TESTING.md
- docs/TESTING-GUIDE.md
- docs/TESTING-QUICK.md
- docs/RELEASE-TAGGING.md
- README.md (Development, Contributing sections)

**Requirements Reference**: FR3, US-6

**_Prompt**:
```
Implement the task for spec keyline-docusaurus-documentation, first run spec-workflow-guide to get the workflow guide then implement the task:

**Role**: Developer Advocate specializing in contributor documentation

**Task**: Create Contributing section for developers

**Context**:
- Development setup and workflow
- Testing guidelines (unit, integration, property-based)
- Release process and version tagging
- Security reporting process

**Restrictions**:
- Merge duplicate testing docs into single guide
- Include Makefile commands
- Add GitHub workflow for releases
- Security process follows responsible disclosure

**Leverage**:
- docs/TESTING*.md files for testing content
- docs/RELEASE-TAGGING.md for release process
- README.md Development section for setup
- Makefile for build/test commands

**Success Criteria**:
1. All 4 files created with proper frontmatter
2. development.md has setup, build, run instructions
3. testing.md consolidates all testing guides
4. release-process.md has versioning and tagging steps
5. security-reports.md has responsible disclosure process
6. All Makefile commands documented
7. "Edit this page" links configured correctly

**Instructions**:
1. Mark task as in-progress [-] in tasks.md
2. Read all TESTING*.md files and RELEASE-TAGGING.md
3. Create 01-development.md with setup guide
4. Create 02-testing.md consolidating all testing docs
5. Create 03-release-process.md with release steps
6. Create 04-security-reports.md with disclosure process
7. Include Makefile commands in development.md
8. Add contribution guidelines
9. Mark task as complete [x] in tasks.md
10. Log implementation with log-implementation tool
```

---

- [x] 15. Create GitHub Actions Workflow

**Description**: Create .github/workflows/docs.yml for automated deployment

**Files**:
- `.github/workflows/docs.yml`

**Requirements Reference**: FR5

**_Prompt**:
```
Implement the task for spec keyline-docusaurus-documentation, first run spec-workflow-guide to get the workflow guide then implement the task:

**Role**: DevOps Engineer specializing in GitHub Actions

**Task**: Create GitHub Actions workflow for documentation deployment

**Context**:
- Two-job pipeline: build and deploy
- Deploy only on main branch (not PRs)
- Use Taskfile for consistency with Secan
- Node.js 24 with npm cache

**Restrictions**:
- Follow Secan .github/workflows/docs.yml pattern exactly
- Use actions/checkout@v6, actions/setup-node@v6, etc.
- Configure Pages environment correctly
- Include manual dispatch trigger

**Leverage**:
- Secan .github/workflows/docs.yml as reference
- Design document GitHub Actions section
- GitHub Pages deployment documentation

**Success Criteria**:
1. .github/workflows/docs.yml created with correct syntax
2. Build job: checkout, setup Node, install deps, build, upload artifact
3. Deploy job: download artifact, setup Pages, deploy
4. Triggers: push to main (docs paths), PR, workflow_dispatch
5. Permissions: pages: write, id-token: write for deploy job
6. Node.js 24 configured with npm cache
7. Taskfile integration (task docs:install, task docs:build)

**Instructions**:
1. Mark task as in-progress [-] in tasks.md
2. Read Secan .github/workflows/docs.yml
3. Create .github/workflows/docs.yml for Keyline
4. Update paths for keyline repo (baseUrl: '/keyline/')
5. Add FORCE_JAVASCRIPT_ACTIONS_TO_NODE24 env var
6. Validate workflow syntax: `act -n` (if available) or manual review
7. Mark task as complete [x] in tasks.md
8. Log implementation with log-implementation tool
```

---

- [x] 16. Update Taskfile and Cleanup

**Description**: Add documentation tasks to Taskfile.yml and clean up old files

**Files**:
- `Taskfile.yml` (update)
- `docs/README.md` (update)
- Old markdown files (archive/delete)

**Requirements Reference**: FR6, FR7

**_Prompt**:
```
Implement the task for spec keyline-docusaurus-documentation, first run spec-workflow-guide to get the workflow guide then implement the task:

**Role**: DevOps Engineer specializing in developer workflow

**Task**: Update Taskfile.yml with documentation tasks and clean up repository

**Context**:
- Add docs:install, docs:dev, docs:build, docs:preview tasks
- Follow Secan Taskfile.yml pattern
- Update root README.md with documentation link
- Archive or remove old markdown files

**Restrictions**:
- Use Taskfile v3 syntax
- Keep essential markdown (README, RELEASE-NOTES, LICENSE)
- Archive spec files remain in .spec-workflow/ and .kiro/
- Update README.md to link to new documentation site

**Leverage**:
- Secan Taskfile.yml documentation tasks as reference
- Design document Taskfile Integration section
- Existing Keyline Taskfile.yml (if exists)

**Success Criteria**:
1. Taskfile.yml updated with 4 documentation tasks
2. docs:install runs npm ci in docs/
3. docs:dev starts development server
4. docs:build builds for production
5. docs:preview serves production build
6. README.md updated with documentation link
7. Old docs/*.md files archived or deleted
8. Repository structure is clean and organized

**Instructions**:
1. Mark task as in-progress [-] in tasks.md
2. Read Secan Taskfile.yml documentation tasks section
3. Add documentation tasks to Taskfile.yml
4. Update docs/README.md with setup instructions
5. Update root README.md:
   - Add documentation site link at top
   - Remove duplicated content (now in docs)
   - Keep: overview, features, quick links
6. Archive old docs/*.md files (move to docs-archive/ or delete)
7. Verify Taskfile syntax: `task --list`
8. Test tasks: task docs:install, task docs:build
9. Mark task as complete [x] in tasks.md
10. Log implementation with log-implementation tool
```

---

## Implementation Order

**Phase 1: Foundation** (Tasks 1-4)
- Initialize Docusaurus, configure site, create structure

**Phase 2: Core Content** (Tasks 5-10)
- Migrate Getting Started, Authentication, User Management, Deployment, Configuration

**Phase 3: Supporting Content** (Tasks 11-14)
- Create Observability, Integrations, Troubleshooting, Contributing

**Phase 4: Deployment & Cleanup** (Tasks 15-16)
- Set up CI/CD, update Taskfile, clean up repository

---

## Verification Checklist

Before marking spec complete, verify:

- V1. All 16 tasks implemented and logged
- V2. Documentation builds successfully: `task docs:build`
- V3. All internal links work
- V4. All diagrams render correctly
- V5. GitHub Actions workflow validated
- V6. Old markdown files cleaned up
- V7. README.md updated with docs link
- V8. Local dev server works: `task docs:dev`
- V9. Mobile-responsive design verified
- V10. Dark mode theme applied
