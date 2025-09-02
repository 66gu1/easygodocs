-- +goose Up
-- +goose StatementBegin
CREATE TABLE entity_versions
(
    entity_id              UUID NOT NULL ,
    version              INT NOT NULL CHECK(version > 0),
    name            TEXT        NOT NULL,
    content         TEXT        NOT NULL,
    parent_id          uuid,
    created_by      UUID        NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (entity_id, version),
    FOREIGN KEY (entity_id) REFERENCES entities(id) ON DELETE CASCADE,
    FOREIGN KEY (parent_id) REFERENCES entities(id) ON DELETE SET NULL, -- parent link nulled; can add snapshot columns later
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE RESTRICT
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE entity_versions;
-- +goose StatementEnd
