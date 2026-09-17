-- +goose Up
CREATE TABLE IF NOT EXISTS chat_members(
    chat_id UUID REFERENCES chats(id),
    user_id UUID REFERENCES users(id),
    role VARCHAR(100),
    joined_at TIMESTAMPTZ,

    PRIMARY KEY(chat_id, user_id)
);
-- +goose Down

DROP TABLE IF EXISTS chat_members;
