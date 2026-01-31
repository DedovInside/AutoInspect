-- +migrate Up
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";  -- для gen_random_uuid()

CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username      VARCHAR(50) UNIQUE NOT NULL,
    email         VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role          VARCHAR(20) NOT NULL DEFAULT 'user' 
                  CHECK (role IN ('user', 'owner', 'admin')),

    -- Дополнительная безопасность
    email_verified BOOLEAN DEFAULT FALSE, -- для регистрации через email
    is_active      BOOLEAN DEFAULT TRUE, -- для блокировки пользователя

    -- Метаданные
    created_at    TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    last_login    TIMESTAMPTZ,

    -- Опционально: для rate limiting
    api_calls_count INTEGER DEFAULT 0,
    api_quota_reset_at TIMESTAMPTZ
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_role ON users(role);

-- Trigger для автоматического updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();