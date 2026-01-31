-- +migrate Down
DROP TRIGGER IF EXISTS update_models_updated_at ON models;
DROP TABLE IF EXISTS models CASCADE;