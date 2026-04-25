-- +migrate Up
CREATE TABLE auth_oauth_identities (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider         VARCHAR(32) NOT NULL,
    provider_user_id VARCHAR(255) NOT NULL,
    email            VARCHAR(255),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (provider, provider_user_id)
);

CREATE INDEX idx_auth_oauth_identities_user_id ON auth_oauth_identities(user_id);

COMMENT ON TABLE auth_oauth_identities IS 'Таблица для хранения связей между пользователями и их OAuth-учетными записями (Google, Facebook и т.д.)';
