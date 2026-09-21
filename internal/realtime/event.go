package realtime

import (
	"encoding/json"
)

type EventType string

const (
	EventPing  EventType = "ping"
	EventPong  EventType = "pong"
	EventError EventType = "error"
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
