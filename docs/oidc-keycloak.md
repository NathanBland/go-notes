# OIDC Setup

`go-notes` connects to an external OIDC provider through environment variables. The repo does not bundle its own identity provider in Docker Compose.

The code uses:

- `go-oidc` for provider discovery and ID token verification
- `golang.org/x/oauth2` for the authorization code flow with PKCE

That means the configured issuer URL must support standard OIDC discovery. In practice, `go-oidc` will fetch provider metadata from the issuer's `/.well-known/openid-configuration` endpoint and use the discovered auth, token, and JWKS URLs automatically.

## Required env vars

- `OIDC_ISSUER_URL`
- `OIDC_CLIENT_ID`
- `OIDC_REDIRECT_URL`

## Optional env vars

- `OIDC_CLIENT_SECRET`
- `OIDC_SCOPES`

`OIDC_CLIENT_SECRET` is optional in this project because the flow already uses PKCE. Some providers still require a client secret for confidential web clients, while public clients may rely on PKCE without one.

## Example external configuration

```env
OIDC_ISSUER_URL=https://your-provider.example/realms/go-notes
OIDC_CLIENT_ID=go-notes
OIDC_CLIENT_SECRET=
OIDC_REDIRECT_URL=http://localhost:8080/api/v1/auth/callback
OIDC_SCOPES=openid,profile,email
SESSION_COOKIE_SECURE=false
```

## What each setting means

- `OIDC_ISSUER_URL`: the provider issuer root, not the `/.well-known/openid-configuration` URL itself
- `OIDC_CLIENT_ID`: the registered client/application ID at the provider
- `OIDC_CLIENT_SECRET`: the provider-issued secret for confidential clients, if required
- `OIDC_REDIRECT_URL`: must exactly match the redirect URI registered with the provider
- `OIDC_SCOPES`: defaults to `openid,profile,email` if unset
- `SESSION_COOKIE_SECURE`: should be `true` when the app is behind HTTPS

## External-provider checklist

1. Register an application/client for `go-notes` with your provider.
2. Enable the authorization code flow.
3. Keep PKCE enabled or allowed.
4. Add `http://localhost:8080/api/v1/auth/callback` as a valid redirect URI for local development.
5. Add your production callback URL separately when deploying.
6. Request at least the `openid` scope, plus `profile` and `email` if you want those claims populated.
7. If your provider requires a client secret for web apps, set `OIDC_CLIENT_SECRET`; otherwise leave it blank.

## Keycloak example

Keycloak is still a useful concrete reference even though it is not bundled into this repo.

1. Create a realm named `go-notes`.
2. Create a client named `go-notes`.
3. Enable the standard authorization code flow.
4. Allow PKCE.
5. Add `http://localhost:8080/api/v1/auth/callback` as a valid redirect URI.
6. Add `http://localhost:8080` as a web origin if the login route is being called from a browser app.
7. Use the realm issuer URL as `OIDC_ISSUER_URL`, for example `https://keycloak.example/realms/go-notes`.

## Runtime flow

1. `GET /api/v1/auth/login` generates state, nonce, and PKCE verifier.
2. The API builds the auth URL with PKCE and redirects the browser to the external provider.
3. The provider sends the browser back to `/api/v1/auth/callback`.
4. The API exchanges the code, verifies the ID token, checks the nonce, upserts the user, stores the session in Valkey, and sets the session cookie.
5. When login started from the minimal web UI, the callback redirects the browser back to `/` after setting the cookie. Otherwise the callback returns the JSON user envelope.

## Common pitfalls

- Using the `/.well-known/openid-configuration` URL directly instead of the issuer root
- Registering a redirect URI that does not exactly match `OIDC_REDIRECT_URL`
- Forgetting to set `SESSION_COOKIE_SECURE=true` when testing behind HTTPS
- Omitting `openid` from the scope list
- Assuming every provider requires a client secret even when PKCE-only public clients are supported

## Troubleshooting login start failures

If `GET /api/v1/auth/login` returns:

```json
{"error":{"code":"login_failed","message":"failed to begin login"}}
```

the failure usually happened before the browser was redirected to the provider. In this project, that typically means one of two things:

- Valkey could not store the pending OIDC state/nonce/verifier
- `go-oidc` could not complete provider discovery from `OIDC_ISSUER_URL`

`go-oidc` performs discovery against the issuer's `/.well-known/openid-configuration` document and validates that the discovery document matches the configured issuer. A mismatch between the configured issuer URL and the issuer reported by the provider is a common cause of startup-login failures.

When this happens, check the API server logs. `go-notes` logs the underlying internal error for login setup failures so you can distinguish between cache problems, network reachability problems, TLS/certificate issues, and issuer-mismatch problems without exposing those raw details to API clients.
