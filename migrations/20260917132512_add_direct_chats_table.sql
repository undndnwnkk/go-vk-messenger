-- +goose Up
CREATE TABLE direct_chats (
    chat_id UUID PRIMARY KEY
        REFERENCES chats(id)
        ON DELETE CASCADE,
    user1_id UUID NOT NULL
        REFERENCES users(id),
    user2_id UUID NOT NULL
        REFERENCES users(id),
        
    UNIQUE(user1_id, user2_id),
    CHECK(user1_id <> user2_id)
);
-- +goose Down

DROP TABLE IF EXISTS direct_chats;
