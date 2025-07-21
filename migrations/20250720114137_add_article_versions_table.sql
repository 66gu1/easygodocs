-- +goose Up
-- +goose StatementBegin
CREATE TABLE article_versions
(
    article_id UUID        NOT NULL,
    version    INT         NOT NULL,
    name      TEXT        NOT NULL,
    content    TEXT        NOT NULL,
    parent_type TEXT      NOT NULL,
    parent_id  UUID        NOT NULL,
    created_by UUID        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,

    PRIMARY KEY (article_id, version)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE article_versions;
-- +goose StatementEnd
