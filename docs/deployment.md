# Deployment

This project now ships three separate container/deployment artifacts on purpose.

## Files and roles

- [`docker-compose.yml`](../docker-compose.yml): local development stack with bind mounts, hot reload, and local build contexts.
- [`docker-compose.prod.yml`](../docker-compose.prod.yml): production-oriented stack for registry-backed deployments such as Portainer.
- [`README.md`](../README.md): the short example entry point for readers who need a quick production compose snippet before opening the full production file.

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
- a one-shot `migrate` service that runs before the API starts

Validate it with:

```bash
make docker-config-prod
```

That target validates [`docker-compose.prod.yml`](../docker-compose.prod.yml) against [`.env.production.example`](../.env.production.example) by default so local checks do not depend on exporting secrets first.

Start the long-running services:

```bash
docker compose --env-file .env.production -f docker-compose.prod.yml up -d
```

The default production startup includes the `migrate` service. The API service waits for that one-shot migration container to complete successfully before it starts. If a migration fails, the API should remain stopped so the deployment fails visibly instead of booting against an incomplete schema.

## Portainer notes

For Portainer-style usage:

- point the stack at [`docker-compose.prod.yml`](../docker-compose.prod.yml)
- provide the required environment variables through Portainer or an env file
- let the default stack start include the `migrate` service; the API is gated on successful migration completion
- after changing image tags or environment values, redeploy the stack rather than starting only the API container, so the migration gate runs for that rollout

## Production hardening

Use this checklist before calling a deployment production-ready:

- terminate TLS at your reverse proxy or load balancer
- set `BASE_URL` to the public `https://...` origin the browser will actually use
- set `SESSION_COOKIE_SECURE=true` so the session cookie is only sent over HTTPS
- keep `OIDC_REDIRECT_URL` aligned with the same public HTTPS origin
- keep PostgreSQL and Valkey off the public internet unless you have a specific network reason to expose them
- pin the API image to a version tag instead of `latest`

### HTTPS and reverse proxies

`go-notes` does not try to terminate TLS itself. The expected production shape is:

- a reverse proxy or ingress handles HTTPS
- the proxy forwards traffic to the API container over an internal network
- the browser only ever sees the public HTTPS origin

That means local development values like `BASE_URL=http://localhost:8080` should not be reused in production. Use the public HTTPS origin instead.

### Cookie implications

The session cookie is server-side and opaque, but the browser still needs secure settings around it.

- use `SESSION_COOKIE_SECURE=true` in production
- keep the app behind HTTPS so modern browsers will consistently send the cookie
- avoid mixing HTTP and HTTPS origins during login flows, or the cookie and OIDC callback behavior will feel inconsistent

### Reverse proxy expectations

At the proxy layer, preserve the basic HTTP behavior the app depends on:

- forward the original host and request path unchanged
- allow the OIDC callback route at `/api/v1/auth/callback`
- avoid stripping cookies or caching authenticated HTML/API responses
- apply request-body limits that still allow normal JSON note payloads

If you later add proxy-specific headers or trusted-proxy logic, document those changes in this file and keep the README production example aligned.

### Rate limiting in production

The built-in throttling currently protects:

- `GET /api/v1/auth/login`
- `GET /api/v1/auth/callback`
- `GET /api/v1/notes/shared/{slug}`

For production, treat that as the baseline rather than the whole policy:

- keep the app-level throttling enabled through `THROTTLE_REQUESTS_PER_SECOND` and `THROTTLE_BURST`
- consider adding proxy or gateway rate limiting in front of the API as a second layer
- review the shared-note route especially if you expect public links to be exposed widely

## Published image model

The GitHub Actions workflows publish:

- `ghcr.io/<owner>/go-notes` for the API
- `ghcr.io/<owner>/go-notes-mcp` for the MCP runtime image
- [`.github/workflows/app-image.yml`](../.github/workflows/app-image.yml) for the API image pipeline
- [`.github/workflows/mcp-release.yml`](../.github/workflows/mcp-release.yml) for MCP image and artifact delivery

Version tags and `main` builds can both produce tagged images. The production compose file defaults to the `latest` API image but should usually be pinned to a version tag in real deployments.

## MCP delivery model

The MCP server now has two deployment forms:

- release archives and checksums built through GoReleaser
- a published container image built from [`Dockerfile.mcp`](../Dockerfile.mcp)

The stdio MCP workflow is usually easiest with the packaged binary. The image is mainly useful when a tool prefers container-based command execution.
