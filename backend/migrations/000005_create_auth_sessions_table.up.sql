-- +migrate Up
CREATE TABLE auth_sessions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash      VARCHAR(64) NOT NULL UNIQUE,
    token_family_id UUID NOT NULL,
    replaced_by_id  UUID REFERENCES auth_sessions(id) ON DELETE SET NULL,
    user_agent      TEXT,
    ip_address      INET,
    expires_at      TIMESTAMPTZ NOT NULL,
    revoked_at      TIMESTAMPTZ,
    revoked_reason  TEXT,
    last_used_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_auth_sessions_user_id ON auth_sessions(user_id);
CREATE INDEX idx_auth_sessions_family_id ON auth_sessions(token_family_id);
CREATE INDEX idx_auth_sessions_expires_at ON auth_sessions(expires_at);

CREATE TRIGGER update_auth_sessions_updated_at
    BEFORE UPDATE ON auth_sessions
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

