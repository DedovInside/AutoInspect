-- +migrate Down
DROP TABLE IF EXISTS car_service_specializations CASCADE;
DROP TABLE IF EXISTS part_categories CASCADE;
DROP TABLE IF EXISTS damage_types CASCADE;
