-- +goose Up
CREATE TABLE chats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type VARCHAR(20) NOT NULL
        CHECK (type IN ('direct', 'group')),
    title VARCHAR(100),
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (
        (type = 'direct' AND title IS NULL)
        OR
        (type = 'group' AND title IS NOT NULL)
    )
);
-- +goose Down
DROP TABLE IF EXISTS chats;