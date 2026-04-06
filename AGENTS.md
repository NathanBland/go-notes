# go-notes Agent Guide

This file captures the working rules for agents and contributors on this project.

## Mission

`go-notes` is a public teaching API. Every change should optimize for:

- correctness
- clarity
- teaching value
- maintainable Go conventions
- practical local development
- up-to-date documentation and code comments

The code should stay small enough to study, but complete enough to model real API work.

## Documentation rule

Use Context7 for current documentation whenever work touches a library, framework, SDK, CLI, database, cache client, container tooling, or auth provider.

That includes:

- Go toolchain behavior when the answer depends on current version details
- `sqlc`
- `pgx`
- PostgreSQL-related client usage
- `valkey-go`
- Docker / Docker Compose
- OIDC / `go-oidc`
- `golang.org/x/oauth2`
- test tooling or coverage workflows

Prefer primary docs and examples over memory.

Keep project documentation in sync with the implementation:

- update `README.md` when setup, workflow, security posture, or local development behavior changes
- update `README.md` and `ROADMAP.md` when roadmap items are completed, re-scoped, added, or otherwise materially changed
- update `CHANGELOG.md` when feature work changes the user-visible, operator-visible, or teaching-visible surface in a way that should be reflected in `Unreleased`
- review the relevant files in `docs/` after every feature is completed, and update them in the same change whenever behavior, setup, architecture, auth, testing, or teaching guidance has shifted
- when the container/runtime shape changes materially, update the dev compose file, the production compose file, and the README compose example together so deployment guidance does not drift
- production container changes must preserve or deliberately update the migration startup gate; the production API should not start against PostgreSQL until migrations have completed successfully
- create a feature plan in `plans/` for each new feature before substantial implementation begins
- update the relevant files under `docs/` when API behavior, architecture, auth flow, filtering, pagination, or testing expectations change
- update teaching comments in code when the implementation changes how nullable fields, UTC timestamps, cache behavior, auth flow, or SQL mapping work
- do not leave docs or comments describing an older design after the code has moved on

## Feature planning rule

- Each new feature should get its own markdown plan file in `plans/`.
- Name plan files clearly so they are easy to scan, for example `plans/throttling-feature-plan.md`.
- Each feature plan should include:
  - the goal of the task
  - what is required to build it
  - the implementation plan
  - acceptance criteria that can be used to verify the work is complete
  - a version-impact assessment, such as `none`, `patch`, `minor`, `major`, or `release-prep`
  - whether `CHANGELOG.md` should be updated as part of the feature
- Update the feature plan if the scope materially changes while the work is in progress.
- Treat the feature plan as part of the feature deliverable, not optional prep work.
- Unless the task is explicitly about cutting a release, do not invent a new version tag during ordinary feature work. Record the expected impact and update `CHANGELOG.md` instead.
- When a task does participate in a real version increment, include the concrete release steps in the work:
  - confirm the next version from the strategy in [`VERSIONING.md`](VERSIONING.md)
  - update `CHANGELOG.md` by promoting `Unreleased` entries into the new versioned section
  - update explicit version surfaces such as [`docs/openapi.yaml`](docs/openapi.yaml) when they should match the release
  - verify release tooling assumptions, including GoReleaser config and tag format

## Repository docs link rule

- Markdown links written inside repository files must use repo-relative paths that resolve from the file that contains the link, not absolute filesystem paths from the local machine.
- Keep canonical documentation in its natural home, usually under `docs/`, instead of adding wrapper markdown files at the repository root just to work around a broken link.
- Prefer linking directly to the real target document instead of adding duplicate entry-point docs unless there is a clear discoverability reason.
- When moving or renaming docs, update every inbound markdown link in the same change.
- Before finishing docs work, run a quick markdown-link sanity check so internal repo links resolve both on GitHub and from a fresh clone.

## Architecture rules

- Prefer standard library HTTP with `net/http` and `ServeMux`.
- Keep handlers thin.
- Put business behavior in small services under `internal/...`.
- Keep SQL explicit and reviewable in `sql/queries`.
- Use `sqlc` as the typed query layer.
- Use `pgxpool` for PostgreSQL access.
- Keep a small repository layer at the DB boundary so HTTP and service inputs can be translated into typed SQL parameters without leaking transport concerns into SQL files.
- Use Valkey explicitly for cache/session behavior instead of hiding it behind abstractions that remove teaching value.
- Keep exported types, interfaces, and constructors documented when they form part of the teaching surface of the project.

## SQL and repository rules

- Let SQL do most of the heavy lifting for filtering, sorting, pagination, and total counts.
- Keep repository methods responsible for adapting validated Go inputs into `sqlc` parameter structs and mapping generated rows back into domain types.
- Never build SQL with string concatenation from request data.
- Prefer `sqlc.arg(...)` and `sqlc.narg(...)` for all user-controlled values so PostgreSQL receives bound parameters instead of interpolated SQL.
- Treat `ORDER BY` as a hard allowlist problem. Only approved sort fields and sort directions may reach the query layer.
- Keep search and filter logic in SQL where PostgreSQL can use indexes and produce authoritative counts.
- Use stable ordering for paginated queries by ending with a deterministic tie-breaker.
- When adding new filters, update the list query, count query, request validation, and tests together.
- Treat applied migrations as append-only history.
- The initial baseline schema belongs in the first migration, such as `000001_init`.
- When the schema changes, add a new numbered migration instead of rewriting an older migration that may already have run on a developer or teaching database.
- Schema changes must be delivered as a paired migration set with both `*.up.sql` and `*.down.sql` files.
- Once the project has moved beyond initial bootstrap, do not edit `000001_init` or any other past migration to represent a new feature or fix.
- Assume newer code may need to be applied to an older local database, so schema evolution should always be expressed through new migrations that can be applied in order.
- Avoid migration patterns that drop or recreate existing tables just to evolve schema shape unless the user explicitly asks for destructive reset behavior.
- Prefer forward-only schema evolution that preserves existing notes and user data, and reflect rollback behavior explicitly in the paired down migration.

## Security and abuse protection

- Validate and normalize request input at the HTTP boundary before it reaches services or SQL parameters.
- Cap request body sizes and reject unknown JSON fields.
- Avoid returning raw PostgreSQL, Valkey, or OIDC provider errors to clients.
- Keep auth required on owner-scoped routes and use explicit public routes for intentionally shared resources only.
- Session cookies must stay opaque, `HttpOnly`, `SameSite=Lax`, and `Secure` when HTTPS is enabled.
- Add throttling around abuse-prone endpoints like login, callback, and public note lookups so expensive downstream work is protected.
- Prefer failing early on malformed input, unsupported sort fields, oversized page sizes, and invalid timestamps.

## Filtering and pagination rules

- Default list behavior: owner-scoped notes, `status=active`, sorted by `updated_at desc`, page `1`, page size `20`.
- Maximum page size is `100` unless a future design document explicitly changes it.
- Sorting is limited to documented fields and directions only.
- Pagination metadata should come from a dedicated SQL count query rather than guessed client-side lengths.
- Filtering should stay server-side for search, tags, archived/shared state, and date windows.
- Normalize date filters to UTC before passing them to SQL.
- When pagination behavior changes, cover it with table-driven tests and integration tests.
- Keep the filtering and pagination docs aligned with the handler validation rules and SQL queries.

## Go-specific implementation rules

- Store timestamps in PostgreSQL as `timestamptz`.
- Force PostgreSQL sessions to UTC on connect.
- Serialize API timestamps in RFC3339/UTC.
- Preserve nullable SQL fields as Go pointers where that teaches important semantics.
- For PATCH-style updates, use pointer fields plus explicit `Set` flags so omitted fields, zero values, and explicit `null` stay distinct.
- Add concise comments on non-obvious Go behaviors, especially:
  - pointer/null handling
  - omitted vs zero values
  - UTC time normalization
  - context propagation
  - cache invalidation behavior
- Add or refresh doc comments on exported structs, interfaces, and constructors when their responsibilities change.
- Keep comments concise and instructional. Explain why the code is shaped this way, especially where Go differs from JavaScript or other dynamic-language APIs.

## Auth and startup rules

- OIDC provider discovery should not make the API fail to boot for local development.
- Prefer lazy external initialization where it improves developer experience without weakening correctness.
- Session cookies should stay opaque and server-backed.

## Data and cache rules

- Prefer integration tests for PostgreSQL and Valkey behavior instead of mocking everything.
- Cache invalidation must be deliberate and documented in code or tests.
- Avoid “magic” ORM patterns; keep data flow readable.
- Cache keys for list endpoints should reflect validated filter inputs so cached pages cannot bleed between users or query variants.

## Testing policy

- Treat tests as part of the feature, not follow-up work.
- New behavior should ship with unit tests and, when storage/cache behavior is involved, integration tests.
- Completed work should leave the handwritten-code coverage gate at or above 80%, including the integration-backed coverage run.
- Do not count generated code toward the coverage target.
- Do not accept changes that reduce meaningful coverage without a strong reason.
- When a task changes behavior, consider documentation updates part of completion alongside tests and coverage.
- When a task has non-`none` version impact, consider the `CHANGELOG.md` update part of completion alongside the docs review.
- A feature is not complete until `docs/` has been reviewed and any impacted documents have been updated.
- A feature is not complete until the relevant test commands have actually been run, not just reasoned about.
- Unless the change is docs-only or the user explicitly says otherwise, run `make test`.
- When the project has Docker-backed integration coverage available for the changed area, run `make test-integration`.
- Before calling a feature complete, run `make coverage-check-integration` so the integration-backed handwritten-code coverage gate stays honest.

### Minimum test expectations by change type

- Handler changes:
  - `httptest` coverage for success and failure paths
  - response envelope checks
  - auth/validation checks where relevant
- Service changes:
  - table-driven tests for business rules
  - cache hit/miss/invalidation behavior where relevant
- SQL / Postgres changes:
  - integration tests against a real PostgreSQL instance
  - migration sanity checks when schema changes
- Valkey changes:
  - integration tests against a real Valkey instance
  - verify round-trip and invalidation behavior

## Local workflow

Preferred local dev commands:

```bash
make docker-up-app
make migrate-up
make test
make test-integration
make coverage
make coverage-check
```

Hot reload uses `air` inside the Compose `api` service. Save a Go file and the API should rebuild automatically.

## Coverage and integration commands

- `make test`: unit tests
- `make test-integration`: integration tests against local Docker services
- `make coverage`: generate a coverage profile for handwritten code
- `make coverage-check`: fail if coverage is below the configured threshold

## Generated code

Generated files should remain generated:

- `internal/platform/db/sqlc/*`

Do not hand-edit generated `sqlc` output unless the task is explicitly about generation debugging.

## When unsure

- choose explicit SQL over indirection
- choose clearer comments over cleverness
- choose real integration tests over brittle mocks for DB/cache behavior
- choose Context7 docs over stale memory
