-- name: CreateCarServiceImage :exec
INSERT INTO car_service_images (
    id, profile_id, s3_key, is_primary, sort_order,
    original_filename, content_type, size_bytes, created_at
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8, $9
);

-- name: ListCarServiceImagesByProfileID :many
SELECT id, profile_id, s3_key, is_primary, sort_order,
       original_filename, content_type, size_bytes, created_at
FROM car_service_images
WHERE profile_id = $1
ORDER BY sort_order ASC, created_at ASC;

-- name: GetCarServiceImageByID :one
SELECT id, profile_id, s3_key, is_primary, sort_order,
       original_filename, content_type, size_bytes, created_at
FROM car_service_images
WHERE id = $1;

-- name: DeleteCarServiceImage :execrows
DELETE FROM car_service_images
WHERE id = $1;

-- name: ClearPrimaryCarServiceImage :exec
UPDATE car_service_images
SET is_primary = FALSE
WHERE profile_id = $1
  AND is_primary = TRUE;

-- name: SetPrimaryCarServiceImage :execrows
UPDATE car_service_images
SET is_primary = TRUE
WHERE id = $1;

-- name: NextCarServiceImageSortOrder :one
SELECT COALESCE(MAX(sort_order), 0)::int + 1
FROM car_service_images
WHERE profile_id = $1;
