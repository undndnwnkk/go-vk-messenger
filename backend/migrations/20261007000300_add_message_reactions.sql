-- +goose Up
ALTER TABLE messages ADD COLUMN reactions_version BIGINT NOT NULL DEFAULT 0;

CREATE TABLE message_reactions (
    message_id BIGINT NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reaction TEXT NOT NULL CHECK (reaction IN ('like', 'heart', 'laugh', 'wow', 'sad', 'angry')),
    PRIMARY KEY (message_id, user_id, reaction)
);

-- +goose Down
DROP TABLE message_reactions;
ALTER TABLE messages DROP COLUMN reactions_version;
