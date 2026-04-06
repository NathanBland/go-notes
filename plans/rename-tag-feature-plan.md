# Rename Tag Feature Plan

## Goal

Add an explicit tag-rename workflow that updates an owner's note tags consistently and teachably across the project.

## What is required

- A SQL-driven bulk tag rename operation that preserves tag order and owner scoping
- Service-layer cache invalidation for all affected notes and lists
- User-facing entry points for the feature where it adds teaching value
- Docs and examples for how to use the feature
- Version impact assessment: `minor`
- `CHANGELOG.md` update required: `yes`

## Plan

1. Add a PostgreSQL-backed rename operation that rewrites matching tags for one owner.
2. Expose the behavior through the notes service with explicit cache invalidation.
3. Add REST, MCP, and minimal UI entry points so tag management feels complete across the teaching surfaces.
4. Add unit and integration coverage for owner scoping, deduplication, and cache behavior.
5. Update docs and examples.

## Acceptance criteria

- One owner can rename a tag across their notes without touching another owner's notes.
- The rename flow is available from MCP and the HTTP/UI surfaces that already teach tag management.
- Affected note/list cache behavior is covered by tests.
- Docs include concrete examples of how to trigger the rename.
- `CHANGELOG.md` reflects the new capability under `Unreleased`.
