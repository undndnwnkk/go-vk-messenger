package service

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/undndnwnkk/go-vk-messenger/internal/model"
	"github.com/undndnwnkk/go-vk-messenger/internal/repository"
)

type chatRepoStub struct {
	ChatRepositoryInterface
	chatType     model.ChatType
	role         model.ChatRole
	memberErr    error
	accessErr    error
	mutationErr  error
	readState    *model.ChatReadState
	readAdvanced bool
	readErr      error
	readCalled   bool
	readChatID   string
	readUserID   string
	readMessage  int64
	added        bool
	removed      bool
	listed       bool
	direct       bool
	group        bool
	groupTitle   string
	groupMembers []string
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
	if chatID != "cccccccc-cccc-4ccc-8ccc-cccccccccccc" || userID != "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb" || role != model.Member {
		panic("wrong membership arguments")
	}
	r.added = true
	return r.mutationErr
}
func (r *chatRepoStub) RemoveMember(context.Context, string, string) error {
	r.removed = true
	return r.mutationErr
}
func (r *chatRepoStub) MarkRead(_ context.Context, chatID string, userID string, messageID int64) (*model.ChatReadState, bool, error) {
	r.readCalled = true
	r.readChatID = chatID
	r.readUserID = userID
	r.readMessage = messageID
	if r.readState == nil {
		r.readState = &model.ChatReadState{ChatID: chatID, UserID: userID, LastReadMessageID: messageID}
	}
	return r.readState, r.readAdvanced, r.readErr
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
	}{{"Self", "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", true, ErrCommonUser}, {"Missing", "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", false, ErrTargetUserNotFound}, {"Success", "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", true, nil}} {
		t.Run(tc.name, func(t *testing.T) {
			users := newFakeUserRepository()
			if tc.exists {
				users.usersByID[tc.target] = model.User{ID: tc.target}
			}
			repo := &chatRepoStub{}
			svc := NewChatService(repo, *newTestUserService(users))
			_, err := svc.CreateDirectChat(context.Background(), "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", tc.target)
			if !errors.Is(err, tc.want) || repo.direct != (tc.want == nil) {
				t.Fatalf("err=%v repo called=%v", err, repo.direct)
			}
		})
	}
}
func TestChatServiceInvalidTitle(t *testing.T) {
	svc := NewChatService(&chatRepoStub{}, UserService{})
	for _, title := range []string{"", "   ", "\t\n"} {
		if _, err := svc.CreateGroupChat(context.Background(), "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", model.CreateGroupChat{Title: title}); !errors.Is(err, ErrInvalidTitle) {
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
		{name: "MissingTarget", kind: model.Group, role: model.Admin, missing: true, want: ErrTargetUserNotFound},
		{name: "Duplicate", kind: model.Group, role: model.Admin, mutationErr: repository.ErrMemberAlreadyExists, want: ErrAlreadyMember},
	} {
		t.Run("Add"+tc.name, func(t *testing.T) {
			users := newFakeUserRepository()
			if !tc.missing {
				users.usersByID["bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"] = model.User{ID: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"}
			}
			repo := &chatRepoStub{chatType: tc.kind, role: tc.role, memberErr: tc.memberErr, mutationErr: tc.mutationErr}
			svc := NewChatService(repo, *newTestUserService(users))
			err := svc.AddMember(context.Background(), "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", "cccccccc-cccc-4ccc-8ccc-cccccccccccc", "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb")
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
			err := svc.RemoveMember(context.Background(), "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", "cccccccc-cccc-4ccc-8ccc-cccccccccccc", "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb")
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
		_, err := svc.GetMembersByChatID(context.Background(), "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", "cccccccc-cccc-4ccc-8ccc-cccccccccccc")
		if accessErr != nil && !errors.Is(err, ErrChatNotFound) {
			t.Fatal(err)
		}
		if repo.listed != (accessErr == nil) {
			t.Fatalf("listed=%v", repo.listed)
		}
	}
}

func TestChatServiceMarkRead(t *testing.T) {
	for _, tc := range []struct {
		name     string
		userID   string
		chatID   string
		message  int64
		repoErr  error
		want     error
		called   bool
		advanced bool
	}{
		{"Success", "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", "cccccccc-cccc-4ccc-8ccc-cccccccccccc", 7, nil, nil, true, true},
		{"Stale", "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", "cccccccc-cccc-4ccc-8ccc-cccccccccccc", 7, nil, nil, true, false},
		{"ZeroMessage", "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", "cccccccc-cccc-4ccc-8ccc-cccccccccccc", 0, nil, ErrInvalidMessageID, false, false},
		{"InvalidUser", "hello", "cccccccc-cccc-4ccc-8ccc-cccccccccccc", 7, nil, ErrInvalidID, false, false},
		{"InvalidChat", "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", "hello", 7, nil, ErrInvalidID, false, false},
		{"Outsider", "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", "cccccccc-cccc-4ccc-8ccc-cccccccccccc", 7, repository.ErrMemberNotFound, ErrChatNotFound, true, false},
		{"WrongMessage", "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", "cccccccc-cccc-4ccc-8ccc-cccccccccccc", 7, repository.ErrMessageNotFound, ErrMessageNotFound, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &chatRepoStub{readAdvanced: tc.advanced, readErr: tc.repoErr}
			svc := NewChatService(repo, UserService{})
			state, advanced, err := svc.MarkRead(context.Background(), tc.userID, tc.chatID, tc.message)
			if !errors.Is(err, tc.want) || repo.readCalled != tc.called || advanced != (tc.want == nil && tc.advanced) {
				t.Fatalf("err=%v called=%v advanced=%v", err, repo.readCalled, advanced)
			}
			if tc.want != nil {
				return
			}
			if state.ChatID != tc.chatID || state.UserID != tc.userID || state.LastReadMessageID != tc.message || repo.readChatID != tc.chatID || repo.readUserID != tc.userID || repo.readMessage != tc.message {
				t.Fatalf("state=%+v repo=%+v", state, repo)
			}
		})
	}
}

func (r *chatRepoStub) CreateGroup(_ context.Context, _ string, title string, ids []string) (*model.Chat, error) {
	r.group = true
	r.groupTitle = title
	r.groupMembers = ids
	return &model.Chat{}, nil
}

func TestGroupInputNormalization(t *testing.T) {
	const creator = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	const bob = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
	const missing = "dddddddd-dddd-4ddd-8ddd-dddddddddddd"
	for _, tc := range []struct {
		name      string
		ids, want []string
		err       error
	}{
		{"Creator", []string{creator, strings.ToUpper(creator), bob}, []string{bob}, nil},
		{"Duplicates", []string{bob, bob, strings.ToUpper(bob)}, []string{bob}, nil},
		{"OnlyCreator", []string{creator}, []string{}, nil},
		{"Missing", []string{bob, missing}, nil, ErrTargetUserNotFound},
		{"Invalid", []string{bob, "hello"}, nil, ErrInvalidID},
	} {
		t.Run(tc.name, func(t *testing.T) {
			users := newFakeUserRepository()
			users.usersByID[bob] = model.User{ID: bob}
			repo := &chatRepoStub{}
			svc := NewChatService(repo, *newTestUserService(users))
			_, err := svc.CreateGroupChat(context.Background(), creator, model.CreateGroupChat{Title: " backend ", UserIDs: tc.ids})
			if !errors.Is(err, tc.err) || repo.group != (tc.err == nil) {
				t.Fatalf("err=%v created=%v", err, repo.group)
			}
			if tc.err == nil && (!reflect.DeepEqual(repo.groupMembers, tc.want) || repo.groupTitle != "backend") {
				t.Fatalf("title=%q members=%v", repo.groupTitle, repo.groupMembers)
			}
		})
	}
}

func TestGroupTitleLength(t *testing.T) {
	for _, tc := range []struct {
		title string
		valid bool
	}{{" x ", true}, {strings.Repeat("?", 100), true}, {"  " + strings.Repeat("?", 100) + "  ", true}, {strings.Repeat("?", 101), false}, {strings.Repeat("a", 101), false}} {
		repo := &chatRepoStub{}
		svc := NewChatService(repo, UserService{})
		_, err := svc.CreateGroupChat(context.Background(), "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", model.CreateGroupChat{Title: tc.title})
		if tc.valid && err != nil || !tc.valid && !errors.Is(err, ErrInvalidTitle) || repo.group != tc.valid {
			t.Fatalf("title %q: %v created=%v", tc.title, err, repo.group)
		}
	}
}

func TestCannotRemoveSelf(t *testing.T) {
	repo := &chatRepoStub{chatType: model.Group, role: model.Admin}
	svc := NewChatService(repo, UserService{})
	err := svc.RemoveMember(context.Background(), "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", "cccccccc-cccc-4ccc-8ccc-cccccccccccc", "AAAAAAAA-AAAA-4AAA-8AAA-AAAAAAAAAAAA")
	if !errors.Is(err, ErrCannotRemoveSelf) || repo.removed {
		t.Fatalf("err=%v removed=%v", err, repo.removed)
	}
}

func TestInvalidChatIDsStopBeforeRepository(t *testing.T) {
	// The embedded nil repository panics if validation lets any call through.
	svc := NewChatService(struct{ ChatRepositoryInterface }{}, UserService{})
	ctx := context.Background()
	const id = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	calls := []func() error{
		func() error { _, err := svc.CreateDirectChat(ctx, id, "hello"); return err },
		func() error { _, err := svc.CreateDirectChat(ctx, "hello", id); return err },
		func() error {
			_, err := svc.CreateGroupChat(ctx, "hello", model.CreateGroupChat{Title: "ok"})
			return err
		},
		func() error {
			_, err := svc.CreateGroupChat(ctx, id, model.CreateGroupChat{Title: "ok", UserIDs: []string{"hello"}})
			return err
		},
		func() error { _, err := svc.GetChats(ctx, "hello"); return err },
		func() error { _, err := svc.GetChatByID(ctx, id, "hello"); return err },
		func() error { _, err := svc.GetMembersByChatID(ctx, id, "hello"); return err },
		func() error { return svc.AddMember(ctx, id, id, "hello") },
		func() error { return svc.RemoveMember(ctx, id, id, "hello") },
	}
	for i, call := range calls {
		if err := call(); !errors.Is(err, ErrInvalidID) {
			t.Fatalf("case %d: %v", i, err)
		}
	}
	for _, bad := range []string{"", strings.Repeat("a", 36), "gggggggg-aaaa-4aaa-8aaa-aaaaaaaaaaaa", "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa", "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaaa"} {
		if err := normalizeChatIDs(&bad); !errors.Is(err, ErrInvalidID) {
			t.Fatalf("accepted %q", bad)
		}
	}
}

type failingChatUserRepo struct {
	UserRepositoryInterface
	err error
}

func (r failingChatUserRepo) GetByID(context.Context, string) (*model.User, error) { return nil, r.err }
func TestTargetLookupPreservesUnexpectedError(t *testing.T) {
	failure := errors.New("database unavailable")
	svc := NewChatService(&chatRepoStub{}, *NewUserService(failingChatUserRepo{err: failure}, nil))
	_, err := svc.CreateDirectChat(context.Background(), "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb")
	if !errors.Is(err, failure) {
		t.Fatalf("unexpected error masked: %v", err)
	}
}
