-- +goose Up
-- +goose StatementBegin
CREATE TABLE entity_hierarchy
(
    entity_id   TEXT NOT NULL,
    entity_type UUID NOT NULL,
    parent_id   UUID,
    parent_type TEXT,
    deleted_at TIMESTAMPTZ,
    PRIMARY KEY (entity_type, entity_id)
);
CREATE INDEX idx_entity_hierarchy_parent ON entity_hierarchy(parent_type, parent_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE entity_hierarchy;
-- +goose StatementEnd
