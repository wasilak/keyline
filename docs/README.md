# Keyline Documentation

This directory contains the Keyline documentation built with [Docusaurus v3](https://docusaurus.io/).

## Quick Start

### Prerequisites

- Node.js 24 or higher
- npm (comes with Node.js)

### Development Server

Start the local development server to preview documentation changes:

```bash
# From project root
task docs:dev

# Or directly in docs directory
cd docs
npm run start
```

This will start the development server at `http://localhost:3000/keyline/`. The server includes hot reloading, so changes to documentation files will be reflected immediately.

### Building Documentation

Build the static documentation site for production:

```bash
# From project root
task docs:build

# Or directly in docs directory
cd docs
npm run build
```

This generates static content in the `build/` directory that can be served by any static hosting service.

### Preview Production Build

Preview the production build locally:

```bash
cd docs
npm run serve
```

This serves the built documentation at `http://localhost:3000/keyline/` using the same base path as production.

### Clear Cache

If you encounter build issues, clear the Docusaurus cache:

```bash
cd docs
npm run clear
```

## Project Structure

```
docs/
├── docusaurus.config.js    # Site configuration
├── sidebars.js              # Navigation structure
├── package.json             # Dependencies
├── README.md                # This file
│
├── docs/                    # Documentation content
│   ├── index.mdx            # Landing page
│   ├── changelog.md         # Release changelog
│   ├── 01-getting-started/  # Section 1
│   ├── 02-authentication/   # Section 2
│   ├── 03-user-management/  # Section 3
│   ├── 04-deployment-modes/ # Section 4
│   ├── 05-deployment/       # Section 5
│   ├── 06-configuration/    # Section 6
│   ├── 07-observability/    # Section 7
│   ├── 08-integrations/     # Section 8
│   ├── 09-troubleshooting/  # Section 9
│   └── 10-contributing/     # Section 10
│
├── static/                  # Static assets
│   └── img/                 # Images, diagrams, screenshots
│
└── src/                     # Custom code
    ├── css/                 # Custom styles
    └── components/          # Custom React components
```

## Contributing

When contributing to documentation:

1. **Create a branch**: `git checkout -b docs/my-improvement`
2. **Make changes**: Edit markdown files in `docs/docs/`
3. **Test locally**: Run `npm run start` and verify changes
4. **Build test**: Run `npm run build` to ensure no errors
5. **Commit**: Use clear commit messages: `docs: add installation guide`
6. **Push and PR**: Push branch and create pull request

### Documentation Style Guide

- Use clear, concise language
- Include code examples where appropriate
- Add diagrams for complex architectures
- Use admonitions for important notes
- Test all commands and code examples
- Keep line length reasonable (80-100 characters)
- Use proper heading hierarchy (H1 → H2 → H3)

## Deployment

Documentation is automatically deployed to GitHub Pages when changes are pushed to the `main` branch via GitHub Actions.

**Live Site**: https://wasilak.github.io/keyline/

## Additional Resources

- [Docusaurus Documentation](https://docusaurus.io/docs)
- [Markdown Guide](https://www.markdownguide.org/)
- [Mermaid Documentation](https://mermaid.js.org/)
- [GitHub Pages Documentation](https://docs.github.com/en/pages)
