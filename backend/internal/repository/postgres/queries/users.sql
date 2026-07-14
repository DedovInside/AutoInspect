-- name: CreateUser :exec
INSERT INTO users (id, username, email, first_name, last_name, display_name, avatar_url,
                   contact_name, contact_phone, contact_email,
                   password_hash, role,
                   email_verified, is_active,
                   created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7,
        $8, $9, $10,
        $11, $12,
        $13, $14,
        $15, $16);

-- name: GetUserByID :one
SELECT id,
       username,
       email,
       first_name,
       last_name,
       display_name,
       avatar_url,
       contact_name,
       contact_phone,
       contact_email,
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
       first_name,
       last_name,
       display_name,
       avatar_url,
       contact_name,
       contact_phone,
       contact_email,
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
    first_name     = $3,
    last_name      = $4,
    display_name   = $5,
    avatar_url     = $6,
    contact_name   = $7,
    contact_phone  = $8,
    contact_email  = $9,
    password_hash  = $10,
    role           = $11,
    email_verified = $12,
    is_active      = $13
WHERE id = $14;

-- name: UpdateUserContactProfile :one
UPDATE users
SET contact_name = $2,
    contact_phone = $3,
    contact_email = $4,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING id,
          username,
          email,
          first_name,
          last_name,
          display_name,
          avatar_url,
          contact_name,
          contact_phone,
          contact_email,
          password_hash,
          role,
          email_verified,
          is_active,
          created_at,
          updated_at,
          last_login;

-- name: DeleteUser :execrows
DELETE FROM users WHERE id = $1;

-- name: ListUsers :many
SELECT id, username, email, first_name, last_name, display_name, avatar_url,
       contact_name, contact_phone, contact_email,
       password_hash, role,
       email_verified, is_active,
       created_at, updated_at, last_login
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
