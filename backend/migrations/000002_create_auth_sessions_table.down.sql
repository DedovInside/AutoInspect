-- +migrate Down
DROP TRIGGER IF EXISTS update_auth_sessions_updated_at ON auth_sessions;
DROP TABLE IF EXISTS auth_sessions CASCADE;
