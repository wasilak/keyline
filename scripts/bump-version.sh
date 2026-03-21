#!/bin/bash
# Bump version and create git tag
# Usage: ./scripts/bump-version.sh [-f] v1.3.0
#   -f, --force    Force recreate tag if it already exists

set -e

FORCE=false
VERSION=""

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -f|--force)
            FORCE=true
            shift
            ;;
        *)
            VERSION="$1"
            shift
            ;;
    esac
done

if [ -z "$VERSION" ]; then
    echo "Error: Version argument required"
    echo "Usage: $0 [-f] v1.3.0"
    echo "  -f, --force    Force recreate tag if it already exists"
    exit 1
fi

# Check if tag already exists
if git rev-parse "$VERSION" >/dev/null 2>&1; then
    if [ "$FORCE" = true ]; then
        echo "Force: Deleting existing tag: $VERSION"
        git tag -d "$VERSION" >/dev/null 2>&1 || true
    else
        echo "Error: Tag '$VERSION' already exists"
        echo "Use --force to recreate the tag"
        echo "Usage: $0 --force $VERSION"
        exit 1
    fi
fi

echo "Creating local git tag: $VERSION"
git tag -a "$VERSION" -m "Release $VERSION"

echo ""
echo "Tag created locally: $VERSION"
echo "Push manually with: git push origin $VERSION"
echo "Or force push with: git push --force origin $VERSION"
