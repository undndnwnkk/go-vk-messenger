package realtime

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/coder/websocket"
)

func TestClientSendReturnsFalseWhenQueueIsFull(t *testing.T) {
	client := NewClient("alice", nil)

	for i := 0; i < sendBufferSize; i++ {
		if !client.Send([]byte("message")) {
			t.Fatalf("Send returned false before buffer was full at message %d", i)
		}
	}

	done := make(chan bool, 1)
	go func() {
		done <- client.Send([]byte("overflow"))
	}()

	select {
	case ok := <-done:
		if ok {
			t.Fatal("Send returned true for a full queue")
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Send blocked when queue was full")
	}
}

func TestClientWriteLoopStopsWhenContextCancelled(t *testing.T) {
	client := NewClient("alice", nil)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)

	go func() {
		done <- client.WriteLoop(ctx)
	}()

	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("WriteLoop error = %v, want context canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("WriteLoop did not stop after context cancellation")
	}
}

func TestClientCloseIsIdempotent(t *testing.T) {
	client := NewClient("alice", nil)

	client.Close(websocket.StatusNormalClosure, "done")
	client.Close(websocket.StatusNormalClosure, "done again")
}
