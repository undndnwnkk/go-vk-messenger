package model

type ReactionSummary struct {
	Reaction string   `json:"reaction"`
	Count    int64    `json:"count"`
	UserIDs  []string `json:"user_ids"`
}

type MessageReactions struct {
	ChatID    string            `json:"chat_id"`
	MessageID int64             `json:"message_id"`
	Version   int64             `json:"reactions_version"`
	Reactions []ReactionSummary `json:"reactions"`
}
