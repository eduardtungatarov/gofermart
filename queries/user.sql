-- name: SaveUser :one
INSERT INTO users (login, password)
VALUES (sqlc.arg(login), sqlc.arg(password))
RETURNING *;

-- name: FindUserByLogin :one
SELECT * FROM users
WHERE login = sqlc.arg(login) LIMIT 1;
