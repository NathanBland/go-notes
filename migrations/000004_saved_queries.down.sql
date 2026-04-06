DROP TRIGGER IF EXISTS saved_queries_set_updated_at ON saved_queries;
DROP INDEX IF EXISTS idx_saved_queries_owner_name;
DROP TABLE IF EXISTS saved_queries;
