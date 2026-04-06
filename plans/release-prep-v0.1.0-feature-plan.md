# Release Prep v0.1.0 Feature Plan

## Goal

Prepare `go-notes` for its first tagged public release without actually creating the Git tag or pushing release artifacts yet.

## What is required

- Confirm the next version from `VERSIONING.md`
- Promote the current `Unreleased` changelog entries into a `v0.1.0` release section
- Leave a fresh `Unreleased` section in place for post-release feature work
- Verify explicit in-repo version surfaces that should reflect the first public release
- Run the project quality and release-validation commands that should be green before tagging
- Summarize the remaining manual release steps so cutting the real release is low-risk
- Version impact assessment: `release-prep`
- `CHANGELOG.md` update required: `yes`

## Plan

1. Confirm the first public release version and capture it in the plan.
2. Update `CHANGELOG.md` from `Unreleased` to a `v0.1.0` section while preserving a fresh `Unreleased` heading.
3. Verify versioned surfaces like `docs/openapi.yaml` and packaging/release docs stay aligned with `v0.1.0`.
4. Run the required project gates plus release-oriented validation commands.
5. Summarize the exact remaining tag-and-push steps without performing them.

## Acceptance criteria

- The repo has a clear first-release target version and documented release-prep state.
- `CHANGELOG.md` is ready for a `v0.1.0` tag.
- Version surfaces that should match the first release are aligned.
- Examples and docs remain discoverable and accurate for the release surface.
- The required quality and release-validation commands have been run successfully.
- The remaining manual release steps are documented clearly enough to execute without guesswork.
