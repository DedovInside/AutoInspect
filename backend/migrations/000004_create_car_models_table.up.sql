-- +migrate Up
CREATE TABLE car_models
(
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    make          VARCHAR(100) NOT NULL,
    model         VARCHAR(100) NOT NULL,
    generation    VARCHAR(100),
    year_from     INTEGER      NOT NULL,
    year_to       INTEGER,

    model_s3_key  VARCHAR(500) NOT NULL,
    model_version VARCHAR(50)  NOT NULL,

    is_universal  BOOLEAN          DEFAULT FALSE,
    is_active     BOOLEAN          DEFAULT TRUE,

    created_at    TIMESTAMPTZ      DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX idx_car_models_unique_active
    ON car_models (make, model, generation, year_from, COALESCE(year_to, 9999), model_version) WHERE is_active = true;

CREATE INDEX idx_car_models_lookup
    ON car_models (make, model, generation, year_from) WHERE is_active = true;

CREATE INDEX idx_car_models_universal
    ON car_models (is_universal) WHERE is_universal = true AND is_active = true;

COMMENT ON TABLE car_models IS 'Реестр ML-моделей, привязанных к маркам/моделям/годам авто';
