# Landing Page Service Overview Feature Plan

## Goal

Refresh the unauthenticated landing page so it describes what a person experiences when they land on the running service: private Markdown notes, tags, saved searches, intentional sharing, OIDC login, and optional API/agent access.

## Required

- Keep the page server-rendered in the existing Go template.
- Keep login routed through the existing OIDC entry point.
- Avoid links to repository files that are not served by the running app.
- Preserve the current Tailwind CDN and small HTMX-oriented UI approach.
- Update docs and tests so the landing page remains discoverable and understandable.

## Implementation Plan

- Add a roadmap item for the landing page update.
- Replace the older UI-focused landing copy with a service-facing product overview.
- Keep the primary action as OIDC login and replace the file-docs link with an app-served API health link.
- Add cards for user-facing capabilities: Markdown notes, search/filtering, tag cleanup, and intentional sharing.
- Update UI tests to lock in the new copy and ensure the removed `/docs/api.md` app link does not return.
- Update `README.md`, `docs/web-ui.md`, `ROADMAP.md`, and `CHANGELOG.md`.

## Acceptance Criteria

- The public landing page clearly describes the current service capabilities, not the internal repository structure.
- The landing page links only to routes served by the app.
- The page still exposes the OIDC login action.
- The page uses the existing Tailwind and HTMX setup without adding a new frontend build pipeline.
- Tests verify the new landing-page copy and action links.
- Docs include examples or notes explaining what the landing page is for.
- `CHANGELOG.md` records the user-visible UI change.
- `make test` passes.
- `make coverage-check-integration` passes.

## Version Impact

- `patch`

## Changelog

- Update `CHANGELOG.md` under `Unreleased`.

## Tooling / Related Functionality

- No new MCP tools are needed. This feature changes the browser-facing landing page only.
