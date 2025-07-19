-- name: FindByUserId :one
SELECT * FROM balance
WHERE user_id = sqlc.arg(user_id);

-- name: DeductFromBalance :one
UPDATE balance
SET withdrawn = withdrawn + sqlc.arg(sum), current = current - sqlc.arg(sum)
WHERE user_id = sqlc.arg(user_id)
RETURNING *;

-- name: AddBalance :one
INSERT INTO balance (user_id, current, withdrawn)
VALUES (sqlc.arg(user_id), sqlc.arg(sum), 0)
ON CONFLICT (user_id)
DO UPDATE SET current = balance.current + EXCLUDED.current
RETURNING *;
