# Primary Tag Sort Feature Plan

## Goal

Add one richer tag-oriented sort mode that stays deterministic, SQL-driven, and easy to teach across the REST API, MCP tools, and the web UI.

## Required to build it

- Keep sorting authoritative in PostgreSQL rather than post-processing notes in Go.
- Reuse the same sort semantics across REST, MCP, and the HTML UI.
- Keep the new sort field in the existing allowlist-based validation boundary.
- Document the behavior with examples so it is discoverable and easy to understand.
- Cover the change with unit tests, integration tests, and the integration-backed coverage gate.

## Implementation plan

1. Add a new sort field, `primary_tag`, to the shared REST and MCP validation logic.
2. Extend the SQL `ORDER BY` allowlist so PostgreSQL sorts by the first stored tag in the note’s `tags` array.
3. Surface the new sort option in the web UI filter form.
4. Update the API, filtering, MCP, and README docs with examples that show how to use the new sort.
5. Add or update unit tests, integration tests, and coverage checks for the new behavior.
6. Update the roadmap to reflect that the first richer tag-oriented sort mode has been implemented while keeping room for future evaluation.

## Acceptance criteria

- REST callers can use `sort=primary_tag` with `order=asc|desc`.
- MCP `list_notes` accepts `sort=primary_tag`.
- The web UI exposes the same sort choice in the tag-browsing controls.
- PostgreSQL remains the source of truth for the sort behavior.
- Docs include examples of the new sort so it is discoverable.
- `make test` passes.
- `make test-integration` passes.
- `make coverage-check-integration` passes.
