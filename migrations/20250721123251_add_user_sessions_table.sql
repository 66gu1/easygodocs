-- +goose Up
-- +goose StatementBegin
CREATE TABLE user_sessions
(
    id         UUID PRIMARY KEY,
    user_id    UUID NOT NULL,
    user_agent TEXT,
    refresh_token      TEXT NOT NULL,
    created_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX idx_user_sessions_user_id ON user_sessions (user_id);
CREATE UNIQUE INDEX idx_user_sessions_refresh_token ON user_sessions (refresh_token);
CREATE UNIQUE INDEX idx_user_sessions_expires_at ON user_sessions (expires_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE user_sessions;
-- +goose StatementEnd
