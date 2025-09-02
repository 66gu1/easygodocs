-- +goose Up
-- +goose StatementBegin
CREATE TABLE user_sessions
(
    id         UUID PRIMARY KEY,
    user_id    UUID NOT NULL,
    refresh_token_hash      TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    session_version INTEGER NOT NULL CHECK ( session_version >= 0 ),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
CREATE INDEX idx_user_sessions_user_id ON user_sessions (user_id);
CREATE INDEX idx_user_sessions_expires_at ON user_sessions (expires_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE user_sessions;
-- +goose StatementEnd
