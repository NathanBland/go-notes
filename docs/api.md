# API Contract

Base path: `/api/v1`

## Auth

- `GET /auth/login`: redirects the browser to the OIDC provider
- `GET /auth/callback`: exchanges the code, verifies the ID token, sets the session cookie, and returns the authenticated user
- `GET /auth/me`: returns the current authenticated user
- `POST /auth/logout`: clears the session cookie and deletes the server-side session

## Notes

- `GET /healthz`
- `POST /notes`
- `GET /notes`
- `GET /notes/{id}`
- `PATCH /notes/{id}`
- `DELETE /notes/{id}`
- `GET /notes/shared/{slug}`

## Saved queries

- `GET /saved-queries`
- `POST /saved-queries`
- `DELETE /saved-queries/{id}`

## Tag management

- `POST /tags/rename`

Saved queries are owner-scoped named list-query presets.

- they store the same note-list query parameters the API already understands
- they do not introduce a second filter language
- `GET /notes?saved_query_id=<uuid>` preloads the saved query first
- explicit request query parameters still win over the saved query’s stored values

Public shared notes intentionally use a reduced response shape:

- no internal note `id`
- no `owner_user_id`
- the public `share_slug` is still included because it is already the public identifier used in the route

Owner-scoped note reads, updates, and deletes intentionally translate cross-owner lookups into the same `not_found` envelope as a truly missing note. That keeps the API from leaking whether another user's note UUID exists.

## List query parameters

- `page`
- `page_size`
- `saved_query_id`
- `search`
- `search_mode=plain|fts`
- `status=active|archived|all`
- `shared=true|false`
- `has_title=true|false`
- `tag`
- `tags`
- `tag_count_min`
- `tag_count_max`
- `tag_mode=any|all`
- `sort=created_at|updated_at|title|tag_count|primary_tag|relevance`
- `order=asc|desc`
- `created_before`
- `created_after`
- `updated_before`
- `updated_after`

## Response style

Single-resource responses:

```json
{
  "data": {
    "id": "uuid",
    "title": null,
    "content": "hello",
    "tags": [],
    "archived": false,
    "shared": false,
    "share_slug": null,
    "created_at": "2026-04-02T03:04:05Z",
    "updated_at": "2026-04-02T03:04:05Z"
  }
}
```

Public shared-note responses:

```json
{
  "data": {
    "title": "Shared note",
    "content": "hello",
    "tags": [],
    "archived": false,
    "shared": true,
    "share_slug": "public-link",
    "created_at": "2026-04-02T03:04:05Z",
    "updated_at": "2026-04-02T03:04:05Z"
  }
}
```

Collection responses:

```json
{
  "data": [],
  "meta": {
    "page": 1,
    "page_size": 20,
    "total": 0,
    "sort": "updated_at",
    "order": "desc"
  }
}
```

Saved query creation:

```json
{
  "name": "Ranked work notes",
  "query": "search=golang&search_mode=fts&tag=work&sort=relevance"
}
```

Saved query list response:

```json
{
  "data": {
    "saved_queries": [
      {
        "id": "uuid",
        "name": "Ranked work notes",
        "query": "search=golang&search_mode=fts&tag=work&sort=relevance",
        "created_at": "2026-04-05T12:00:00Z",
        "updated_at": "2026-04-05T12:00:00Z"
      }
    ]
  }
}
```

Tag rename request:

```json
{
  "old_tag": "planning",
  "new_tag": "roadmap"
}
```

Tag rename response:

```json
{
  "data": {
    "old_tag": "planning",
    "new_tag": "roadmap",
    "affected_notes": 2
  }
}
```

Validation errors:

```json
{
  "error": {
    "code": "validation_error",
    "message": "invalid request",
    "fields": {
      "page_size": "must be between 1 and 100"
    }
  }
}
```

Ownership-preserving not-found error:

```json
{
  "error": {
    "code": "not_found",
    "message": "note not found"
  }
}
```

Cache behavior example:

- note-by-id and shared-note caches are refreshed after owner-scoped updates such as `PATCH /notes/{id}` and `POST /tags/rename`
- list caches use short TTLs instead of trying to fan out invalidation to every possible filter combination
