package model

import (
	"time"
)

type ChatType string
type ChatRole string

const (
	Direct ChatType = "direct"
	Group  ChatType = "group"
	Admin  ChatRole = "admin"
	Member ChatRole = "member"
)

type Chat struct {
	ID                string    `db:"id" json:"id"`
	Type              ChatType  `db:"type" json:"type"`
	Title             *string   `db:"title" json:"title"`
	CreatedBy         string    `db:"created_by" json:"created_by"`
	CreatedAt         time.Time `db:"created_at" json:"created_at"`
	LastReadMessageID *int64    `db:"last_read_message_id" json:"last_read_message_id,omitempty"`
	UnreadCount       int64     `db:"unread_count" json:"unread_count"`
}

type ChatMember struct {
	ChatID            string    `db:"chat_id" json:"chat_id"`
	UserID            string    `db:"user_id" json:"user_id"`
	Role              ChatRole  `db:"role" json:"role"`
	JoinedAt          time.Time `db:"joined_at" json:"joined_at"`
	LastReadMessageID *int64    `db:"last_read_message_id" json:"last_read_message_id,omitempty"`
}

type DirectChat struct {
	ChatID  string `db:"chat_id" json:"chat_id"`
	User1ID string `db:"user1_id" json:"user1_id"`
	User2ID string `db:"user2_id" json:"user2_id"`
}

type CreateDirectChat struct {
	UserID2 string `json:"user2_id"`
}

type CreateGroupChat struct {
	Title   string   `json:"title"`
	UserIDs []string `json:"user_ids"`
}

type MarkReadRequest struct {
	MessageID int64 `json:"message_id"`
}

type ChatReadState struct {
	ChatID            string `json:"chat_id"`
	UserID            string `json:"user_id"`
	LastReadMessageID int64  `json:"last_read_message_id"`
}
