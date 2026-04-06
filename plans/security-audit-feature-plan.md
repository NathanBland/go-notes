# Security Audit Feature Plan

## Goal

Run the roadmap security audit and turn the meaningful findings into code, tests, and docs so `go-notes` stays safe as a public teaching API.

## What is required

- Review owner-scoped note access across REST, UI, service, store, and MCP flows.
- Review the public shared-note route for data leakage or unintended discoverability.
- Review request parsing and query handling for common API hardening gaps, especially JSON parsing and SQL-shaping inputs.
- Add regression tests for any issue we fix.
- Update docs and roadmap status so the audit findings are discoverable to readers.

## Implementation plan

1. Audit the shared-note route and owner-scoped note access rules.
2. Fix any public data leakage, especially internal note identifiers or owner identifiers leaking through public shared responses.
3. Tighten JSON parsing so request bodies reject trailing payload data rather than silently accepting the first JSON value.
4. Add regression tests for:
   - owner-scoped note access by a different user
   - public shared-note response shape
   - strict JSON parsing behavior
5. Update the relevant docs, README, and roadmap to capture the audit findings and new guarantees.

## Acceptance criteria

- Public shared-note reads no longer expose internal note identifiers or owner identifiers.
- Owner-scoped notes remain inaccessible by raw UUID to other authenticated users, with regression coverage proving it.
- JSON note create/patch parsing rejects malformed trailing payloads.
- The project docs explain the security behavior and include examples or explicit guarantees where relevant.
- `make test` passes.
- `make test-integration` passes.
- `make coverage-check-integration` passes.
