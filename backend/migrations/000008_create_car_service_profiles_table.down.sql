-- +migrate Down
ALTER TABLE car_service_applications
    DROP CONSTRAINT IF EXISTS fk_car_service_applications_created_profile;

DROP TABLE IF EXISTS car_service_profiles CASCADE;
