-- name: SaveUser :one
INSERT INTO users (login, password, token)
VALUES (sqlc.arg(login), sqlc.arg(password), sqlc.arg(token))
RETURNING *;
