# Saved Query Patterns Feature Plan

## Goal

Add owner-scoped saved query patterns so users and local MCP clients can reuse named note filters across the REST API, the web UI, and MCP without inventing a second filtering model.

## Required to build it

- Store saved queries in PostgreSQL as owner-scoped named records.
- Keep the saved-query payload aligned with the existing list filter surface instead of creating a separate query language.
- Let SQL-backed note listing continue to be authoritative; saved queries should only preload normalized filter parameters.
- Support saved-query creation, listing, deletion, and application across REST, UI, and MCP.
- Keep merge behavior simple and teachable: saved query first, explicit request parameters override second.
- Add examples and docs so the feature is discoverable.
- Verify the feature with unit tests, Docker-backed integration tests, and the integration-backed coverage gate.

## Implementation plan

1. Add a new migration and SQL queries for owner-scoped saved queries.
2. Extend the notes store and service with saved-query CRUD methods.
3. Add HTTP endpoints for listing, creating, and deleting saved queries, plus support for `saved_query_id` on note listing.
4. Extend the web UI with a saved-query section and a form to save the current filter set.
5. Add MCP tools for listing, saving, and deleting queries, and allow `list_notes` to accept `saved_query_id`.
6. Update docs, examples, and the roadmap to reflect the new feature.
7. Run `make test`, `make test-integration`, and `make coverage-check-integration`.

## Acceptance criteria

- The project stores owner-scoped saved queries in PostgreSQL with both up and down migrations.
- `GET /api/v1/notes` accepts `saved_query_id`, and explicit query parameters override the saved query’s stored values.
- REST exposes endpoints to list, create, and delete saved queries.
- The web UI lets a signed-in user save the current filter set and reapply saved queries.
- MCP exposes tools to list, save, and delete saved queries, and `list_notes` can use `saved_query_id`.
- Docs include examples for REST, UI, and MCP saved-query usage.
- `README.md`, `ROADMAP.md`, and the relevant files under `docs/` are updated.
- `make test` passes.
- `make test-integration` passes.
- `make coverage-check-integration` passes.
