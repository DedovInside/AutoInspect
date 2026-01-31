CREATE TABLE audit_logs (
    id          BIGSERIAL PRIMARY KEY,  -- BIGSERIAL для высокой нагрузки
    
    user_id     UUID REFERENCES users(id),
    action      VARCHAR(100) NOT NULL,
    entity_type VARCHAR(50),   -- "analysis", "model", "dataset"
    entity_id   UUID,
    
    -- Контекст
    ip_address  INET,
    user_agent  TEXT,
    request_id  UUID,  -- для трейсинга
    
    -- Детали
    details     JSONB,
    status_code INTEGER,  -- HTTP status
    
    created_at  TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at DESC);
CREATE INDEX idx_audit_logs_entity ON audit_logs(entity_type, entity_id);

-- Партиционирование по месяцам (для больших объёмов)
-- CREATE TABLE audit_logs_2025_01 PARTITION OF audit_logs 
--   FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');