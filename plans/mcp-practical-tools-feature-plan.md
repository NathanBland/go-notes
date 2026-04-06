# MCP Practical Tools Feature Plan

## Goal

Expand the MCP surface so local LLM clients can complete more of the note lifecycle without dropping down to REST, while keeping the actual note rules in the shared Go service layer and the tag-vocabulary work in PostgreSQL.

## Required to build it

- Keep MCP behavior aligned with the existing `notes.Service` so owner scoping, share-slug rules, and cache invalidation stay identical across UI, REST, and MCP.
- Add the practical note-lifecycle tools already called out on the roadmap:
  - `delete_note`
  - `list_tags`
  - `set_note_tags`
  - `share_note`
  - `unshare_note`
  - `archive_note`
  - `unarchive_note`
- Push tag discovery into SQL instead of calculating it in Go.
- Add examples and docs so the new MCP tools are discoverable and easy to understand.
- Verify the feature with unit tests, Docker-backed integration tests where appropriate, and the integrated coverage gate.

## Implementation plan

1. Extend the notes store/service boundary with an owner-scoped tag-summary operation.
2. Add a SQL query that unnests `tags`, groups by tag, counts usage, and orders results deterministically.
3. Extend the MCP notes-service interface with delete and tag-summary behavior.
4. Add focused MCP tools that reuse `Patch` or `Delete` instead of inventing new business logic in the transport layer.
5. Add MCP unit tests for the new tool definitions and handlers.
6. Add or extend integration coverage for the SQL-backed tag-summary path.
7. Update `README.md`, `ROADMAP.md`, `docs/mcp.md`, and any impacted testing docs with examples of how the new tools are used.

## Acceptance criteria

- MCP exposes `delete_note`, `list_tags`, `set_note_tags`, `share_note`, `unshare_note`, `archive_note`, and `unarchive_note`.
- `list_tags` returns owner-scoped tag counts produced by PostgreSQL.
- The new MCP tools reuse the shared note service for note-state changes instead of duplicating note rules.
- Docs include examples for the new tools and clearly explain what they do.
- `make test` passes.
- `make test-integration` passes.
- `make coverage-check-integration` passes.
