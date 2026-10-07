package restapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/undndnwnkk/go-vk-messenger/internal/model"
	"github.com/undndnwnkk/go-vk-messenger/internal/realtime"
	"github.com/undndnwnkk/go-vk-messenger/internal/repository"
)

func TestMessageHTTPMutationErrors(t *testing.T) {
	for _, method := range []string{http.MethodPatch, http.MethodDelete} {
		for _, tc := range []struct {
			name, id, body, code            string
			status                          int
			noToken                         bool
			accessErr, repoErr, mutationErr error
			message                         *model.Message
		}{
			{name: "NoJWT", id: "42", noToken: true, status: 401, code: "unauthorized"},
			{name: "BadID", id: "abc", status: 400, code: "invalid_message_id"},
			{name: "ZeroID", id: "0", status: 400, code: "invalid_message_id"},
			{name: "NegativeID", id: "-1", status: 400, code: "invalid_message_id"},
			{name: "OverflowID", id: "9223372036854775808", status: 400, code: "invalid_message_id"},
			{name: "Outsider", id: "42", accessErr: repository.ErrChatNotFound, status: 404, code: "chat_not_found"},
			{name: "Missing", id: "42", repoErr: repository.ErrMessageNotFound, status: 404, code: "message_not_found"},
			{name: "NotAuthor", id: "42", message: &model.Message{SenderID: wsMessageBob}, status: 403, code: "forbidden"},
			{name: "Deleted", id: "42", message: &model.Message{SenderID: httpMessageUser, Deleted: true}, status: 404, code: "message_not_found"},
			{name: "ReadFailure", id: "42", repoErr: errors.New("secret DB details"), status: 500, code: "internal_error"},
			{name: "WriteFailure", id: "42", mutationErr: errors.New("secret DB details"), status: 500, code: "internal_error"},
			{name: "ConcurrentDelete", id: "42", mutationErr: repository.ErrMessageNotFound, status: 404, code: "message_not_found"},
		} {
			t.Run(method+tc.name, func(t *testing.T) {
				repo := &httpMessageRepo{message: tc.message, err: tc.repoErr, mutationErr: tc.mutationErr}
				h, token := messageHTTPHandler(t, repo, tc.accessErr)
				if tc.noToken {
					token = ""
				}
				rec := messageHTTPRequest(h, token, method, httpMessagePath+"/"+tc.id, `{"content":"updated","sender_id":"`+httpMessageUser+`"}`)
				if tc.noToken {
					if rec.Code != 401 || repo.called {
						t.Fatalf("status=%d called=%v", rec.Code, repo.called)
					}
					return
				}
				assertMessageHTTPError(t, rec, tc.status, tc.code)
				if tc.mutationErr == nil && repo.mutated {
					t.Fatal("rejected request mutated a message")
				}
			})
		}
	}
	for _, tc := range []struct{ body, code string }{
		{"{", "invalid_request"}, {`{}`, "empty_message"}, {`{"content":" "}`, "empty_message"},
		{`{"content":"` + strings.Repeat("я", 4001) + `"}`, "message_too_long"},
	} {
		repo := &httpMessageRepo{}
		h, token := messageHTTPHandler(t, repo, nil)
		assertMessageHTTPError(t, messageHTTPRequest(h, token, http.MethodPatch, httpMessagePath+"/42", tc.body), 400, tc.code)
		if repo.called {
			t.Fatal("invalid content reached repository")
		}
	}
}

func TestMessageHTTPMutationsBroadcast(t *testing.T) {
	for _, method := range []string{http.MethodPatch, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			repo := &httpMessageRepo{}
			chatRepo := &httpChatRepo{members: []model.ChatMember{{UserID: httpMessageUser}, {UserID: wsMessageBob}}}
			hub := realtime.NewHub()
			handler, jwt := webSocketMessageHandlerWithChat(t, repo, chatRepo, hub)
			server := httptest.NewServer(handler)
			defer server.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			connections := make([]*websocket.Conn, 0, 3)
			for _, user := range []string{httpMessageUser, wsMessageBob, wsMessageBob} {
				conn := dialWebSocket(t, ctx, server.URL, generateWebSocketToken(t, jwt, user))
				defer conn.CloseNow()
				connections = append(connections, conn)
			}
			carol := dialWebSocket(t, ctx, server.URL, generateWebSocketToken(t, jwt, wsMessageCarol))
			defer carol.CloseNow()
			waitForConnectionCount(t, ctx, hub, httpMessageUser, 1)
			waitForConnectionCount(t, ctx, hub, wsMessageBob, 2)
			waitForConnectionCount(t, ctx, hub, wsMessageCarol, 1)
			rec := messageHTTPRequest(handler, generateWebSocketToken(t, jwt, httpMessageUser), method, httpMessagePath+"/42", `{"content":"updated"}`)
			var msg model.Message
			if err := json.Unmarshal(rec.Body.Bytes(), &msg); err != nil {
				t.Fatal(err)
			}
			if rec.Code != 200 || msg.ID != 42 || msg.ChatID != httpMessageChat || msg.SenderID != httpMessageUser {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body)
			}
			eventType := realtime.EventMessageEdited
			if method == http.MethodPatch {
				if !msg.Edited || msg.EditedAt == nil || msg.Content != "updated" || msg.Deleted {
					t.Fatalf("edited=%+v", msg)
				}
			} else {
				eventType = realtime.EventMessageDeleted
				if !msg.Deleted || msg.DeletedAt == nil || msg.Content != "" {
					t.Fatalf("deleted=%+v", msg)
				}
			}
			for _, conn := range connections {
				if got := readMessageEvent(t, ctx, conn, eventType); !reflect.DeepEqual(got, msg) {
					t.Fatalf("event=%+v response=%+v", got, msg)
				}
			}
			assertNoWebSocketMessage(t, carol)
		})
	}
}

func TestMessageHTTPMutationFailureBroadcasts(t *testing.T) {
	for _, method := range []string{http.MethodPatch, http.MethodDelete} {
		for _, notifierFailure := range []bool{false, true} {
			t.Run(method+map[bool]string{false: "WriteFailure", true: "NotifierFailure"}[notifierFailure], func(t *testing.T) {
				repo := &httpMessageRepo{}
				chatRepo := &httpChatRepo{members: []model.ChatMember{{UserID: wsMessageBob}}}
				wantStatus := 500
				if notifierFailure {
					chatRepo.listMembersErr = errors.New("notifier failure")
					wantStatus = 200
				} else {
					repo.mutationErr = errors.New("write failure")
				}
				hub := realtime.NewHub()
				handler, jwt := webSocketMessageHandlerWithChat(t, repo, chatRepo, hub)
				server := httptest.NewServer(handler)
				defer server.Close()
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				bob := dialWebSocket(t, ctx, server.URL, generateWebSocketToken(t, jwt, wsMessageBob))
				defer bob.CloseNow()
				waitForConnectionCount(t, ctx, hub, wsMessageBob, 1)
				rec := messageHTTPRequest(handler, generateWebSocketToken(t, jwt, httpMessageUser), method, httpMessagePath+"/42", `{"content":"updated"}`)
				if rec.Code != wantStatus {
					t.Fatalf("status=%d body=%s", rec.Code, rec.Body)
				}
				assertNoWebSocketMessage(t, bob)
			})
		}
	}
}
