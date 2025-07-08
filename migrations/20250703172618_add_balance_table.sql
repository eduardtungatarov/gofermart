-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS balance (
    id SERIAL PRIMARY KEY,
    user_id INTEGER UNIQUE NOT NULL,
    current INTEGER NOT NULL DEFAULT 0 CHECK (current >= 0),
    withdrawn INTEGER NOT NULL DEFAULT 0 CHECK (withdrawn >= 0)
);

CREATE INDEX IF NOT EXISTS idx_balance_user_id ON balance(user_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS balance;
-- +goose StatementEnd
