# Versioning

`go-notes` uses tag-driven releases with a changelog-first workflow.

## Source of truth

- Release versions come from Git tags in the form `vMAJOR.MINOR.PATCH`.
- GoReleaser reads those tags when building release artifacts.
- Snapshot builds keep using the existing GoReleaser template, which currently produces `incpatch(version)-next`.

For example:

- tag `v0.1.0` produces release version `0.1.0`
- snapshot builds from that line produce `0.1.1-next`

## Current release line

Until the project reaches `1.0.0`, `go-notes` uses a simplified pre-1.0 SemVer policy so contributors can make consistent decisions without overfitting every change.

### While the project is `0.x`

- `0.MINOR.0`
  - use for new user-visible capabilities
  - use for new endpoints, MCP tools, resources, prompts, schema changes, deployment model changes, or other meaningful public-surface additions
  - use for breaking changes too, because pre-1.0 compatibility is still evolving
- `0.MINOR.PATCH`
  - use for bug fixes
  - use for security hardening
  - use for documentation corrections tied to already-shipped behavior
  - use for non-breaking operational or packaging fixes

### Once the project reaches `1.0.0`

Move to normal SemVer:

- `MAJOR` for breaking changes
- `MINOR` for backward-compatible features
- `PATCH` for backward-compatible fixes

## Changelog workflow

The repository keeps a root [`CHANGELOG.md`](CHANGELOG.md).

Feature work should update the `Unreleased` section when the change is user-visible, operator-visible, or meaningfully changes the teaching surface.

Recommended headings:

- `Added`
- `Changed`
- `Fixed`
- `Security`
- `Docs`

When preparing a release:

1. Decide the next version from the accumulated `Unreleased` changes.
2. Move the `Unreleased` entries into a new versioned section.
3. Update any explicit in-repo version surfaces, such as [`docs/openapi.yaml`](docs/openapi.yaml), if they should reflect the new public release.
4. Create the Git tag in `vMAJOR.MINOR.PATCH` form.
5. Let GoReleaser build artifacts and release notes from that tag.

## Practical release heuristics for this repo

Use a new minor release when the change adds or materially changes:

- REST endpoints or request/response behavior
- MCP tools, resources, or prompts
- schema or migration behavior
- deployment/runtime shape
- local developer workflows that readers are expected to follow

Use a patch release when the change mainly:

- fixes an existing behavior
- tightens security without broadening the public surface
- corrects docs for existing behavior
- fixes packaging or release mechanics without changing features

## What feature work should do

Most feature work should not mint a new version tag immediately.

Instead, each feature should:

1. assess the expected version impact
2. update `CHANGELOG.md` under `Unreleased` when appropriate
3. leave the actual version tag/release cut to a dedicated release-preparation step unless the task explicitly includes shipping a release

That keeps feature delivery lightweight while still making release prep predictable.
