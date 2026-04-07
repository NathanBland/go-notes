# Public Shared Note UI Feature Plan

## Goal

Add a public browser page for intentionally shared notes so someone with a share link can read the note without signing in.

## Required

- Keep owner-scoped workspace routes behind authentication.
- Reuse the existing shared-note service path so the UI has the same authorization boundary as `GET /api/v1/notes/shared/{slug}`.
- Render Markdown safely with the same Goldmark configuration as the authenticated note detail view.
- Avoid exposing internal note IDs or owner IDs in the public HTML page.
- Throttle the public shared-note UI route because it is unauthenticated.
- Update docs, examples, and tests.

## Implementation Plan

- Add `GET /shared/{slug}` to the HTTP router with the same public throttle middleware used for the shared JSON endpoint.
- Add a handler that validates the slug, loads the note with `notes.Service.GetByShareSlug`, and renders a public HTML template.
- Add a reusable slug validator for public shared-note routes.
- Add templates for the shared note and shared-note error page.
- Add tests for success, missing notes, malformed slugs, and unauthenticated access.
- Update `README.md`, `docs/web-ui.md`, `docs/api.md`, `ROADMAP.md`, and `CHANGELOG.md`.

## Acceptance Criteria

- `GET /shared/{slug}` renders an intentionally shared note without requiring a session.
- Missing or unshared notes return a not-found page instead of redirecting to login.
- Malformed slugs return a bad-request page.
- The public page renders Markdown and tags but does not include internal note IDs or owner IDs.
- The existing `GET /api/v1/notes/shared/{slug}` JSON endpoint still works.
- Tests cover shared, missing/unshared, and malformed slug paths.
- Docs include examples showing how to open a public shared-note link.
- `make test` passes.
- `make coverage-check-integration` passes.

## Version Impact

- `minor`

## Changelog

- Update `CHANGELOG.md` under `Unreleased`.

## Tooling / Related Functionality

- No new MCP tools are required. MCP already has `share_note` and `unshare_note`; this feature makes the resulting browser share link usable without authentication.
