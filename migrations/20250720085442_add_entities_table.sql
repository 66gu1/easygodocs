-- +goose Up
-- +goose StatementBegin
CREATE TABLE entities
(
    id              UUID PRIMARY KEY,
    type            TEXT        NOT NULL,
    name            TEXT        NOT NULL,
    content         TEXT        NOT NULL,
    parent_id          uuid,
    created_by      UUID        NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL,
    updated_by      UUID        NOT NULL,
    updated_at      TIMESTAMPTZ NOT NULL,
    current_version INT CHECK (current_version ISNULL OR current_version > 0),
    deleted_at      TIMESTAMPTZ,
    FOREIGN KEY (parent_id) REFERENCES entities(id) ON DELETE CASCADE,
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE RESTRICT,
    FOREIGN KEY (updated_by) REFERENCES users(id) ON DELETE RESTRICT
);
CREATE INDEX idx_entities_parent ON entities (parent_id) WHERE deleted_at ISNULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE entities;
-- +goose StatementEnd
