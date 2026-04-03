-- name: CreateAuthSession :exec
INSERT INTO auth_sessions (id, user_id, token_hash, token_family_id, replaced_by_id,
                                   user_agent, ip_address,
                                   expires_at, revoked_at, revoked_reason,
                                   created_at, last_used_at)
VALUES ($1, $2, $3, $4, $5,
        $6, $7,
        $8, $9, $10,
        $11, $12);

-- name: GetAuthSessionByTokenHash :one
SELECT id,
       user_id,
       token_hash,
       token_family_id,
       replaced_by_id,
       user_agent,
       ip_address,
       expires_at,
       revoked_at,
       revoked_reason,
       last_used_at,
       created_at,
       updated_at
FROM auth_sessions
WHERE token_hash = $1 LIMIT 1;

-- name: RevokeAuthSession :execrows
UPDATE auth_sessions
SET revoked_at     = NOW(),
    revoked_reason = $1,
    replaced_by_id = $2
WHERE id = $3
  AND revoked_at IS NULL;

-- name: TouchLastUsed :execrows
UPDATE auth_sessions
SET last_used_at = $1
WHERE id = $2;

-- name: RevokeFamily :exec
UPDATE auth_sessions
SET revoked_at     = NOW(),
    revoked_reason = $1
WHERE token_family_id = $2
  AND revoked_at IS NULL;