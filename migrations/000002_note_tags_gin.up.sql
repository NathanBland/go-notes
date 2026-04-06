CREATE INDEX idx_notes_tags_gin ON notes USING GIN (tags);
