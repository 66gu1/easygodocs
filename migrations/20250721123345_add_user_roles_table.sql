-- +goose Up
-- +goose StatementBegin
CREATE TABLE user_roles
(
    user_id       UUID NOT NULL,
    role          TEXT NOT NULL,
    entity_id UUID,

    FOREIGN KEY (entity_id) REFERENCES entities(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX uq_user_roles_global
    ON user_roles (user_id, role)
    WHERE entity_id IS NULL;
CREATE UNIQUE INDEX uq_user_roles_scoped
    ON user_roles (user_id, role, entity_id)
    WHERE entity_id IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE user_roles;
-- +goose StatementEnd
