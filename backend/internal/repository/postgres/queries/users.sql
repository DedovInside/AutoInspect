-- name: CreateUser :exec
INSERT INTO users (id, username, email, password_hash, role,
                   email_verified, is_active,
                   created_at, updated_at)
VALUES ($1, $2, $3, $4, $5,
        $6, $7,
        $8, $9);

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
       last_login
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
       last_login
FROM users
WHERE email = $1 LIMIT 1;

-- name: UpdateUser :execrows
UPDATE users
SET username       = $1,
    email          = $2,
    password_hash  = $3,
    role           = $4,
    email_verified = $5,
    is_active      = $6
WHERE id = $7;

-- name: DeleteUser :execrows
DELETE FROM users WHERE id = $1;

-- name: ListUsers :many
SELECT id, username, email, password_hash, role,
       email_verified, is_active,
       created_at, updated_at, last_login
FROM users
ORDER BY created_at DESC
    LIMIT $1 OFFSET $2;

-- name: UpdateLastLogin :execrows
UPDATE users SET last_login = NOW() WHERE id = $1;
