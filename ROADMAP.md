# go-notes Roadmap

This file tracks the current roadmap for `go-notes` beyond what is already implemented today.

## Recently completed

- Improve deployment-guide discoverability from the repository root
  - add a top-level deployment entry point instead of relying only on a nested `docs/` path
  - update README deployment links so readers can find the guide immediately
- Define a coherent versioning strategy and changelog workflow
  - document tag-driven releases and pre-`1.0.0` version rules
  - add a root changelog with an `Unreleased` section
  - update `AGENTS.md` so future feature work records version impact and follows the correct release-step expectations
- Add deployment packaging for general consumption
  - set up a GitHub Actions pipeline to build and publish the root application image
  - set up a GitHub Actions pipeline to build and publish the MCP image and packaged MCP distribution artifacts
  - add a production-oriented `docker-compose.prod.yml` file suitable for environments like Portainer
  - add and maintain a README production compose example that matches the supported container layout
  - document how the dev compose stack, production compose stack, and README example relate to each other so contributors know which one to update

- Add request-throttling middleware for login, callback, and public shared-note routes
- Expand integration coverage around note CRUD, filtering, pagination, and cache invalidation
- Keep OIDC configuration env-driven and well-documented so the app can connect cleanly to an external provider without bundling one into local Docker Compose
- Add the first MCP stdio interface with `list_notes`, `get_note`, and `create_note`, reusing the shared notes service
- Expand the MCP surface with `update_note` plus tag-management tools so local LLM clients can revise notes and adjust tags without leaving MCP
- Add practical MCP note-lifecycle and discovery tools
  - `delete_note` to complete CRUD over MCP
  - `list_tags` so LLM clients can discover the current tag vocabulary and counts
  - `set_note_tags` as an explicit “replace the authoritative tag set” tool
  - `share_note` and `unshare_note` for publish/unpublish workflows without dropping to REST
  - `archive_note` and `unarchive_note` as focused note-state tools when they stay clearer than a generic patch
- Add a minimal server-rendered web interface using HTMX and Tailwind CSS for browser-based login, note reading, note creation, and note updates
- Restore the integration-backed handwritten-code coverage gate to `80%+`
  - the project now clears `make coverage-check-integration` at `85%+`
  - the coverage target continues to exclude generated code and thin command entrypoints like `cmd/api` and `cmd/mcp`
- Expand tag-aware filtering across the REST API, MCP tools, and web UI
  - support filtering by one or more tags instead of a single tag only
  - support explicit `any` versus `all` tag-match modes
  - support browsing and re-filtering notes by tag in the web UI
  - expose the same tag filter semantics through MCP so agent workflows and HTTP clients stay aligned
  - keep the heavy lifting in PostgreSQL with a GIN index and array operators so tag filtering, counts, and pagination remain authoritative and cacheable
  - add deterministic `tag_count` sorting where it teaches something useful without obscuring the SQL
  - add deterministic `primary_tag` sorting based on the first stored tag so tag-derived ordering stays explicit and teachable
  - document the PostgreSQL array query tradeoffs as part of the teaching surface
- Run a focused security audit across the API, UI, SQL, and MCP surfaces
  - verify owner-scoped notes are never discoverable by raw note ID unless the authenticated session is actually authorized for that note
  - verify shared-note routes only expose intentionally published notes and do not leak private note existence or internal identifiers
  - review list, get, patch, delete, and MCP note flows for IDOR-style issues
  - review SQL and query-shaping code for SQL injection risks, especially dynamic filtering and ordering paths
  - review request parsing, auth/session handling, error responses, and cache behavior for common API security mistakes
  - add regression tests for shared-note response shape, strict JSON parsing, and cross-owner note access
- Add advanced filtering and search on notes
  - expand beyond the current baseline filters into richer combinations that still stay SQL-driven and teachable
  - add explicit search semantics across REST, UI, and MCP with clear validation rules
  - add PostgreSQL full-text search support with ranked `relevance` sorting
  - add title-presence and tag-count range filters without moving filter logic out of SQL
  - document the PostgreSQL indexing and query-planning tradeoffs that come with more advanced search behavior
- Add binary distribution for the stdio MCP server
  - build installable `go-notes-mcp` binaries for macOS and Linux
  - publish archives and checksums suitable for GitHub Releases or similar distribution
  - add local snapshot-build commands so contributors can validate release packaging before publishing
  - add setup instructions for major MCP-capable AI tools such as Codex, Claude Code, Cursor, and Windsurf
- Add saved query patterns that can be represented cleanly across REST, UI, and MCP
  - store owner-scoped named saved queries in PostgreSQL
  - let `saved_query_id` preload a canonical filter preset before explicit request parameters override it
  - expose saved-query creation, listing, deletion, and reuse through REST, the server-rendered web UI, and MCP
  - keep saved queries aligned with the existing list-filter grammar instead of introducing a second query language
  - verify the feature with unit tests, Docker-backed integration tests, and the coverage gate
- Add higher-level MCP discovery and organization helpers where they stay teachable
  - `find_related_notes` based on overlapping tags, ranked in PostgreSQL with deterministic tie-breaks
  - return the related note plus the shared tags and overlap count so agent output stays explainable
  - verify owner scoping so other users' notes never participate in the ranking
- Add owner-scoped bulk tag cleanup with `rename_tag`
  - expose a shared tag-rename workflow through REST, the server-rendered UI, and MCP
  - keep the rewrite logic in PostgreSQL so order preservation and deduplication stay consistent across surfaces
  - verify cache refresh and owner scoping with unit and integration coverage
- Run a focused UI consistency audit for the server-rendered workspace
  - move note creation to the top of the sidebar because it is the primary action
  - remove decorative section chips that do not add teaching value
  - keep the HTMX behavior but simplify hierarchy, wording, and section consistency
  - add tests that lock in the sidebar order and the removal of low-value labels
- Broaden the teaching surface around ownership boundaries, cache behavior, and error translation
  - add and document examples showing how cross-owner note access still returns the same public `not_found` envelope
  - document how note caches are refreshed while list caches rely on short TTLs
  - keep the examples aligned with tests so the behavior is easy to discover and trust
- Add a focused production-hardening documentation pass
  - document HTTPS expectations, reverse proxy behavior, secure-cookie implications, and production throttling guidance
  - keep the README and deployment docs aligned so public readers can find the hardening guidance quickly

## Near-term roadmap
- Continue higher-level MCP discovery and organization helpers where they stay teachable
  - `suggest_tags_for_note` if the implementation remains simple and explainable

## MCP and LLM roadmap
- Consider note resources and prompt templates once the core MCP tool flow is stable
- Add MCP resources that help agents read note state into context cleanly
  - `notes://{id}` for single-note reads
  - `notes://shared/{slug}` for shared-note reads
  - `notes://tags` for tag and count discovery
  - `notes://filters` for the supported list/filter/sort teaching surface
- Add MCP prompts for common note workflows
  - `summarize_note`
  - `extract_action_items`
  - `propose_tags`
  - `organize_notes_by_theme`
- Evaluate `streamable-HTTP` after the stdio slice proves useful locally
- Design a clear MCP-specific auth story instead of forcing browser-oriented OIDC flows onto non-browser MCP clients

## Planned teaching improvements

- Add more documentation around indexing strategy, SQL performance tradeoffs, and when to push logic into PostgreSQL

## Longer-term ideas

- Full-text search examples using PostgreSQL text search
- More advanced cache patterns beyond the explicit cache-aside baseline
- Optional bearer-token or machine-to-machine auth examples, if they add teaching value without overcomplicating the baseline project
