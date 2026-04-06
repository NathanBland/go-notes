# Production Compose Feature Plan

## Goal

Add a production-oriented Compose file that is suitable for Portainer-style deployments of the API stack.

## What is required

- A production compose file that uses published images instead of bind-mounted source code
- Durable volumes for PostgreSQL and Valkey
- Env-driven configuration for secrets, URLs, and image tags
- A migration strategy that works in production

## Plan

1. Add `docker-compose.prod.yml` for API, migrations, PostgreSQL, and Valkey.
2. Use published images and environment variables for deployment-time configuration.
3. Add healthchecks and restart policies that fit a long-running deployment.
4. Document how to apply the compose file in tools like Portainer.

## Acceptance criteria

- A production compose file exists and validates with Compose.
- The compose file is suitable for a registry-backed deployment flow.
- The docs explain the expected env vars and runtime model.
- The feature includes an example deployment configuration.
