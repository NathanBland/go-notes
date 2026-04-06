# UI Markdown Rendering Feature Plan

## Goal

Render note content as Markdown in the minimal web UI read view so notes are easier to read while still storing raw Markdown in PostgreSQL.

## Required to build it

- Keep note storage and API responses as raw Markdown text.
- Render Markdown only at the HTML UI boundary.
- Use a small Go-native renderer with safe HTML defaults.
- Cover the rendering behavior with tests.
- Update the UI docs after implementation.

## Implementation plan

1. Add a Markdown renderer for the HTML UI layer.
2. Render note content as HTML in the read-only note detail template.
3. Keep edit mode unchanged so users still edit raw Markdown.
4. Add tests that verify Markdown formatting appears in rendered HTML.
5. Update the README and web UI docs.

## Acceptance criteria

- Reading a note in the HTML UI renders Markdown formatting like headings, lists, emphasis, and code blocks.
- Editing a note still shows the original Markdown source text.
- Raw HTML remains disabled by default in the renderer.
- `make test` passes.
