-- name: CreateUser :one
INSERT INTO users ("name", "email", "password_hash", "bio")
    VALUES ($1, $2 , $3, $4)
RETURNING id;

-- name: GetUserById :one
SELECT * FROM users WHERE id = $1;

-- name: GetAllUsers :many
SELECT * FROM users;

-- name: UpdateUser :one
UPDATE users
    SET "name" = $2, "email" = $3, "bio" = $4, updated_at = now()
    WHERE id = $1
RETURNING *;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;