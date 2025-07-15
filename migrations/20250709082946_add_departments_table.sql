-- +goose Up
CREATE TABLE departments (
                             id UUID PRIMARY KEY,
                             name TEXT NOT NULL,
                             parent_id UUID,
                             created_at TIMESTAMP NOT NULL,
                             updated_at TIMESTAMP NOT NULL,
                             deleted_at TIMESTAMP
);

-- +goose Down
DROP TABLE departments;