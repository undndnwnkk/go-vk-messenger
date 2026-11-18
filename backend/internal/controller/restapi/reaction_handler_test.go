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

func (r *httpMessageRepo) SetReaction(_ context.Context, chat, user string, id int64, reaction string, add bool) (*model.MessageReactions, bool, error) {
	r.called, r.chat, r.sender, r.reactionID, r.reactionName, r.reactionAdd = true, chat, user, id, reaction, add
	return &model.MessageReactions{ChatID: chat, MessageID: id, Version: 1, Reactions: []model.ReactionSummary{}}, r.reactionChanged, r.err
}

func TestReactionHTTP(t *testing.T) {
	for _, method := range []string{"PUT", "DELETE"} {
		for _, tc := range []struct {
			name, suffix, code string
			status             int
			noToken            bool
			accessErr, repoErr error
		}{
			{name: "Success", suffix: "/42/reactions/like", status: 200},
			{name: "NoJWT", suffix: "/42/reactions/like", status: 401, noToken: true},
			{name: "MalformedID", suffix: "/abc/reactions/like", status: 400, code: "invalid_message_id"},
			{name: "NegativeID", suffix: "/-1/reactions/like", status: 400, code: "invalid_message_id"},
			{name: "OverflowID", suffix: "/9223372036854775808/reactions/like", status: 400, code: "invalid_message_id"},
			{name: "InvalidReaction", suffix: "/42/reactions/unknown", status: 400, code: "invalid_reaction"},
			{name: "Outsider", suffix: "/42/reactions/like", status: 404, code: "chat_not_found", accessErr: repository.ErrChatNotFound},
			{name: "MissingMessage", suffix: "/42/reactions/like", status: 404, code: "message_not_found", repoErr: repository.ErrMessageNotFound},
			{name: "DatabaseError", suffix: "/42/reactions/like", status: 500, code: "internal_error", repoErr: errors.New("secret DB info")},
		} {
			t.Run(method+tc.name, func(t *testing.T) {
				repo := &httpMessageRepo{err: tc.repoErr}
				h, token := messageHTTPHandler(t, repo, tc.accessErr)
				if tc.noToken {
					token = ""
				}
				rec := messageHTTPRequest(h, token, method, httpMessagePath+tc.suffix, `{"user_id":"`+wsMessageBob+`"}`)
				if rec.Code != tc.status {
					t.Fatalf("status=%d body=%s", rec.Code, rec.Body)
				}
				if tc.code != "" {
					assertMessageHTTPError(t, rec, tc.status, tc.code)
				}
				if tc.status == 200 && (repo.sender != httpMessageUser || repo.reactionAdd != (method == "PUT") || repo.reactionID != 42) {
					t.Fatalf("repo=%+v", repo)
				}
				if (tc.status == 400 || tc.status == 401 || tc.accessErr != nil) && repo.called {
					t.Fatal("invalid request reached repository")
				}
			})
		}
	}
}

func TestReactionsRealtime(t *testing.T) {
	for _, tc := range []struct {
		name               string
		changed            bool
		repoErr, notifyErr error
		status             int
	}{
		{name: "Changed", changed: true, status: 200},
		{name: "Idempotent", status: 200},
		{name: "DatabaseError", changed: true, repoErr: errors.New("write failure"), status: 500},
		{name: "NotifierError", changed: true, notifyErr: errors.New("notifier failure"), status: 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &httpMessageRepo{reactionChanged: tc.changed, err: tc.repoErr}
			chat := &httpChatRepo{members: []model.ChatMember{{UserID: httpMessageUser}, {UserID: wsMessageBob}}, listMembersErr: tc.notifyErr}
			hub := realtime.NewHub()
			h, jwt := webSocketMessageHandlerWithChat(t, repo, chat, hub)
			server := httptest.NewServer(h)
			defer server.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			var clients []*websocket.Conn
			for _, user := range []string{httpMessageUser, wsMessageBob, wsMessageBob, wsMessageCarol} {
				c := dialWebSocket(t, ctx, server.URL, generateWebSocketToken(t, jwt, user))
				defer c.CloseNow()
				clients = append(clients, c)
			}
			waitForConnectionCount(t, ctx, hub, httpMessageUser, 1)
			waitForConnectionCount(t, ctx, hub, wsMessageBob, 2)
			waitForConnectionCount(t, ctx, hub, wsMessageCarol, 1)
			rec := messageHTTPRequest(h, generateWebSocketToken(t, jwt, httpMessageUser), "PUT", httpMessagePath+"/42/reactions/like", "")
			if rec.Code != tc.status {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body)
			}
			for i, c := range clients {
				if i == 3 || !tc.changed || tc.repoErr != nil || tc.notifyErr != nil {
					assertNoWebSocketMessage(t, c)
					continue
				}
				_, data, err := c.Read(ctx)
				if err != nil {
					t.Fatal(err)
				}
				var e struct {
					Type string
					Data model.MessageReactions
				}
				if err := json.Unmarshal(data, &e); err != nil {
					t.Fatal(err)
				}
				if e.Type != "reactions_updated" || e.Data.ChatID != httpMessageChat || e.Data.MessageID != 42 || e.Data.Version != 1 || e.Data.Reactions == nil {
					t.Fatalf("event=%s", data)
				}
			}
		})
	}
}
