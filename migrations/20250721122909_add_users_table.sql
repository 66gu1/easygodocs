-- +goose Up
-- +goose StatementBegin
CREATE TABLE users (
                       id UUID PRIMARY KEY,
                       email TEXT UNIQUE NOT NULL,
                       password_hash TEXT NOT NULL,
                       name TEXT,
                       created_at TIMESTAMPTZ,
                       updated_at TIMESTAMPTZ,
                       deleted_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX idx_users_email ON users (email);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE users;
-- +goose StatementEnd
