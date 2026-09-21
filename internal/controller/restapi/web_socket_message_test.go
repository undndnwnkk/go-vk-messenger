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

	"github.com/coder/websocket"
	"github.com/undndnwnkk/go-vk-messenger/internal/model"
	"github.com/undndnwnkk/go-vk-messenger/internal/realtime"
	"github.com/undndnwnkk/go-vk-messenger/internal/repository"
	"github.com/undndnwnkk/go-vk-messenger/internal/service"
)

const wsMessageBob = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
const wsMessageCarol = "dddddddd-dddd-4ddd-8ddd-dddddddddddd"

func TestWebSocketSendMessageAckAndSpoofing(t *testing.T) {
	repo := &httpMessageRepo{}
	handler, token := webSocketMessageHandler(t, repo, nil, httpMessageUser)
	server := httptest.NewServer(handler)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn := dialWebSocket(t, ctx, server.URL, token)
	defer conn.CloseNow()

	writeTextMessage(t, ctx, conn, `{
		"type":"send_message",
		"data":{
			"chat_id":"`+httpMessageChat+`",
			"content":"hello",
			"sender_id":"bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
		}
	}`)

	ack := readMessageAck(t, ctx, conn)
	if repo.chat != httpMessageChat || repo.sender != httpMessageUser || repo.content != "hello" {
		t.Fatalf("message repo got chat=%q sender=%q content=%q", repo.chat, repo.sender, repo.content)
	}
	if ack.ID != 1 || ack.ChatID != httpMessageChat || ack.SenderID != httpMessageUser || ack.Content != "hello" {
		t.Fatalf("ack data = %+v", ack)
	}
}

func TestWebSocketSendMessageBroadcastsMessageCreatedToOnlineChatMembers(t *testing.T) {
	repo := &httpMessageRepo{}
	chatRepo := &httpChatRepo{members: []model.ChatMember{
		{ChatID: httpMessageChat, UserID: httpMessageUser},
		{ChatID: httpMessageChat, UserID: wsMessageBob},
	}}
	hub := realtime.NewHub()
	handler, jwt := webSocketMessageHandlerWithChat(t, repo, chatRepo, hub)
	server := httptest.NewServer(handler)
	defer server.Close()

	aliceToken := generateWebSocketToken(t, jwt, httpMessageUser)
	bobToken := generateWebSocketToken(t, jwt, wsMessageBob)
	carolToken := generateWebSocketToken(t, jwt, wsMessageCarol)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	alice := dialWebSocket(t, ctx, server.URL, aliceToken)
	defer alice.CloseNow()
	bob1 := dialWebSocket(t, ctx, server.URL, bobToken)
	defer bob1.CloseNow()
	bob2 := dialWebSocket(t, ctx, server.URL, bobToken)
	defer bob2.CloseNow()
	carol := dialWebSocket(t, ctx, server.URL, carolToken)
	defer carol.CloseNow()

	waitForConnectionCount(t, ctx, hub, httpMessageUser, 1)
	waitForConnectionCount(t, ctx, hub, wsMessageBob, 2)
	waitForConnectionCount(t, ctx, hub, wsMessageCarol, 1)

	writeTextMessage(t, ctx, alice, `{
		"type":"send_message",
		"data":{
			"chat_id":"`+httpMessageChat+`",
			"content":"hello"
		}
	}`)

	ack := readMessageAck(t, ctx, alice)
	if ack.ID != 1 || ack.SenderID != httpMessageUser || ack.Content != "hello" {
		t.Fatalf("ack data = %+v", ack)
	}
	for _, conn := range []*websocket.Conn{bob1, bob2} {
		created := readMessageCreated(t, ctx, conn)
		if created.ID != ack.ID || created.ChatID != httpMessageChat || created.SenderID != httpMessageUser || created.Content != "hello" {
			t.Fatalf("message_created data = %+v, ack = %+v", created, ack)
		}
	}
	assertNoWebSocketMessage(t, carol)
}

func TestWebSocketSendMessageInvalidPayloadKeepsConnectionAlive(t *testing.T) {
	repo := &httpMessageRepo{}
	handler, token := webSocketMessageHandler(t, repo, nil, httpMessageUser)
	server := httptest.NewServer(handler)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn := dialWebSocket(t, ctx, server.URL, token)
	defer conn.CloseNow()

	writeTextMessage(t, ctx, conn, `{"type":"send_message","data":"lol"}`)
	assertWebSocketEvent(t, ctx, conn, "error", "invalid_payload")
	if repo.called {
		t.Fatal("invalid payload reached message repository")
	}

	writeTextMessage(t, ctx, conn, `{"type":"ping"}`)
	assertWebSocketEvent(t, ctx, conn, "pong", "")
}

func TestWebSocketSendMessageRepositoryErrorDoesNotBroadcast(t *testing.T) {
	repo := &httpMessageRepo{err: errors.New("secret database connection details")}
	chatRepo := &httpChatRepo{members: []model.ChatMember{
		{ChatID: httpMessageChat, UserID: httpMessageUser},
		{ChatID: httpMessageChat, UserID: wsMessageBob},
	}}
	hub := realtime.NewHub()
	handler, jwt := webSocketMessageHandlerWithChat(t, repo, chatRepo, hub)
	server := httptest.NewServer(handler)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	alice := dialWebSocket(t, ctx, server.URL, generateWebSocketToken(t, jwt, httpMessageUser))
	defer alice.CloseNow()
	bob := dialWebSocket(t, ctx, server.URL, generateWebSocketToken(t, jwt, wsMessageBob))
	defer bob.CloseNow()

	waitForConnectionCount(t, ctx, hub, httpMessageUser, 1)
	waitForConnectionCount(t, ctx, hub, wsMessageBob, 1)

	writeTextMessage(t, ctx, alice, `{
		"type":"send_message",
		"data":{
			"chat_id":"`+httpMessageChat+`",
			"content":"hello"
		}
	}`)

	assertWebSocketEvent(t, ctx, alice, "error", "internal_error")
	if !repo.called {
		t.Fatal("message repository was not called")
	}
	assertNoWebSocketMessage(t, bob)
}

func TestWebSocketSendMessageChatNotFoundDoesNotCreateMessage(t *testing.T) {
	repo := &httpMessageRepo{}
	handler, token := webSocketMessageHandler(t, repo, repository.ErrChatNotFound, httpMessageUser)
	server := httptest.NewServer(handler)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn := dialWebSocket(t, ctx, server.URL, token)
	defer conn.CloseNow()

	writeTextMessage(t, ctx, conn, `{
		"type":"send_message",
		"data":{
			"chat_id":"`+httpMessageChat+`",
			"content":"hello"
		}
	}`)

	assertWebSocketEvent(t, ctx, conn, "error", "chat_not_found")
	if repo.called {
		t.Fatal("message repository was called for inaccessible chat")
	}

	writeTextMessage(t, ctx, conn, `{"type":"ping"}`)
	assertWebSocketEvent(t, ctx, conn, "pong", "")
}

func webSocketMessageHandler(t *testing.T, repo *httpMessageRepo, accessErr error, userID string) (http.Handler, string) {
	t.Helper()

	jwt := service.NewJWTService("ws-message-test", time.Minute)
	user := service.NewUserService(newFakeUserRepo(), jwt)
	chat := service.NewChatService(&httpChatRepo{err: accessErr}, *user)
	message := service.NewMessageService(repo, *chat)
	token, err := jwt.GenerateToken(userID)
	if err != nil {
		t.Fatal(err)
	}

	return NewHandler(*user, jwt, fakeHealthChecker{}, *chat, *message, realtime.NewHub()), token
}

func webSocketMessageHandlerWithChat(t *testing.T, repo *httpMessageRepo, chatRepo *httpChatRepo, hub *realtime.Hub) (http.Handler, *service.JWTService) {
	t.Helper()

	jwt := service.NewJWTService("ws-message-test", time.Minute)
	user := service.NewUserService(newFakeUserRepo(), jwt)
	chat := service.NewChatService(chatRepo, *user)
	message := service.NewMessageService(repo, *chat)

	return NewHandler(*user, jwt, fakeHealthChecker{}, *chat, *message, hub), jwt
}

func generateWebSocketToken(t *testing.T, jwt *service.JWTService, userID string) string {
	t.Helper()

	token, err := jwt.GenerateToken(userID)
	if err != nil {
		t.Fatal(err)
	}

	return token
}

func dialWebSocket(t *testing.T, ctx context.Context, serverURL, token string) *websocket.Conn {
	t.Helper()

	conn, resp, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(serverURL, "http")+"/api/v1/ws", &websocket.DialOptions{
		HTTPHeader: http.Header{"Authorization": {"Bearer " + token}},
	})
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	if resp.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("status = %d, want 101", resp.StatusCode)
	}

	return conn
}

func readMessageAck(t *testing.T, ctx context.Context, conn *websocket.Conn) model.Message {
	t.Helper()

	return readMessageEvent(t, ctx, conn, realtime.EventMessageAck)
}

func readMessageCreated(t *testing.T, ctx context.Context, conn *websocket.Conn) model.Message {
	t.Helper()

	return readMessageEvent(t, ctx, conn, realtime.EventMessageCreated)
}

func readMessageEvent(t *testing.T, ctx context.Context, conn *websocket.Conn, eventType realtime.EventType) model.Message {
	t.Helper()

	msgType, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read %s: %v", eventType, err)
	}
	if msgType != websocket.MessageText {
		t.Fatalf("message type = %v, want %v", msgType, websocket.MessageText)
	}

	var event struct {
		Type string        `json:"type"`
		Data model.Message `json:"data"`
	}
	if err := json.Unmarshal(data, &event); err != nil {
		t.Fatalf("decode %s %q: %v", eventType, data, err)
	}
	if event.Type != string(eventType) {
		t.Fatalf("event type = %q, want %q; body: %s", event.Type, eventType, data)
	}

	return event.Data
}

func assertNoWebSocketMessage(t *testing.T, conn *websocket.Conn) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, data, err := conn.Read(ctx)
	if err == nil {
		t.Fatalf("unexpected websocket message: %s", data)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("read unexpected error = %v, want deadline exceeded", err)
	}
}
