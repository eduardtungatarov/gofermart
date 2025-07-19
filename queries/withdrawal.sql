-- name: FindByUserId :many
SELECT * FROM withdrawals
WHERE user_id = sqlc.arg(user_id)
ORDER BY processed_at desc;

-- name: SaveWithdrawal :one
INSERT INTO withdrawals (user_id, order_number, sum)
VALUES (sqlc.arg(user_id), sqlc.arg(order_number), sqlc.arg(sum))
RETURNING *;