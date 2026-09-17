-- +goose Up
CREATE TABLE IF NOT EXISTS chat_members(
CREATE TABLE chat_members (
    chat_id UUID NOT NULL
        REFERENCES chats(id)
        ON DELETE CASCADE,
    user_id UUID NOT NULL
        REFERENCES users(id),
    role VARCHAR(20) NOT NULL
        CHECK (role IN ('admin', 'member')),
    joined_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY(chat_id, user_id)
);
);
-- +goose Down

DROP TABLE IF EXISTS chat_members;
