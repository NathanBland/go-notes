# MCP Feature Plan

## Goal

Add the first MCP interface to `go-notes` so LLM clients can use the project through MCP without replacing the REST API.

## What is required

- Current `mcp-go` guidance for stdio servers, tools, and structured tool results
- A small MCP package that reuses the existing `notes.Service`
- A local-only owner-selection strategy for the first MCP slice
- A `cmd/mcp` entrypoint and a Make target to run it
- Tests for the MCP tool handlers
- Documentation and roadmap updates once the first MCP slice lands

## Implementation plan

1. Use Context7 to confirm the current `mcp-go` server and tool patterns.
2. Add `github.com/mark3labs/mcp-go` to the project.
3. Create `internal/mcpapi` with a small tool surface:
   - `list_notes`
   - `get_note`
   - `create_note`
4. Reuse the existing `notes.Service` so MCP and REST share the same business behavior.
5. Add `cmd/mcp` using stdio transport and a local-development owner UUID from env.
6. Add a Make target to run the MCP server locally.
7. Add tests for the MCP tool behavior and validation.
8. Update `README.md`, `ROADMAP.md`, and relevant docs under `docs/`.

## Acceptance criteria

- A local MCP server can be started with stdio transport
- `list_notes`, `get_note`, and `create_note` are exposed as MCP tools
- The MCP tools call the existing notes service instead of reimplementing note behavior
- The first MCP slice is documented as local/development oriented with a temporary owner-UUID auth approach
- Tests pass and documentation is updated to reflect the new capability
