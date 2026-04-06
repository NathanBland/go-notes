# Filtering And Pagination

## Defaults

- page: `1`
- page size: `20`
- max page size: `100`
- saved query: optional, owner-scoped preset
- search mode: `plain`
- status: `active`
- tag mode: `any`
- sort: `updated_at`
- order: `desc`

## Rules

- Query validation happens before the repository layer sees any values.
- Only documented sort fields and directions are accepted.
- Pagination always uses a bounded page size and a stable SQL tie-breaker.
- Search, filtering, sorting, pagination, and total counts happen in PostgreSQL rather than in Go loops.
- `saved_query_id` lets the API preload a named owner-scoped filter preset before applying any explicit query parameters from the current request.
- `search_mode=plain` keeps the earlier substring-style behavior, while `search_mode=fts` uses PostgreSQL full-text search with `websearch_to_tsquery`.
- `sort=relevance` is only valid when `search_mode=fts` and `search` is present.
- `has_title=true|false` filters notes based on whether a normalized title is present.
- Tag filters can be supplied as repeated `tag` values or a comma-separated `tags` value.
- `tag_count_min` and `tag_count_max` filter by the number of tags on each note.
- `tag_mode=any` uses PostgreSQL array overlap semantics, while `tag_mode=all` uses array containment semantics.
- `sort=primary_tag` sorts by the first stored tag in `tags[]`, which keeps the behavior deterministic and SQL-driven.
- When you want a human-friendly order inside a tag-filtered result set, prefer `sort=title` instead of inventing another tag-derived sort rule.
- Filter timestamps are parsed as RFC3339 and normalized to UTC before they reach SQL.
- This keeps behavior consistent, protects against SQL injection mistakes, and makes cached list keys deterministic.

## Examples

Newest active notes:

```bash
curl 'http://localhost:8080/api/v1/notes?page=1&page_size=20'
```

Archived notes sorted by title:

```bash
curl 'http://localhost:8080/api/v1/notes?status=archived&sort=title&order=asc'
```

Shared notes containing a search term:

```bash
curl 'http://localhost:8080/api/v1/notes?shared=true&search=golang'
```

Ranked full-text search results:

```bash
curl 'http://localhost:8080/api/v1/notes?search=%22full+text+search%22+ranking&search_mode=fts&sort=relevance&order=desc'
```

Untitled notes with a small tag count:

```bash
curl 'http://localhost:8080/api/v1/notes?has_title=false&tag_count_min=1&tag_count_max=2&status=all'
```

Notes tagged `work` after a date:

```bash
curl 'http://localhost:8080/api/v1/notes?tag=work&created_after=2026-04-01T00:00:00Z'
```

Notes matching any of several tags and sorted by tag count:

```bash
curl 'http://localhost:8080/api/v1/notes?tags=work,planning&tag_mode=any&sort=tag_count&order=desc'
```

Reuse a saved query and override its sort order:

```bash
curl 'http://localhost:8080/api/v1/notes?saved_query_id=<uuid>&order=asc'
```

Notes tagged `work` and then sorted alphabetically by title:

```bash
curl 'http://localhost:8080/api/v1/notes?tags=work&tag_mode=any&sort=title&order=asc'
```

Notes sorted alphabetically by their first stored tag:

```bash
curl 'http://localhost:8080/api/v1/notes?status=all&sort=primary_tag&order=asc'
```

Notes that must contain all requested tags:

```bash
curl 'http://localhost:8080/api/v1/notes?tag=work&tag=backend&tag_mode=all'
```

## Why normalize filters

The note list cache key is derived from the normalized filter object. That means two requests only share a cache entry when their page, sort, filter, and owner values really match.

Saved queries reuse that same normalization step. The project stores a canonical query string after validation so REST, UI, and MCP can all replay the same filter preset without drifting into separate filter rules.
