CREATE INDEX idx_notes_search_fts_gin
ON notes
USING GIN (to_tsvector('english', COALESCE(title, '') || ' ' || content));
