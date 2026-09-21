package realtime

import (
	"context"
	"log"

	"github.com/coder/websocket"
)

const sendBufferSize = 20

type Client struct {
	userID string
	conn   *websocket.Conn
	send   chan []byte
}

func NewClient(userID string, conn *websocket.Conn) *Client {
	return &Client{userID: userID, conn: conn, send: make(chan []byte, sendBufferSize)}
}

func (c *Client) ReadLoop(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			_, _, err := c.conn.Read(ctx)
			if err != nil {
				if websocket.CloseStatus(err) == websocket.StatusNormalClosure ||
					websocket.CloseStatus(err) == websocket.StatusGoingAway {
					log.Printf("websocket closed normally")
				} else {
					log.Printf("websocket read err: %v", err)
				}
				return err
			}
		}

	}
}

func (c *Client) Send(data []byte) bool {
	select {
	case c.send <- data:
		return true
	default:
		return false
	}
}

func (c *Client) WriteLoop(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case data := <-c.send:
			if err := c.conn.Write(ctx, websocket.MessageText, data); err != nil {
				return err
			}
		}
	}
}
