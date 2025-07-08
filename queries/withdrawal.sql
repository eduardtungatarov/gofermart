-- name: FindByUserId :many
SELECT * FROM withdrawals
WHERE user_id = sqlc.arg(user_id)
ORDER BY processed_at desc;