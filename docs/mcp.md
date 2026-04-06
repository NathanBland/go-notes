# MCP

`go-notes` now includes a first MCP interface alongside the REST API.

## Current scope

Transport:

- `stdio`

Tools:

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
- `rename_tag`
- `set_note_tags`
- `share_note`
- `unshare_note`
- `archive_note`
- `unarchive_note`
- `add_note_tags`
- `remove_note_tags`

This first slice is intentionally local-development oriented. It reuses the existing `notes.Service` so MCP and REST share the same note behavior, cache rules, and PostgreSQL access.

## Temporary auth model

The MCP server currently uses a fixed owner UUID from environment variables:

```env
MCP_OWNER_USER_ID=11111111-1111-1111-1111-111111111111
```

That means:

- the MCP server is currently local/dev focused
- all MCP note operations are scoped to one configured note owner
- a fuller MCP-specific auth story is still a roadmap item

## Required env vars

- `DATABASE_URL`
- `VALKEY_ADDR`
- `MCP_OWNER_USER_ID`

Optional:

- `VALKEY_PASSWORD`
- `NOTE_CACHE_TTL`
- `LIST_CACHE_TTL`

## Running the MCP server

```bash
make docker-up
make migrate-up
make run-mcp
```

The MCP server runs over stdio, so it is meant to be launched by an MCP-capable client rather than visited in a browser.

For packaged binary installation and client setup examples, see [MCP Install](mcp-install.md).

## Tool behavior

### `list_notes`

Supports the same filter concepts as the HTTP API:

- `saved_query_id`
- `page`
- `page_size`
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
- `created_after`
- `created_before`
- `updated_after`
- `updated_before`

The MCP list tool now mirrors the HTTP API's multi-tag behavior:

- use `tag` for the older single-tag input
- use `tags` for multi-tag filtering
- use `saved_query_id` to preload a named saved query before applying explicit MCP arguments
- use `search_mode=plain|fts` to choose between substring matching and PostgreSQL full-text search
- use `sort=relevance` with `search_mode=fts` when you want ranked PostgreSQL results
- use `has_title` when you need titled-only or untitled-only results
- use `tag_count_min` and `tag_count_max` to filter by the number of tags on a note
- use `tag_mode=any|all` to control whether a note may match any requested tag or must contain them all
- use `sort=tag_count` when you want notes with more tags to rise or fall together
- use `sort=primary_tag` when you want a deterministic alphabetical order based on the note's first stored tag

### `get_note`

- requires a note UUID
- returns the owner-scoped note

### `find_related_notes`

- requires a source note UUID
- accepts an optional `limit` from `1` to `20`, defaulting to `5`
- only considers notes owned by the configured MCP owner
- ranks results in PostgreSQL by overlapping tag count, then by title, then by note ID for deterministic output
- returns the related note plus `shared_tags` and `shared_tag_count` so MCP clients can explain why a note was considered related

### `create_note`

- requires `content`
- accepts optional `title`
- accepts optional `tags`
- accepts `archived` and `shared`

### `update_note`

- requires a note UUID
- applies owner-scoped partial updates
- accepts optional `title`
- accepts `clear_title` when you want to explicitly remove the current title
- accepts optional `content`
- accepts optional `tags` plus `replace_tags` when you want to replace the whole tag list
- accepts optional `archived` and `shared`

This tool mirrors the project’s PATCH-style teaching semantics: omitted fields are left alone, while explicitly provided fields are changed.

### `delete_note`

- requires a note UUID
- deletes the owner-scoped note
- returns a small structured success payload so MCP clients can confirm the deletion explicitly

### `list_saved_queries`

- returns the current owner-scoped saved queries
- each entry includes the saved query name and canonical query string

### `save_query`

- requires `name`
- accepts the same filter arguments as `list_notes`
- stores the normalized query string so the saved preset stays aligned with the main note-list grammar

### `delete_saved_query`

- requires a saved query UUID
- deletes the owner-scoped saved query

### `list_tags`

- returns the current owner-scoped tag vocabulary plus note counts
- uses PostgreSQL aggregation instead of rebuilding the vocabulary in Go
- orders tags by descending usage count, then alphabetically for deterministic agent output

### `rename_tag`

- requires `old_tag`
- requires `new_tag`
- rewrites one owner-scoped tag across matching notes
- uses PostgreSQL to preserve ordered tag arrays while deduplicating the rewritten result
- refreshes note and shared-note caches through the shared notes service

### `set_note_tags`

- requires a note UUID
- requires a `tags` array
- trims and deduplicates tags before replacing the note’s authoritative tag set
- is useful when an agent wants to converge on a final curated set instead of incrementally adding/removing tags

### `share_note`

- requires a note UUID
- turns sharing on for the owner-scoped note
- reuses the shared note service so share-slug generation and cache invalidation stay centralized

### `unshare_note`

- requires a note UUID
- turns sharing off for the owner-scoped note
- reuses the shared note service so the share slug is cleared consistently

### `archive_note`

- requires a note UUID
- archives the owner-scoped note through the shared PATCH rules

### `unarchive_note`

- requires a note UUID
- unarchives the owner-scoped note through the shared PATCH rules

### `add_note_tags`

- requires a note UUID
- requires a `tags` array
- trims and deduplicates tags before patching the note
- preserves the note’s other fields and reuses the shared note service cache behavior

### `remove_note_tags`

- requires a note UUID
- requires a `tags` array
- removes matching tags from the current note tag set
- leaves unrelated tags in place and reuses the shared note service cache behavior

## Teaching purpose

The MCP implementation is intentionally small so readers can see:

- how an MCP server can sit beside a REST API
- how to keep business logic in one service layer
- how stdio MCP servers fit local agent workflows
- how a temporary auth strategy can be made explicit instead of hidden

## Examples

- list the available owner-scoped tags and counts before proposing cleanup:
  - `list_tags`
- bulk rename a tag after standardizing vocabulary:
  - `rename_tag` with `old_tag="planning"` and `new_tag="roadmap"`
- replace a note's tag set after reclassifying it:
  - `set_note_tags` with `id=<uuid>` and `tags=["go","mcp","teaching"]`
- publish a note without having to build a generic patch payload:
  - `share_note` with `id=<uuid>`
- archive a note after extracting its action items:
  - `archive_note` with `id=<uuid>`
- find notes related to a source note before summarizing a topic cluster:
  - `find_related_notes` with `id=<uuid>` and optional `limit=5`
