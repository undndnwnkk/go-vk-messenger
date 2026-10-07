package model

import (
	"time"
)

type Message struct {
	ID               int64         `json:"id" db:"id"`
	ChatID           string        `json:"chat_id" db:"chat_id"`
	SenderID         string        `json:"sender_id" db:"sender_id"`
	Content          string        `json:"content" db:"content"`
	CreatedAt        time.Time     `json:"created_at" db:"created_at"`
	Edited           bool          `json:"edited"`
	EditedAt         *time.Time    `json:"edited_at" db:"edited_at"`
	Deleted          bool          `json:"deleted"`
	DeletedAt        *time.Time    `json:"deleted_at" db:"deleted_at"`
	ReplyToMessageID *int64        `json:"reply_to_message_id" db:"reply_to_message_id"`
	ReplyTo          *MessageReply `json:"reply_to"`
}

// MessageReply is a live, one-level preview, not a stored copy of the original text.
type MessageReply struct {
	ID        int64      `json:"id"`
	SenderID  string     `json:"sender_id"`
	Content   string     `json:"content"`
	Edited    bool       `json:"edited"`
	EditedAt  *time.Time `json:"edited_at"`
	Deleted   bool       `json:"deleted"`
	DeletedAt *time.Time `json:"deleted_at"`
}

type CreateMessageRequest struct {
	Content          string `json:"content"`
	ReplyToMessageID *int64 `json:"reply_to_message_id"`
}

type EditMessageRequest struct {
	Content string `json:"content"`
}

type MessagePage struct {
	Messages   []Message `json:"messages"`
	NextCursor *int64    `json:"next_cursor,omitempty"`
}
