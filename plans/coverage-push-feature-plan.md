# Coverage Push Feature Plan

## Goal

Raise handwritten-code coverage in `go-notes` toward the project target of `80%+` by adding meaningful tests around the largest remaining gaps.

## What is required

- A fresh coverage report to identify the biggest low-coverage packages and files
- Additional tests that exercise real behavior rather than shallow line coverage
- A focus on the highest-leverage areas first, especially handlers, services, and cache-adjacent logic
- Documentation updates once the coverage milestone materially changes roadmap status

## Implementation plan

1. Run the current coverage report and identify the largest gaps in handwritten code.
2. Add tests for the highest-impact uncovered behavior first.
3. Re-run coverage and repeat until the project reaches the target or we identify a concrete blocker.
4. Update the README and roadmap if the work materially changes the viability status.

## Acceptance criteria

- Handwritten-code coverage increases meaningfully from the previous baseline.
- New tests cover real behavior and failure paths, not just trivial getters or wrappers.
- `make test` passes.
- `make coverage` runs successfully.
- If the `80%` threshold is reached, `make coverage-check` passes.
