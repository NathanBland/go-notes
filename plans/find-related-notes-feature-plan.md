# Find Related Notes Feature Plan

## Goal

Add a teachable, owner-scoped `find_related_notes` MCP tool that helps LLM clients discover notes related to a source note by overlapping tags without moving ranking logic out of PostgreSQL.

## Required to build it

- Keep related-note discovery owner-scoped so one user's notes never influence another user's results.
- Let PostgreSQL compute overlap and ranking instead of rebuilding relatedness in Go.
- Reuse the existing notes service and repository boundaries rather than introducing MCP-only business logic.
- Return enough context to make the tool useful and understandable, including the related note plus the shared tags and overlap count.
- Document the behavior and provide examples so the tool is discoverable.
- Verify the feature with unit tests, Docker-backed integration tests, and the integration-backed coverage gate.

## Implementation plan

1. Add a SQL query that finds owner-scoped related notes for a source note using overlapping tags and deterministic ordering.
2. Extend the notes store and service with a `FindRelatedNotes` method and a domain model for related-note results.
3. Add an MCP tool named `find_related_notes` that validates input, calls the shared notes service, and returns structured results.
4. Update MCP docs, README, roadmap, and any teaching docs impacted by the new discovery workflow.
5. Add unit tests for the notes service, repository adapter, and MCP server plus a Docker-backed integration test for the SQL behavior.
6. Run `make fmt`, `make test`, `make test-integration`, and `make coverage-check-integration`.

## Acceptance criteria

- MCP exposes `find_related_notes` with an owner-scoped note UUID and an optional bounded limit.
- The related-note ranking is computed in PostgreSQL using overlapping tags and deterministic tie-breaks.
- Results include the related note, the shared tags, and the overlap count.
- A note owned by another user is never returned or used as the source note.
- Docs include examples and explain how relatedness is determined.
- `README.md`, `ROADMAP.md`, and the relevant files under `docs/` are updated.
- `make test` passes.
- `make test-integration` passes.
- `make coverage-check-integration` passes.
