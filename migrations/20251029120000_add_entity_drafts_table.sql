-- +goose Up
-- +goose StatementBegin
-- A draft holds a user's unpublished changes to an entity. Readers keep seeing
-- the published version from the entities table until the draft is published.
CREATE TABLE entity_drafts
(
    entity_id    UUID        NOT NULL,
    user_id      UUID        NOT NULL,
    name         TEXT        NOT NULL,
    content      TEXT        NOT NULL,
    -- the published version the draft was started from (NULL for a never published entity);
    -- publishing fails if the entity has been published since then
    base_version INT CHECK (base_version ISNULL OR base_version > 0),
    updated_at   TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (entity_id, user_id),
    FOREIGN KEY (entity_id) REFERENCES entities (id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

-- Before this migration a published entity could be turned back into a draft:
-- its row was overwritten with the draft and current_version was set to NULL,
-- which hid the entity from everyone but its last editor.
-- Move such drafts to entity_drafts and restore the last published version.
WITH latest AS (
    SELECT DISTINCT ON (entity_id) entity_id, version, name, content, created_by, created_at
    FROM entity_versions
    ORDER BY entity_id, version DESC
),
moved AS (
    INSERT INTO entity_drafts (entity_id, user_id, name, content, base_version, updated_at)
    SELECT e.id, e.updated_by, e.name, e.content, l.version, e.updated_at
    FROM entities e
    JOIN latest l ON l.entity_id = e.id
    WHERE e.current_version ISNULL AND e.deleted_at ISNULL
)
UPDATE entities e
SET name            = l.name,
    content         = l.content,
    current_version = l.version,
    updated_by      = l.created_by,
    updated_at      = l.created_at
FROM latest l
WHERE l.entity_id = e.id AND e.current_version ISNULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Drafts are dropped: the old model can't represent a draft of a published entity.
DROP TABLE entity_drafts;
-- +goose StatementEnd
