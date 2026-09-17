package service

import (
	"context"
	"errors"
	"testing"

	"github.com/undndnwnkk/go-vk-messenger/internal/model"
	"github.com/undndnwnkk/go-vk-messenger/internal/repository"
)

type chatRepoStub struct {
	ChatRepositoryInterface
	chatType    model.ChatType
	role        model.ChatRole
	memberErr   error
	accessErr   error
	mutationErr error
	added       bool
	removed     bool
	listed      bool
	direct      bool
}

func (r *chatRepoStub) GetByID(context.Context, string) (*model.Chat, error) {
	return &model.Chat{Type: r.chatType}, nil
}
func (r *chatRepoStub) GetMember(context.Context, string, string) (*model.ChatMember, error) {
	return &model.ChatMember{Role: r.role}, r.memberErr
}
func (r *chatRepoStub) GetByIDForUser(context.Context, string, string) (*model.Chat, error) {
	return &model.Chat{}, r.accessErr
}
func (r *chatRepoStub) ListMembers(context.Context, string) ([]model.ChatMember, error) {
	r.listed = true
	return []model.ChatMember{}, nil
}
func (r *chatRepoStub) AddMember(_ context.Context, chatID, userID string, role model.ChatRole) error {
	if chatID != "chat" || userID != "target" || role != model.Member {
		panic("wrong membership arguments")
	}
	r.added = true
	return r.mutationErr
}
func (r *chatRepoStub) RemoveMember(context.Context, string, string) error {
	r.removed = true
	return r.mutationErr
}
func (r *chatRepoStub) GetOrCreateDirect(context.Context, string, string) (*model.Chat, error) {
	r.direct = true
	return &model.Chat{}, nil
}

func TestChatServiceDirectValidation(t *testing.T) {
	for _, tc := range []struct {
		name, target string
		exists       bool
		want         error
	}{{"Self", "alice", true, ErrCommonUser}, {"Missing", "target", false, ErrUserNotFound}, {"Success", "target", true, nil}} {
		t.Run(tc.name, func(t *testing.T) {
			users := newFakeUserRepository()
			if tc.exists {
				users.usersByID[tc.target] = model.User{ID: tc.target}
			}
			repo := &chatRepoStub{}
			svc := NewChatService(repo, *newTestUserService(users))
			_, err := svc.CreateDirectChat(context.Background(), "alice", tc.target)
			if !errors.Is(err, tc.want) || repo.direct != (tc.want == nil) {
				t.Fatalf("err=%v repo called=%v", err, repo.direct)
			}
		})
	}
}
func TestChatServiceInvalidTitle(t *testing.T) {
	svc := NewChatService(&chatRepoStub{}, UserService{})
	for _, title := range []string{"", "   ", "\t\n"} {
		if _, err := svc.CreateGroupChat(context.Background(), "alice", model.CreateGroupChat{Title: title}); !errors.Is(err, ErrInvalidTitle) {
			t.Fatal(err)
		}
	}
}
func TestChatServiceMembership(t *testing.T) {
	for _, tc := range []struct {
		name                         string
		kind                         model.ChatType
		role                         model.ChatRole
		memberErr, mutationErr, want error
		missing                      bool
	}{
		{name: "Success", kind: model.Group, role: model.Admin},
		{name: "NotAdmin", kind: model.Group, role: model.Member, want: ErrNotAdmin},
		{name: "Outsider", kind: model.Group, memberErr: repository.ErrMemberNotFound, want: ErrNotAdmin},
		{name: "Direct", kind: model.Direct, role: model.Admin, want: ErrCannotModifyDirect},
		{name: "MissingTarget", kind: model.Group, role: model.Admin, missing: true, want: ErrUserNotFound},
		{name: "Duplicate", kind: model.Group, role: model.Admin, mutationErr: repository.ErrMemberAlreadyExists, want: ErrAlreadyMember},
	} {
		t.Run("Add"+tc.name, func(t *testing.T) {
			users := newFakeUserRepository()
			if !tc.missing {
				users.usersByID["target"] = model.User{ID: "target"}
			}
			repo := &chatRepoStub{chatType: tc.kind, role: tc.role, memberErr: tc.memberErr, mutationErr: tc.mutationErr}
			svc := NewChatService(repo, *newTestUserService(users))
			err := svc.AddMember(context.Background(), "caller", "chat", "target")
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v want %v", err, tc.want)
			}
			allowed := tc.want == nil || tc.mutationErr != nil
			if repo.added != allowed {
				t.Fatalf("mutation called=%v", repo.added)
			}
		})
	}
	for _, tc := range []struct {
		name              string
		kind              model.ChatType
		role              model.ChatRole
		mutationErr, want error
	}{{"Success", model.Group, model.Admin, nil, nil}, {"NotAdmin", model.Group, model.Member, nil, ErrNotAdmin}, {"Direct", model.Direct, model.Admin, nil, ErrCannotModifyDirect}, {"Missing", model.Group, model.Admin, repository.ErrMemberNotFound, ErrMemberNotFound}} {
		t.Run("Remove"+tc.name, func(t *testing.T) {
			repo := &chatRepoStub{chatType: tc.kind, role: tc.role, mutationErr: tc.mutationErr}
			svc := NewChatService(repo, UserService{})
			err := svc.RemoveMember(context.Background(), "caller", "chat", "target")
			if !errors.Is(err, tc.want) || repo.removed != (tc.want == nil || tc.mutationErr != nil) {
				t.Fatalf("err=%v removed=%v", err, repo.removed)
			}
		})
	}
}
func TestChatServiceMembersAccess(t *testing.T) {
	for _, accessErr := range []error{nil, repository.ErrChatNotFound} {
		repo := &chatRepoStub{accessErr: accessErr}
		svc := NewChatService(repo, UserService{})
		_, err := svc.GetMembersByChatID(context.Background(), "caller", "chat")
		if accessErr != nil && !errors.Is(err, ErrChatNotFound) {
			t.Fatal(err)
		}
		if repo.listed != (accessErr == nil) {
			t.Fatalf("listed=%v", repo.listed)
		}
	}
}
