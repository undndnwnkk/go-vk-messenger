package realtime

import (
	"encoding/json"
)

type EventType string

const (
	EventPing             EventType = "ping"
	EventPong             EventType = "pong"
	EventError            EventType = "error"
	EventSendMessage      EventType = "send_message"
	EventMessageAck       EventType = "message_ack"
	EventMessageCreated   EventType = "message_created"
	EventMessageEdited    EventType = "message_edited"
	EventMessageDeleted   EventType = "message_deleted"
	EventReadUpdated      EventType = "read_updated"
	EventReactionsUpdated EventType = "reactions_updated"
)

type Event struct {
	Type EventType       `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

type ErrorEvent struct {
	Type  EventType `json:"type"`
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type SendMessagePayload struct {
	ChatID           string `json:"chat_id"`
	Content          string `json:"content"`
	ReplyToMessageID *int64 `json:"reply_to_message_id"`
}

func NewErrorEvent(code, message string) []byte {
	var event ErrorEvent
	event.Type = EventError
	event.Error.Code = code
	event.Error.Message = message

	res, _ := json.Marshal(event)
	return res
}

func MustJSON(event Event) []byte {
	res, _ := json.Marshal(event)
	return res
}

func MustEventJSON(eventType EventType, data any) []byte {
	raw, _ := json.Marshal(data)
	return MustJSON(Event{Type: eventType, Data: raw})
}
