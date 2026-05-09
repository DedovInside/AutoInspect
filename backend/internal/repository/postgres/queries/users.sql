-- name: CreateUser :exec
INSERT INTO users (id, username, email, avatar_url, password_hash, role,
                   email_verified, is_active,
                   created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6,
        $7, $8,
        $9, $10);

-- name: GetUserByID :one
SELECT id,
       username,
       email,
       password_hash,
       role,
       email_verified,
       is_active,
       created_at,
       updated_at,
       last_login,
       avatar_url
FROM users
WHERE id = $1 LIMIT 1;

-- name: GetUserByEmail :one
SELECT id,
       username,
       email,
       password_hash,
       role,
       email_verified,
       is_active,
       created_at,
       updated_at,
       last_login,
       avatar_url
FROM users
WHERE email = $1 LIMIT 1;

-- name: UpdateUser :execrows
UPDATE users
SET username       = $1,
    email          = $2,
    avatar_url     = $3,
    password_hash  = $4,
    role           = $5,
    email_verified = $6,
    is_active      = $7
WHERE id = $8;

-- name: DeleteUser :execrows
DELETE FROM users WHERE id = $1;

-- name: ListUsers :many
SELECT id, username, email, password_hash, role,
       email_verified, is_active,
       created_at, updated_at, last_login, avatar_url
FROM users
ORDER BY created_at DESC
    LIMIT $1 OFFSET $2;

-- name: UpdateLastLogin :execrows
UPDATE users SET last_login = NOW() WHERE id = $1;

-- name: UpdateUserRole :execrows
UPDATE users
SET role = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1;
