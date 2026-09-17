package model

import (
	"time"
)

type ChatType string
type MemberRole string

const (
	Direct ChatType   = "direct"
	Group  ChatType   = "group"
	Admin  MemberRole = "admin"
	Member MemberRole = "member"
)

type Chat struct {
	ID        string    `db:"id"`
	Type      ChatType  `db:"type"`
	Title     *string   `db:"title"`
	CreatedBy string    `db:"created_by"`
	CreatedAt time.Time `db:"created_at"`
}

type ChatMember struct {
	ChatID string     `db:"chat_id"`
	UserID string     `db:"user_id"`
	Role   MemberRole `db:"role"`
}

type DirectChat struct {
	ChatID  string `db:"chat_id"`
	User1ID string `db:"user1_id"`
	User2ID string `db:"user2_id"`
}
