package realtime

import (
	"context"
	"crypto/rand"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestRedisRelayAcrossInstances(t *testing.T) {
	url := os.Getenv("TEST_REDIS_URL")
	if url == "" {
		t.Skip("set TEST_REDIS_URL for Redis integration tests")
	}
	opts, err := redis.ParseURL(url)
	if err != nil {
		t.Fatal(err)
	}
	client := redis.NewClient(opts)
	defer client.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	h1, h2 := NewHub(), NewHub()
	channel := "test:" + rand.Text()
	r1, err := NewRedisRelay(ctx, client, h1, channel)
	if err != nil {
		t.Fatal(err)
	}
	defer r1.Close()
	r2, err := NewRedisRelay(ctx, client, h2, channel)
	if err != nil {
		t.Fatal(err)
	}
	defer r2.Close()
	alice1, alice2, bob, outsider := NewClient("alice", nil), NewClient("alice", nil), NewClient("bob", nil), NewClient("outsider", nil)
	h1.Register(alice1)
	h2.Register(alice2)
	h2.Register(bob)
	h2.Register(outsider)
	read := func(c *Client, want string) {
		t.Helper()
		select {
		case got := <-c.send:
			if string(got) != want {
				t.Fatalf("got %s want %s", got, want)
			}
		case <-ctx.Done():
			t.Fatal("event not delivered across instances")
		}
	}
	for _, event := range []EventType{EventMessageCreated, EventMessageEdited, EventMessageDeleted, EventReactionsUpdated, EventReadUpdated} {
		data := MustEventJSON(event, map[string]string{"chat_id": "chat"})
		h1.SendToUsers([]string{"alice", "bob", "bob"}, data)
		read(alice1, string(data))
		read(alice2, string(data))
		read(bob, string(data))
	}
	private := MustEventJSON(EventChatMuteUpdated, map[string]bool{"muted": true})
	h2.SendToUser("alice", private)
	read(alice1, string(private))
	read(alice2, string(private))
	for _, c := range []*Client{alice1, alice2, bob, outsider} {
		select {
		case data := <-c.send:
			t.Fatalf("duplicate or leaked event: %s", data)
		default:
		}
	}
	// Redis failure does not prevent local delivery or panic on shutdown.
	_ = client.Close()
	h1.SendToUser("alice", private)
	read(alice1, string(private))
	r1.Close()
	r1.Close()
}
