# Versioning Strategy Feature Plan

## Goal

Define a coherent versioning and changelog strategy for `go-notes` so releases, documentation, and contributor expectations stay aligned.

## What is required

- A written versioning strategy that matches the repo's current tag-driven release tooling
- A changelog structure that contributors can update as feature work lands
- Clear release-preparation steps for promoting `Unreleased` work into a tagged release
- Contributor guidance in `AGENTS.md` so feature plans and feature completion include version-impact expectations
- A clear version-impact assessment for this feature, which is `none` for runtime behavior but `patch` for the next release process/docs line
- A `CHANGELOG.md` update because the strategy changes contributor and release workflow

## Plan

1. Document the project's versioning rules in a dedicated versioning guide.
2. Add a root `CHANGELOG.md` with an `Unreleased` section and release-note categories.
3. Align the strategy with the current GoReleaser tag and snapshot behavior.
4. Update `README.md`, `ROADMAP.md`, and any affected docs so the strategy is discoverable.
5. Update `AGENTS.md` so future feature plans include version-impact and changelog expectations.

## Acceptance criteria

- A dedicated versioning guide exists and explains how versions are chosen.
- A root changelog exists and is structured for ongoing feature work.
- The roadmap item is marked complete and the README links readers to the new docs.
- `AGENTS.md` requires future feature plans to assess version impact and changelog updates.
- `AGENTS.md` includes the steps contributors should follow when a task participates in a real version increment.
