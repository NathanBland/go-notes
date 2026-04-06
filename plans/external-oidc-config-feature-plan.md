# External OIDC Config Feature Plan

## Goal

Make `go-notes` easier to connect to a real external OIDC provider by tightening the environment-variable examples, setup docs, and configuration guidance.

## What is required

- Review the current `go-oidc` and `golang.org/x/oauth2` guidance for provider discovery, ID token verification, and PKCE
- Update `.env.example` so it reflects an external provider instead of implying a local in-repo OIDC service
- Update the README and OIDC documentation so issuer URL, redirect URI, scopes, secure cookies, and client secret expectations are clear
- Add or refresh tests if configuration behavior needs to be made more explicit

## Implementation plan

1. Review current OIDC docs and config behavior in the repo.
2. Pull the current `go-oidc` and `oauth2` guidance from Context7.
3. Update `.env.example` with provider-agnostic placeholders and concise teaching comments.
4. Rewrite the OIDC setup doc around external-provider configuration, while still keeping a Keycloak example.
5. Update the README and roadmap to reflect the improved env-driven OIDC setup.
6. Run the test suite if any code or config behavior changes.

## Acceptance criteria

- `.env.example` clearly shows how to point the app at an external OIDC provider
- The OIDC doc explains provider discovery, redirect URI setup, scopes, PKCE, and secure-cookie expectations
- The README points developers to the right OIDC setup flow without implying an in-repo provider
- The roadmap reflects the new status of the external OIDC configuration work
- Tests still pass if any code or config behavior changed
