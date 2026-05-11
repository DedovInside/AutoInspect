-- name: CreateModelTrainingRequest :exec
INSERT INTO model_training_requests (
    id, initiator_user_id, initiator_role,
    make, model, generation, year_from, year_to, description,
    status, admin_comment, reviewed_by, reviewed_at,
    created_model_id, idempotency_key, created_at, updated_at
) VALUES (
    $1, $2, $3,
    $4, $5, $6, $7, $8, $9,
    $10, $11, $12, $13,
    $14, $15, $16, $17
);

-- name: GetModelTrainingRequestByID :one
SELECT id, initiator_user_id, initiator_role,
       make, model, generation, year_from, year_to, description,
       status, admin_comment, reviewed_by, reviewed_at,
       created_model_id, idempotency_key, created_at, updated_at
FROM model_training_requests
WHERE id = $1;

-- name: GetModelTrainingRequestByUserAndIdempotencyKey :one
SELECT id, initiator_user_id, initiator_role,
       make, model, generation, year_from, year_to, description,
       status, admin_comment, reviewed_by, reviewed_at,
       created_model_id, idempotency_key, created_at, updated_at
FROM model_training_requests
WHERE initiator_user_id = $1
  AND idempotency_key = $2;

-- name: ListModelTrainingRequestsByUserID :many
SELECT id, initiator_user_id, initiator_role,
       make, model, generation, year_from, year_to, description,
       status, admin_comment, reviewed_by, reviewed_at,
       created_model_id, idempotency_key, created_at, updated_at
FROM model_training_requests
WHERE initiator_user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListModelTrainingRequestsForAdmin :many
SELECT id, initiator_user_id, initiator_role,
       make, model, generation, year_from, year_to, description,
       status, admin_comment, reviewed_by, reviewed_at,
       created_model_id, idempotency_key, created_at, updated_at
FROM model_training_requests
WHERE sqlc.narg('status')::text IS NULL
   OR status = sqlc.narg('status')::text
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountActiveModelTrainingRequestsByUserID :one
SELECT count(*)::int
FROM model_training_requests
WHERE initiator_user_id = $1
  AND status IN ('pending', 'approved', 'in_progress');

-- name: UpdateModelTrainingRequestStatus :execrows
UPDATE model_training_requests
SET status = $2,
    admin_comment = $3,
    reviewed_by = $4,
    reviewed_at = $5,
    created_model_id = $6,
    updated_at = $7
WHERE id = $1;
