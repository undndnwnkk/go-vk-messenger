package service

import (
	"context"
	"errors"
	"testing"

	"github.com/undndnwnkk/go-vk-messenger/internal/model"
)

func TestUserServiceSearchUsers(t *testing.T) {
	repo := newFakeUserRepository()
	for id, name := range map[string]string{"u1": "alice", "u2": "alina", "u3": "bob", "me": "alex"} {
		repo.usersByID[id] = model.User{ID: id, Username: name}
	}
	svc := newTestUserService(repo)
	ctx := context.Background()

	users, err := svc.SearchUsers(ctx, "me", "  @ALI ", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 2 || users[0].Username != "alice" || users[1].Username != "alina" {
		t.Fatalf("users = %+v, want alice and alina", users)
	}
	if call := repo.searchCalls[0]; call.userID != "me" || call.prefix != "ali" || call.limit != 20 {
		t.Fatalf("search call = %+v", call)
	}

	if _, err := svc.SearchUsers(ctx, "me", "al", 500); err != nil {
		t.Fatal(err)
	}
	if got := repo.searchCalls[1].limit; got != 50 {
		t.Fatalf("capped limit = %d, want 50", got)
	}

	for _, query := range []string{"", "   ", "@"} {
		if _, err := svc.SearchUsers(ctx, "me", query, 10); err != nil {
			t.Fatal(err)
		}
	}
	if len(repo.contactCalls) != 3 || len(repo.searchCalls) != 2 {
		t.Fatalf("blank queries must list contacts: contacts=%d searches=%d", len(repo.contactCalls), len(repo.searchCalls))
	}
	if call := repo.contactCalls[0]; call.userID != "me" || call.limit != 10 {
		t.Fatalf("contacts call = %+v", call)
	}

	for _, limit := range []int{0, -1} {
		if _, err := svc.SearchUsers(ctx, "me", "al", limit); !errors.Is(err, ErrInvalidLimit) {
			t.Fatalf("limit %d: err = %v, want ErrInvalidLimit", limit, err)
		}
	}
}
