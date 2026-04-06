# OIDC Login Diagnostics Feature Plan

## Goal

Make OIDC login failures easier to diagnose during development without leaking provider internals to API clients.

## Required to build it

- Keep the public JSON error envelope stable and safe.
- Log the underlying internal error on failed login start and callback finish.
- Document the most common OIDC discovery misconfiguration issues for local development.
- Add tests so the logging behavior does not regress.

## Implementation plan

1. Update the auth HTTP handlers to log internal login and callback failures with the request ID.
2. Add handler tests that verify a failed login still returns the generic error envelope while logging the underlying cause.
3. Update the OIDC setup docs with common discovery and issuer-mismatch troubleshooting guidance.

## Acceptance criteria

- `GET /api/v1/auth/login` still returns the existing generic JSON error response on failure.
- The server logs include the underlying OIDC error when login setup fails.
- The OIDC docs explain that the issuer URL must be the issuer root and that discovery/issuer mismatches are a common cause of login-start failures.
- Tests pass.
