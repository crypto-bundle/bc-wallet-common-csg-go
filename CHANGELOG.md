# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v0.0.6] — 2026-09-26
### Added
- Detailed function writing rules (signatures, parameters, body organisation)
- Error formatting and handling best practices
- Error code classification rules

### Changed
- gRPC client/server documentation with health check guidance
- PostgreSQL/sqlx operation examples (SELECT, INSERT, UPDATE, DELETE, transactions)
- Code layout conventions and repository workflow guidelines

### Fixed
- Various corrections in existing documentation

> Original commits: bcef8d2 61dff4f f356c47 dd13303 dba0da1 8526940 4f5411d

## [v0.0.5] — 2026-09-25
### Added
- Service component pattern with factory and unified EFS (Embedded File System)
- Comprehensive application development docs: architecture, DI, code-writing, gRPC, HTTP, NATS, PostgreSQL/sqlx

### Changed
- Complete restructuring of `pkg/csg/` into service-component with embedded documentation
- Deep reorganisation of `docs/` and `pkg/csg/docs/` subdirectories (~120+ files)

> Change committed in: dfee5eb

## [v0.0.4] — 2026-03-08
### Added
- Go module (`go.mod`) for import as dependency
- Embedded CSG documentation bundle under `pkg/csg/docs/`
- CLI entrypoint (`cmd/csg/main.go`) for serving documentation references

> Change committed in: 5d4098c

## [v0.0.3] — 2025-06-13
### Changed
- Major documentation reorganisation: renamed and regrouped all markdown files
- Updated project README to match new structure

> Change committed in: 9498610

## [v0.0.2] — 2025-04-17
### Added
- Error codes documentation and coding rules

### Fixed
- Insert one record request example correction

> Change committed in: b4771f1

## [v0.0.1] — 2024-12-27
### Added
- Initial CSG documentation package migrated from ca-developer-tasks repository
- PostgreSQL/sqlx examples: SELECT, INSERT, UPDATE, DELETE, transactions
- Git setup guide, git flow conventions, GitHub code review process
- Toolkit references: copywrite, golangci-lint, migrator, gvm
- Repository layout guides: builders, license management, CI/CD standards

> Root commit: 008b4af
