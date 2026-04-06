# Changelog

All notable changes to `go-notes` should be recorded in this file.

The project uses an `Unreleased` section during feature work and promotes those entries into a versioned section when a release tag is cut.

## [Unreleased]

## [0.1.2] - 2026-04-06

### Changed

- Production Compose deployments now run the migration service as a default startup gate, and the API waits for migrations to complete successfully before starting.

## [0.1.1] - 2026-04-06

### Changed

- Deployment documentation links now point directly to the canonical guide in [`docs/deployment.md`](docs/deployment.md), so they work cleanly on GitHub and in local clones.
- The unauthenticated landing page now describes the running service experience around private Markdown notes, tag filtering, saved searches, intentional sharing, and optional API/agent access, while only linking to routes served by the running app.

## [0.1.0] - 2026-04-06

### Added

- Versioning and changelog guidance now live in [`VERSIONING.md`](VERSIONING.md).
- A production deployment path now exists with published-image workflows, production Dockerfiles, and a Portainer-friendly compose file.
- Initial public baseline for the REST API, server-rendered UI, PostgreSQL persistence, Valkey caching, OIDC login flow, and stdio MCP support is now tracked here pending the first tagged release.
- Owner-scoped tag rename is now available through REST, the server-rendered UI, and MCP with shared PostgreSQL-backed rewrite rules.

### Changed

- Contributors now track version impact and changelog expectations as part of feature planning and completion.
- The docs now include clearer ownership, cache-behavior, error-envelope, HTTPS, reverse-proxy, cookie, and production throttling guidance.
