package ws

import (
	"context"
	"time"

	"github.com/coder/websocket"
)

type Client struct {
	Conn *websocket.Conn
	Send chan []byte
	Hub  *Hub
}

func NewClient(conn *websocket.Conn, hub *Hub) *Client {
	return &Client{
		Conn: conn,
		Send: make(chan []byte, 64),
		Hub:  hub,
	}
}

func (c *Client) WritePump() {
	defer func() {
		c.Conn.Close(websocket.StatusNormalClosure, "write pump done")
	}()

	for {
		msg, ok := <-c.Send
		if !ok {
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		err := c.Conn.Write(ctx, websocket.MessageText, msg)
		cancel()

		if err != nil {
			return
		}
	}
}