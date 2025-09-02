-- +goose Up
-- +goose StatementBegin
CREATE TABLE users
(
    id            UUID PRIMARY KEY,
    email         TEXT NOT NULL,
    password_hash TEXT        NOT NULL,
    name          TEXT,
    session_version INTEGER NOT NULL CHECK (session_version >= 0),
    created_at    TIMESTAMPTZ,
    updated_at    TIMESTAMPTZ,
    deleted_at    TIMESTAMPTZ
);
CREATE UNIQUE INDEX idx_users_email ON users (lower(email)) WHERE deleted_at ISNULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE users;
-- +goose StatementEnd
