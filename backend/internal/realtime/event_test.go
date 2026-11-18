package realtime

import (
	"encoding/json"
	"testing"
)

func TestEventJSONEncodeDecode(t *testing.T) {
	raw := []byte(`{"type":"ping","data":{"hello":"world"}}`)

	var event Event
	if err := json.Unmarshal(raw, &event); err != nil {
		t.Fatalf("unmarshal event: %v", err)
	}
	if event.Type != EventPing {
		t.Fatalf("event type = %q, want %q", event.Type, EventPing)
	}
	if string(event.Data) != `{"hello":"world"}` {
		t.Fatalf("event data = %s, want raw data object", event.Data)
	}

	encoded := MustJSON(Event{Type: EventPong})
	var decoded Event
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal encoded event: %v", err)
	}
	if decoded.Type != EventPong {
		t.Fatalf("encoded event type = %q, want %q", decoded.Type, EventPong)
	}
}

func TestNewErrorEventJSON(t *testing.T) {
	encoded := NewErrorEvent("invalid_event", "invalid websocket event")

	var event ErrorEvent
	if err := json.Unmarshal(encoded, &event); err != nil {
		t.Fatalf("unmarshal error event: %v", err)
	}
	if event.Type != EventError {
		t.Fatalf("error event type = %q, want %q", event.Type, EventError)
	}
	if event.Error.Code != "invalid_event" {
		t.Fatalf("error code = %q, want invalid_event", event.Error.Code)
	}
	if event.Error.Message != "invalid websocket event" {
		t.Fatalf("error message = %q, want invalid websocket event", event.Error.Message)
	}
}
