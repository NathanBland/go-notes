# UI Consistency Audit Feature Plan

## Goal

Run a focused consistency pass on the server-rendered web UI so the sidebar reflects the real task priority: creating a note first, then working with saved queries and filters, without decorative labels that do not teach anything useful.

## Required to build it

- Move the note-creation form to the top of the sidebar because it is the primary action in the workspace.
- Remove superficial section badges that do not add meaningful instruction.
- Keep the HTMX behavior and server-rendered flow unchanged while improving hierarchy and wording.
- Use consistent section headers and supporting copy so each sidebar block explains its purpose.
- Add tests that lock in the intended section order and prevent decorative badge text from creeping back in.
- Update README, ROADMAP, and web UI docs so the audit is discoverable as part of the teaching surface.

## Implementation plan

1. Add the UI consistency audit item to the roadmap and record the expected outcome.
2. Rework the sidebar template so the note creation form appears first and the section headers use consistent instructional copy.
3. Remove low-value visual chips such as `HTMX`, `Reusable`, and similar decorative labels.
4. Add or update UI tests to verify section order and the absence of those old badge labels.
5. Update README and docs to reflect the refined sidebar hierarchy.
6. Run `make test`, `make test-integration`, and `make coverage-check-integration`.

## Acceptance criteria

- The note creation section is the first section in the workspace sidebar.
- Decorative sidebar chips like `HTMX` and `Reusable` are removed.
- The sidebar sections use consistent, instructional headings or helper copy instead of ornamental labels.
- UI tests verify the section order and the removal of those old labels.
- `README.md`, `ROADMAP.md`, and the relevant files under `docs/` are updated.
- `make test` passes.
- `make test-integration` passes.
- `make coverage-check-integration` passes.
