package model

import (
	"time"
)

type Message struct {
	ID        int64     `json:"id"`
	ChatID    string    `json:"chat_id"`
	SenderID  string    `json:"sender_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateMessageRequest struct {
	Content string `json:"content"`
}
