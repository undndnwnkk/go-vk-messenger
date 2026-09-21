package restapi

import (
	"log"
	"net/http"

	"github.com/coder/websocket"
	"github.com/undndnwnkk/go-vk-messenger/internal/realtime"
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

	ctx := r.Context()

	client := realtime.NewClient(userID, conn)

	client.ReadLoop(ctx)
}
