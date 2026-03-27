-- +migrate Down
DROP TRIGGER IF EXISTS update_auth_refresh_sessions_updated_at ON auth_refresh_sessions;
DROP TABLE IF EXISTS auth_refresh_sessions CASCADE;

