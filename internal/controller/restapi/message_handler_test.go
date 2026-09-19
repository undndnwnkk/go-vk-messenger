package restapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/undndnwnkk/go-vk-messenger/internal/model"
	"github.com/undndnwnkk/go-vk-messenger/internal/repository"
	"github.com/undndnwnkk/go-vk-messenger/internal/service"
)

const httpMessageChat = "cccccccc-cccc-4ccc-8ccc-cccccccccccc"
const httpMessageUser = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
const httpMessagePath = "/api/v1/chats/" + httpMessageChat + "/messages"

type httpMessageRepo struct {
	called                       bool
	chat, sender, content, query string
	limit                        int
	before                       *int64
	messages                     []model.Message
	err                          error
}

func (r *httpMessageRepo) Create(_ context.Context, chat, sender, content string) (*model.Message, error) {
	r.called, r.chat, r.sender, r.content = true, chat, sender, content
	return &model.Message{ID: 1, ChatID: chat, SenderID: sender, Content: content}, r.err
}
func (r *httpMessageRepo) ListBefore(_ context.Context, chat string, before *int64, limit int) ([]model.Message, error) {
	r.called, r.chat, r.before, r.limit = true, chat, before, limit
	if r.messages == nil {
		return []model.Message{}, r.err
	}
	return r.messages, r.err
}
func (r *httpMessageRepo) Search(_ context.Context, chat, query string, limit int) ([]model.Message, error) {
	r.called, r.chat, r.query, r.limit = true, chat, query, limit
	if r.messages == nil {
		return []model.Message{}, r.err
	}
	return r.messages, r.err
}
func messageHTTPHandler(t *testing.T, repo *httpMessageRepo, accessErr error) (http.Handler, string) {
	t.Helper()
	jwt := service.NewJWTService("message-test", time.Minute)
	user := service.NewUserService(newFakeUserRepo(), jwt)
	chat := service.NewChatService(&httpChatRepo{err: accessErr}, *user)
	message := service.NewMessageService(repo, *chat)
	token, err := jwt.GenerateToken(httpMessageUser)
	if err != nil {
		t.Fatal(err)
	}
	return NewHandler(*user, jwt, fakeHealthChecker{}, *chat, *message), token
}
func messageHTTPRequest(handler http.Handler, token, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}
func TestMessageHTTPNoJWT(t *testing.T) {
	for _, tc := range []struct{ method, path string }{{"POST", httpMessagePath}, {"GET", httpMessagePath}, {"GET", httpMessagePath + "/search?q=hello"}} {
		t.Run(tc.method+tc.path, func(t *testing.T) {
			repo := &httpMessageRepo{}
			h, _ := messageHTTPHandler(t, repo, nil)
			rec := messageHTTPRequest(h, "", tc.method, tc.path, `{"content":"hello"}`)
			if rec.Code != 401 || repo.called {
				t.Fatalf("status=%d called=%v", rec.Code, repo.called)
			}
		})
	}
}
func TestMessageHTTPSend(t *testing.T) {
	repo := &httpMessageRepo{}
	h, token := messageHTTPHandler(t, repo, nil)
	rec := messageHTTPRequest(h, token, "POST", httpMessagePath, `{"content":"hello","chat_id":"ignored"}`)
	var msg model.Message
	if err := json.Unmarshal(rec.Body.Bytes(), &msg); err != nil {
		t.Fatal(err)
	}
	if rec.Code != 201 || msg.Content != "hello" || msg.ID != 1 || msg.ChatID != httpMessageChat || msg.SenderID != httpMessageUser || repo.chat != httpMessageChat || repo.sender != httpMessageUser || repo.content != "hello" {
		t.Fatalf("status=%d body=%s repo=%+v", rec.Code, rec.Body, repo)
	}
}
func TestMessageHTTPHistory(t *testing.T) {
	for _, tc := range []struct {
		name, suffix string
		limit        int
		before       int64
	}{
		{"Default", "", 51, 0}, {"Explicit", "?limit=10", 11, 0}, {"Clamped", "?limit=101", 101, 0}, {"Cursor", "?limit=3&before_id=8", 4, 8},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &httpMessageRepo{}
			h, token := messageHTTPHandler(t, repo, nil)
			rec := messageHTTPRequest(h, token, "GET", httpMessagePath+tc.suffix, "")
			if rec.Code != 200 || repo.chat != httpMessageChat || repo.limit != tc.limit {
				t.Fatalf("status=%d body=%s repo=%+v", rec.Code, rec.Body, repo)
			}
			if tc.before == 0 && repo.before != nil || tc.before != 0 && (repo.before == nil || *repo.before != tc.before) {
				t.Fatalf("cursor=%v", repo.before)
			}
			var page model.MessagePage
			if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
				t.Fatal(err)
			}
			if page.Messages == nil || len(page.Messages) != 0 || page.NextCursor != nil {
				t.Fatalf("page=%+v", page)
			}
		})
	}
	t.Run("NextCursor", func(t *testing.T) {
		repo := &httpMessageRepo{messages: []model.Message{{ID: 10}, {ID: 9}, {ID: 8}, {ID: 7}}}
		h, token := messageHTTPHandler(t, repo, nil)
		rec := messageHTTPRequest(h, token, "GET", httpMessagePath+"?limit=3", "")
		var page model.MessagePage
		if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
			t.Fatal(err)
		}
		if rec.Code != 200 || len(page.Messages) != 3 || page.Messages[0].ID != 10 || page.NextCursor == nil || *page.NextCursor != 8 {
			t.Fatalf("status=%d page=%+v", rec.Code, page)
		}
	})
}
func TestMessageHTTPSearch(t *testing.T) {
	repo := &httpMessageRepo{messages: []model.Message{{ID: 5, Content: "hello"}}}
	h, token := messageHTTPHandler(t, repo, nil)
	rec := messageHTTPRequest(h, token, "GET", httpMessagePath+"/search?q=%20hello%20", "")
	var msgs []model.Message
	if err := json.Unmarshal(rec.Body.Bytes(), &msgs); err != nil {
		t.Fatal(err)
	}
	if rec.Code != 200 || len(msgs) != 1 || msgs[0].ID != 5 || repo.chat != httpMessageChat || repo.query != "hello" || repo.limit != 50 {
		t.Fatalf("status=%d body=%s repo=%+v", rec.Code, rec.Body, repo)
	}
}
func TestMessageHTTPInputErrors(t *testing.T) {
	for _, tc := range []struct{ name, method, path, body, code string }{
		{"BadJSON", "POST", httpMessagePath, "{", "invalid_request"},
		{"Empty", "POST", httpMessagePath, `{"content":" "}`, "empty_message"},
		{"TooLong", "POST", httpMessagePath, `{"content":"` + strings.Repeat("a", 4001) + `"}`, "message_too_long"},
		{"BadLimit", "GET", httpMessagePath + "?limit=abc", "", "invalid_limit"},
		{"ZeroLimit", "GET", httpMessagePath + "?limit=0", "", "invalid_limit"},
		{"NegativeLimit", "GET", httpMessagePath + "?limit=-1", "", "invalid_limit"},
		{"BadCursor", "GET", httpMessagePath + "?before_id=abc", "", "invalid_cursor"},
		{"ZeroCursor", "GET", httpMessagePath + "?before_id=0", "", "invalid_cursor"},
		{"OverflowCursor", "GET", httpMessagePath + "?before_id=9223372036854775808", "", "invalid_cursor"},
		{"EmptyQuery", "GET", httpMessagePath + "/search", "", "empty_search_query"},
		{"WhitespaceQuery", "GET", httpMessagePath + "/search?q=%20%20", "", "empty_search_query"},
		{"SendUUID", "POST", "/api/v1/chats/invalid/messages", `{"content":"hello"}`, "invalid_id"},
		{"HistoryUUID", "GET", "/api/v1/chats/invalid/messages", "", "invalid_id"},
		{"SearchUUID", "GET", "/api/v1/chats/invalid/messages/search?q=hello", "", "invalid_id"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &httpMessageRepo{}
			h, token := messageHTTPHandler(t, repo, nil)
			rec := messageHTTPRequest(h, token, tc.method, tc.path, tc.body)
			assertMessageHTTPError(t, rec, 400, tc.code)
			if repo.called {
				t.Fatal("invalid request reached message repository")
			}
		})
	}
}
func assertMessageHTTPError(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	var result struct {
		Error struct{ Code, Message string }
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if rec.Code != status || result.Error.Code != code {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body)
	}
	if status == 500 && result.Error.Message != "internal server error" {
		t.Fatalf("unsafe error: %s", rec.Body)
	}
}
func TestMessageHTTPAccessAndSafeErrors(t *testing.T) {
	failure := errors.New("secret database connection details")
	for _, op := range []struct{ method, path string }{{"POST", httpMessagePath}, {"GET", httpMessagePath}, {"GET", httpMessagePath + "/search?q=hello"}} {
		for _, tc := range []struct {
			name               string
			accessErr, repoErr error
			status             int
			code               string
			called             bool
		}{
			{"Outsider", repository.ErrChatNotFound, nil, 404, "chat_not_found", false},
			{"AccessFailure", failure, nil, 500, "internal_error", false},
			{"RepositoryFailure", nil, failure, 500, "internal_error", true},
		} {
			t.Run(op.method+op.path+tc.name, func(t *testing.T) {
				repo := &httpMessageRepo{err: tc.repoErr}
				h, token := messageHTTPHandler(t, repo, tc.accessErr)
				rec := messageHTTPRequest(h, token, op.method, op.path, `{"content":"hello"}`)
				assertMessageHTTPError(t, rec, tc.status, tc.code)
				if repo.called != tc.called {
					t.Fatalf("repository called=%v", repo.called)
				}
			})
		}
	}
}

func TestMessageHTTPRejectsTrailingSlash(t *testing.T) {
	for _, method := range []string{"GET", "POST"} {
		t.Run(method, func(t *testing.T) {
			repo := &httpMessageRepo{}
			h, token := messageHTTPHandler(t, repo, nil)
			rec := messageHTTPRequest(h, token, method, httpMessagePath+"/", `{"content":"hello"}`)
			if rec.Code != http.StatusNotFound || repo.called {
				t.Fatalf("status=%d called=%v", rec.Code, repo.called)
			}
		})
	}
}
