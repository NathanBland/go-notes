# Tag Sort Guidance Feature Plan

## Goal

Close the remaining tag-oriented sort roadmap item by confirming the one additional useful case we want to endorse: sorting by title inside a tag-filtered result set.

## Required to build it

- Keep tag-related ordering deterministic and easy to explain.
- Avoid adding ambiguous or expensive new tag-derived sort behavior.
- Add examples and tests that show the recommended pattern clearly across the teaching docs.
- Update the roadmap so the project no longer implies more tag sort work is still pending.

## Implementation plan

1. Add an integration test that proves title sorting works cleanly inside a tag-filtered list.
2. Update filtering docs with a concrete example and a short explanation of why this is the preferred “tag-adjacent” sort.
3. Update the README and roadmap to reflect that the project intentionally stops short of adding more complex tag-sorting rules.

## Acceptance criteria

- Integration coverage proves a tag-filtered list can be sorted by title deterministically.
- Docs include an example of title sorting within a tag-filtered result set.
- The roadmap item for further tag-oriented sorting is closed with an explicit explanation instead of being left ambiguous.
- `make test` passes.
- `make test-integration` passes.
- `make coverage-check-integration` passes.
