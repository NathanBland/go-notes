# Architecture

## Layers

- `cmd/api`: process startup and graceful shutdown
- `cmd/mcp`: stdio MCP startup for local LLM tooling
- `internal/app`: env-driven configuration and dependency wiring
- `internal/httpapi`: handlers, middleware, request parsing, JSON envelopes, HTML templates, and HTMX fragments
- `internal/mcpapi`: MCP tools that reuse the notes service
- `internal/auth`: OIDC/session business rules
- `internal/notes`: note business rules, cache-aside behavior, list metadata
- `internal/platform/db`: PostgreSQL pool + sqlc-backed repository adapter
- `internal/platform/cache`: Valkey client wrapper
- `internal/platform/oidc`: provider discovery, PKCE, token verification
- `docker-compose.yml`: local PostgreSQL, Valkey, migration helper container, and hot-reloading API
- `docker-compose.prod.yml`: production-oriented compose stack for registry-backed deployments
- `Dockerfile`: production API image with the app binary plus bundled migrations and migrate CLI
- `Dockerfile.mcp`: production MCP runtime image
- `.github/workflows/`: CI pipelines for API image publication and MCP release delivery
- `.air.toml`: file-watch rules for rebuild and restart during development

## Flow

1. A handler parses HTTP input and validates it close to the transport edge.
2. The handler calls a small service method.
3. The service enforces ownership rules, nullable-field behavior, and cache strategy.
4. The repository uses sqlc-generated queries for database access.
5. The response is wrapped in a consistent JSON envelope.

For the minimal web UI:

1. A browser request reaches the same `internal/httpapi` package.
2. The HTML handlers reuse the existing auth and notes services instead of calling the JSON API over HTTP.
3. HTMX endpoints return small HTML fragments for note detail and edit flows.
4. Full-page routes render server-side templates with the current workspace state.

For MCP:

1. An MCP tool request is received over stdio.
2. `internal/mcpapi` validates and normalizes tool arguments.
3. The MCP layer calls the same `notes.Service` used by HTTP.
4. Structured tool results are returned to the MCP client.

## Query rules

- Handlers validate and normalize filter input before it reaches the repository layer.
- The repository translates those validated values into typed `sqlc` parameter structs.
- PostgreSQL does the heavy lifting for filtering, searching, sorting, pagination, and total counts.
- Saved queries are stored as canonical owner-scoped query strings, then replayed through the same validation path before explicit request parameters override them.
- Related-note discovery also stays in PostgreSQL so tag overlap ranking and deterministic ordering do not drift between transports.
- Sort fields and sort directions are allowlisted before they can influence SQL behavior.
- Query values stay in bound parameters instead of being interpolated into raw SQL strings.

## Teaching choices

- `net/http` is used directly so the project shows the standard library clearly.
- Request-scoped `context.Context` flows from handlers into DB and Valkey operations.
- `sqlc` keeps SQL visible and reviewable instead of hiding it behind a heavy ORM.
- Valkey is used explicitly rather than magically so cache invalidation stays understandable.
- PostgreSQL and Valkey live in Docker during local development so the host machine stays clean and the stack is easy to stop.
- The repository layer stays intentionally small so the project teaches how route inputs become SQL parameters without hiding the queries themselves.
- The MCP layer is intentionally thin and local-first so the project shows how a second interface can reuse the same service boundary without duplicating note logic.
- The minimal web UI stays server-rendered on purpose so the project can teach HTMX-style progressive enhancement without introducing a full SPA build step.
