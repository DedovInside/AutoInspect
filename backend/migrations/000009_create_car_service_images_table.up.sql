-- +migrate Up
CREATE TABLE car_service_images
(
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id        UUID         NOT NULL REFERENCES car_service_profiles (id) ON DELETE CASCADE,

    s3_key            TEXT         NOT NULL,
    is_primary        BOOLEAN      NOT NULL DEFAULT FALSE,
    sort_order        INTEGER      NOT NULL,
    original_filename VARCHAR(255) NOT NULL,
    content_type      VARCHAR(100) NOT NULL,
    size_bytes        BIGINT       NOT NULL CHECK (size_bytes > 0),

    created_at        TIMESTAMPTZ           DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT chk_car_service_images_sort_order_positive
        CHECK (sort_order > 0)
);

CREATE INDEX idx_car_service_images_profile_order
    ON car_service_images (profile_id, sort_order ASC, created_at ASC);

CREATE UNIQUE INDEX idx_car_service_images_profile_s3_key
    ON car_service_images (profile_id, s3_key);

CREATE UNIQUE INDEX idx_car_service_images_one_primary_per_profile
    ON car_service_images (profile_id)
    WHERE is_primary = TRUE;

COMMENT ON TABLE car_service_images IS 'Изображения профилей автосервисов, хранящиеся в MinIO';
