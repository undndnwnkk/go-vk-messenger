-- +goose Up
ALTER TABLE messages
    ADD COLUMN edited_at TIMESTAMPTZ,
    ADD COLUMN deleted_at TIMESTAMPTZ;

ALTER TABLE messages DROP CONSTRAINT messages_content_check;
ALTER TABLE messages ADD CONSTRAINT messages_content_check CHECK (
    (deleted_at IS NULL AND char_length(content) BETWEEN 1 AND 4000)
    OR (deleted_at IS NOT NULL AND content = '')
);

-- +goose Down
-- Keep message IDs and read cursors intact; deleted text cannot be restored.
UPDATE messages SET content = '[deleted]', deleted_at = NULL WHERE deleted_at IS NOT NULL;
ALTER TABLE messages DROP CONSTRAINT messages_content_check;
ALTER TABLE messages ADD CONSTRAINT messages_content_check CHECK (char_length(content) BETWEEN 1 AND 4000);
ALTER TABLE messages DROP COLUMN deleted_at, DROP COLUMN edited_at;
