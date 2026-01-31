-- +migrate Down
DROP TRIGGER IF EXISTS update_datasets_updated_at ON datasets;
DROP TABLE IF EXISTS datasets CASCADE;