-- +migrate Up
CREATE TABLE vehicle_makes
(
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       VARCHAR(100) NOT NULL,
    slug       VARCHAR(120) NOT NULL UNIQUE,
    is_active  BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT chk_vehicle_makes_name_not_blank
        CHECK (char_length(trim(name)) > 0),
    CONSTRAINT chk_vehicle_makes_slug_not_blank
        CHECK (char_length(trim(slug)) > 0)
);

CREATE TABLE vehicle_models
(
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    make_id    UUID         NOT NULL REFERENCES vehicle_makes (id) ON DELETE CASCADE,
    name       VARCHAR(100) NOT NULL,
    slug       VARCHAR(120) NOT NULL,
    is_active  BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT chk_vehicle_models_name_not_blank
        CHECK (char_length(trim(name)) > 0),
    CONSTRAINT chk_vehicle_models_slug_not_blank
        CHECK (char_length(trim(slug)) > 0),
    CONSTRAINT uq_vehicle_models_make_slug
        UNIQUE (make_id, slug)
);

CREATE TABLE vehicle_generations
(
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    model_id   UUID         NOT NULL REFERENCES vehicle_models (id) ON DELETE CASCADE,
    name       VARCHAR(100) NOT NULL,
    slug       VARCHAR(120) NOT NULL,
    year_from  INTEGER      NOT NULL,
    year_to    INTEGER,
    is_active  BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT chk_vehicle_generations_name_not_blank
        CHECK (char_length(trim(name)) > 0),
    CONSTRAINT chk_vehicle_generations_slug_not_blank
        CHECK (char_length(trim(slug)) > 0),
    CONSTRAINT chk_vehicle_generations_year_from_range
        CHECK (year_from BETWEEN 1900 AND 2100),
    CONSTRAINT chk_vehicle_generations_year_to_range
        CHECK (year_to IS NULL OR year_to BETWEEN 1900 AND 2100),
    CONSTRAINT chk_vehicle_generations_year_range
        CHECK (year_to IS NULL OR year_to >= year_from),
    CONSTRAINT uq_vehicle_generations_model_slug_year_from
        UNIQUE (model_id, slug, year_from)
);

CREATE INDEX idx_vehicle_makes_active_sort
    ON vehicle_makes (is_active, name);

CREATE INDEX idx_vehicle_models_make_active_sort
    ON vehicle_models (make_id, is_active, name);

CREATE INDEX idx_vehicle_generations_model_active_sort
    ON vehicle_generations (model_id, is_active, year_from DESC, name);

COMMENT ON TABLE vehicle_makes IS 'Справочник марок автомобилей для нормализованного выбора при создании анализа';
COMMENT ON TABLE vehicle_models IS 'Справочник моделей автомобилей, связанных с марками';
COMMENT ON TABLE vehicle_generations IS 'Справочник поколений автомобилей с диапазонами годов выпуска';
COMMENT ON COLUMN vehicle_makes.slug IS 'Нормализованный технический код марки автомобиля';
COMMENT ON COLUMN vehicle_models.slug IS 'Нормализованный технический код модели автомобиля в рамках марки';
COMMENT ON COLUMN vehicle_generations.slug IS 'Нормализованный технический код поколения автомобиля в рамках модели';
