-- name: CreateCarServiceApplication :exec
INSERT INTO car_service_applications (
    id, user_id,
    organization_name, city, address, phone, email, contact_info, description,
    status, rejection_reason, reviewed_by, reviewed_at, created_profile_id,
    created_at, updated_at
) VALUES (
    $1, $2,
    $3, $4, $5, $6, $7, $8, $9,
    $10, $11, $12, $13, $14,
    $15, $16
);

-- name: GetCarServiceApplicationByID :one
SELECT id, user_id,
       organization_name, city, address, phone, email, contact_info, description,
       status, rejection_reason, reviewed_by, reviewed_at, created_profile_id,
       created_at, updated_at
FROM car_service_applications
WHERE id = $1;

-- name: GetPendingCarServiceApplicationByUserID :one
SELECT id, user_id,
       organization_name, city, address, phone, email, contact_info, description,
       status, rejection_reason, reviewed_by, reviewed_at, created_profile_id,
       created_at, updated_at
FROM car_service_applications
WHERE user_id = $1
  AND status = 'pending'
ORDER BY created_at DESC
LIMIT 1;

-- name: ListCarServiceApplicationsByUserID :many
SELECT id, user_id,
       organization_name, city, address, phone, email, contact_info, description,
       status, rejection_reason, reviewed_by, reviewed_at, created_profile_id,
       created_at, updated_at
FROM car_service_applications
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;
