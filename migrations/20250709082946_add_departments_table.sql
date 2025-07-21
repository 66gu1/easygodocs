-- +goose Up
-- +goose StatementBegin
CREATE TABLE departments
(
    id         UUID PRIMARY KEY,
    name       TEXT      NOT NULL,
    parent_id  UUID,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    deleted_at TIMESTAMPTZ
);
CREATE INDEX idx_departments_parent_id ON departments (parent_id);
CREATE INDEX idx_departments_deleted_at ON departments (deleted_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE departments;
-- +goose StatementEnd