-- name: ListVehicleMakes :many
SELECT id,
       name,
       slug,
       is_active,
       created_at,
       updated_at
FROM vehicle_makes
WHERE is_active = TRUE
ORDER BY name;

-- name: ListVehicleMakesForAdmin :many
SELECT id,
       name,
       slug,
       is_active,
       created_at,
       updated_at
FROM vehicle_makes
ORDER BY name;

-- name: GetVehicleMakeByID :one
SELECT id,
       name,
       slug,
       is_active,
       created_at,
       updated_at
FROM vehicle_makes
WHERE id = $1;

-- name: CreateVehicleMake :exec
INSERT INTO vehicle_makes (
    id,
    name,
    slug,
    is_active,
    created_at,
    updated_at
) VALUES ($1, $2, $3, $4, $5, $6);

-- name: UpdateVehicleMake :execrows
UPDATE vehicle_makes
SET name = $2,
    slug = $3,
    updated_at = $4
WHERE id = $1;

-- name: SetVehicleMakeActive :execrows
UPDATE vehicle_makes
SET is_active = $2,
    updated_at = $3
WHERE id = $1;

-- name: ListVehicleModelsByMakeID :many
SELECT id,
       make_id,
       name,
       slug,
       is_active,
       created_at,
       updated_at
FROM vehicle_models
WHERE make_id = $1
  AND is_active = TRUE
ORDER BY name;

-- name: ListVehicleModelsByMakeIDForAdmin :many
SELECT id,
       make_id,
       name,
       slug,
       is_active,
       created_at,
       updated_at
FROM vehicle_models
WHERE make_id = $1
ORDER BY name;

-- name: GetVehicleModelByID :one
SELECT id,
       make_id,
       name,
       slug,
       is_active,
       created_at,
       updated_at
FROM vehicle_models
WHERE id = $1;

-- name: CreateVehicleModel :exec
INSERT INTO vehicle_models (
    id,
    make_id,
    name,
    slug,
    is_active,
    created_at,
    updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: UpdateVehicleModel :execrows
UPDATE vehicle_models
SET make_id = $2,
    name = $3,
    slug = $4,
    updated_at = $5
WHERE id = $1;

-- name: SetVehicleModelActive :execrows
UPDATE vehicle_models
SET is_active = $2,
    updated_at = $3
WHERE id = $1;

-- name: ListVehicleGenerationsByModelID :many
SELECT id,
       model_id,
       name,
       slug,
       year_from,
       year_to,
       is_active,
       created_at,
       updated_at
FROM vehicle_generations
WHERE model_id = $1
  AND is_active = TRUE
ORDER BY year_from DESC, name;

-- name: ListVehicleGenerationsByModelIDForAdmin :many
SELECT id,
       model_id,
       name,
       slug,
       year_from,
       year_to,
       is_active,
       created_at,
       updated_at
FROM vehicle_generations
WHERE model_id = $1
ORDER BY year_from DESC, name;

-- name: GetVehicleGenerationByID :one
SELECT id,
       model_id,
       name,
       slug,
       year_from,
       year_to,
       is_active,
       created_at,
       updated_at
FROM vehicle_generations
WHERE id = $1;

-- name: GetVehicleGenerationDetailsByID :one
SELECT make.id                AS make_id,
       make.name              AS make_name,
       make.slug              AS make_slug,
       make.is_active         AS make_is_active,
       make.created_at        AS make_created_at,
       make.updated_at        AS make_updated_at,
       model.id               AS model_id,
       model.make_id          AS model_make_id,
       model.name             AS model_name,
       model.slug             AS model_slug,
       model.is_active        AS model_is_active,
       model.created_at       AS model_created_at,
       model.updated_at       AS model_updated_at,
       generation.id          AS generation_id,
       generation.model_id    AS generation_model_id,
       generation.name        AS generation_name,
       generation.slug        AS generation_slug,
       generation.year_from   AS generation_year_from,
       generation.year_to     AS generation_year_to,
       generation.is_active   AS generation_is_active,
       generation.created_at  AS generation_created_at,
       generation.updated_at  AS generation_updated_at
FROM vehicle_generations generation
JOIN vehicle_models model ON model.id = generation.model_id
JOIN vehicle_makes make ON make.id = model.make_id
WHERE generation.id = $1
  AND generation.is_active = TRUE
  AND model.is_active = TRUE
  AND make.is_active = TRUE;

-- name: CreateVehicleGeneration :exec
INSERT INTO vehicle_generations (
    id,
    model_id,
    name,
    slug,
    year_from,
    year_to,
    is_active,
    created_at,
    updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: UpdateVehicleGeneration :execrows
UPDATE vehicle_generations
SET model_id = $2,
    name = $3,
    slug = $4,
    year_from = $5,
    year_to = $6,
    updated_at = $7
WHERE id = $1;

-- name: SetVehicleGenerationActive :execrows
UPDATE vehicle_generations
SET is_active = $2,
    updated_at = $3
WHERE id = $1;
