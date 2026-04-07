# Release Prep v0.2.0 Feature Plan

## Goal

Prepare the local `v0.2.0` release for the public shared-note UI feature without pushing commits or tags.

## Required

- Promote the public shared-note UI changelog entry out of `Unreleased`.
- Update explicit version surfaces that should match the new release.
- Run the project release validation commands.
- Create a release commit and local annotated `v0.2.0` tag.
- Do not push to `origin`.

## Implementation Plan

- Move the current `Unreleased` entry into a `0.2.0` changelog section dated `2026-04-07`.
- Update the OpenAPI metadata version and shared-note slug schema to reflect the stricter shared slug validation.
- Run formatting, tests, integration-backed coverage, production compose validation, and release config validation.
- Commit the feature and release-prep changes.
- Create the local `v0.2.0` tag.

## Acceptance Criteria

- `CHANGELOG.md` contains a `0.2.0` section and leaves `Unreleased` ready for future work.
- `docs/openapi.yaml` has version `0.2.0`.
- The public shared-note UI feature remains documented and tested.
- `make test` passes.
- `make coverage-check-integration` passes.
- `make docker-config-prod` passes.
- `make release-check-mcp` passes.
- The release commit exists locally.
- The local `v0.2.0` tag points at the release commit.
- Nothing is pushed.

## Version Impact

- `release-prep`

## Changelog

- Promote the public shared-note UI entry into `0.2.0`.

## Tooling / Related Functionality

- No MCP tool changes are needed for release prep.
