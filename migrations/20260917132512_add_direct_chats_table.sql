-- +goose Up
CREATE TABLE IF NOT EXISTS direct_chats(
    chat_id UUID PRIMARY KEY REFERENCES chats(id),
    user1_id UUID REFERENCES users(id),
    user2_id UUID REFERENCES users(id),

    UNIQUE(user1_id, user2_id)
);
-- +goose Down

DROP TABLE IF EXISTS direct_chats;
