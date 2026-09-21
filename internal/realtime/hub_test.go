package realtime_test

import (
	"sync"
	"testing"

	"github.com/undndnwnkk/go-vk-messenger/internal/realtime"
)

func TestHubConnections(t *testing.T) {
	hub := realtime.NewHub()
	alice1 := realtime.NewClient("alice", nil)
	alice2 := realtime.NewClient("alice", nil)
	bob := realtime.NewClient("bob", nil)
	unknown := realtime.NewClient("alice", nil)

	steps := []struct {
		name       string
		operation  func(*realtime.Client)
		client     *realtime.Client
		alice, bob int
	}{
		{"unregister unknown user", hub.Unregister, unknown, 0, 0},
		{"register first", hub.Register, alice1, 1, 0},
		{"register same client again", hub.Register, alice1, 1, 0},
		{"register second connection", hub.Register, alice2, 2, 0},
		{"register another user", hub.Register, bob, 2, 1},
		{"unregister unknown connection", hub.Unregister, unknown, 2, 1},
		{"unregister one of two", hub.Unregister, alice1, 1, 1},
		{"unregister same client again", hub.Unregister, alice1, 1, 1},
		{"unregister last connection", hub.Unregister, alice2, 0, 1},
		{"unregister last again", hub.Unregister, alice2, 0, 1},
		{"reconnect after last removed", hub.Register, alice1, 1, 1},
		{"remove reconnected client", hub.Unregister, alice1, 0, 1},
		{"remove other user", hub.Unregister, bob, 0, 0},
	}
	for _, step := range steps {
		t.Run(step.name, func(t *testing.T) {
			step.operation(step.client)
			for user, want := range map[string]int{"alice": step.alice, "bob": step.bob, "unknown": 0} {
				if got := hub.ConnectionCount(user); got != want {
					t.Fatalf("ConnectionCount(%q) = %d, want %d", user, got, want)
				}
			}
		})
	}
}

func TestHubConcurrentAccess(t *testing.T) {
	hub := realtime.NewHub()
	const perUser = 32
	users := []string{"alice", "bob"}
	var registered, finished sync.WaitGroup
	registered.Add(perUser * len(users))
	finished.Add(perUser * len(users))
	start := make(chan struct{})
	remove := make(chan struct{})
	for _, user := range users {
		for i := 0; i < perUser; i++ {
			client := realtime.NewClient(user, nil)
			go func() {
				defer finished.Done()
				<-start
				hub.Register(client)
				hub.ConnectionCount(user)
				registered.Done()
				<-remove
				hub.Unregister(client)
				hub.ConnectionCount(user)
				// Exercise mixed reads and writes while other clients disconnect.
				for j := 0; j < 50; j++ {
					hub.Register(client)
					hub.ConnectionCount(user)
					hub.Unregister(client)
				}
			}()
		}
	}
	close(start)
	registered.Wait()
	for _, user := range users {
		if got := hub.ConnectionCount(user); got != perUser {
			t.Errorf("ConnectionCount(%q) = %d, want %d", user, got, perUser)
		}
	}
	close(remove)
	finished.Wait()
	for _, user := range users {
		if got := hub.ConnectionCount(user); got != 0 {
			t.Errorf("ConnectionCount(%q) after disconnects = %d, want 0", user, got)
		}
	}
}
