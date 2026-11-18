package restapi

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/undndnwnkk/go-vk-messenger/internal/model"
	"github.com/undndnwnkk/go-vk-messenger/internal/realtime"
	"github.com/undndnwnkk/go-vk-messenger/internal/repository"
)

func TestReplySendAndBroadcast(t *testing.T) {
	for _, transport := range []string{"REST", "WebSocket"} {
		t.Run(transport, func(t *testing.T) {
			preview := &model.MessageReply{ID: 42, SenderID: wsMessageBob, Content: "original text"}
			repo := &httpMessageRepo{reply: preview}
			chatRepo := &httpChatRepo{members: []model.ChatMember{{UserID: httpMessageUser}, {UserID: wsMessageBob}}}
			hub := realtime.NewHub()
			handler, jwt := webSocketMessageHandlerWithChat(t, repo, chatRepo, hub)
			server := httptest.NewServer(handler)
			defer server.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			token := generateWebSocketToken(t, jwt, httpMessageUser)
			alice := dialWebSocket(t, ctx, server.URL, token)
			defer alice.CloseNow()
			bob := dialWebSocket(t, ctx, server.URL, generateWebSocketToken(t, jwt, wsMessageBob))
			defer bob.CloseNow()
			carol := dialWebSocket(t, ctx, server.URL, generateWebSocketToken(t, jwt, wsMessageCarol))
			defer carol.CloseNow()
			waitForConnectionCount(t, ctx, hub, httpMessageUser, 1)
			waitForConnectionCount(t, ctx, hub, wsMessageBob, 1)
			waitForConnectionCount(t, ctx, hub, wsMessageCarol, 1)
			// The parent preview and sender must come from the server, not the payload.
			body := `{"content":"answer","reply_to_message_id":42,"reply_to":{"content":"forged"},"sender_id":"` + wsMessageCarol + `","chat_id":"` + httpMessageChat + `"}`
			var result model.Message
			if transport == "REST" {
				rec := messageHTTPRequest(handler, token, "POST", httpMessagePath, body)
				if rec.Code != 201 {
					t.Fatalf("status=%d body=%s", rec.Code, rec.Body)
				}
				if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
					t.Fatal(err)
				}
			} else {
				writeTextMessage(t, ctx, alice, `{"type":"send_message","data":`+body+`}`)
				result = readMessageAck(t, ctx, alice)
			}
			if result.ReplyToMessageID == nil || *result.ReplyToMessageID != 42 || !reflect.DeepEqual(result.ReplyTo, preview) || result.SenderID != httpMessageUser || result.ChatID != httpMessageChat {
				t.Fatalf("reply=%+v preview=%+v", result, result.ReplyTo)
			}
			if got := readMessageCreated(t, ctx, alice); !reflect.DeepEqual(got, result) {
				t.Fatalf("author event=%+v", got)
			}
			if got := readMessageCreated(t, ctx, bob); !reflect.DeepEqual(got, result) {
				t.Fatalf("recipient event=%+v", got)
			}
			assertNoWebSocketMessage(t, carol)
		})
	}
}

func TestReplyErrorsDoNotBroadcast(t *testing.T) {
	for _, transport := range []string{"REST", "WebSocket"} {
		for _, tc := range []struct {
			name, value, code  string
			status             int
			accessErr, repoErr error
			called             bool
		}{
			{name: "Zero", value: "0", status: 400, code: "invalid_message_id"},
			{name: "Negative", value: "-1", status: 400, code: "invalid_message_id"},
			{name: "String", value: `"42"`, status: 400, code: "invalid_request"},
			{name: "Fraction", value: "1.5", status: 400, code: "invalid_request"},
			{name: "Overflow", value: "9223372036854775808", status: 400, code: "invalid_request"},
			{name: "MissingDeletedOrOtherChat", value: "42", repoErr: repository.ErrMessageNotFound, status: 404, code: "message_not_found", called: true},
			{name: "Outsider", value: "42", accessErr: repository.ErrChatNotFound, status: 404, code: "chat_not_found"},
		} {
			t.Run(transport+tc.name, func(t *testing.T) {
				repo := &httpMessageRepo{err: tc.repoErr}
				chatRepo := &httpChatRepo{err: tc.accessErr, members: []model.ChatMember{{UserID: wsMessageBob}}}
				hub := realtime.NewHub()
				handler, jwt := webSocketMessageHandlerWithChat(t, repo, chatRepo, hub)
				server := httptest.NewServer(handler)
				defer server.Close()
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				token := generateWebSocketToken(t, jwt, httpMessageUser)
				bob := dialWebSocket(t, ctx, server.URL, generateWebSocketToken(t, jwt, wsMessageBob))
				defer bob.CloseNow()
				waitForConnectionCount(t, ctx, hub, wsMessageBob, 1)
				body := `{"content":"answer","chat_id":"` + httpMessageChat + `","reply_to_message_id":` + tc.value + `}`
				if transport == "REST" {
					assertMessageHTTPError(t, messageHTTPRequest(handler, token, "POST", httpMessagePath, body), tc.status, tc.code)
				} else {
					alice := dialWebSocket(t, ctx, server.URL, token)
					defer alice.CloseNow()
					writeTextMessage(t, ctx, alice, `{"type":"send_message","data":`+body+`}`)
					code := tc.code
					if code == "invalid_request" {
						code = "invalid_payload"
					}
					assertWebSocketEvent(t, ctx, alice, "error", code)
					writeTextMessage(t, ctx, alice, `{"type":"ping"}`)
					assertWebSocketEvent(t, ctx, alice, "pong", "")
				}
				if repo.called != tc.called {
					t.Fatalf("repository called=%v", repo.called)
				}
				assertNoWebSocketMessage(t, bob)
			})
		}
	}
}
