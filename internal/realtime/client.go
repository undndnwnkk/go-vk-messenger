package realtime

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	"github.com/coder/websocket"
)

const sendBufferSize = 20

type Client struct {
	userID    string
	conn      *websocket.Conn
	send      chan []byte
	closeOnce sync.Once
}

func NewClient(userID string, conn *websocket.Conn) *Client {
	return &Client{userID: userID, conn: conn, send: make(chan []byte, sendBufferSize)}
}

func (c *Client) ReadLoop(ctx context.Context, handle func(Event)) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			msgType, data, err := c.conn.Read(ctx)
			if err != nil {
				if websocket.CloseStatus(err) != websocket.StatusNormalClosure &&
					websocket.CloseStatus(err) != websocket.StatusGoingAway {
					log.Printf("websocket read err: %v", err)
				}
				return err
			}

			if msgType != websocket.MessageText {
				c.Send(NewErrorEvent("invalid_event", "invalid websocket event"))
				continue
			}

			var event Event
			if err := json.Unmarshal(data, &event); err != nil {
				c.Send(NewErrorEvent("invalid_event", "invalid websocket event"))
				continue
			}

			handle(event)
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

func (c *Client) Close(status websocket.StatusCode, reason string) {
	c.closeOnce.Do(func() {
		if c.conn != nil {
			go func() {
				_ = c.conn.Close(status, reason)
			}()
		}
	})
}
