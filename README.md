# go-notes

`go-notes` is a public teaching project for building a practical REST API in Go with:

- `net/http` and `ServeMux`
- PostgreSQL
- `sqlc` + `pgx/v5`
- Valkey
- `air` for hot reload in Docker-based development
- OIDC with authorization code flow + PKCE

The goal is to keep the code small enough to study, while still covering real API concerns like filtering, pagination, caching, structured errors, secure cookies, and graceful shutdown.

## Current status

What is available today:

- `GET /api/v1/healthz`
- OIDC login flow endpoints:
  - `GET /api/v1/auth/login`
  - `GET /api/v1/auth/callback`
  - `GET /api/v1/auth/me`
  - `POST /api/v1/auth/logout`
- Note endpoints:
  - `POST /api/v1/notes`
  - `GET /api/v1/notes`
  - `GET /api/v1/notes/{id}`
  - `PATCH /api/v1/notes/{id}`
  - `DELETE /api/v1/notes/{id}`
  - `GET /api/v1/notes/shared/{slug}`
- Saved query endpoints:
  - `GET /api/v1/saved-queries`
  - `POST /api/v1/saved-queries`
  - `DELETE /api/v1/saved-queries/{id}`
- PostgreSQL-backed persistence through checked-in SQL and generated `sqlc` code
- Valkey-backed session storage, pending OIDC state storage, note cache, shared-note cache, and list cache
- Request throttling on login, callback, and public shared-note routes
- Expanded Docker-backed integration coverage for note CRUD, filtering, pagination, and observable cache behavior
- Coverage tooling and an `80%` handwritten-code target enforced through `make coverage-check-integration`, currently passing at `85.3%`
- GoReleaser-based packaging for installable `go-notes-mcp` binaries on macOS and Linux, including checksums and snapshot build support
- Server-side filtering, sorting, and pagination:
  - `page`
  - `page_size`
  - `saved_query_id`
  - `search`
  - `search_mode`
  - `status`
  - `shared`
  - `has_title`
  - `tag`
  - `tags`
  - `tag_count_min`
  - `tag_count_max`
  - `tag_mode`
  - `sort`
  - `order`
  - `created_before`
  - `created_after`
  - `updated_before`
  - `updated_after`
- Multi-tag filtering with `tag_mode=any|all` plus deterministic `sort=tag_count` and `sort=primary_tag`, with PostgreSQL doing the authoritative tag matching and ordering work
- Advanced note search now supports both plain substring matching and PostgreSQL full-text search with `sort=relevance`
- Owner-scoped saved queries can store canonical list filters in PostgreSQL and be reused from REST, the web UI, and MCP with `saved_query_id`
- Public shared-note responses that intentionally omit internal note IDs and owner IDs so a share link does not leak private identifiers
- Docker-based local development for PostgreSQL, Valkey, migrations, and the hot-reloading API
- A first local MCP interface over stdio with note, tag, saved-query, and related-note discovery tools, including `find_related_notes`
- A minimal server-rendered web interface with HTMX and Tailwind CSS for login, note reading, note creation, and note updates
- Unit tests, integration tests, coverage tooling, and project rules captured in [AGENTS.md](/Users/nathanbland/projects/codex-workspace/go-notes/AGENTS.md)

The current roadmap lives in [ROADMAP.md](/Users/nathanbland/projects/codex-workspace/go-notes/ROADMAP.md), including near-term API work, testing goals, the new MCP/LLM direction, and follow-up work around coverage and richer tag-aware behavior.
That roadmap now also includes a minimal HTMX + Tailwind CSS web interface for local login and note interactions, while keeping the REST API as the primary teaching surface.

OIDC provider connectivity stays env-driven. The project is intended to connect to an external provider through configured environment variables rather than running an OIDC provider inside this repository's Docker Compose stack.

Recent completion:

- request throttling is now in place for `GET /api/v1/auth/login`, `GET /api/v1/auth/callback`, and `GET /api/v1/notes/shared/{slug}`
- throttle settings are configurable with `THROTTLE_REQUESTS_PER_SECOND` and `THROTTLE_BURST`
- integration tests now verify real note CRUD, filtering, pagination, and cache behavior against PostgreSQL and Valkey
- the OIDC setup docs and env example now assume an external provider and explain issuer discovery, PKCE, redirect URIs, and when a client secret is optional
- the MCP slice now runs over stdio and reuses the existing note service for note CRUD, note-state updates, and owner-scoped tag discovery
- the project now has a small browser-based UI at `/` that reuses the existing auth and notes services instead of introducing a separate frontend app
- tag filtering now works consistently across REST, MCP, and the web UI, including multi-tag `any`/`all` matching and SQL-driven `tag_count` sorting
- tag-oriented sorting now also includes `primary_tag`, which orders notes by the first stored tag without moving sort logic out of PostgreSQL
- the project intentionally stops there for tag-derived sorts; when a tag-filtered result set needs a human-friendly order, the recommended pattern is still `sort=title`
- advanced note filtering now also includes explicit `has_title` and tag-count range filters, while full-text search stays opt-in through `search_mode=fts`
- saved queries now store normalized owner-scoped list presets and replay them through the same validation path before explicit request values override them
- MCP now includes `find_related_notes`, which ranks owner-scoped related notes by overlapping tags in PostgreSQL and returns the shared tags that explain each match
- the integration-backed handwritten-code coverage gate is green again after the recent UI, tag, and MCP additions
- the security audit pass now includes regression coverage for cross-owner note access and a hardened shared-note response shape
- the MCP server now has validated GoReleaser packaging plus install docs for Codex, Claude Code, Cursor, and Windsurf

## Docs

- [Roadmap](/Users/nathanbland/projects/codex-workspace/go-notes/ROADMAP.md)
- [Parity matrix](docs/parity-matrix.md)
- [Architecture](docs/architecture.md)
- [API contract](docs/api.md)
- [OpenAPI](docs/openapi.yaml)
- [Filtering and pagination](docs/filtering-pagination.md)
- [MCP](docs/mcp.md)
- [MCP install](docs/mcp-install.md)
- [Web UI](docs/web-ui.md)
- [OIDC and Keycloak](docs/oidc-keycloak.md)
- [Testing](docs/testing.md)
- [Go for Node developers](docs/go-for-node-devs.md)

## Quickstart

1. Copy `.env.example` to `.env` and adjust the values.
   The OIDC values should point at your external provider's issuer URL and registered client settings.
   `make run` and `make run-mcp` automatically load `.env` and `.env.local` when those files exist.
2. Install host dependencies:

```bash
make deps
```

3. Start PostgreSQL and Valkey in Docker:

```bash
make docker-up
```

If `make docker-up` says the daemon is not running, start Docker Desktop or `dockerd` first.

4. Apply migrations:

```bash
make migrate-up
```

5. Generate the typed SQL layer and tidy modules:

```bash
make sqlc-generate
make go-mod-tidy
```

6. Run the API:

```bash
make run
```

Then visit [http://localhost:8080/](http://localhost:8080/) for the minimal web UI or [http://localhost:8080/api/v1/healthz](http://localhost:8080/api/v1/healthz) for the API health check.

7. Stop the containers when you are done:

```bash
make docker-down
```

## Local workflow

```bash
make check-deps
make docker-up
make migrate-up
make sqlc-generate
make test
make test-integration
make coverage
make coverage-integration
make coverage-check-integration
make release-check-mcp
make release-snapshot-mcp
make run
make run-mcp
```

## Hot reload with Docker

The recommended dev reload tool is `air`. It watches Go files, rebuilds `cmd/api`, and restarts the process when you save.

To run the full stack, including the hot-reloading API, in Docker:

```bash
make docker-up-app
make migrate-up
make docker-logs-api
```

Then hit [http://localhost:8080/api/v1/healthz](http://localhost:8080/api/v1/healthz).

Key files:

- [`.air.toml`](/Users/nathanbland/projects/codex-workspace/go-notes/.air.toml)
- [`Dockerfile.dev`](/Users/nathanbland/projects/codex-workspace/go-notes/Dockerfile.dev)
- [`docker-compose.yml`](/Users/nathanbland/projects/codex-workspace/go-notes/docker-compose.yml)

## MCP

The project now includes a first MCP server over stdio for local LLM tooling.

Current MCP tools:

- `list_notes`
- `get_note`
- `find_related_notes`
- `create_note`
- `update_note`
- `delete_note`
- `list_saved_queries`
- `save_query`
- `delete_saved_query`
- `list_tags`
- `set_note_tags`
- `share_note`
- `unshare_note`
- `archive_note`
- `unarchive_note`
- `add_note_tags`
- `remove_note_tags`

Run it with:

```bash
make run-mcp
```

This first MCP slice is local-only and currently uses `MCP_OWNER_USER_ID` to scope note access to one configured owner until a fuller MCP-specific auth model is added.

For local development, both `make run` and `make run-mcp` automatically source `.env` first and then `.env.local`, so `.env.local` can safely override machine-specific values.

If you want a packaged binary instead of `go run`, the project now also includes GoReleaser scaffolding for `go-notes-mcp`. See [MCP install](/Users/nathanbland/projects/codex-workspace/go-notes/docs/mcp-install.md) for snapshot builds and client setup examples for Codex, Claude Code, Cursor, and Windsurf.

## Web UI

The minimal browser UI now lives at `/`.

It is intentionally small and teaching-focused:

- unauthenticated visitors see a landing page with an OIDC login action
- authenticated users see a server-rendered notes workspace
- HTMX is used for targeted note reads and edit/create updates
- Tailwind CSS is loaded from the CDN to keep the asset setup light
- the UI login action uses the same OIDC callback endpoint, but carries a safe local redirect target so successful browser login returns to `/`
- note content is stored as raw Markdown and rendered to HTML only in the browser-facing read view
- rendered Markdown now gets a built-in prose-style treatment for headings, lists, quotes, code blocks, and tables in the UI
- the workspace now includes tag browsing controls, preserved filter state during note edits, and clickable note tags that re-filter the list
- the workspace also includes a saved-query card so the current filter set can be named, reused, and deleted without inventing a second client-side filter language

The JSON API is still the primary teaching surface. The HTML UI exists to make local learning and demos easier without introducing a separate SPA.

## API guardrails

- Keep handlers focused on HTTP concerns like decoding, validation, auth, and response formatting.
- Use the repository layer in [`internal/platform/db`](/Users/nathanbland/projects/codex-workspace/go-notes/internal/platform/db) to translate validated route inputs into typed `sqlc` parameters.
- Let PostgreSQL do the heavy lifting for filtering, sorting, pagination, and counting so the API stays consistent and efficient under load.
- Never build SQL by concatenating raw user input. All user-controlled values should stay in bound parameters through `sqlc.arg(...)` or `sqlc.narg(...)`.
- Treat sort fields and directions as strict allowlists. `ORDER BY` is one of the easiest places to accidentally re-introduce SQL injection if raw strings leak through.
- Keep pagination bounded. Default list behavior is page `1`, page size `20`, max page size `100`.
- Keep filtering server-side. Search, archived/shared filters, tags, date ranges, and counts should come from SQL so clients cannot drift from the source of truth.
- Use a stable tie-breaker in paginated queries. This project uses `id ASC` after the selected sort so page boundaries stay predictable.
- Normalize timestamps to UTC before storing or comparing them, and serialize them as RFC3339 in the API.

## Security notes

- Validate and normalize input at the HTTP boundary before it reaches services or SQL parameters.
- Cap JSON request bodies so a single request cannot force the API to read unbounded input into memory.
- Keep owner-scoped routes behind auth and use public shared routes only where the resource is intentionally published.
- Public shared-note reads should expose a reduced public shape only. Do not leak internal note IDs or owner IDs through shared links.
- Avoid leaking database, cache, or OIDC provider internals in API error responses.
- Keep session cookies `HttpOnly`, `SameSite=Lax`, and `Secure` when running behind HTTPS.
- Use Valkey-backed session and OIDC state storage with short TTLs for temporary auth state.
- Throttling is enabled for abuse-prone public routes like login, callback, and shared-note access so the API can fail fast before expensive DB or OIDC work starts.

## Notes

- PostgreSQL and Valkey run in Docker so local development does not leave database daemons running in the background on the host machine.
- The Compose `api` service overrides `DATABASE_URL` and `VALKEY_ADDR` so the Go app talks to container hostnames instead of `localhost`.
- OIDC is not part of local Compose. Configure `OIDC_ISSUER_URL`, `OIDC_CLIENT_ID`, `OIDC_REDIRECT_URL`, and optionally `OIDC_CLIENT_SECRET` to connect to an external provider.
- MCP is also separate from local Compose. It runs as a stdio process and currently requires `MCP_OWNER_USER_ID` for local owner scoping.
- The handwritten-code coverage gate excludes generated `sqlc` output and the thin `cmd/api` / `cmd/mcp` entrypoints so it focuses on the project’s actual teaching logic.
- The web UI lives in the same Go server as the API and reuses the same auth and notes services rather than calling the API over HTTP from a second frontend app.
- [AGENTS.md](/Users/nathanbland/projects/codex-workspace/go-notes/AGENTS.md) captures the project’s standing rules for architecture, docs usage, testing, and coverage.
- Timestamps are stored as PostgreSQL `timestamptz` values and always normalized to UTC.
- Nullable SQL columns intentionally become Go pointers so the code can show the difference between `null`, an omitted value, and a zero value.
- PATCH semantics use pointer fields plus explicit `Set` flags because Go needs a little more help than JavaScript when you need to tell “field missing” from “field intentionally set”.
