-- This schema snapshot exists so sqlc can analyze the current database shape
-- without reading both up and down migration files.
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    oidc_issuer text NOT NULL,
    oidc_subject text NOT NULL,
    email text,
    email_verified boolean NOT NULL DEFAULT false,
    display_name text,
    picture_url text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (oidc_issuer, oidc_subject)
);

CREATE TABLE notes (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title text,
    content text NOT NULL,
    tags text[] NOT NULL DEFAULT '{}',
    archived boolean NOT NULL DEFAULT false,
    shared boolean NOT NULL DEFAULT false,
    share_slug text UNIQUE,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_notes_owner_updated_at ON notes (owner_user_id, updated_at DESC);
CREATE INDEX idx_notes_owner_archived ON notes (owner_user_id, archived);
CREATE INDEX idx_notes_owner_shared ON notes (owner_user_id, shared);
CREATE INDEX idx_notes_share_slug ON notes (share_slug) WHERE share_slug IS NOT NULL;
