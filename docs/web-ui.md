# Minimal Web UI

`go-notes` now includes a deliberately small server-rendered web interface alongside the JSON API and MCP server.

## Why it exists

The UI is meant to make local development and teaching easier:

- trigger the OIDC login flow from a browser
- read notes without a separate API client
- create and update notes with simple HTML forms
- filter notes by one or more tags without leaving the page
- save and reapply named filter presets without rebuilding them by hand
- show how HTMX can add targeted interactivity without replacing server-rendered HTML

The sidebar now follows the intended task priority:

- the note-creation form appears first because creating notes is the primary workspace action
- saved queries come next because they help reuse browsing context
- filters follow after that as a secondary narrowing tool
- section headers use short instructional copy instead of decorative chips

The login action still starts at the shared `GET /api/v1/auth/login` endpoint. The UI adds a safe local redirect target so the shared OIDC callback can send a browser back to `/` after the session cookie has been set.

## Markdown rendering

Note content is still stored and edited as raw Markdown text.

The read-only note detail view renders that Markdown to HTML in the Go server before sending the fragment to the browser. This keeps the data model simple while making the workspace easier to read during demos and local development.

The renderer intentionally keeps Goldmark's safe default HTML behavior, so raw HTML is not enabled in note content.

For styling, the project follows the same idea as Tailwind Typography's `prose` treatment: rendered Markdown is wrapped in a dedicated container and common Markdown elements get opinionated typography and spacing. Since this project currently uses the Tailwind CDN instead of a Tailwind build pipeline, the UI uses a small handcrafted prose-style stylesheet rather than the Typography plugin itself.

The UI is intentionally not a separate frontend application. The note behavior still lives in the shared auth and notes services.

## Routes

- `GET /`
  - public landing page when unauthenticated
  - authenticated workspace when a valid session cookie is present
- `POST /app/logout`
- `POST /app/saved-queries`
  - save the current filter set as a named saved query
- `POST /app/saved-queries/{id}/delete`
  - delete one saved query
- `GET /app/notes/{id}`
  - HTMX detail fragment for the selected note
- `GET /app/notes/{id}/edit`
  - HTMX edit-form fragment
- `POST /app/notes`
  - create note from HTML form data
- `POST /app/notes/{id}`
  - update note from HTML form data

## HTMX usage

The interface uses a small set of HTMX patterns:

- note links use `hx-get` to load a detail fragment into the note panel
- the edit action uses `hx-get` to load an edit form into the same panel
- create and update forms use `hx-post` and replace the workspace region after success
- the saved-query form also uses `hx-post`, which keeps the filter sidebar and note panel in sync after a preset is created
- active tag filters are preserved across create, read, and edit flows so the workspace stays in the same browsing context

## Filters

The workspace now includes a filter form:

- `search` supports the current search term
- `search_mode=plain|fts` lets the UI demonstrate the difference between substring matching and PostgreSQL full-text search
- `has_title` filters titled versus untitled notes
- `tags` accepts a comma-separated list of tags
- `tag_count_min` and `tag_count_max` let the UI demonstrate count-based filtering without moving those rules into Go
- `tag_mode=any|all` controls whether notes may match any requested tag or must contain them all
- `sort=tag_count` lets the UI demonstrate a simple deterministic tag-derived order without inventing application-only sorting rules
- `sort=primary_tag` lets the UI demonstrate another SQL-driven tag ordering rule based on the note's first stored tag
- `sort=relevance` becomes available when the search mode is full-text

The note detail view also turns each rendered tag into a small filter link so the UI can demonstrate a lightweight "browse notes like this one" interaction.

This keeps the UI dynamic while preserving normal links and form actions as simple browser fallbacks.

## Saved queries

The workspace now includes a saved-query card above the filter form.

- users can name the current filter set and save it
- saved queries render as small reusable links in the sidebar
- deleting a saved query happens with a normal form post so the behavior stays understandable without hidden client state
- the saved query stores the normalized list-query string, not a separate UI-only payload

That keeps the browser experience aligned with the API and MCP surfaces.

## Styling approach

Tailwind CSS is loaded from the CDN for this teaching-oriented interface. That keeps setup light while the project stays focused on Go, SQL, caching, and auth examples.

If the UI grows into a production-oriented surface later, the project should move to a built asset pipeline instead of the CDN approach.

## Relationship to the API

The HTML routes do not replace the JSON API.

- JSON routes remain the primary public teaching interface
- the web UI is a thin convenience layer for browser-based local development
- MCP remains a separate local agent surface over stdio

This three-surface setup is intentional: it lets the project teach how one set of business rules can support API, HTML, and MCP interfaces without duplicating logic.
