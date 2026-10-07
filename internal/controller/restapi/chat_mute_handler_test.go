package restapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/undndnwnkk/go-vk-messenger/internal/model"
	"github.com/undndnwnkk/go-vk-messenger/internal/realtime"
	"github.com/undndnwnkk/go-vk-messenger/internal/repository"
)

const mutePath = "/api/v1/chats/" + httpMessageChat + "/mute"

func (r *httpChatRepo) SetMute(_ context.Context, chat, user string, muted bool) (*model.ChatMuteState, bool, error) {
	r.chatID, r.caller, r.muteCalled = chat, user, true
	if r.err != nil {
		return nil, false, r.err
	}
	changed := r.muted != muted
	if changed {
		r.muted = muted
		r.muteVersion++
	}
	return &model.ChatMuteState{ChatID: chat, Muted: r.muted, Version: r.muteVersion}, changed, nil
}

func TestChatMuteHTTP(t *testing.T) {
	for _, tc := range []struct {
		name, body, code string
		status           int
		noToken          bool
		err              error
	}{
		{name: "Mute", body: `{"muted":true,"user_id":"` + wsMessageBob + `"}`, status: 200},
		{name: "Unmute", body: `{"muted":false}`, status: 200},
		{name: "NoJWT", body: `{"muted":true}`, status: 401, noToken: true},
		{name: "Missing", body: `{}`, status: 400, code: "invalid_request"},
		{name: "Null", body: `{"muted":null}`, status: 400, code: "invalid_request"},
		{name: "String", body: `{"muted":"false"}`, status: 400, code: "invalid_request"},
		{name: "Malformed", body: `{`, status: 400, code: "invalid_request"},
		{name: "NonMember", body: `{"muted":true}`, status: 404, code: "chat_not_found", err: repository.ErrMemberNotFound},
		{name: "DatabaseError", body: `{"muted":true}`, status: 500, code: "internal_error", err: errors.New("secret database info")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &httpChatRepo{err: tc.err, muted: tc.name == "Unmute"}
			h, token := chatTestHandler(t, repo)
			if tc.noToken {
				token = ""
			}
			rec := messageHTTPRequest(h, token, "PATCH", mutePath, tc.body)
			if rec.Code != tc.status {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body)
			}
			if tc.code != "" {
				assertMessageHTTPError(t, rec, tc.status, tc.code)
			}
			if tc.status == 200 {
				var state model.ChatMuteState
				if err := json.Unmarshal(rec.Body.Bytes(), &state); err != nil {
					t.Fatal(err)
				}
				if state.Muted != (tc.name == "Mute") || state.Version != 1 || repo.caller != httpMessageUser {
					t.Fatalf("state=%+v repo=%+v", state, repo)
				}
			}
			if (tc.status == 400 || tc.status == 401) && repo.muteCalled {
				t.Fatal("invalid request reached repo")
			}
		})
	}
}

func TestChatMutePrivateRealtimeAndMessageDelivery(t *testing.T) {
	chat := &httpChatRepo{members: []model.ChatMember{{UserID: httpMessageUser}, {UserID: wsMessageBob}}}
	hub := realtime.NewHub()
	h, jwt := webSocketMessageHandlerWithChat(t, &httpMessageRepo{}, chat, hub)
	server := httptest.NewServer(h)
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	token := generateWebSocketToken(t, jwt, httpMessageUser)
	alice1 := dialWebSocket(t, ctx, server.URL, token)
	defer alice1.CloseNow()
	alice2 := dialWebSocket(t, ctx, server.URL, token)
	defer alice2.CloseNow()
	bob := dialWebSocket(t, ctx, server.URL, generateWebSocketToken(t, jwt, wsMessageBob))
	defer bob.CloseNow()
	carol := dialWebSocket(t, ctx, server.URL, generateWebSocketToken(t, jwt, wsMessageCarol))
	defer carol.CloseNow()
	waitForConnectionCount(t, ctx, hub, httpMessageUser, 2)
	waitForConnectionCount(t, ctx, hub, wsMessageBob, 1)
	waitForConnectionCount(t, ctx, hub, wsMessageCarol, 1)
	readMute := func(conn *websocket.Conn, muted bool, version int64) {
		t.Helper()
		_, data, err := conn.Read(ctx)
		if err != nil {
			t.Fatal(err)
		}
		var e struct {
			Type string
			Data model.ChatMuteState
		}
		if err := json.Unmarshal(data, &e); err != nil {
			t.Fatal(err)
		}
		if e.Type != "chat_mute_updated" || e.Data.ChatID != httpMessageChat || e.Data.Muted != muted || e.Data.Version != version {
			t.Fatalf("event=%s", data)
		}
	}
	rec := messageHTTPRequest(h, token, "PATCH", mutePath, `{"muted":true}`)
	if rec.Code != 200 {
		t.Fatalf("mute=%s", rec.Body)
	}
	readMute(alice1, true, 1)
	readMute(alice2, true, 1)
	// Duplicate mute must not enqueue another settings event.
	rec = messageHTTPRequest(h, token, "PATCH", mutePath, `{"muted":true}`)
	if rec.Code != 200 || chat.muteVersion != 1 {
		t.Fatalf("repeat mute=%s", rec.Body)
	}
	rec = messageHTTPRequest(h, token, "GET", "/api/v1/chats/"+httpMessageChat, "")
	var got model.Chat
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if !got.Muted || got.MuteVersion != 1 {
		t.Fatalf("chat=%+v", got)
	}
	rec = messageHTTPRequest(h, generateWebSocketToken(t, jwt, wsMessageBob), "POST", httpMessagePath, `{"content":"still delivered"}`)
	if rec.Code != 201 {
		t.Fatalf("send=%s", rec.Body)
	}
	for _, c := range []*websocket.Conn{alice1, alice2, bob} {
		msg := readMessageCreated(t, ctx, c)
		if msg.Content != "still delivered" {
			t.Fatalf("message=%+v", msg)
		}
	}
	rec = messageHTTPRequest(h, token, "PATCH", mutePath, `{"muted":false}`)
	if rec.Code != 200 {
		t.Fatalf("unmute=%s", rec.Body)
	}
	readMute(alice1, false, 2)
	readMute(alice2, false, 2)
	assertNoWebSocketMessage(t, bob)
	assertNoWebSocketMessage(t, carol)
}
