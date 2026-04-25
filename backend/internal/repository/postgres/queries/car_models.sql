-- name: FindActiveCarModel :one
SELECT id, make, model, generation, year_from, year_to,
       model_s3_key, parts_catalog_s3_key, model_version, is_universal, is_active, created_at
FROM car_models
WHERE is_active = true
  AND make = $1
  AND model = $2
  AND (generation = $3 OR generation = '' OR generation IS NULL)
  AND year_from <= $4
  AND (year_to >= $4 OR year_to IS NULL)
ORDER BY
    CASE WHEN generation = $3 THEN 0 ELSE 1 END,
    model_version DESC
LIMIT 1;

-- name: GetUniversalCarModel :one
SELECT id, make, model, generation, year_from, year_to,
       model_s3_key, parts_catalog_s3_key, model_version, is_universal, is_active, created_at
FROM car_models
WHERE is_active = true
  AND is_universal = true
    LIMIT 1;

-- name: CreateCarModel :exec
INSERT INTO car_models (
    id, make, model, generation, year_from, year_to,
    model_s3_key, parts_catalog_s3_key, model_version, is_universal, is_active, created_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12);
