-- name: CreateAnalysisJob :exec
INSERT INTO analysis_jobs (
    id, user_id,
    idempotency_key,
    car_make, car_model, car_generation, car_year,
    image_keys,
    correlation_id,
    status,
    requested_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11);

-- name: GetAnalysisJobByID :one
SELECT id, user_id,
       idempotency_key,
       car_make, car_model, car_generation, car_year,
       image_keys,
       correlation_id,
       status, error_message,
       result, used_model_version,
       requested_at, started_at, completed_at
FROM analysis_jobs
WHERE id = $1;

-- name: GetAnalysisJobByCorrelationID :one
SELECT id, user_id,
       idempotency_key,
       car_make, car_model, car_generation, car_year,
       image_keys,
       correlation_id,
       status, error_message,
       result, used_model_version,
       requested_at, started_at, completed_at
FROM analysis_jobs
WHERE correlation_id = $1;

-- name: GetAnalysisJobByUserAndIdempotencyKey :one
SELECT id, user_id,
       idempotency_key,
       car_make, car_model, car_generation, car_year,
       image_keys,
       correlation_id,
       status, error_message,
       result, used_model_version,
       requested_at, started_at, completed_at
FROM analysis_jobs
WHERE user_id = $1
  AND idempotency_key = $2;

-- name: ListAnalysisJobsByUserID :many
SELECT id, user_id,
       idempotency_key,
       car_make, car_model, car_generation, car_year,
       image_keys,
       correlation_id,
       status, error_message,
       result, used_model_version,
       requested_at, started_at, completed_at
FROM analysis_jobs
WHERE user_id = $1
ORDER BY requested_at DESC
    LIMIT $2 OFFSET $3;

-- name: UpdateAnalysisJobStatus :execrows
UPDATE analysis_jobs
SET status = $1,
    error_message = $2,
    completed_at = CASE WHEN $1 IN ('completed', 'failed') THEN NOW() ELSE completed_at END
WHERE id = $3
  AND status = 'pending';

-- name: UpdateAnalysisJobStatusByCorrelationID :execrows
UPDATE analysis_jobs
SET status = $1,
    error_message = $2,
    completed_at = CASE WHEN $1 IN ('completed', 'failed') THEN NOW() ELSE completed_at END
WHERE correlation_id = $3
  AND status = 'pending';

-- name: UpdateAnalysisJobResult :execrows
UPDATE analysis_jobs
SET status = 'completed',
    result = $1,
    used_model_version = $2,
    completed_at = NOW()
WHERE id = $3
  AND status = 'pending';

-- name: UpdateAnalysisJobResultByCorrelationID :execrows
UPDATE analysis_jobs
SET status = 'completed',
    result = $1,
    used_model_version = $2,
    completed_at = NOW()
WHERE correlation_id = $3
  AND status = 'pending';
