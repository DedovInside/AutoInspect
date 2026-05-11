-- +migrate Up
CREATE TABLE car_service_profiles
(
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID         NOT NULL UNIQUE REFERENCES users (id) ON DELETE CASCADE,

    organization_name VARCHAR(255) NOT NULL CHECK (char_length(trim(organization_name)) > 0),
    city              VARCHAR(100) NOT NULL CHECK (char_length(trim(city)) > 0),
    address           VARCHAR(255) NOT NULL CHECK (char_length(trim(address)) > 0),
    phone             VARCHAR(50),
    email             VARCHAR(255),
    website_url       VARCHAR(500),
    contact_info      TEXT,
    description       TEXT,
    is_active         BOOLEAN      NOT NULL DEFAULT TRUE,

    created_at        TIMESTAMPTZ           DEFAULT CURRENT_TIMESTAMP,
    updated_at        TIMESTAMPTZ           DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT chk_car_service_profiles_contact_present
        CHECK (
            NULLIF(trim(COALESCE(phone, '')), '') IS NOT NULL OR
            NULLIF(trim(COALESCE(email, '')), '') IS NOT NULL OR
            NULLIF(trim(COALESCE(contact_info, '')), '') IS NOT NULL OR
            NULLIF(trim(COALESCE(website_url, '')), '') IS NOT NULL
        )
);

CREATE INDEX idx_car_service_profiles_city
    ON car_service_profiles (lower(city));

CREATE INDEX idx_car_service_profiles_active
    ON car_service_profiles (is_active)
    WHERE is_active = TRUE;

ALTER TABLE car_service_applications
    ADD CONSTRAINT fk_car_service_applications_created_profile
        FOREIGN KEY (created_profile_id)
            REFERENCES car_service_profiles (id)
            ON DELETE SET NULL;

COMMENT ON TABLE car_service_profiles IS 'Профили автосервисов, созданные после одобрения заявки пользователя';
