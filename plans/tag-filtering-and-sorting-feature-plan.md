# Tag Filtering And Sorting Feature Plan

## Goal

Expand tag-aware note discovery across the REST API, MCP tools, and web UI with multi-tag filtering, explicit tag-match modes, and a simple tag-derived sort that remains easy to explain.

## Required to build it

- Keep filtering and counts SQL-driven in PostgreSQL.
- Preserve backward compatibility for the existing single `tag` filter while adding multi-tag support.
- Expose the same tag semantics across REST, MCP, and the web UI.
- Add index support for array-tag queries.
- Cover the new behavior with unit tests and integration tests.
- Update the README and relevant docs after implementation.

## Implementation plan

1. Extend the shared filter model to support multiple tags and `any`/`all` tag matching.
2. Update SQL queries to use PostgreSQL array overlap and containment operators.
3. Add a GIN index on the `tags` column for tag-filter queries.
4. Extend the REST query parser with:
   - repeated `tag` values
   - `tags` CSV input
   - `tag_mode=any|all`
   - `sort=tag_count`
5. Extend the MCP list tool with the same multi-tag and tag-match semantics.
6. Add a small UI filter form plus clickable tag links that reuse the same query rules.
7. Update tests, docs, and roadmap status.

## Acceptance criteria

- REST callers can filter by one or more tags and choose `tag_mode=any|all`.
- MCP `list_notes` supports the same tag behavior.
- The web UI can browse/filter notes by tags and preserve those filters in the workspace.
- Sorting by `tag_count` works with stable pagination semantics.
- PostgreSQL remains the source of truth for filtering, counts, and ordering.
- `make test`, `make test-integration`, and `make coverage-check-integration` pass.
