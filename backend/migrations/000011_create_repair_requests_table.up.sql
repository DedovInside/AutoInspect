-- +migrate Up
CREATE TABLE repair_requests
(
    id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id                UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    analysis_job_id        UUID        NOT NULL REFERENCES analysis_jobs (id) ON DELETE CASCADE,
    car_service_profile_id UUID        NOT NULL REFERENCES car_service_profiles (id) ON DELETE CASCADE,

    status                 VARCHAR(30) NOT NULL DEFAULT 'pending',

    repair_summary         JSONB       NOT NULL,
    service_estimate       JSONB,

    customer_name          VARCHAR(255),
    customer_phone         VARCHAR(50),
    customer_email         VARCHAR(255),
    customer_comment       TEXT,

    service_comment        TEXT,
    estimated_price_min    NUMERIC(12, 2),
    estimated_price_max    NUMERIC(12, 2),

    created_at             TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    responded_at           TIMESTAMPTZ,

    CONSTRAINT chk_repair_requests_status
        CHECK (status IN ('pending', 'accepted', 'rejected', 'canceled', 'completed')),
    CONSTRAINT chk_repair_requests_estimated_price_min_non_negative
        CHECK (estimated_price_min IS NULL OR estimated_price_min >= 0),
    CONSTRAINT chk_repair_requests_estimated_price_max_non_negative
        CHECK (estimated_price_max IS NULL OR estimated_price_max >= 0),
    CONSTRAINT chk_repair_requests_estimated_price_range
        CHECK (
            estimated_price_min IS NULL
            OR estimated_price_max IS NULL
            OR estimated_price_min <= estimated_price_max
        )
);

CREATE UNIQUE INDEX idx_repair_requests_unique_active
    ON repair_requests (user_id, analysis_job_id, car_service_profile_id)
    WHERE status = 'pending';

CREATE INDEX idx_repair_requests_user_created
    ON repair_requests (user_id, created_at DESC);

CREATE INDEX idx_repair_requests_service_status_created
    ON repair_requests (car_service_profile_id, status, created_at DESC);

CREATE INDEX idx_repair_requests_analysis_job
    ON repair_requests (analysis_job_id);

COMMENT ON TABLE repair_requests IS 'Заявки пользователей на ремонт автомобиля по результатам анализа';
COMMENT ON COLUMN repair_requests.repair_summary IS 'Снимок ремонтной сводки, сформированной из результата анализа на момент создания заявки';
COMMENT ON COLUMN repair_requests.service_estimate IS 'Предварительная смета автосервиса по парам деталь и тип повреждения';
