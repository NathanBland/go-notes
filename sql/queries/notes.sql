-- name: CreateNote :one
INSERT INTO notes (
    owner_user_id,
    title,
    content,
    tags,
    archived,
    shared,
    share_slug
)
VALUES (
    sqlc.arg(owner_user_id),
    sqlc.narg(title),
    sqlc.arg(content),
    sqlc.arg(tags),
    sqlc.arg(archived),
    sqlc.arg(shared),
    sqlc.narg(share_slug)
)
RETURNING *;

-- name: GetNoteByIDForOwner :one
SELECT *
FROM notes
WHERE id = sqlc.arg(id)
  AND owner_user_id = sqlc.arg(owner_user_id);

-- name: GetNoteByShareSlug :one
SELECT *
FROM notes
WHERE share_slug = sqlc.arg(share_slug)
  AND shared = true;

-- name: UpdateNotePatch :one
UPDATE notes
SET
    title = CASE
        WHEN sqlc.arg(title_is_set)::boolean THEN sqlc.narg(title)::text
        ELSE title
    END,
    content = CASE
        WHEN sqlc.arg(content_is_set)::boolean THEN sqlc.narg(content)::text
        ELSE content
    END,
    tags = CASE
        WHEN sqlc.arg(tags_is_set)::boolean THEN sqlc.arg(tags)::text[]
        ELSE tags
    END,
    archived = CASE
        WHEN sqlc.arg(archived_is_set)::boolean THEN sqlc.arg(archived)::boolean
        ELSE archived
    END,
    shared = CASE
        WHEN sqlc.arg(shared_is_set)::boolean THEN sqlc.arg(shared)::boolean
        ELSE shared
    END,
    share_slug = CASE
        WHEN sqlc.arg(share_slug_is_set)::boolean THEN sqlc.narg(share_slug)::text
        ELSE share_slug
    END,
    updated_at = now()
WHERE id = sqlc.arg(id)
  AND owner_user_id = sqlc.arg(owner_user_id)
RETURNING *;

-- name: DeleteNoteForOwner :execrows
DELETE FROM notes
WHERE id = sqlc.arg(id)
  AND owner_user_id = sqlc.arg(owner_user_id);

-- Filtering, sorting, pagination, and total counts intentionally live in SQL.
-- That keeps the API behavior authoritative and lets PostgreSQL do the work it
-- is good at. User-controlled values stay bound through sqlc args/nargs, and
-- sort behavior is limited to CASE-based allowlists instead of interpolated SQL
-- strings.
-- name: ListNotesForOwner :many
SELECT *
FROM notes
WHERE owner_user_id = sqlc.arg(owner_user_id)
  AND (
    sqlc.arg(status_filter)::text = 'all'
    OR (sqlc.arg(status_filter)::text = 'active' AND archived = false)
    OR (sqlc.arg(status_filter)::text = 'archived' AND archived = true)
  )
  AND (sqlc.narg(shared_filter)::boolean IS NULL OR shared = sqlc.narg(shared_filter)::boolean)
  AND (
    sqlc.narg(has_title_filter)::boolean IS NULL
    OR (
      sqlc.narg(has_title_filter)::boolean = true
      AND NULLIF(BTRIM(COALESCE(title, '')), '') IS NOT NULL
    )
    OR (
      sqlc.narg(has_title_filter)::boolean = false
      AND NULLIF(BTRIM(COALESCE(title, '')), '') IS NULL
    )
  )
  AND (
    NOT sqlc.arg(tag_filter_enabled)::boolean
    OR (
      sqlc.arg(tag_match_mode)::text = 'all'
      AND tags @> sqlc.arg(tag_filters)::text[]
    )
    OR (
      sqlc.arg(tag_match_mode)::text = 'any'
      AND tags && sqlc.arg(tag_filters)::text[]
    )
  )
  AND (sqlc.narg(tag_count_min_filter)::int IS NULL OR cardinality(tags) >= sqlc.narg(tag_count_min_filter)::int)
  AND (sqlc.narg(tag_count_max_filter)::int IS NULL OR cardinality(tags) <= sqlc.narg(tag_count_max_filter)::int)
  AND (
    sqlc.narg(search_filter)::text IS NULL
    OR (
      sqlc.arg(search_mode)::text = 'plain'
      AND (
        COALESCE(title, '') ILIKE '%' || sqlc.narg(search_filter)::text || '%'
        OR content ILIKE '%' || sqlc.narg(search_filter)::text || '%'
      )
    )
    OR (
      sqlc.arg(search_mode)::text = 'fts'
      AND websearch_to_tsquery('english', sqlc.narg(search_filter)::text) @@ to_tsvector('english', COALESCE(title, '') || ' ' || content)
    )
  )
  AND (sqlc.narg(created_after_filter)::timestamptz IS NULL OR created_at >= sqlc.narg(created_after_filter)::timestamptz)
  AND (sqlc.narg(created_before_filter)::timestamptz IS NULL OR created_at <= sqlc.narg(created_before_filter)::timestamptz)
  AND (sqlc.narg(updated_after_filter)::timestamptz IS NULL OR updated_at >= sqlc.narg(updated_after_filter)::timestamptz)
  AND (sqlc.narg(updated_before_filter)::timestamptz IS NULL OR updated_at <= sqlc.narg(updated_before_filter)::timestamptz)
ORDER BY
  CASE WHEN sqlc.arg(sort_field)::text = 'title' AND sqlc.arg(sort_direction)::text = 'asc' THEN lower(COALESCE(title, '')) END ASC,
  CASE WHEN sqlc.arg(sort_field)::text = 'title' AND sqlc.arg(sort_direction)::text = 'desc' THEN lower(COALESCE(title, '')) END DESC,
  -- PostgreSQL arrays are 1-based, so tags[1] means "the first stored tag".
  -- Empty arrays return NULL here, which lets us keep a deterministic
  -- SQL-side primary-tag sort without pushing custom ordering logic into Go.
  CASE WHEN sqlc.arg(sort_field)::text = 'primary_tag' AND sqlc.arg(sort_direction)::text = 'asc' THEN lower(COALESCE(tags[1], '')) END ASC,
  CASE WHEN sqlc.arg(sort_field)::text = 'primary_tag' AND sqlc.arg(sort_direction)::text = 'desc' THEN lower(COALESCE(tags[1], '')) END DESC,
  CASE WHEN sqlc.arg(sort_field)::text = 'created_at' AND sqlc.arg(sort_direction)::text = 'asc' THEN created_at END ASC,
  CASE WHEN sqlc.arg(sort_field)::text = 'created_at' AND sqlc.arg(sort_direction)::text = 'desc' THEN created_at END DESC,
  CASE WHEN sqlc.arg(sort_field)::text = 'updated_at' AND sqlc.arg(sort_direction)::text = 'asc' THEN updated_at END ASC,
  CASE WHEN sqlc.arg(sort_field)::text = 'updated_at' AND sqlc.arg(sort_direction)::text = 'desc' THEN updated_at END DESC,
  CASE WHEN sqlc.arg(sort_field)::text = 'tag_count' AND sqlc.arg(sort_direction)::text = 'asc' THEN cardinality(tags) END ASC,
  CASE WHEN sqlc.arg(sort_field)::text = 'tag_count' AND sqlc.arg(sort_direction)::text = 'desc' THEN cardinality(tags) END DESC,
  CASE WHEN sqlc.arg(sort_field)::text = 'relevance' AND sqlc.arg(sort_direction)::text = 'asc' THEN ts_rank_cd(to_tsvector('english', COALESCE(title, '') || ' ' || content), websearch_to_tsquery('english', COALESCE(sqlc.narg(search_filter)::text, '')), 32) END ASC,
  CASE WHEN sqlc.arg(sort_field)::text = 'relevance' AND sqlc.arg(sort_direction)::text = 'desc' THEN ts_rank_cd(to_tsvector('english', COALESCE(title, '') || ' ' || content), websearch_to_tsquery('english', COALESCE(sqlc.narg(search_filter)::text, '')), 32) END DESC,
  id ASC
LIMIT sqlc.arg(limit_count)
OFFSET sqlc.arg(offset_count);

-- Count uses the same filters as ListNotesForOwner so pagination metadata
-- stays consistent with the actual result set.
-- name: CountNotesForOwner :one
SELECT count(*)
FROM notes
WHERE owner_user_id = sqlc.arg(owner_user_id)
  AND (
    sqlc.arg(status_filter)::text = 'all'
    OR (sqlc.arg(status_filter)::text = 'active' AND archived = false)
    OR (sqlc.arg(status_filter)::text = 'archived' AND archived = true)
  )
  AND (sqlc.narg(shared_filter)::boolean IS NULL OR shared = sqlc.narg(shared_filter)::boolean)
  AND (
    sqlc.narg(has_title_filter)::boolean IS NULL
    OR (
      sqlc.narg(has_title_filter)::boolean = true
      AND NULLIF(BTRIM(COALESCE(title, '')), '') IS NOT NULL
    )
    OR (
      sqlc.narg(has_title_filter)::boolean = false
      AND NULLIF(BTRIM(COALESCE(title, '')), '') IS NULL
    )
  )
  AND (
    NOT sqlc.arg(tag_filter_enabled)::boolean
    OR (
      sqlc.arg(tag_match_mode)::text = 'all'
      AND tags @> sqlc.arg(tag_filters)::text[]
    )
    OR (
      sqlc.arg(tag_match_mode)::text = 'any'
      AND tags && sqlc.arg(tag_filters)::text[]
    )
  )
  AND (sqlc.narg(tag_count_min_filter)::int IS NULL OR cardinality(tags) >= sqlc.narg(tag_count_min_filter)::int)
  AND (sqlc.narg(tag_count_max_filter)::int IS NULL OR cardinality(tags) <= sqlc.narg(tag_count_max_filter)::int)
  AND (
    sqlc.narg(search_filter)::text IS NULL
    OR (
      sqlc.arg(search_mode)::text = 'plain'
      AND (
        COALESCE(title, '') ILIKE '%' || sqlc.narg(search_filter)::text || '%'
        OR content ILIKE '%' || sqlc.narg(search_filter)::text || '%'
      )
    )
    OR (
      sqlc.arg(search_mode)::text = 'fts'
      AND websearch_to_tsquery('english', sqlc.narg(search_filter)::text) @@ to_tsvector('english', COALESCE(title, '') || ' ' || content)
    )
  )
  AND (sqlc.narg(created_after_filter)::timestamptz IS NULL OR created_at >= sqlc.narg(created_after_filter)::timestamptz)
  AND (sqlc.narg(created_before_filter)::timestamptz IS NULL OR created_at <= sqlc.narg(created_before_filter)::timestamptz)
  AND (sqlc.narg(updated_after_filter)::timestamptz IS NULL OR updated_at >= sqlc.narg(updated_after_filter)::timestamptz)
  AND (sqlc.narg(updated_before_filter)::timestamptz IS NULL OR updated_at <= sqlc.narg(updated_before_filter)::timestamptz);

-- Tag discovery intentionally happens in SQL so every caller gets the same
-- owner scoping, counting, and deterministic ordering behavior.
-- name: ListTagsForOwner :many
SELECT
    expanded.tag::text AS tag,
    count(*)::bigint AS count
FROM notes
CROSS JOIN LATERAL unnest(tags) AS expanded(tag)
WHERE owner_user_id = sqlc.arg(owner_user_id)
  AND btrim(expanded.tag) <> ''
GROUP BY expanded.tag
ORDER BY count(*) DESC, lower(expanded.tag) ASC, expanded.tag ASC;

-- Related-note discovery intentionally stays in SQL so overlap ranking,
-- owner scoping, and deterministic ordering are shared by every caller.
-- name: FindRelatedNotesForOwner :many
WITH source AS (
    SELECT tags
    FROM notes
    WHERE notes.id = sqlc.arg(note_id)
      AND notes.owner_user_id = sqlc.arg(owner_user_id)
),
related AS (
    SELECT
        n.*,
        COALESCE(shared.shared_tags, ARRAY[]::text[])::text[] AS shared_tags,
        cardinality(COALESCE(shared.shared_tags, ARRAY[]::text[])::text[])::int AS shared_tag_count
    FROM notes AS n
    CROSS JOIN source
    CROSS JOIN LATERAL (
        SELECT array_agg(tag ORDER BY lower(tag), tag) AS shared_tags
        FROM (
            SELECT DISTINCT tag
            FROM unnest(n.tags) AS note_tag(tag)
            WHERE btrim(tag) <> ''
            INTERSECT
            SELECT DISTINCT tag
            FROM unnest(source.tags) AS source_tag(tag)
            WHERE btrim(tag) <> ''
        ) AS overlap
    ) AS shared
    WHERE n.owner_user_id = sqlc.arg(owner_user_id)
      AND n.id <> sqlc.arg(note_id)
)
SELECT *
FROM related
WHERE shared_tag_count > 0
ORDER BY shared_tag_count DESC, lower(COALESCE(title, '')) ASC, related.id ASC
LIMIT sqlc.arg(limit_count);
