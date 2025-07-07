-- name: FindByUserId :one
SELECT * FROM balance
WHERE user_id = sqlc.arg(user_id);