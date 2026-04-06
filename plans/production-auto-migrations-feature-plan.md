# Production Auto Migrations Feature Plan

## Goal

Make production deployments run database migrations as part of the normal Compose startup path so operators do not have to remember a separate manual migration step before the API starts.

## Required

- Keep migrations explicit and visible in `docker-compose.prod.yml`.
- Make the API depend on successful migration completion.
- Keep the migration command idempotent by using the checked-in migration tool and migration files.
- Update deployment docs and README production examples so Portainer-style users understand the startup sequence.
- Update project rules so future production runtime changes keep migration behavior in mind.

## Implementation Plan

- Remove the production `migrate` service from the optional `ops` profile.
- Add `api.depends_on.migrate.condition=service_completed_successfully`.
- Keep `migrate.depends_on.postgres.condition=service_healthy`.
- Update `README.md` and `docs/deployment.md` to explain that the production stack runs migrations automatically before API startup.
- Update `AGENTS.md` with a production migration-startup rule.
- Record the behavior in `CHANGELOG.md`.
- Validate the production Compose file.

## Acceptance Criteria

- `docker-compose.prod.yml` includes a default `migrate` service.
- The API service waits for the migrate service to complete successfully before starting.
- Production docs no longer tell operators to run migrations as a separate required pre-step for normal startup.
- README production example mentions the migration gate.
- The feature includes docs/examples, and no MCP tool changes are required because this is deployment behavior only.
- `make docker-config-prod` passes.
- Production compose config validation does not require a running Docker daemon, so YAML checks can run even when Colima or Docker Desktop is stopped.

## Version Impact

- `patch`

## Changelog

- Update `CHANGELOG.md` under the next release section.

## Tooling / Related Functionality

- No new MCP tools are needed.
- The change affects production Compose orchestration only.
