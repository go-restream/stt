# Version Management Guide

**🌐 Language:** [English](VERSION_EN.md) | [中文](VERSION.md)

This document describes the version management specifications and usage methods for the streamASR project.

## Version Format

The project uses Semantic Versioning format: `vMAJOR.MINOR.PATCH`

- **MAJOR**: Incompatible API changes
- **MINOR**: Backward compatible functional additions
- **PATCH**: Backward compatible bug fixes

Current version: **v0.1.1**

## Version Management Commands

### View Version Information

```bash
# View current version information
make version

# Show only version number
make version-show

# View all Git tags
make tag-list
```

### Version Upgrades

```bash
# Patch version upgrade (v0.1.1 -> v0.1.2)
make version-bump-patch

# Minor version upgrade (v0.1.1 -> v0.2.0)
make version-bump-minor

# Major version upgrade (v0.1.1 -> v1.0.0)
make version-bump-major

# Manually set version
make version-set NEW_VERSION=v0.2.0
```

### Git Tag Management

```bash
# Create Git tag for current version
make tag

# View all version tags
make tag-list
```

## Build-time Version Information

Version information is automatically injected into the binary at build time:

- `version`: Version number (read from VERSION file)
- `buildTime`: Build time (ISO 8601 format)
- `gitCommit`: Git commit hash (short format)

### Runtime Version Viewing

```bash
# View version after build
./build/streamASR -v
./build/streamASR --version

# Version information is also displayed in startup logs
./build/streamASR -c config.yaml
```

## Docker Version Management

### Build Versioned Docker Images

```bash
# Build production version image (automatically creates two tags)
make docker-build
# Generates: streamasr:latest and streamasr:v0.1.1

# Build development version image
make docker-build-dev
# Generates: streamasr:dev

# Build versioned image via docker-compose
make docker-compose-build
```

### Docker Image Version Tags

Docker images use the following tag strategy:

- `streamasr:latest` - Latest version
- `streamasr:v0.1.1` - Specific version number
- `streamasr:dev` - Development version

## Release Process

### Development Version Release

1. Update version number:
   ```bash
   make version-bump-patch
   ```

2. Build and test:
   ```bash
   make clean
   make build
   make test
   ```

3. Create Git tag:
   ```bash
   make tag
   ```

4. Build Docker image:
   ```bash
   make docker-build
   ```

### Production Version Release

1. Update version number (choose based on change type):
   ```bash
   make version-bump-patch    # Fix version
   make version-bump-minor    # Feature version
   make version-bump-major    # Breaking change version
   ```

2. Commit code changes:
   ```bash
   git add VERSION
   git commit -m "Bump version to v0.1.2"
   ```

3. Create tag:
   ```bash
   make tag
   ```

4. Build production image:
   ```bash
   make docker-deploy
   ```

## Version Examples

### Patch Version Release (v0.1.1 -> v0.1.2)

```bash
# 1. Upgrade patch version
make version-bump-patch

# 2. Commit changes
git add VERSION
git commit -m "Bump version to v0.1.2"

# 3. Create tag
make tag

# 4. Build and deploy
make docker-deploy
```

### Minor Version Release (v0.1.1 -> v0.2.0)

```bash
# 1. Upgrade minor version
make version-bump-minor

# 2. Commit changes
git add VERSION
git commit -m "Add new feature, bump version to v0.2.0"

# 3. Create tag
make tag

# 4. Build and deploy
make docker-deploy
```

## Important Notes

1. **Version File**: Version information is stored in the `VERSION` file, do not edit manually
2. **Git Status**: Ensure working directory is clean before creating tags
3. **Tag Push**: `make tag` automatically pushes tags to remote repository
4. **Build Order**: Upgrade version first, then build images to ensure correct version information
5. **Version Rollback**: If you need to rollback version:
   ```bash
   git checkout v0.1.1  # Checkout to specific tag
   make version-set NEW_VERSION=v0.1.1  # Reset version file
   ```

## Environment Variables

In CI/CD environments, version information can be overridden via environment variables:

```bash
export VERSION="v0.1.2-custom"
export BUILD_TIME="2024-01-01T00:00:00Z"
export GIT_COMMIT="abc123def"

make docker-build
```