-- +goose Up
CREATE TABLE IF NOT EXISTS messages (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,

    chat_id UUID NOT NULL
        REFERENCES chats(id)
        ON DELETE CASCADE,

    sender_id UUID NOT NULL
        REFERENCES users(id),

    content TEXT NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CHECK (char_length(content) BETWEEN 1 AND 4000)
);

CREATE INDEX idx_messages_chat_id_id
ON messages(chat_id, id DESC);
-- +goose Down
DROP TABLE IF EXISTS messages;