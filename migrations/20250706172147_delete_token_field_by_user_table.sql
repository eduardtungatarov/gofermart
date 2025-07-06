-- +goose Up
-- +goose StatementBegin
ALTER TABLE users
DROP COLUMN token;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users
    ADD COLUMN token VARCHAR(255) UNIQUE;
-- +goose StatementEnd
