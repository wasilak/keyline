# Release Tagging Guide

This document describes the release tagging process for Keyline.

## Version Numbering

Keyline follows [Semantic Versioning](https://semver.org/) (SemVer):

- **MAJOR.MINOR.PATCH** (e.g., 2.0.0)
- **MAJOR**: Breaking changes
- **MINOR**: New features (backward compatible)
- **PATCH**: Bug fixes (backward compatible)

## Release Process

### 1. Pre-Release Checklist

Before tagging a release, ensure:

- [ ] All tests pass (unit, integration, property-based)
- [ ] Code builds successfully
- [ ] Documentation is updated
- [ ] Release notes are prepared
- [ ] Migration guide is complete (for breaking changes)
- [ ] Rollback plan is documented (for major changes)
- [ ] CI/CD pipeline passes
- [ ] Security scan passes
- [ ] Performance benchmarks meet targets

### 2. Version Determination

Determine the version number based on changes:

**Major Version (X.0.0)** - Breaking changes:
- Configuration format changes
- API changes
- Removed features
- Incompatible behavior changes

**Minor Version (x.Y.0)** - New features:
- New authentication methods
- New configuration options
- New metrics or observability features
- Performance improvements

**Patch Version (x.y.Z)** - Bug fixes:
- Security fixes
- Bug fixes
- Documentation fixes
- Dependency updates (security)

### 3. Create Release Tag

#### For Major/Minor Releases

```bash
# Ensure you're on the main branch
git checkout main
git pull origin main

# Create annotated tag
git tag -a v2.0.0 -m "Release v2.0.0 - Dynamic User Management

Major Features:
- Dynamic Elasticsearch user management
- Role-based access control with flexible mappings
- Credential caching with AES-256-GCM encryption
- Horizontal scaling support with Redis

Breaking Changes:
- Removed elasticsearch.users configuration
- Removed oidc.mappings configuration
- Removed local_users[].es_user field

See RELEASE-NOTES.md for full details."

# Push tag to remote
git push origin v2.0.0
```

#### For Patch Releases

```bash
# Ensure you're on the main branch
git checkout main
git pull origin main

# Create annotated tag
git tag -a v2.0.1 -m "Release v2.0.1 - Bug Fixes

Bug Fixes:
- Fixed cache encryption key validation
- Fixed role mapping pattern matching
- Fixed ES API retry logic

See RELEASE-NOTES.md for full details."

# Push tag to remote
git push origin v2.0.1
```

### 4. Create GitHub Release

After pushing the tag:

1. Go to GitHub repository
2. Click "Releases" → "Draft a new release"
3. Select the tag you just created
4. Set release title: "v2.0.0 - Dynamic User Management"
5. Copy release notes from RELEASE-NOTES.md
6. Attach binaries (if available)
7. Mark as pre-release (if applicable)
8. Publish release

### 5. Update Documentation

After release:

- [ ] Update README.md with latest version
- [ ] Update installation instructions
- [ ] Update Docker image tags in examples
- [ ] Update Kubernetes manifests
- [ ] Update Helm chart version (if applicable)

### 6. Announce Release

- [ ] Post to GitHub Discussions
- [ ] Update project website (if applicable)
- [ ] Notify users via mailing list (if applicable)
- [ ] Post to relevant communities (if applicable)

## Tag Naming Convention

### Release Tags

- **Format**: `vX.Y.Z`
- **Examples**: `v2.0.0`, `v2.1.0`, `v2.0.1`

### Pre-Release Tags

- **Format**: `vX.Y.Z-rc.N` (release candidate)
- **Examples**: `v2.0.0-rc.1`, `v2.0.0-rc.2`

### Development Tags

- **Format**: `vX.Y.Z-alpha.N` or `vX.Y.Z-beta.N`
- **Examples**: `v2.0.0-alpha.1`, `v2.0.0-beta.1`

## Release Branches

### Main Branch

- Always stable and releasable
- All releases are tagged from main
- Protected branch with required reviews

### Release Branches (Optional)

For long-term support:

```bash
# Create release branch for v2.0.x
git checkout -b release/v2.0 v2.0.0
git push origin release/v2.0

# Apply patch to release branch
git checkout release/v2.0
git cherry-pick <commit-hash>
git tag -a v2.0.1 -m "Release v2.0.1"
git push origin release/v2.0 v2.0.1
```

## Rollback a Release

If a release has critical issues:

### 1. Create Hotfix

```bash
# Create hotfix branch from previous tag
git checkout -b hotfix/v2.0.1 v2.0.0

# Apply fixes
git commit -m "fix: critical issue"

# Tag hotfix
git tag -a v2.0.1 -m "Hotfix v2.0.1"
git push origin hotfix/v2.0.1 v2.0.1
```

### 2. Deprecate Bad Release

- Mark GitHub release as "deprecated" in description
- Add warning to release notes
- Point users to fixed version

### 3. Communicate

- Post incident report
- Notify users of issue and fix
- Update documentation

## Automated Release Process

### GitHub Actions (Recommended)

Create `.github/workflows/release.yml`:

```yaml
name: Release

on:
  push:
    tags:
      - 'v*'

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.21'

      - name: Build binaries
        run: make build-all

      - name: Create Release
        uses: softprops/action-gh-release@v1
        with:
          files: bin/*
          body_path: RELEASE-NOTES.md
          draft: false
          prerelease: false
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}

      - name: Build and push Docker image
        uses: docker/build-push-action@v5
        with:
          context: .
          push: true
          tags: |
            keyline:${{ github.ref_name }}
            keyline:latest
```

## Version Management in Code

### Update Version in Code

```go
// cmd/keyline/main.go
var (
    Version   = "dev"
    BuildTime = "unknown"
)

func main() {
    fmt.Printf("Keyline %s (built %s)\n", Version, BuildTime)
    // ...
}
```

### Build with Version

```bash
# Makefile
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-w -s -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

build:
    go build $(LDFLAGS) -o bin/keyline ./cmd/keyline
```

## Release Checklist Template

```markdown
## Release vX.Y.Z Checklist

### Pre-Release
- [ ] All tests pass
- [ ] Code builds successfully
- [ ] Documentation updated
- [ ] Release notes prepared
- [ ] Migration guide complete (if breaking changes)
- [ ] CI/CD pipeline passes

### Release
- [ ] Version number determined
- [ ] Tag created and pushed
- [ ] GitHub release created
- [ ] Binaries attached to release
- [ ] Docker image built and pushed

### Post-Release
- [ ] Documentation updated
- [ ] Installation instructions updated
- [ ] Release announced
- [ ] Users notified

### Verification
- [ ] Docker image works
- [ ] Binaries work on all platforms
- [ ] Installation instructions work
- [ ] Migration guide works (if applicable)
```

## Example: v2.0.0 Release

### Commands

```bash
# 1. Ensure main is up to date
git checkout main
git pull origin main

# 2. Verify everything is ready
make ci
make build-all

# 3. Create tag
git tag -a v2.0.0 -m "Release v2.0.0 - Dynamic User Management

Major Features:
- Dynamic Elasticsearch user management
- Role-based access control
- Credential caching with encryption
- Horizontal scaling support

Breaking Changes:
- Removed static user mapping
- New configuration format

See RELEASE-NOTES.md for details."

# 4. Push tag
git push origin v2.0.0

# 5. Build and push Docker image
docker build -t keyline:v2.0.0 .
docker tag keyline:v2.0.0 keyline:latest
docker push keyline:v2.0.0
docker push keyline:latest

# 6. Create GitHub release (via UI or CLI)
gh release create v2.0.0 \
  --title "v2.0.0 - Dynamic User Management" \
  --notes-file RELEASE-NOTES.md \
  bin/*
```

## Troubleshooting

### Tag Already Exists

```bash
# Delete local tag
git tag -d v2.0.0

# Delete remote tag
git push origin :refs/tags/v2.0.0

# Recreate tag
git tag -a v2.0.0 -m "Release v2.0.0"
git push origin v2.0.0
```

### Wrong Tag

```bash
# Move tag to different commit
git tag -f -a v2.0.0 <commit-hash> -m "Release v2.0.0"
git push -f origin v2.0.0
```

### Release Failed

```bash
# Delete GitHub release
gh release delete v2.0.0

# Delete tag
git tag -d v2.0.0
git push origin :refs/tags/v2.0.0

# Fix issues and retry
```

## Best Practices

1. **Always use annotated tags** (`-a` flag)
2. **Include meaningful tag messages**
3. **Test before tagging**
4. **Tag from main branch only**
5. **Never force-push tags** (unless absolutely necessary)
6. **Document breaking changes clearly**
7. **Provide migration guides for major versions**
8. **Test installation instructions**
9. **Verify Docker images work**
10. **Communicate releases to users**

## References

- [Semantic Versioning](https://semver.org/)
- [Git Tagging](https://git-scm.com/book/en/v2/Git-Basics-Tagging)
- [GitHub Releases](https://docs.github.com/en/repositories/releasing-projects-on-github)
- [Conventional Commits](https://www.conventionalcommits.org/)
