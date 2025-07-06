-- name: SaveUser :one
INSERT INTO users (login, password, token)
VALUES (sqlc.arg(login), sqlc.arg(password), sqlc.arg(token))
RETURNING *;

-- name: FindUserByLogin :one
SELECT * FROM users
WHERE login = sqlc.arg(login) LIMIT 1;

-- name: UpdateTokenByUser :one
UPDATE users
SET token = sqlc.arg(token)
WHERE login = sqlc.arg(login)
RETURNING *;