-- +migrate Up
CREATE TABLE damage_types
(
    code       VARCHAR(100) PRIMARY KEY,
    name_ru    VARCHAR(255) NOT NULL CHECK (char_length(trim(name_ru)) > 0),
    is_active  BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ           DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ           DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE part_categories
(
    code       VARCHAR(100) PRIMARY KEY,
    name_ru    VARCHAR(255) NOT NULL CHECK (char_length(trim(name_ru)) > 0),
    is_pair    BOOLEAN      NOT NULL DEFAULT FALSE,
    is_active  BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ           DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ           DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE car_service_specializations
(
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id         UUID         NOT NULL REFERENCES car_service_profiles (id) ON DELETE CASCADE,
    damage_type_code   VARCHAR(100) NOT NULL REFERENCES damage_types (code),
    part_category_code VARCHAR(100) NOT NULL,
    created_at         TIMESTAMPTZ           DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT chk_car_service_specializations_part_category
        CHECK (part_category_code = '*' OR char_length(trim(part_category_code)) > 0)
);

CREATE INDEX idx_damage_types_active
    ON damage_types (is_active)
    WHERE is_active = TRUE;

CREATE INDEX idx_part_categories_active
    ON part_categories (is_active)
    WHERE is_active = TRUE;

CREATE INDEX idx_car_service_specializations_profile
    ON car_service_specializations (profile_id);

CREATE INDEX idx_car_service_specializations_matching
    ON car_service_specializations (damage_type_code, part_category_code);

CREATE UNIQUE INDEX idx_car_service_specializations_unique
    ON car_service_specializations (profile_id, damage_type_code, part_category_code);

INSERT INTO damage_types (code, name_ru, is_active)
VALUES
    ('dent', 'вмятина', TRUE),
    ('scratch', 'царапина', TRUE),
    ('crack', 'трещина', TRUE),
    ('glass-shatter', 'разбитое стекло', TRUE),
    ('lamp-broken', 'повреждённая фара', TRUE),
    ('tire-flat', 'спущенное колесо', TRUE)
ON CONFLICT (code) DO NOTHING;

INSERT INTO part_categories (code, name_ru, is_pair, is_active)
VALUES
    ('*', 'любая деталь', FALSE, TRUE),
    ('back-bumper', 'задний бампер', FALSE, TRUE),
    ('back-windshield', 'заднее стекло', FALSE, TRUE),
    ('front-bumper', 'передний бампер', FALSE, TRUE),
    ('grille', 'решётка радиатора', FALSE, TRUE),
    ('hood', 'капот', FALSE, TRUE),
    ('back-door', 'задняя дверь', TRUE, TRUE),
    ('back-wheel', 'заднее колесо', TRUE, TRUE),
    ('back-window', 'заднее окно', TRUE, TRUE),
    ('fender', 'крыло', TRUE, TRUE),
    ('front-door', 'передняя дверь', TRUE, TRUE),
    ('front-wheel', 'переднее колесо', TRUE, TRUE),
    ('front-window', 'переднее окно', TRUE, TRUE),
    ('headlight', 'фара', TRUE, TRUE),
    ('mirror', 'зеркало', TRUE, TRUE),
    ('quarter-panel', 'задняя боковая панель', TRUE, TRUE),
    ('rocker-panel', 'порог', TRUE, TRUE),
    ('tail-light', 'задний фонарь', TRUE, TRUE),
    ('license-plate', 'номерной знак', FALSE, TRUE),
    ('roof', 'крыша', FALSE, TRUE),
    ('trunk', 'багажник', FALSE, TRUE),
    ('windshield', 'лобовое стекло', FALSE, TRUE),
    ('wheel', 'колесо', TRUE, TRUE)
ON CONFLICT (code) DO NOTHING;

ALTER TABLE car_service_specializations
    ADD CONSTRAINT fk_car_service_specializations_part_category
        FOREIGN KEY (part_category_code)
            REFERENCES part_categories (code);

COMMENT ON TABLE damage_types IS 'Справочник типов повреждений, соответствующих результатам ML-сервиса';
COMMENT ON TABLE part_categories IS 'Справочник обобщённых категорий деталей автомобиля для специализации автосервисов';
COMMENT ON TABLE car_service_specializations IS 'Специализации автосервисов по типам повреждений и категориям деталей';
