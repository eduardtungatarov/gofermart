-- name: SaveOrder :one
INSERT INTO orders (user_id, order_number, status, accrual)
VALUES (sqlc.arg(user_id), sqlc.arg(order_number), sqlc.arg(status), sqlc.arg(accrual))
RETURNING *;

-- name: FindOrderByOrderNumber :one
SELECT * FROM orders
WHERE order_number = sqlc.arg(order_number) LIMIT 1;

-- name: FindByUserId :many
SELECT * FROM orders
WHERE user_id = sqlc.arg(user_id)
ORDER BY uploaded_at desc;