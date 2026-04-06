# Integration Coverage Feature Plan

## Goal

Expand the Docker-backed integration suite so `go-notes` verifies real note CRUD behavior, server-side filtering and pagination, and cache behavior against PostgreSQL and Valkey.

## What is required

- A reusable integration test setup for PostgreSQL, Valkey, and a clean database state
- Real note lifecycle tests that exercise the `notes.Service` and DB store together
- Filter and pagination tests that prove PostgreSQL is doing the expected query work
- Cache assertions that verify note, shared-note, and list cache behavior where it is externally observable
- Documentation updates describing the broader integration coverage

## Implementation plan

1. Add integration-test helpers for environment setup, table reset, and test user creation.
2. Add a CRUD-focused integration test that covers create, get, patch, share/unshare, and delete behavior.
3. Add filtering and pagination integration tests that verify owner scoping, status/shared/tag/search filters, date windows, sorting, and totals.
4. Assert cache behavior by checking read-after-write behavior and inspecting expected cache keys where practical.
5. Update the README, roadmap, and testing docs once the new integration coverage is complete.
6. Run unit and integration tests to verify the feature end to end.

## Acceptance criteria

- Integration tests verify note CRUD behavior against real PostgreSQL and Valkey services.
- Integration tests verify filtering, sorting, pagination, and total-count behavior against real PostgreSQL queries.
- Integration tests verify at least the observable note, shared-note, and list cache behavior.
- `make test` passes.
- `make test-integration` passes.
- The README, roadmap, and testing docs reflect the expanded integration coverage.
