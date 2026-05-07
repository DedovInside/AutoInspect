-- name: ListActiveDamageTypes :many
SELECT code, name_ru, is_active, created_at, updated_at
FROM damage_types
WHERE is_active = TRUE
ORDER BY name_ru ASC;

-- name: ExistsActiveDamageType :one
SELECT EXISTS(
    SELECT 1
    FROM damage_types
    WHERE code = $1
      AND is_active = TRUE
)::bool;

-- name: ListActivePartCategories :many
SELECT code, name_ru, is_pair, is_active, created_at, updated_at
FROM part_categories
WHERE is_active = TRUE
ORDER BY CASE WHEN code = '*' THEN 0 ELSE 1 END, name_ru ASC;

-- name: ExistsActivePartCategory :one
SELECT EXISTS(
    SELECT 1
    FROM part_categories
    WHERE code = $1
      AND is_active = TRUE
)::bool;

-- name: CreateCarServiceSpecialization :exec
INSERT INTO car_service_specializations (
    id, profile_id, damage_type_code, part_category_code, created_at
) VALUES (
    $1, $2, $3, $4, $5
);

-- name: ListCarServiceSpecializationsByProfileID :many
SELECT id, profile_id, damage_type_code, part_category_code, created_at
FROM car_service_specializations
WHERE profile_id = $1
ORDER BY damage_type_code ASC, part_category_code ASC;

-- name: DeleteCarServiceSpecializationsByProfileID :exec
DELETE FROM car_service_specializations
WHERE profile_id = $1;
