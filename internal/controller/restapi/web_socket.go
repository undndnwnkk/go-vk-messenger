package restapi

import (
	"log"
	"net/http"

	"github.com/coder/websocket"
)

func (h *Handler) webSocketHandler(w http.ResponseWriter, r *http.Request) {
	if _, ok := GetUserIDFromContext(r.Context()); !ok {
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
	for {
		_, _, err := conn.Read(ctx)
		if err != nil {
			if websocket.CloseStatus(err) == websocket.StatusNormalClosure ||
				websocket.CloseStatus(err) == websocket.StatusGoingAway {
				log.Printf("websocket closed normally")
			} else {
				log.Printf("websocket read err: %v", err)
			}
			return
		}
	}
}
