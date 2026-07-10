-- +migrate Down
DROP TABLE IF EXISTS vehicle_generations CASCADE;
DROP TABLE IF EXISTS vehicle_models CASCADE;
DROP TABLE IF EXISTS vehicle_makes CASCADE;
