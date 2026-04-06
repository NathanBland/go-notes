# UI Auth Redirect Feature Plan

## Goal

Return browser users to the minimal web UI after a successful OIDC callback instead of leaving them on the JSON callback response.

## Required to build it

- Preserve the existing OIDC security model with state, nonce, and PKCE.
- Carry a safe post-login redirect target through the pending auth state.
- Keep API callback behavior available when no UI redirect target is present.
- Cover the new behavior with tests and update the docs.

## Implementation plan

1. Extend the pending OIDC auth state to store an optional post-login redirect target.
2. Update login initiation to accept a safe redirect target and persist it with the pending state.
3. Update the callback handler to redirect to the stored target after setting the session cookie.
4. Point the UI login action at the login endpoint with a redirect target back to the root workspace.
5. Add tests for UI redirect behavior and refresh the relevant docs.

## Acceptance criteria

- UI-triggered login redirects back to `/` after a successful callback.
- The session cookie is still set before the redirect.
- Unsafe redirect targets are ignored in favor of a safe local default.
- The existing API callback JSON response still works when no redirect target is present.
- `make test` passes.
