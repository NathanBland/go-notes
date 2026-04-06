# Throttling Feature Plan

## Goal

Add request-throttling middleware to protect the most abuse-prone routes in `go-notes` without changing the existing API contract for normal traffic.

## What is required

- A small, explicit middleware implementation that fits the existing `net/http` stack
- Configurable throttle settings exposed through environment-driven app config
- Route wiring for login, callback, and shared-note access
- Clear error responses when requests exceed the limit
- Unit tests for allowed and throttled requests
- Documentation updates in the README and roadmap once the feature is in place

## Implementation plan

1. Add throttle configuration fields to app config with safe development defaults.
2. Implement a small HTTP middleware that tracks per-client request rate and burst allowance.
3. Apply the middleware only to the routes called out in the roadmap: login, callback, and public shared-note access.
4. Return a consistent JSON `429` response for throttled requests.
5. Add tests covering normal access, repeated requests, and route scoping.
6. Update the README and roadmap to reflect the completed feature.

## Acceptance criteria

- Repeated requests to the throttled routes eventually return `429 Too Many Requests`.
- Non-throttled routes keep their current behavior.
- Throttled responses use the project's JSON error envelope.
- Throttle settings are configurable through environment variables.
- `make test` passes after the change.
- The README and roadmap reflect the new throttling support.
