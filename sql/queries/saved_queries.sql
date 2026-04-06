-- name: CreateSavedQuery :one
INSERT INTO saved_queries (
    owner_user_id,
    name,
    query_string
)
VALUES (
    sqlc.arg(owner_user_id),
    sqlc.arg(name),
    sqlc.arg(query_string)
)
RETURNING *;

-- name: GetSavedQueryForOwner :one
SELECT *
FROM saved_queries
WHERE id = sqlc.arg(id)
  AND owner_user_id = sqlc.arg(owner_user_id);

-- name: ListSavedQueriesForOwner :many
SELECT *
FROM saved_queries
WHERE owner_user_id = sqlc.arg(owner_user_id)
ORDER BY lower(name) ASC, name ASC, id ASC;

-- name: DeleteSavedQueryForOwner :execrows
DELETE FROM saved_queries
WHERE id = sqlc.arg(id)
  AND owner_user_id = sqlc.arg(owner_user_id);
