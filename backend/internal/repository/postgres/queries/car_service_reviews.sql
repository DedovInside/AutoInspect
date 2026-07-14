-- name: CreateCarServiceReview :exec
INSERT INTO car_service_reviews (
    id, repair_request_id, car_service_profile_id, user_id,
    rating, author_name, comment,
    created_at, updated_at
) VALUES (
    $1, $2, $3, $4,
    $5, $6, $7,
    $8, $9
);

-- name: GetCarServiceReviewByID :one
SELECT id, repair_request_id, car_service_profile_id, user_id,
       rating, author_name, comment,
       created_at, updated_at
FROM car_service_reviews
WHERE id = $1;

-- name: GetCarServiceReviewByRepairRequestID :one
SELECT id, repair_request_id, car_service_profile_id, user_id,
       rating, author_name, comment,
       created_at, updated_at
FROM car_service_reviews
WHERE repair_request_id = $1;

-- name: ListCarServiceReviewsByProfileID :many
SELECT id, repair_request_id, car_service_profile_id, user_id,
       rating, author_name, comment,
       created_at, updated_at
FROM car_service_reviews
WHERE car_service_profile_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListCarServiceReviewsByUserID :many
SELECT id, repair_request_id, car_service_profile_id, user_id,
       rating, author_name, comment,
       created_at, updated_at
FROM car_service_reviews
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: UpdateCarServiceReviewByRepairRequestIDAndUserID :one
UPDATE car_service_reviews
SET rating = $3,
    author_name = $4,
    comment = $5
WHERE repair_request_id = $1
  AND user_id = $2
RETURNING id, repair_request_id, car_service_profile_id, user_id,
          rating, author_name, comment,
          created_at, updated_at;

-- name: DeleteCarServiceReviewByRepairRequestIDAndUserID :execrows
DELETE FROM car_service_reviews
WHERE repair_request_id = $1
  AND user_id = $2;
