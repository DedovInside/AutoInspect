-- +migrate Up
CREATE TABLE car_service_applications
(
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID         NOT NULL REFERENCES users (id) ON DELETE CASCADE,

    organization_name VARCHAR(255) NOT NULL CHECK (char_length(trim(organization_name)) > 0),
    city              VARCHAR(100) NOT NULL CHECK (char_length(trim(city)) > 0),
    address           VARCHAR(255) NOT NULL CHECK (char_length(trim(address)) > 0),
    phone             VARCHAR(50),
    email             VARCHAR(255),
    contact_info      TEXT,
    description       TEXT         NOT NULL CHECK (char_length(trim(description)) > 0),

    status            VARCHAR(50)  NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'approved', 'rejected')),
    rejection_reason  TEXT,
    reviewed_by       UUID REFERENCES users (id) ON DELETE SET NULL,
    reviewed_at       TIMESTAMPTZ,

    created_profile_id UUID,

    created_at        TIMESTAMPTZ           DEFAULT CURRENT_TIMESTAMP,
    updated_at        TIMESTAMPTZ           DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT chk_car_service_applications_contact_present
        CHECK (
            NULLIF(trim(COALESCE(phone, '')), '') IS NOT NULL OR
            NULLIF(trim(COALESCE(email, '')), '') IS NOT NULL OR
            NULLIF(trim(COALESCE(contact_info, '')), '') IS NOT NULL
        )
);

CREATE INDEX idx_car_service_applications_user_history
    ON car_service_applications (user_id, created_at DESC);

CREATE INDEX idx_car_service_applications_status
    ON car_service_applications (status, created_at DESC);

CREATE INDEX idx_car_service_applications_admin_queue
    ON car_service_applications (status, created_at ASC)
    WHERE status = 'pending';

CREATE UNIQUE INDEX idx_car_service_applications_one_pending_per_user
    ON car_service_applications (user_id)
    WHERE status = 'pending';

COMMENT ON TABLE car_service_applications IS 'Заявки пользователей на получение роли автосервиса';
