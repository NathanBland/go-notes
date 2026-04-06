# go-notes Plan Snapshot

This document is now a concise snapshot of the implemented baseline rather than the active roadmap. For current next-step planning, use [ROADMAP.md](../ROADMAP.md) and the feature plans under `plans/`.

## Current baseline

`go-notes` is now a public teaching API with:

- `net/http` + `ServeMux`
- PostgreSQL + `sqlc` + `pgx/v5`
- Valkey-backed sessions and cache-aside note caching
- OIDC authorization code flow with PKCE
- server-side filtering, sorting, and pagination
- Docker-based local PostgreSQL, Valkey, migrations, and hot-reload API workflow

## Implemented API surface

- `GET /api/v1/healthz`
- `GET /api/v1/auth/login`
- `GET /api/v1/auth/callback`
- `GET /api/v1/auth/me`
- `POST /api/v1/auth/logout`
- `POST /api/v1/notes`
- `GET /api/v1/notes`
- `GET /api/v1/notes/{id}`
- `PATCH /api/v1/notes/{id}`
- `DELETE /api/v1/notes/{id}`
- `GET /api/v1/notes/shared/{slug}`

## Implemented list behavior

- `page`
- `page_size`
- `search`
- `status=active|archived|all`
- `shared=true|false`
- `tag`
- `sort=created_at|updated_at|title`
- `order=asc|desc`
- `created_before`
- `created_after`
- `updated_before`
- `updated_after`

Defaults:

- page: `1`
- page size: `20`
- max page size: `100`
- sort: `updated_at desc`
- status: `active`

## Design choices that stayed true

- Keep handlers thin and push business logic into services.
- Keep SQL explicit and reviewed through checked-in query files.
- Let PostgreSQL do the heavy lifting for filtering, sorting, pagination, and counts.
- Use a small repository layer only to adapt validated Go inputs into typed `sqlc` parameters.
- Keep cache behavior explicit so invalidation is visible and teachable.
- Keep timestamps in UTC and represent nullable DB fields with pointers where the distinction matters.

## Current quality bar

- Request throttling is in place for abuse-prone public auth/shared routes.
- Docker-backed integration tests cover CRUD, filtering, pagination, and cache behavior.
- Handwritten-code coverage is enforced and currently passes above the `80%` project target.

## Where to look next

- [README.md](../README.md) for the current developer workflow
- [docs/architecture.md](architecture.md) for the current shape of the codebase
- [docs/testing.md](testing.md) for the current test and coverage workflow
- [ROADMAP.md](../ROADMAP.md) for upcoming work

- handler tests with `httptest`
- table-driven tests for success and failure cases
- response envelope assertions

### Integration tests

- sqlc-generated queries against a local PostgreSQL instance
- cache integration tests against local Valkey

Keep business logic testable without requiring the server to run.

## Delivery phases

### Phase 0: foundation

- create repo layout
- add Makefile and dependency bootstrap
- add sqlc config
- write schema and query plan

### Phase 1: core API

- health endpoint
- note CRUD
- PostgreSQL connection wiring
- sqlc code generation
- standard JSON helpers

### Phase 2: query ergonomics

- paging
- sorting
- filter/search
- response metadata

### Phase 3: production-minded basics

- validation
- structured logging
- graceful shutdown
- request IDs / recovery middleware
- tests for handlers and services

### Phase 4: caching

- Valkey client wiring
- note-by-id cache
- list cache for filtered queries
- invalidation on writes

### Phase 5: parity-plus extensions

- auth
- HTTPS examples
- optional OAuth example
- Docker and compose for local stacks

## Immediate next build steps

1. initialize the Go module and choose the final module path
2. write the first schema migration for `notes`
3. add `sqlc` queries and generate code
4. build `GET /healthz` and `POST /api/v1/notes`
5. add `GET /api/v1/notes` with pagination, sorting, and filtering

## Sources

- [Go by Example](https://gobyexample.com/)
- [Go by Example: HTTP Server](https://gobyexample.com/http-server)
- [Go by Example: JSON](https://gobyexample.com/json)
- [Go by Example: Context](https://gobyexample.com/context)
- [Go by Example: Logging](https://gobyexample.com/logging)
- [Go by Example: Testing and Benchmarking](https://gobyexample.com/testing-and-benchmarking)
- [Nathan Bland DEV series: Building a JSON API](https://dev.to/nathanbland/episode-9-building-a-json-api---filtersearch-5fi8)
