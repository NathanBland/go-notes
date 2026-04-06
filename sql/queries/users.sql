-- name: UpsertUserFromOIDC :one
INSERT INTO users (
    oidc_issuer,
    oidc_subject,
    email,
    email_verified,
    display_name,
    picture_url
)
VALUES (
    sqlc.arg(oidc_issuer),
    sqlc.arg(oidc_subject),
    sqlc.narg(email),
    sqlc.arg(email_verified),
    sqlc.narg(display_name),
    sqlc.narg(picture_url)
)
ON CONFLICT (oidc_issuer, oidc_subject)
DO UPDATE SET
    email = EXCLUDED.email,
    email_verified = EXCLUDED.email_verified,
    display_name = EXCLUDED.display_name,
    picture_url = EXCLUDED.picture_url,
    updated_at = now()
RETURNING *;

-- name: GetUserByID :one
SELECT *
FROM users
WHERE id = $1;
