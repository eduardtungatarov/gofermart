-- name: FindByUserId :one
SELECT * FROM balance
WHERE user_id = sqlc.arg(user_id);

-- name: DeductFromBalance :one
UPDATE balance
SET withdrawn = withdrawn + sqlc.arg(sum), current = current - sqlc.arg(sum)
WHERE user_id = sqlc.arg(user_id)
RETURNING *;