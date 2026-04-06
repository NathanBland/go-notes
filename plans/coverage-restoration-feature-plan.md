# Coverage Restoration Feature Plan

## Goal

Restore the integration-backed handwritten-code coverage gate to `80%+` after the recent UI, tag, and MCP surface-area growth.

## Required to build it

- Keep the coverage gate honest instead of masking untested behavior.
- Add tests around the highest-value low-coverage branches first.
- Keep coverage exclusions consistent with the existing policy for command entrypoints and generated code.
- Update docs if the coverage workflow or current status changes.

## Implementation plan

1. Add focused tests for the HTML/UI and auth branches that still have thin coverage.
2. Add direct tests for MCP helper/tool-registration behavior where recent features expanded the surface.
3. Exclude the `cmd/mcp` entrypoint from the handwritten-code coverage gate, matching the existing exclusion for `cmd/api`.
4. Re-run `make test` and `make coverage-check-integration`.
5. Update docs and roadmap status based on the actual outcome.

## Acceptance criteria

- `make test` passes.
- `make coverage-check-integration` passes at `80%+`.
- Coverage exclusions remain limited to generated code and thin command entrypoints.
- Coverage docs and roadmap status accurately describe the current state.
