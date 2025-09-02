-- +goose Up
-- +goose StatementBegin
ALTER TABLE entities
    ADD CONSTRAINT entities_current_version_fk
        FOREIGN KEY (id, current_version)
            REFERENCES entity_versions(entity_id, version)
            DEFERRABLE INITIALLY DEFERRED
        NOT VALID;

ALTER TABLE entities
    VALIDATE CONSTRAINT entities_current_version_fk;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE entities DROP CONSTRAINT entities_current_version_fk;
-- +goose StatementEnd
