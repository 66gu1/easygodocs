-- +goose Up
-- +goose StatementBegin
CREATE TABLE user_roles
(
    user_id       UUID NOT NULL,
    role          TEXT NOT NULL,
    entity_id UUID,
    entity_type    TEXT,
    PRIMARY KEY (user_id, role, entity_id, entity_type)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE user_roles;
-- +goose StatementEnd
