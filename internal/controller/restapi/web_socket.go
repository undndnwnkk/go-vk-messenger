package restapi

import (
	"context"
	"log"
	"net/http"

	"github.com/coder/websocket"
	"github.com/undndnwnkk/go-vk-messenger/internal/realtime"
	"golang.org/x/sync/errgroup"
)

func (h *Handler) webSocketHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusInternalServerError, "id_not_found", "user id not found from context")
		return
	}

	conn, err := websocket.Accept(w, r, nil)
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
			default:
				client.Send(realtime.NewErrorEvent("unsupported_event", "unsupported event type"))
			}
		})
	})
	g.Go(func() error {
		return client.WriteLoop(ctx)
	})

	if err := g.Wait(); err != nil {
		log.Printf("read loop or write loop ended with error: %v", err)
	} else {
		log.Printf("all loops ended successfully")
	}
}
