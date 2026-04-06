# UI Markdown Styling Feature Plan

## Goal

Style rendered Markdown in the minimal web UI so headings, paragraphs, lists, blockquotes, code, and tables read like real note content instead of raw unstyled HTML.

## Required to build it

- Keep `goldmark` as the Markdown renderer.
- Apply styles only at the HTML UI layer.
- Use a small prose-style CSS treatment that fits the existing visual design.
- Add tests for the rendered Markdown container and update docs.

## Implementation plan

1. Add a dedicated Markdown content wrapper class in the note detail template.
2. Add a small stylesheet for Markdown elements like headings, paragraphs, lists, blockquotes, code, preformatted blocks, and tables.
3. Keep edit mode unchanged so raw Markdown stays visible in the form.
4. Add handler tests that verify the Markdown wrapper is present.
5. Update the README and web UI docs.

## Acceptance criteria

- Rendered Markdown in the note detail view has consistent styling for common block types.
- Inline code and fenced code blocks are visually distinct.
- Lists, blockquotes, and tables are readable in the dark UI.
- Edit mode still shows raw Markdown.
- `make test` passes.
