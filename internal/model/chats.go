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
	ID        string    `db:"id"`
	Type      ChatType  `db:"type"`
	Title     *string   `db:"title"`
	CreatedBy string    `db:"created_by"`
	CreatedAt time.Time `db:"created_at"`
}

type ChatMember struct {
	ChatID   string    `db:"chat_id"`
	UserID   string    `db:"user_id"`
	Role     ChatRole  `db:"role"`
	JoinedAt time.Time `db:"joined_at"`
}

type DirectChat struct {
	ChatID  string `db:"chat_id"`
	User1ID string `db:"user1_id"`
	User2ID string `db:"user2_id"`
}

type CreateDirectChat struct {
	UserID2 string `json:"user2_id"`
}

type CreateGroupChat struct {
	Title   string   `json:"title"`
	UserIDs []string `json:"user_ids"`
}
