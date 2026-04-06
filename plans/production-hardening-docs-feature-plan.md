# Production Hardening Docs Feature Plan

## Goal

Add a release-readiness documentation pass covering HTTPS, reverse proxies, secure cookies, and production rate-limiting expectations.

## What is required

- Production hardening guidance in the deployment docs
- Clear notes on reverse proxy headers and secure cookie behavior
- Rate-limiting expectations for production deployments
- README discoverability for the new guidance
- Version impact assessment: `patch`
- `CHANGELOG.md` update required: `yes`

## Plan

1. Add a focused production hardening section to the deployment docs.
2. Cover HTTPS, reverse proxy expectations, cookie behavior, and rate limiting.
3. Link the guidance from README and any related docs.

## Acceptance criteria

- The docs explain the expected production posture clearly.
- README links readers to the hardening guidance.
- The guidance includes concrete examples or checklists.
- `CHANGELOG.md` reflects the documentation improvement under `Unreleased`.
