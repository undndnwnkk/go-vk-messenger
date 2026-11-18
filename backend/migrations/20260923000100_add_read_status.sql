-- +goose Up
ALTER TABLE chat_members
    ADD COLUMN last_read_message_id BIGINT
        REFERENCES messages(id)
        ON DELETE SET NULL;

CREATE INDEX idx_chat_members_user_chat_read
ON chat_members(user_id, chat_id, last_read_message_id);

-- +goose Down
DROP INDEX IF EXISTS idx_chat_members_user_chat_read;

ALTER TABLE chat_members
    DROP COLUMN IF EXISTS last_read_message_id;
