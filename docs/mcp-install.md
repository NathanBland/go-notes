# MCP Install

This guide covers the packaged `go-notes-mcp` binary for local stdio MCP clients.
The project also now publishes a container-friendly MCP runtime image for environments that prefer `docker run` style command execution.

## What gets distributed

The release config now builds `go-notes-mcp` archives for:

- macOS `amd64`
- macOS `arm64`
- Linux `amd64`
- Linux `arm64`

Each release archive includes:

- the `go-notes-mcp` binary
- `README.md`
- `docs/mcp.md`
- this install guide
- a `checksums.txt` file published alongside the archives

## Build locally

Validate the release config:

```bash
make release-check-mcp
```

Build local snapshot archives under `dist/`:

```bash
make release-snapshot-mcp
```

The first run may download GoReleaser through `go run`.

## Required runtime environment

The packaged binary still talks directly to PostgreSQL and Valkey, so it needs the same core MCP runtime environment as `make run-mcp`.

Required:

- `DATABASE_URL`
- `VALKEY_ADDR`
- `MCP_OWNER_USER_ID`

Optional:

- `VALKEY_PASSWORD`
- `NOTE_CACHE_TTL`
- `LIST_CACHE_TTL`

You can inspect the packaged binary version metadata with:

```bash
./go-notes-mcp -version
```

## Generic stdio launch pattern

For stdio MCP clients, the important shape is:

- command: absolute path to `go-notes-mcp`
- args: usually none
- env: database, cache, and owner-scoping values
- working directory: optional, but keeping it at the project root makes local paths and logs easier to reason about

Example:

```text
command=/Users/nathanbland/bin/go-notes-mcp
args=[]
env:
  DATABASE_URL=postgres://postgres:postgres@localhost:5432/go_notes?sslmode=disable
  VALKEY_ADDR=127.0.0.1:6379
  MCP_OWNER_USER_ID=<user-uuid>
```

## Container image launch pattern

The published MCP image is most useful when a client can run a Docker command as its stdio transport.

Example:

```text
command=docker
args=[
  "run",
  "--rm",
  "-i",
  "-e", "DATABASE_URL=postgres://postgres:postgres@host.docker.internal:5432/go_notes?sslmode=disable",
  "-e", "VALKEY_ADDR=host.docker.internal:6379",
  "-e", "MCP_OWNER_USER_ID=<user-uuid>",
  "ghcr.io/nathanbland/go-notes-mcp:latest"
]
```

For Linux hosts, replace `host.docker.internal` with the hostname or bridge IP that reaches your PostgreSQL and Valkey services.

## Codex

The current Codex desktop custom MCP form maps cleanly to the packaged binary:

- Name: `go-notes`
- Transport: `STDIO`
- Command to launch: `/absolute/path/to/go-notes-mcp`
- Arguments: none
- Environment variables:
  - `DATABASE_URL`
  - `VALKEY_ADDR`
  - `MCP_OWNER_USER_ID`
- Working directory: your local `go-notes` project root, or another stable directory if you prefer

This matches the same stdio command pattern the project already uses during local development.

## Claude Code

Anthropic’s MCP docs describe project-level `.mcp.json` configuration for stdio servers. A packaged `go-notes-mcp` entry looks like this:

```json
{
  "mcpServers": {
    "go-notes": {
      "command": "/absolute/path/to/go-notes-mcp",
      "args": [],
      "env": {
        "DATABASE_URL": "postgres://postgres:postgres@localhost:5432/go_notes?sslmode=disable",
        "VALKEY_ADDR": "127.0.0.1:6379",
        "MCP_OWNER_USER_ID": "11111111-1111-1111-1111-111111111111"
      }
    }
  }
}
```

## Cursor

Cursor’s MCP docs describe a `mcp.json` entry for local stdio servers. A packaged setup looks like:

```json
{
  "mcpServers": {
    "go-notes": {
      "command": "/absolute/path/to/go-notes-mcp",
      "args": [],
      "env": {
        "DATABASE_URL": "postgres://postgres:postgres@localhost:5432/go_notes?sslmode=disable",
        "VALKEY_ADDR": "127.0.0.1:6379",
        "MCP_OWNER_USER_ID": "11111111-1111-1111-1111-111111111111"
      }
    }
  }
}
```

## Windsurf

Windsurf’s Cascade MCP docs describe `~/.codeium/windsurf/mcp_config.json` for stdio servers. A packaged setup looks like:

```json
{
  "mcpServers": {
    "go-notes": {
      "command": "/absolute/path/to/go-notes-mcp",
      "args": [],
      "env": {
        "DATABASE_URL": "postgres://postgres:postgres@localhost:5432/go_notes?sslmode=disable",
        "VALKEY_ADDR": "127.0.0.1:6379",
        "MCP_OWNER_USER_ID": "11111111-1111-1111-1111-111111111111"
      }
    }
  }
}
```

## Operational notes

- Keep the binary local-only until the MCP auth roadmap item is complete. The current `MCP_OWNER_USER_ID` model is intentionally single-user.
- Prefer absolute binary paths in client configs so restarts and editor launches do not depend on shell `PATH` state.
- Keep PostgreSQL and Valkey running before launching the MCP binary.

## Sources

- GoReleaser docs via Context7: `/goreleaser/goreleaser`
- `mcp-go` stdio docs via Context7: `/mark3labs/mcp-go`
- OpenAI Docs MCP page for Codex MCP configuration examples: [Docs MCP](https://developers.openai.com/learn/docs-mcp)
- Anthropic Claude Code MCP docs: [MCP in the SDK](https://docs.claude.com/en/docs/agent-sdk/mcp)
- Cursor MCP docs: [Model Context Protocol](https://docs.cursor.com/context/model-context-protocol)
- Windsurf Cascade MCP docs: [Cascade MCP Integration](https://docs.windsurf.com/windsurf/cascade/mcp)
