-- +migrate Down
DROP TRIGGER IF EXISTS update_analyses_updated_at ON analyses;
DROP TABLE IF EXISTS analyses CASCADE;