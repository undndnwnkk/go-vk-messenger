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
