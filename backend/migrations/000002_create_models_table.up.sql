-- +migrate Up
CREATE TABLE models (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    version        VARCHAR(50) UNIQUE NOT NULL,
    name           VARCHAR(255) NOT NULL,

    -- Спецификация автомобиля
    car_make       VARCHAR(100) NOT NULL,
    car_model      VARCHAR(100) NOT NULL,
    car_generation VARCHAR(100),
    year_from      INTEGER,
    year_to        INTEGER,

    -- Хранилище
    weights_path   TEXT NOT NULL,
    config_path    TEXT,

    -- Runtime-статус модели
    status         VARCHAR(20) NOT NULL DEFAULT 'ready'
                   CHECK (status IN ('ready', 'active', 'deprecated')),
    active         BOOLEAN NOT NULL DEFAULT FALSE,

    created_at     TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CHECK (year_from IS NULL OR year_from >= 1900),
    CHECK (year_to IS NULL OR year_to >= 1900),
    CHECK (year_from IS NULL OR year_to IS NULL OR year_from <= year_to)
);

-- Только одна активная модель для конкретной спецификации авто
CREATE UNIQUE INDEX idx_models_active_per_car_spec
ON models (
    car_make,
    car_model,
    COALESCE(car_generation, ''),
    COALESCE(year_from, -1),
    COALESCE(year_to, -1)
)
WHERE active = TRUE;

-- Индексы для быстрого подбора нужной модели
CREATE INDEX idx_models_car_lookup ON models(car_make, car_model, car_generation);
CREATE INDEX idx_models_year_range ON models(year_from, year_to);
CREATE INDEX idx_models_status ON models(status);

-- Trigger для автоматического updated_at (функция создана в 000001)
CREATE TRIGGER update_models_updated_at
    BEFORE UPDATE ON models
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();