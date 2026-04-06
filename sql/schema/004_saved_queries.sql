CREATE TABLE saved_queries (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name text NOT NULL,
    query_string text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (owner_user_id, name)
);

CREATE INDEX idx_saved_queries_owner_name ON saved_queries (owner_user_id, lower(name), id);

CREATE TRIGGER saved_queries_set_updated_at
BEFORE UPDATE ON saved_queries
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();
