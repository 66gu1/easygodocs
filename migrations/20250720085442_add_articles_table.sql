-- +goose Up
-- +goose StatementBegin
CREATE TABLE articles
(
    id              UUID PRIMARY KEY,
    name            TEXT        NOT NULL,
    content         TEXT        NOT NULL,
    parent_type      TEXT        NOT NULL,
    parent_id        UUID        NOT NULL,
    created_by      UUID        NOT NULL,
    updated_by      UUID        NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL,
    updated_at      TIMESTAMPTZ NOT NULL,
    deleted_at      TIMESTAMPTZ,
    current_version INT
);
CREATE INDEX idx_articles_parent ON articles (parent_type, parent_id);
CREATE INDEX idx_articles_deleted_at ON articles (deleted_at);
CREATE INDEX idx_articles_published ON articles (current_version);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE articles
-- +goose StatementEnd
