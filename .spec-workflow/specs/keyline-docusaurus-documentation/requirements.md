# Keyline Docusaurus Documentation - Requirements

## Overview

Implement a comprehensive Docusaurus-based documentation site for Keyline, replacing the current scattered markdown files with a structured, searchable, and versioned documentation system. The new documentation will follow the Secan project's proven GitHub Pages deployment pattern and integrate with Keyline's existing development workflow.

**Key principle**: One source of truth for all user-facing documentation, with clear navigation, search capabilities, and automated deployment.

## Goals

- Create professional, searchable documentation site using Docusaurus
- Migrate all existing markdown content to structured documentation
- Implement automated GitHub Pages deployment
- Establish documentation maintenance workflow
- Clean up scattered markdown files across repository

## Non-Goals

- API reference documentation (separate future task)
- Video tutorials or interactive guides
- Multi-language support (English only for now)
- Blog or news section

## Functional Requirements

### FR1: Docusaurus Site Setup
- Initialize Docusaurus v3 in `docs/` directory
- Configure for GitHub Pages deployment
- Set `baseUrl: '/keyline/'` to match repository name
- Enable dark mode with Catppuccin-inspired theme
- Configure Mermaid.js for diagrams
- Enable documentation versioning for future releases

### FR2: Documentation Structure
- Implement 10-section navigation structure:
  1. Getting Started
  2. Authentication
  3. User Management
  4. Deployment Modes
  5. Deployment
  6. Configuration
  7. Observability
  8. Integrations
  9. Troubleshooting
  10. Contributing
- Use numbered prefixes for consistent ordering
- Create sidebar configuration matching structure

### FR3: Content Migration
- Migrate all existing markdown content from:
  - `docs/*.md` (16 files)
  - `README.md`
  - `RELEASE-NOTES.md`
  - `.spec-workflow/specs/**/*.md`
  - `.kiro/specs/**/*.md` (archived specs)
- Transform content to Docusaurus format with frontmatter
- Preserve all technical accuracy during migration
- Create new content to fill gaps

### FR4: Visual Assets
- Create architecture diagrams using Mermaid:
  - High-level architecture
  - OIDC authentication flow
  - Dynamic user upsert flow
  - ForwardAuth mode diagram
  - Standalone mode diagram
- Generate screenshots for key features
- Create favicon and logo assets
- Design social card for sharing

### FR5: GitHub Actions Deployment
- Create `.github/workflows/docs.yml` workflow
- Implement two-job pipeline (build + deploy)
- Configure deployment only on `main` branch
- Support manual dispatch for testing
- Upload build artifacts with 1-day retention

### FR6: Taskfile Integration
- Add documentation tasks to `Taskfile.yml`:
  - `docs:install` - Install dependencies
  - `docs:dev` - Start development server
  - `docs:build` - Build for production
  - `docs:preview` - Preview production build
- Use consistent patterns with Secan project
- Include Node.js version pinning (Node 24)

### FR7: Content Cleanup
- Remove or archive all original markdown files after migration
- Update `README.md` to link to documentation site
- Keep only essential markdown in root:
  - `README.md` (updated with docs link)
  - `RELEASE-NOTES.md` (changelog)
  - `LICENSE`
  - `CONTRIBUTING.md` (if created)
- Archive spec files remain in `.spec-workflow/` and `.kiro/`

### FR8: Search and Navigation
- Enable Docusaurus built-in search
- Configure breadcrumb navigation
- Implement "Edit this page" links to GitHub
- Add "On this page" table of contents
- Configure previous/next page navigation

## Non-Functional Requirements

### NFR1: Performance
- Build time under 2 minutes
- Page load time under 2 seconds
- Search results in under 500ms

### NFR2: Maintainability
- Clear separation of content and configuration
- Reusable components for common patterns
- Consistent formatting across all pages
- Easy to add new sections

### NFR3: Accessibility
- WCAG 2.1 AA compliance
- Keyboard navigation support
- Screen reader compatibility
- Sufficient color contrast

### NFR4: SEO
- Proper meta tags for all pages
- Open Graph social cards
- Sitemap generation
- robots.txt configuration

## User Stories

### US-1: As a new Keyline user, I want to quickly understand what Keyline does
- [ ] **US-1.1**: Landing page explains Keyline's purpose in 2-3 sentences
- [ ] **US-1.2**: Architecture diagram shows how Keyline fits in infrastructure
- [ ] **US-1.3**: Quick start guide gets me running in under 5 minutes
- [ ] **US-1.4**: Use cases show common deployment scenarios

### US-2: As an elastauth user, I want to migrate to Keyline smoothly
- [ ] **US-2.1**: Dedicated migration guide explains differences
- [ ] **US-2.2**: Configuration mapping table shows elastauth → Keyline equivalents
- [ ] **US-2.3**: Step-by-step migration checklist provided
- [ ] **US-2.4**: Troubleshooting section addresses common migration issues

### US-3: As a developer, I want to configure Keyline for my environment
- [ ] **US-3.1**: Configuration reference documents all options
- [ ] **US-3.2**: Example configurations for common scenarios
- [ ] **US-3.3**: Environment variable substitution explained
- [ ] **US-3.4**: Configuration validation process documented

### US-4: As an operator, I want to deploy Keyline to my infrastructure
- [ ] **US-4.1**: Docker deployment guide with compose examples
- [ ] **US-4.2**: Kubernetes deployment manifests and instructions
- [ ] **US-4.3**: Binary installation for bare-metal deployments
- [ ] **US-4.4**: High-availability configuration guide

### US-5: As a user, I want to troubleshoot issues independently
- [ ] **US-5.1**: Search finds relevant documentation quickly
- [ ] **US-5.2**: Troubleshooting section organized by symptom
- [ ] **US-5.3**: FAQ answers common questions
- [ ] **US-5.4**: Clear escalation path for unresolved issues

### US-6: As a contributor, I want to improve the documentation
- [ ] **US-6.1**: "Edit this page" links go to correct GitHub file
- [ ] **US-6.2**: Contribution guidelines explain documentation process
- [ ] **US-6.3**: Local development setup is straightforward
- [ ] **US-6.4**: Style guide ensures consistency

## Glossary

| Term | Definition |
|------|------------|
| Docusaurus | React-based documentation site generator |
| GitHub Pages | Static site hosting by GitHub |
| ForwardAuth | Traefik/Nginx authentication middleware pattern |
| OIDC | OpenID Connect authentication protocol |
| Mermaid | JavaScript diagramming library |
| Frontmatter | YAML metadata at top of markdown files |

## Traceability Matrix

| Requirement | Design Section | Tasks |
|-------------|----------------|-------|
| FR1 | Site Configuration | 1, 2 |
| FR2 | Information Architecture | 3, 4 |
| FR3 | Content Migration | 5, 6, 7 |
| FR4 | Visual Design | 8, 9 |
| FR5 | CI/CD Pipeline | 10 |
| FR6 | Developer Workflow | 11 |
| FR7 | Repository Cleanup | 12 |
| FR8 | User Experience | 4, 13 |
| NFR1 | Performance section | 14 |
| NFR2 | Maintainability section | 3, 5 |
| NFR3 | Accessibility section | 15 |
| NFR4 | SEO section | 16 |
| US-1 | Getting Started section | 5, 6, 8 |
| US-2 | Migration Guide | 7 |
| US-3 | Configuration section | 5, 8 |
| US-4 | Deployment section | 5, 8 |
| US-5 | Troubleshooting section | 7, 13 |
| US-6 | Contributing section | 11, 12 |

## Dependencies

- Node.js 24 or higher
- npm package manager
- GitHub repository with Pages enabled
- Docusaurus v3 dependencies
- Mermaid.js for diagrams
- Existing Keyline markdown content

## Success Criteria

1. Documentation site builds successfully
2. All existing content migrated accurately
3. GitHub Actions deploys to GitHub Pages automatically
4. Navigation and search work correctly
5. All original markdown files cleaned up or archived
6. README.md updated with documentation link
7. Local development workflow functional
8. Mobile-responsive design verified
9. Dark mode theme applied consistently
10. All diagrams render correctly
