# Testing

## Unit tests

Use standard `go test` table-driven tests for:

- list filter parsing
- PATCH body parsing
- service-level cache behavior
- MCP tool argument normalization and MCP tool behavior, including note updates, delete/state helpers, and tag discovery/mutation helpers
- OIDC/session boundary behavior with fakes
- HTML form parsing and minimal UI helpers
- config parsing and helper behavior
- conversion helpers that map pgx/sqlc types into API-facing models

## Handler tests

`httptest` is enough for most route-level behavior because the handlers depend on services instead of talking directly to the database.

That includes both:

- JSON API routes
- HTML/HTMX routes for the minimal web UI

## Integration tests

Integration tests should be enabled when these environment variables are present:

- `DATABASE_URL`
- `VALKEY_ADDR`

Recommended local setup:

```bash
make docker-up
make migrate-up
make test
make test-integration
```

Project targets:

```bash
make test
make test-integration
make coverage
make coverage-integration
make coverage-check
make coverage-check-integration
```

For MCP-specific local development:

```bash
make run-mcp
```

For MCP packaging work:

```bash
make release-check-mcp
make release-snapshot-mcp
```

Current status:

- `make coverage-check-integration` is the strongest project gate right now because it includes the Docker-backed integration suite.
- The project target remains `80%+` integration-backed handwritten-code coverage.
- Feature work should leave the repo with this gate passing before it is considered fully complete.
- The current baseline is back above the threshold at `85%+`.
- The coverage commands now measure the handwritten packages directly instead of using a duplicated cross-package profile, which keeps the gate aligned with the actual packages we maintain.

Suggested coverage:

- migrations apply cleanly
- sqlc queries behave as expected
- session storage round-trips in Valkey
- note CRUD works against real PostgreSQL
- filtering, sorting, pagination, and total counts behave as expected
- plain substring search and PostgreSQL full-text search behave as expected
- ranked relevance sorting is only accepted with the explicit full-text search mode
- owner-scoped tag summaries are aggregated correctly from PostgreSQL
- related-note discovery is ranked correctly from PostgreSQL tag overlap and never crosses owner boundaries
- saved queries persist, list, load, and delete correctly in PostgreSQL
- owner-scoped notes stay unreadable, unpatchable, and undeletable by a different authenticated user
- note, shared-note, and list cache behavior is observable against real Valkey-backed services
- public shared-note responses do not expose internal note or owner identifiers
- MCP tool handlers are covered with unit tests against the shared note service boundary
- saved-query behavior is covered in REST, UI helper, MCP, and integration tests
- the minimal web UI is covered for guest rendering, authenticated rendering, and note create/read/update flows

## Coverage policy

- Generated code is excluded from the handwritten-code coverage target.
- Thin command entrypoints in `cmd/api` and `cmd/mcp` are also excluded so the gate measures shared teaching logic rather than bootstrap wrappers.
- The project target is `80%` or higher handwritten-code coverage.
- Use `make coverage` to inspect the current total.
- Use `make coverage-check` to enforce the threshold.
- Use `make coverage-integration` when you want the handwritten-code total with Docker-backed integration tests included.
- Use `make coverage-check-integration` to enforce the threshold against the integration-backed coverage run.
