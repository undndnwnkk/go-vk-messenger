package model

import (
	"time"
)

type Message struct {
	ID        int64     `json:"id" db:"id"`
	ChatID    string    `json:"chat_id" db:"chat_id"`
	SenderID  string    `json:"sender_id" db:"sender_id"`
	Content   string    `json:"content" db:"content"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type CreateMessageRequest struct {
	ChatID  string `json:"chat_id"`
	Content string `json:"content"`
}

type HistoryRequest struct {
	ChatID string `json:"chat_id"`
}

type MessagePage struct {
	Messages   []Message `json:"messages"`
	NextCursor *int64    `json:"next_cursor,omitempty"`
}
