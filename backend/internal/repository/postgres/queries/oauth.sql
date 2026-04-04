-- name: CreateOAuthIdentity :exec
INSERT INTO auth_oauth_identities (id, user_id, provider, provider_user_id, email, created_at)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: GetOAuthIdentityByProviderSubject :one
SELECT id, user_id, provider, provider_user_id, email, created_at
FROM auth_oauth_identities
WHERE provider = $1 AND provider_user_id = $2
    LIMIT 1;
