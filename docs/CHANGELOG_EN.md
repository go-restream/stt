# Changelog

**🌐 Language:** [English](CHANGELOG_EN.md) | [中文](CHANGELOG.md)

This document records all important changes to the streamASR project.

Format based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project follows [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added
- Docker containerization support
- Complete version management system
- Automated build scripts

### Changed
- Project structure optimization
- Configuration file format standardization

## [v0.1.2] - 2025-11-02

### Added
- Version management system
- Support for automatic version number injection and display
- Docker and Docker Compose configuration
- Makefile integration of version management commands
- Startup log displays version information
- Command-line version viewing feature (`-v` and `--version`)

### Changed
- Optimized build process to support version information injection
- Updated Dockerfile to support multi-stage builds
- Improved Makefile to support cross-platform builds

### Documentation
- Added Docker deployment guide
- Added version management documentation
- Updated project documentation structure

## [v0.1.1] - 2024-XX-XX

### Added
- Initial version release
- OpenAI Realtime API integration
- WebSocket server support
- VAD (Voice Activity Detection) functionality
- Audio processing and resampling
- Health check system
- Configuration file management
- Structured logging

---

## Version Number Description

- **Major Version**: Incompatible API changes
- **Minor Version**: Backward compatible functional additions
- **Patch Version**: Backward compatible bug fixes

## Version Management Commands

```bash
# View version information
make version

# Version upgrades
make version-bump-patch    # v0.1.1 -> v0.1.2
make version-bump-minor    # v0.1.2 -> v0.2.0
make version-bump-major    # v0.2.0 -> v1.0.0

# Create Git tag
make tag

# Build Docker image
make docker-build
```