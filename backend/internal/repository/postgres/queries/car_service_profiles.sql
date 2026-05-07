-- name: CreateCarServiceProfile :exec
INSERT INTO car_service_profiles (
    id, user_id,
    organization_name, city, address, phone, email, website_url, contact_info, description,
    is_active, created_at, updated_at
) VALUES (
    $1, $2,
    $3, $4, $5, $6, $7, $8, $9, $10,
    $11, $12, $13
);

-- name: GetCarServiceProfileByID :one
SELECT id, user_id,
       organization_name, city, address, phone, email, website_url, contact_info, description,
       is_active, created_at, updated_at
FROM car_service_profiles
WHERE id = $1;

-- name: GetCarServiceProfileByUserID :one
SELECT id, user_id,
       organization_name, city, address, phone, email, website_url, contact_info, description,
       is_active, created_at, updated_at
FROM car_service_profiles
WHERE user_id = $1;

-- name: UpdateCarServiceProfile :execrows
UPDATE car_service_profiles
SET organization_name = $2,
    city = $3,
    address = $4,
    phone = $5,
    email = $6,
    website_url = $7,
    contact_info = $8,
    description = $9,
    is_active = $10,
    updated_at = $11
WHERE user_id = $1;

-- name: SetCarServiceProfileActive :execrows
UPDATE car_service_profiles
SET is_active = $2,
    updated_at = $3
WHERE user_id = $1;
