# MCP Update And Tag Tools Feature Plan

## Goal

Expand the local MCP surface so LLM clients can update notes and manage note tags without dropping down to the REST API.

## Required to build it

- Keep MCP behavior aligned with the existing `notes.Service` so note rules stay consistent across HTTP, UI, and MCP.
- Add an `update_note` tool with clear PATCH-style semantics.
- Add tag-focused tools that are useful to agent workflows without duplicating the whole notes API.
- Keep tool schemas explicit and well-documented in `docs/mcp.md` and `README.md`.
- Cover the new tool handlers with unit tests.

## Implementation plan

1. Add a roadmap item for MCP note updates and tag-management tools.
2. Extend the MCP note service interface to include patch behavior.
3. Add an `update_note` tool that supports:
   - optional `title`
   - `clear_title`
   - optional `content`
   - optional `tags` plus `replace_tags`
   - optional `archived`
   - optional `shared`
4. Add tag-focused tools:
   - `add_note_tags`
   - `remove_note_tags`
5. Reuse owner-scoped note reads plus `Patch` so tag tools preserve the same cache and share-slug behavior as the rest of the app.
6. Update MCP docs, README, roadmap, and testing guidance.
7. Run tests and report current coverage-gate status honestly.

## Acceptance criteria

- MCP exposes `update_note`, `add_note_tags`, and `remove_note_tags`.
- MCP note updates remain owner-scoped and reuse the shared notes service.
- Tag tools deduplicate and normalize tags instead of blindly appending duplicates.
- `make test` passes.
- Relevant docs and roadmap entries are updated in the same change.
