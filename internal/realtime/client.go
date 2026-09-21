package realtime

import (
	"context"
	"github.com/coder/websocket"
	"log"
)

type Client struct {
	userID string
	conn   *websocket.Conn
}

func NewClient(userID string, conn *websocket.Conn) *Client {
	return &Client{userID: userID, conn: conn}
}

func (c *Client) ReadLoop(ctx context.Context) {
	for {
		_, _, err := c.conn.Read(ctx)
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
