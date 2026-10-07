-- +goose Up
ALTER TABLE chat_members
    ADD COLUMN muted BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN mute_version BIGINT NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE chat_members DROP COLUMN mute_version, DROP COLUMN muted;
