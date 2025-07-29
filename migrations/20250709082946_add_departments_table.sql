-- +goose Up
-- +goose StatementBegin
CREATE TABLE departments
(
    id         UUID PRIMARY KEY,
    name       TEXT      NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    deleted_at TIMESTAMPTZ
);
CREATE INDEX idx_departments_deleted_at ON departments (deleted_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE departments;
-- +goose StatementEnd