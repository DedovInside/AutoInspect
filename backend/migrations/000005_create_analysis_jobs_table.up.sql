-- +migrate Up
CREATE TABLE analysis_jobs
(
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id            UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,

    idempotency_key    VARCHAR(255),

    car_make           VARCHAR(100),
    car_model          VARCHAR(100),
    car_generation     VARCHAR(100),
    car_year           INT,

    image_keys         JSONB       NOT NULL CHECK (jsonb_typeof(image_keys) = 'array'),
    correlation_id     UUID UNIQUE NOT NULL,

    status             VARCHAR(50) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
    error_message      TEXT,

    result             JSONB,
    used_model_version VARCHAR(50),

    requested_at       TIMESTAMPTZ          DEFAULT CURRENT_TIMESTAMP,
    started_at         TIMESTAMPTZ,
    completed_at       TIMESTAMPTZ
);

CREATE INDEX idx_analysis_jobs_user_history
    ON analysis_jobs (user_id, requested_at DESC);

CREATE INDEX idx_analysis_jobs_status_pending
    ON analysis_jobs (status) WHERE status IN ('pending', 'processing');

CREATE INDEX idx_analysis_jobs_result_gin
    ON analysis_jobs USING GIN (result);

CREATE UNIQUE INDEX idx_analysis_jobs_idempotency
    ON analysis_jobs(user_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL;

COMMENT
ON TABLE analysis_jobs IS 'Асинхронные задачи анализа изображений и результаты ML';
