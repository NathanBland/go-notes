# MCP Binary Distribution Feature Plan

## Goal

Make the stdio MCP server shippable as an installable binary so people can use `go-notes` from common AI tools without cloning the repo or running `go run` locally.

## Required to build it

- Add release automation that builds the MCP server from `cmd/mcp`.
- Produce deterministic archives and checksums for macOS and Linux on `amd64` and `arm64`.
- Add a simple version surface to the MCP binary so packaged builds can be identified.
- Add local release commands to the `Makefile` so contributors can validate snapshot packaging.
- Document install and setup flows for major MCP-capable AI tools.
- Update the roadmap, README, and MCP docs so the feature is discoverable.
- Verify the project still passes unit tests, integration tests, and the integration-backed coverage gate.

## Implementation plan

1. Add a GoReleaser config that builds a single `go-notes-mcp` binary from `cmd/mcp`.
2. Inject version, commit, and build-date metadata into the MCP binary.
3. Add `make` targets for local snapshot packaging and release-config validation.
4. Add a dedicated install guide covering packaged binary setup for Codex, Claude Code, Cursor, and Windsurf.
5. Update README, roadmap, and MCP docs with examples and distribution guidance.
6. Run `make test`, `make test-integration`, and `make coverage-check-integration`.

## Acceptance criteria

- The repo contains a GoReleaser config for building and archiving `go-notes-mcp`.
- The packaged MCP binary exposes build metadata through a version flag.
- The `Makefile` includes discoverable commands for validating and snapshot-building MCP release artifacts.
- Docs include example setup instructions for packaged MCP binaries in Codex, Claude Code, Cursor, and Windsurf.
- README and roadmap reflect that packaged MCP distribution is now part of the implemented project surface.
- `make test` passes.
- `make test-integration` passes.
- `make coverage-check-integration` passes.
