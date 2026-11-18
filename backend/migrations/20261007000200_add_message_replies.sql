-- +goose Up
ALTER TABLE messages
    ADD COLUMN reply_to_message_id BIGINT,
    ADD CONSTRAINT messages_chat_id_id_key UNIQUE (chat_id, id),
    ADD CONSTRAINT messages_reply_same_chat_fk
        FOREIGN KEY (chat_id, reply_to_message_id) REFERENCES messages(chat_id, id),
    ADD CONSTRAINT messages_reply_not_self_check CHECK (reply_to_message_id <> id);

CREATE INDEX idx_messages_reply_to
    ON messages(chat_id, reply_to_message_id) WHERE reply_to_message_id IS NOT NULL;

-- +goose Down
DROP INDEX idx_messages_reply_to;
ALTER TABLE messages
    DROP CONSTRAINT messages_reply_same_chat_fk,
    DROP CONSTRAINT messages_reply_not_self_check,
    DROP CONSTRAINT messages_chat_id_id_key,
    DROP COLUMN reply_to_message_id;
