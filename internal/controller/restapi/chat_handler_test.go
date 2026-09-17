package restapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/undndnwnkk/go-vk-messenger/internal/model"
	"github.com/undndnwnkk/go-vk-messenger/internal/service"
)

type httpChatRepo struct {
	service.ChatRepositoryInterface
	caller, chatID, target, title string
	role                          model.ChatRole
	memberIDs                     []string
	err                           error
}

func (r *httpChatRepo) CreateGroup(_ context.Context, creator, title string, ids []string) (*model.Chat, error) {
	r.caller = creator
	r.title = title
	r.memberIDs = ids
	return &model.Chat{ID: "created"}, r.err
}
func (r *httpChatRepo) GetOrCreateDirect(_ context.Context, a, b string) (*model.Chat, error) {
	r.caller = a
	r.target = b
	return &model.Chat{ID: "direct"}, r.err
}
func (r *httpChatRepo) GetByID(_ context.Context, id string) (*model.Chat, error) {
	r.chatID = id
	return &model.Chat{ID: id, Type: model.Group}, r.err
}
func (r *httpChatRepo) GetByIDForUser(_ context.Context, id, user string) (*model.Chat, error) {
	r.chatID = id
	r.caller = user
	return &model.Chat{ID: id}, r.err
}
func (r *httpChatRepo) ListByUser(_ context.Context, user string) ([]model.Chat, error) {
	r.caller = user
	return []model.Chat{}, r.err
}
func (r *httpChatRepo) GetMember(_ context.Context, id, user string) (*model.ChatMember, error) {
	r.chatID = id
	r.caller = user
	return &model.ChatMember{Role: model.Admin}, r.err
}
func (r *httpChatRepo) ListMembers(_ context.Context, id string) ([]model.ChatMember, error) {
	r.chatID = id
	return []model.ChatMember{}, r.err
}
func (r *httpChatRepo) AddMember(_ context.Context, id, user string, role model.ChatRole) error {
	r.chatID = id
	r.target = user
	r.role = role
	return r.err
}
func (r *httpChatRepo) RemoveMember(_ context.Context, id, user string) error {
	r.chatID = id
	r.target = user
	return r.err
}

func chatTestHandler(t *testing.T, repo *httpChatRepo) (http.Handler, string) {
	t.Helper()
	jwt := service.NewJWTService("chat-test", time.Minute)
	users := newFakeUserRepo()
	users.usersByID["target"] = model.User{ID: "target"}
	userSvc := service.NewUserService(users, jwt)
	chatSvc := service.NewChatService(repo, *userSvc)
	token, err := jwt.GenerateToken("caller")
	if err != nil {
		t.Fatal(err)
	}
	return NewHandler(*userSvc, jwt, fakeHealthChecker{}, *chatSvc), token
}
func TestChatHTTPRoutes(t *testing.T) {
	cases := []struct {
		name, method, path, body string
		status                   int
	}{
		{"Direct", "POST", "/api/v1/chats/direct", `{"user2_id":"target"}`, 200},
		{"Group", "POST", "/api/v1/chats/group", `{"title":"Group","user_ids":["target"]}`, 201},
		{"List", "GET", "/api/v1/chats/", "", 200},
		{"Get", "GET", "/api/v1/chats/chat", "", 200},
		{"Members", "GET", "/api/v1/chats/chat/members/", "", 200},
		{"Add", "POST", "/api/v1/chats/chat/members/target", `{"role":"admin","chat_id":"wrong","user_id":"wrong"}`, 204},
		{"Delete", "DELETE", "/api/v1/chats/chat/members/target", "", 204},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &httpChatRepo{}
			handler, token := chatTestHandler(t, repo)
			for _, authenticated := range []bool{false, true} {
				req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
				if authenticated {
					req.Header.Set("Authorization", "Bearer "+token)
				}
				rec := httptest.NewRecorder()
				handler.ServeHTTP(rec, req)
				want := 401
				if authenticated {
					want = tc.status
				}
				if rec.Code != want {
					t.Fatalf("status %d want %d: %s", rec.Code, want, rec.Body.String())
				}
				if want == 204 && rec.Body.Len() != 0 {
					t.Fatal("204 has body")
				}
			}
			if repo.caller != "caller" {
				t.Fatalf("JWT caller=%q", repo.caller)
			}
			if tc.name == "Group" && (repo.title != "Group" || len(repo.memberIDs) != 1 || repo.memberIDs[0] != "target") {
				t.Fatalf("body not decoded: %+v", repo)
			}
			if tc.name == "Direct" && repo.target != "target" {
				t.Fatal("direct body not decoded")
			}
			if tc.name == "Get" || tc.name == "Members" || tc.name == "Add" || tc.name == "Delete" {
				if repo.chatID != "chat" {
					t.Fatal("wrong chat URL param")
				}
			}
			if tc.name == "Add" && repo.role != model.Member {
				t.Fatal("client chose role")
			}
			if (tc.name == "Add" || tc.name == "Delete") && repo.target != "target" {
				t.Fatal("wrong user URL param")
			}
		})
	}
}
func TestChatHTTPBadJSON(t *testing.T) {
	for _, path := range []string{"/api/v1/chats/direct", "/api/v1/chats/group"} {
		repo := &httpChatRepo{}
		handler, token := chatTestHandler(t, repo)
		req := httptest.NewRequest("POST", path, strings.NewReader("{"))
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != 400 || repo.caller != "" {
			t.Fatalf("status=%d repo=%+v", rec.Code, repo)
		}
	}
}
func TestChatHTTPErrorMapping(t *testing.T) {
	for _, tc := range []struct {
		err    error
		status int
		code   string
	}{{service.ErrChatNotFound, 404, "chat_not_found"}, {service.ErrMemberNotFound, 404, "member_not_found"}, {service.ErrNotAdmin, 403, "forbidden"}, {service.ErrAlreadyMember, 409, "already_member"}, {service.ErrInvalidTitle, 400, "invalid_title"}, {service.ErrCannotModifyDirect, 400, "cannot_modify_direct"}, {errors.New("secret database details"), 500, "internal_error"}} {
		t.Run(tc.code, func(t *testing.T) {
			repo := &httpChatRepo{err: tc.err}
			handler, token := chatTestHandler(t, repo)
			req := httptest.NewRequest("GET", "/api/v1/chats/chat", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != tc.status || !strings.Contains(rec.Body.String(), tc.code) {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
			if strings.Contains(rec.Body.String(), "secret") {
				t.Fatal("internal detail leaked")
			}
		})
	}
}
