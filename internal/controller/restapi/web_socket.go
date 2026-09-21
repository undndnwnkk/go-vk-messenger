package restapi

import (
	"github.com/coder/websocket"
	"log"
	"net/http"
)

func (h *Handler) webSocketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		log.Printf("websocket accept err: %v", err)
		return
	}
	defer conn.CloseNow()

	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusInternalServerError, "id_not_found", "user id not found from context")
		return
	}

	ctx := r.Context()
	for {
		msgType, data, err := conn.Read(ctx)
		if err != nil {
			if websocket.CloseStatus(err) == websocket.StatusNormalClosure ||
				websocket.CloseStatus(err) == websocket.StatusGoingAway {
				log.Printf("websocket closed normally")
			} else {
				log.Printf("websocket read err: %v", err)
			}
			return
		}

		log.Printf("websocket received data: %s", data)

		if err := conn.Write(ctx, msgType, []byte(userID)); err != nil {
			log.Printf("websocket write err: %v", err)
			return
		}
	}
}
