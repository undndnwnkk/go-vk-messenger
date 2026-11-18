package restapi

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/coder/websocket"
	"github.com/undndnwnkk/go-vk-messenger/internal/model"
	"github.com/undndnwnkk/go-vk-messenger/internal/realtime"
	"golang.org/x/sync/errgroup"
)

func (h *Handler) webSocketHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusInternalServerError, "id_not_found", "user id not found from context")
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		// Echo only the application subprotocol, never the bearer.<JWT> entry.
		Subprotocols:   []string{WebSocketSubprotocol},
		OriginPatterns: h.wsOriginPatterns,
	})
	if err != nil {
		log.Printf("websocket accept err: %v", err)
		return
	}
	defer conn.CloseNow()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	g, ctx := errgroup.WithContext(ctx)

	client := realtime.NewClient(userID, conn)
	h.hub.Register(client)
	defer h.hub.Unregister(client)

	g.Go(func() error {
		defer cancel()
		return client.ReadLoop(ctx, func(event realtime.Event) {
			switch event.Type {
			case realtime.EventPing:
				client.Send(realtime.MustJSON(realtime.Event{Type: realtime.EventPong}))
			case realtime.EventSendMessage:
				h.handleWebSocketSendMessage(ctx, client, userID, event)
			default:
				client.Send(realtime.NewErrorEvent("unsupported_event", "unsupported event type"))
			}
		})
	})
	g.Go(func() error {
		return client.WriteLoop(ctx)
	})

	if err := g.Wait(); err != nil && !isExpectedWebSocketClose(err) {
		log.Printf("read loop or write loop ended with error: %v", err)
	}
}

// webSocketOriginPatterns turns exact origins ("https://app.example.com") into
// websocket.AcceptOptions.OriginPatterns. Patterns containing "://" are matched
// against scheme://host with path.Match, so its metacharacters are escaped
// (IPv6 hosts contain "[" and "]").
func webSocketOriginPatterns(origins []string) []string {
	if len(origins) == 0 {
		return nil
	}
	escape := strings.NewReplacer(`\`, `\\`, `*`, `\*`, `?`, `\?`, `[`, `\[`, `]`, `\]`)
	patterns := make([]string, 0, len(origins))
	for _, origin := range origins {
		patterns = append(patterns, escape.Replace(origin))
	}
	return patterns
}

func (h *Handler) handleWebSocketSendMessage(ctx context.Context, client *realtime.Client, userID string, event realtime.Event) {
	var payload realtime.SendMessagePayload
	if err := json.Unmarshal(event.Data, &payload); err != nil {
		client.Send(realtime.NewErrorEvent("invalid_payload", "invalid websocket event payload"))
		return
	}

	msg, err := h.message.Send(ctx, userID, payload.ChatID, model.CreateMessageRequest{
		Content:          payload.Content,
		ReplyToMessageID: payload.ReplyToMessageID,
	})
	if err != nil {
		_, code, message := publicError(err)
		client.Send(realtime.NewErrorEvent(code, message))
		return
	}

	client.Send(realtime.MustEventJSON(realtime.EventMessageAck, msg))
	if err := h.notifier.NotifyMessageCreated(ctx, userID, msg); err != nil {
		log.Printf("websocket message created broadcast failed: %v", err)
	}
}

func isExpectedWebSocketClose(err error) bool {
	if errors.Is(err, context.Canceled) {
		return true
	}
	switch websocket.CloseStatus(err) {
	case websocket.StatusNormalClosure, websocket.StatusGoingAway, websocket.StatusPolicyViolation:
		return true
	default:
		return false
	}
}
