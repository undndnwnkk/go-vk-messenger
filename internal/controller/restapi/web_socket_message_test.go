package restapi

import (
	"context"
	"encoding/json"
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

	msgType, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read message ack: %v", err)
	}
	if msgType != websocket.MessageText {
		t.Fatalf("message type = %v, want %v", msgType, websocket.MessageText)
	}

	var event struct {
		Type string        `json:"type"`
		Data model.Message `json:"data"`
	}
	if err := json.Unmarshal(data, &event); err != nil {
		t.Fatalf("decode message ack %q: %v", data, err)
	}
	if event.Type != string(realtime.EventMessageAck) {
		t.Fatalf("event type = %q, want %q; body: %s", event.Type, realtime.EventMessageAck, data)
	}

	return event.Data
}
