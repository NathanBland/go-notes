# Advanced Filtering And Search Feature Plan

## Goal

Expand note filtering and search beyond the current baseline while keeping the behavior SQL-driven, deterministic, and teachable across REST, MCP, and the minimal web UI.

## Required to build it

- Keep plain substring search available for the simpler baseline behavior.
- Add an explicit PostgreSQL full-text search mode with ranked results.
- Add richer filters that are still easy to explain and index:
  - `has_title`
  - `tag_count_min`
  - `tag_count_max`
- Keep request validation, SQL queries, MCP tool inputs, UI controls, docs, and tests aligned.
- Add forward-only migrations for any new indexes needed to support the search path.

## Implementation plan

1. Extend `notes.ListFilters` with explicit search/filter fields and safe defaults.
2. Add request parsing and MCP argument normalization for:
   - `search_mode=plain|fts`
   - `sort=relevance`
   - `has_title`
   - `tag_count_min`
   - `tag_count_max`
3. Add a new migration for a PostgreSQL GIN expression index to support full-text search.
4. Update the list/count SQL queries to support:
   - plain substring search
   - full-text search with `websearch_to_tsquery`
   - `ts_rank_cd` relevance sorting
   - title-presence and tag-count filters
5. Extend the minimal UI filter form so the new search/filter behavior is discoverable without leaving the browser.
6. Update README, API/filtering docs, OpenAPI, MCP docs, and testing guidance with examples.
7. Run `make sqlc-generate`, `make test`, `make test-integration`, and `make coverage-check-integration`.

## Acceptance criteria

- REST, MCP, and the browser UI can all drive the new advanced search/filter fields.
- PostgreSQL remains the authoritative source for matching, ranking, counting, and sorting behavior.
- A new migration pair adds the supporting search index without rewriting older migrations.
- Docs include clear examples of plain search versus full-text search and the new filter fields.
- `make test` passes.
- `make test-integration` passes.
- `make coverage-check-integration` passes.
