-- +goose Up
-- +goose StatementBegin
CREATE TABLE user_roles
(
    user_id       UUID NOT NULL,
    role          TEXT NOT NULL,
    department_id UUID, -- nullable; if set, role applies to that department
    article_id    UUID, -- nullable; if set, role applies to that article
    PRIMARY KEY (user_id, role, department_id, article_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE user_roles;
-- +goose StatementEnd
