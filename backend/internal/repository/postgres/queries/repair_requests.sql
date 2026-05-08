-- name: CreateRepairRequest :exec
INSERT INTO repair_requests (
    id, user_id, analysis_job_id, car_service_profile_id,
    status, repair_summary, service_estimate,
    customer_name, customer_phone, customer_email, customer_comment,
    service_comment, estimated_price_min, estimated_price_max,
    created_at, updated_at, responded_at
) VALUES (
    $1, $2, $3, $4,
    $5, $6, $7,
    $8, $9, $10, $11,
    $12, $13, $14,
    $15, $16, $17
);

-- name: GetRepairRequestByID :one
SELECT id, user_id, analysis_job_id, car_service_profile_id,
       status, repair_summary, service_estimate,
       customer_name, customer_phone, customer_email, customer_comment,
       service_comment, estimated_price_min, estimated_price_max,
       created_at, updated_at, responded_at
FROM repair_requests
WHERE id = $1;

-- name: GetPendingRepairRequestByUserAnalysisAndService :one
SELECT id, user_id, analysis_job_id, car_service_profile_id,
       status, repair_summary, service_estimate,
       customer_name, customer_phone, customer_email, customer_comment,
       service_comment, estimated_price_min, estimated_price_max,
       created_at, updated_at, responded_at
FROM repair_requests
WHERE user_id = $1
  AND analysis_job_id = $2
  AND car_service_profile_id = $3
  AND status = 'pending';

-- name: ListRepairRequestsByUserID :many
SELECT id, user_id, analysis_job_id, car_service_profile_id,
       status, repair_summary, service_estimate,
       customer_name, customer_phone, customer_email, customer_comment,
       service_comment, estimated_price_min, estimated_price_max,
       created_at, updated_at, responded_at
FROM repair_requests
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CancelPendingRepairRequestByUserID :execrows
UPDATE repair_requests
SET status = 'canceled',
    updated_at = $3
WHERE id = $1
  AND user_id = $2
  AND status = 'pending';
