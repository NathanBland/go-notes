# Deployment

This project now ships three separate container/deployment artifacts on purpose.

## Files and roles

- [`docker-compose.yml`](/Users/nathanbland/projects/codex-workspace/go-notes/docker-compose.yml): local development stack with bind mounts, hot reload, and local build contexts.
- [`docker-compose.prod.yml`](/Users/nathanbland/projects/codex-workspace/go-notes/docker-compose.prod.yml): production-oriented stack for registry-backed deployments such as Portainer.
- [`README.md`](/Users/nathanbland/projects/codex-workspace/go-notes/README.md): the short example entry point for readers who need a quick production compose snippet before opening the full production file.

Keep these three in sync whenever the runtime shape changes.

## Development stack

The development stack is optimized for editing code:

- `Dockerfile.dev`
- source bind mounts
- `air` for hot reload
- local build context instead of published images
- direct port exposure for local iteration

Use it with:

```bash
make docker-up-app
make migrate-up
```

## Production stack

The production stack is optimized for deployment:

- published images instead of source mounts
- durable named volumes for PostgreSQL and Valkey
- `restart: unless-stopped`
- healthchecks for service orchestration
- a separate `migrate` service under the `ops` profile

Validate it with:

```bash
make docker-config-prod
```

That target validates [`docker-compose.prod.yml`](/Users/nathanbland/projects/codex-workspace/go-notes/docker-compose.prod.yml) against [`.env.production.example`](/Users/nathanbland/projects/codex-workspace/go-notes/.env.production.example) by default so local checks do not depend on exporting secrets first.

Run migrations explicitly before or during a rollout:

```bash
docker compose --env-file .env.production -f docker-compose.prod.yml --profile ops run --rm migrate
```

Start the long-running services:

```bash
docker compose --env-file .env.production -f docker-compose.prod.yml up -d api postgres valkey
```

## Portainer notes

For Portainer-style usage:

- point the stack at [`docker-compose.prod.yml`](/Users/nathanbland/projects/codex-workspace/go-notes/docker-compose.prod.yml)
- provide the required environment variables through Portainer or an env file
- run the `migrate` service intentionally as an operational step before promoting the API service

## Published image model

The GitHub Actions workflows publish:

- `ghcr.io/<owner>/go-notes` for the API
- `ghcr.io/<owner>/go-notes-mcp` for the MCP runtime image
- [`.github/workflows/app-image.yml`](/Users/nathanbland/projects/codex-workspace/go-notes/.github/workflows/app-image.yml) for the API image pipeline
- [`.github/workflows/mcp-release.yml`](/Users/nathanbland/projects/codex-workspace/go-notes/.github/workflows/mcp-release.yml) for MCP image and artifact delivery

Version tags and `main` builds can both produce tagged images. The production compose file defaults to the `latest` API image but should usually be pinned to a version tag in real deployments.

## MCP delivery model

The MCP server now has two deployment forms:

- release archives and checksums built through GoReleaser
- a published container image built from [`Dockerfile.mcp`](/Users/nathanbland/projects/codex-workspace/go-notes/Dockerfile.mcp)

The stdio MCP workflow is usually easiest with the packaged binary. The image is mainly useful when a tool prefers container-based command execution.
