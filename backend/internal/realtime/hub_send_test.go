package realtime

import (
	"testing"
	"time"
)

func TestHubSendToUserSendsToAllUserConnections(t *testing.T) {
	hub := NewHub()
	alice1 := NewClient("alice", nil)
	alice2 := NewClient("alice", nil)
	bob := NewClient("bob", nil)

	hub.Register(alice1)
	hub.Register(alice2)
	hub.Register(bob)

	hub.SendToUser("alice", []byte("hello"))

	assertQueuedMessage(t, alice1, "hello")
	assertQueuedMessage(t, alice2, "hello")
	assertNoQueuedMessage(t, bob)
}

func TestHubSendToUsersSendsOnlyToListedUsers(t *testing.T) {
	hub := NewHub()
	alice := NewClient("alice", nil)
	bob1 := NewClient("bob", nil)
	bob2 := NewClient("bob", nil)
	carol := NewClient("carol", nil)

	hub.Register(alice)
	hub.Register(bob1)
	hub.Register(bob2)
	hub.Register(carol)

	hub.SendToUsers([]string{"alice", "bob"}, []byte("created"))

	assertQueuedMessage(t, alice, "created")
	assertQueuedMessage(t, bob1, "created")
	assertQueuedMessage(t, bob2, "created")
	assertNoQueuedMessage(t, carol)
}

func TestHubSendToUserRemovesSlowClientWithoutBlockingOthers(t *testing.T) {
	hub := NewHub()
	slow := NewClient("bob", nil)
	fast := NewClient("bob", nil)

	for i := 0; i < sendBufferSize; i++ {
		if !slow.Send([]byte("backlog")) {
			t.Fatalf("slow client queue filled early at %d", i)
		}
	}
	hub.Register(slow)
	hub.Register(fast)

	hub.SendToUser("bob", []byte("created"))

	assertQueuedMessage(t, fast, "created")
	if got := hub.ConnectionCount("bob"); got != 1 {
		t.Fatalf("ConnectionCount(%q) = %d, want 1", "bob", got)
	}
}

func TestHubShutdownClearsClientsAndIsIdempotent(t *testing.T) {
	hub := NewHub()
	hub.Register(NewClient("alice", nil))
	hub.Register(NewClient("alice", nil))
	hub.Register(NewClient("bob", nil))

	hub.Shutdown()
	hub.Shutdown()

	if got := hub.ConnectionCount("alice"); got != 0 {
		t.Fatalf("ConnectionCount(%q) = %d, want 0", "alice", got)
	}
	if got := hub.ConnectionCount("bob"); got != 0 {
		t.Fatalf("ConnectionCount(%q) = %d, want 0", "bob", got)
	}
}

func assertQueuedMessage(t *testing.T, client *Client, want string) {
	t.Helper()

	select {
	case got := <-client.send:
		if string(got) != want {
			t.Fatalf("queued message = %q, want %q", got, want)
		}
	case <-time.After(time.Second):
		t.Fatalf("client did not receive queued message %q", want)
	}
}

func assertNoQueuedMessage(t *testing.T, client *Client) {
	t.Helper()

	select {
	case got := <-client.send:
		t.Fatalf("unexpected queued message: %q", got)
	default:
	}
}
