-- +migrate Up
CREATE TABLE model_training_requests
(
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    initiator_user_id  UUID         NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    initiator_role     VARCHAR(50)  NOT NULL CHECK (initiator_role IN ('user', 'car_service', 'admin')),

    make               VARCHAR(100) NOT NULL,
    model              VARCHAR(100) NOT NULL,
    generation         VARCHAR(100),
    year_from          INTEGER      NOT NULL,
    year_to            INTEGER,
    description        TEXT         NOT NULL CHECK (char_length(trim(description)) > 0),

    status             VARCHAR(50)  NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'approved', 'rejected', 'in_progress', 'completed')),
    admin_comment      TEXT,
    reviewed_by        UUID REFERENCES users (id) ON DELETE SET NULL,
    reviewed_at        TIMESTAMPTZ,

    created_model_id   UUID REFERENCES car_models (id) ON DELETE SET NULL,
    idempotency_key    VARCHAR(255),

    created_at         TIMESTAMPTZ           DEFAULT CURRENT_TIMESTAMP,
    updated_at         TIMESTAMPTZ           DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT chk_model_training_requests_year_range
        CHECK (year_to IS NULL OR year_to >= year_from)
);

CREATE INDEX idx_model_training_requests_initiator_history
    ON model_training_requests (initiator_user_id, created_at DESC);

CREATE INDEX idx_model_training_requests_status
    ON model_training_requests (status, created_at DESC);

CREATE INDEX idx_model_training_requests_admin_queue
    ON model_training_requests (status, created_at ASC)
    WHERE status IN ('pending', 'approved', 'in_progress');

CREATE UNIQUE INDEX idx_model_training_requests_idempotency
    ON model_training_requests (initiator_user_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL;

CREATE UNIQUE INDEX idx_model_training_requests_unique_active_car
    ON model_training_requests (
        initiator_user_id,
        lower(make),
        lower(model),
        COALESCE(lower(generation), ''),
        year_from,
        COALESCE(year_to, 9999)
    )
    WHERE status IN ('pending', 'approved', 'in_progress');

COMMENT ON TABLE model_training_requests IS 'Заявки пользователей на добавление или обучение моделей сегментации деталей';
