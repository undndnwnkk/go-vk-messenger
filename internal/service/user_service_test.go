package service

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/undndnwnkk/go-vk-messenger/internal/model"
	"github.com/undndnwnkk/go-vk-messenger/internal/repository"
)

type fakeUserRepository struct {
	usersByID       map[string]model.User
	usersByUsername map[string]model.User
	nextID          int
}

func newFakeUserRepository() *fakeUserRepository {
	return &fakeUserRepository{
		usersByID:       make(map[string]model.User),
		usersByUsername: make(map[string]model.User),
	}
}

func (r *fakeUserRepository) Create(_ context.Context, user model.User) (string, error) {
	if _, ok := r.usersByUsername[user.Username]; ok {
		return "", repository.ErrUsernameTaken
	}
	r.nextID++
	user.ID = "user-" + strconv.Itoa(r.nextID)
	user.CreatedAt = time.Now().UTC()
	r.usersByID[user.ID] = user
	r.usersByUsername[user.Username] = user
	return user.ID, nil
}

func (r *fakeUserRepository) GetByUsername(_ context.Context, username string) (*model.User, error) {
	user, ok := r.usersByUsername[username]
	if !ok {
		return nil, repository.ErrUserNotFound
	}
	return &user, nil
}

func (r *fakeUserRepository) GetByID(_ context.Context, id string) (*model.User, error) {
	user, ok := r.usersByID[id]
	if !ok {
		return nil, repository.ErrUserNotFound
	}
	return &user, nil
}

func newTestUserService(repo *fakeUserRepository) *UserService {
	return NewUserService(repo, NewJWTService("test-secret", 15*time.Minute))
}

func TestUserServiceRegister(t *testing.T) {
	repo := newFakeUserRepository()
	svc := newTestUserService(repo)

	token, err := svc.Register(context.Background(), model.CreateUserRequest{
		Username: " Alice ",
		Password: "verysecret",
	})
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}
	if token == "" {
		t.Fatal("Register returned empty token")
	}

	user := repo.usersByUsername["alice"]
	if user.PasswordHash == "" || user.PasswordHash == "verysecret" {
		t.Fatalf("password was not hashed: %q", user.PasswordHash)
	}
	if !checkPasswordHash("verysecret", user.PasswordHash) {
		t.Fatal("stored password hash does not match the password")
	}
}

func TestUserServiceRegisterRejectsInvalidUsernames(t *testing.T) {
	tests := []string{
		"   ",
		"ab",
		"hello world",
		"!!!!",
		"abcdefghijklmnopqrstuvwxyzabcdefg",
	}

	for _, username := range tests {
		t.Run(username, func(t *testing.T) {
			repo := newFakeUserRepository()
			svc := newTestUserService(repo)

			_, err := svc.Register(context.Background(), model.CreateUserRequest{
				Username: username,
				Password: "verysecret",
			})
			if !errors.Is(err, ErrInvalidUsername) {
				t.Fatalf("Register error = %v, want %v", err, ErrInvalidUsername)
			}
		})
	}
}

func TestUserServiceRegisterDuplicate(t *testing.T) {
	repo := newFakeUserRepository()
	svc := newTestUserService(repo)
	req := model.CreateUserRequest{Username: "alice", Password: "verysecret"}

	if _, err := svc.Register(context.Background(), req); err != nil {
		t.Fatalf("first Register returned error: %v", err)
	}
	if _, err := svc.Register(context.Background(), req); !errors.Is(err, ErrUserAlreadyExists) {
		t.Fatalf("duplicate Register error = %v, want %v", err, ErrUserAlreadyExists)
	}
}

func TestUserServiceLoginSuccess(t *testing.T) {
	repo := newFakeUserRepository()
	svc := newTestUserService(repo)
	if _, err := svc.Register(context.Background(), model.CreateUserRequest{Username: "alice", Password: "verysecret"}); err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	token, err := svc.Login(context.Background(), model.CreateUserRequest{
		Username: "ALICE",
		Password: "verysecret",
	})
	if err != nil {
		t.Fatalf("Login returned error: %v", err)
	}
	if token == "" {
		t.Fatal("Login returned empty token")
	}
}

func TestUserServiceLoginInvalidCredentials(t *testing.T) {
	repo := newFakeUserRepository()
	svc := newTestUserService(repo)
	if _, err := svc.Register(context.Background(), model.CreateUserRequest{Username: "alice", Password: "verysecret"}); err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	if _, err := svc.Login(context.Background(), model.CreateUserRequest{Username: "alice", Password: "wrongpassword"}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("wrong password error = %v, want %v", err, ErrInvalidCredentials)
	}
	if _, err := svc.Login(context.Background(), model.CreateUserRequest{Username: "missing", Password: "verysecret"}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("wrong username error = %v, want %v", err, ErrInvalidCredentials)
	}
}
