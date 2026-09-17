-- +goose Up
CREATE TABLE IF NOT EXISTS chats(
    id UUID PRIMARY KET DEFAULT gen_random_uuid(),
    type VARCHAR(100) NOT NULL,
    title VARCHAR(100),
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- +goose Down
DROP TABLE IF EXISTS chats;