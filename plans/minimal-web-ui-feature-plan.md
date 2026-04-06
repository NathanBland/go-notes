# Minimal Web UI Feature Plan

## Goal

Add a small server-rendered web interface using HTMX and Tailwind CSS so local development can trigger login, read notes, create notes, and update notes without needing a separate frontend application.

## Required to build it

- Keep the REST API as the primary teaching surface.
- Reuse the existing auth and notes services instead of duplicating note behavior in a separate UI layer.
- Use HTMX only for the small interactive pieces that benefit from partial page updates.
- Keep the UI teachable and intentionally small.
- Add tests for the landing page, authenticated workspace, and core note interactions.
- Update the README, roadmap, and docs after implementation.

## Implementation plan

1. Add public and authenticated HTML routes alongside the existing JSON API routes.
2. Create server-rendered templates for:
   - a public landing page with a login action
   - an authenticated workspace with a create form, note list, and detail panel
   - HTMX partials for note detail and edit form updates
3. Add small form parsers that map HTML form fields into the existing `notes.CreateInput` and `notes.PatchInput` models.
4. Use HTMX for:
   - loading note details into the detail panel
   - loading an edit form into the detail panel
   - refreshing the workspace after create and update
5. Add handler tests covering guest rendering, authenticated rendering, note detail loading, note creation, and note update flows.
6. Update docs and roadmap status.

## Acceptance criteria

- Visiting `/` without a session shows a public landing page with a login action.
- Visiting `/` with a valid session shows a minimal notes workspace.
- The UI can create, read, and update notes using the existing service layer.
- HTMX partial routes return HTML fragments rather than JSON.
- `make test` passes.
- README and relevant docs describe the new web UI and how it fits beside the REST API.
