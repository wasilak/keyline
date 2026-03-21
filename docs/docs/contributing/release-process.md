---
sidebar_label: Release Process
sidebar_position: 3
---

# Release Process

Procedure for releasing new versions of Keyline.

## Version Numbering

Keyline follows Semantic Versioning (MAJOR.MINOR.PATCH):
- **MAJOR**: Breaking changes
- **MINOR**: New features (backward compatible)
- **PATCH**: Bug fixes (backward compatible)

## Release Checklist

- [ ] All tests passing
- [ ] Build succeeds for all platforms
- [ ] Documentation updated
- [ ] CHANGELOG.md updated
- [ ] Git tag created
- [ ] GitHub release published
- [ ] Docker images pushed

## Creating a Release

```bash
# Tag the release
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0

# Build release binaries
make build-all

# Create GitHub release
# Upload binaries to GitHub Releases
```

## Next Steps

- **[Development](./development.md)** - Development guide
- **[Testing](./testing.md)** - Testing guide
- **[Security Reports](./security-reports.md)** - Reporting vulnerabilities
