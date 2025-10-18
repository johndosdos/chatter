package websocket

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/a-h/templ"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	components "github.com/johndosdos/chatter/components/chat"
	"github.com/johndosdos/chatter/internal/chat"
)

type Client struct {
	Userid   uuid.UUID
	Username string
	conn     *websocket.Conn
	Hub      *Hub
	Recv     chan chat.Message
}

const pongWait = 60 * time.Second

func NewClient(conn *websocket.Conn) *Client {
	return &Client{
		conn: conn,
		Recv: make(chan chat.Message, 64),
	}
}

func (c *Client) WriteMessage() {
	t := time.NewTicker((pongWait * 9) / 10)
	defer t.Stop()

	// In order to group messages by sender, we need to reference the
	// previous message. We can achieve this by setting the current
	// message as the previous after processing.
	var prevMsg chat.Message
	for {
		select {
		case message, ok := <-c.Recv:
			// Stop the process if the recv channel closed.
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// Invoke a new writer from the current connection.
			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				log.Printf("[write error] %v", err)
				break
			}

			// Check if current and previous messages have the same userid.
			sameUser := false
			if message.Userid == prevMsg.Userid {
				sameUser = true
			}

			// Render message as sender or receiver.
			var content templ.Component
			if message.Userid == c.Userid {
				content = components.SenderBubble(message.Username, message.Content, sameUser, message.CreatedAt)
			} else {
				content = components.ReceiverBubble(message.Username, message.Content, sameUser, message.CreatedAt)
			}
			content.Render(context.Background(), w)

			w.Close()

			prevMsg = message

		case <-t.C:
			err := c.conn.WriteMessage(websocket.PingMessage, nil)
			if err != nil {
				log.Printf("[write error] failed to send ping signal: %v", err)
				return
			}
		}
	}
}

func (c *Client) ReadMessage() {
	defer func() {
		c.Hub.Unregister <- c
		_ = c.conn.Close()
	}()

	// The default connection behavior is to wait indefinitely for incoming data.
	// Firewalls, proxies, and other services have their own system to invalidate
	// a stale connection. Therefore, we must keep the connection alive by sending
	// ping pong signals between the server and the client (to simulate network traffic)
	// within a set deadline.
	err := c.conn.SetReadDeadline(time.Now().UTC().Add(pongWait))
	if err != nil {
		log.Printf("[conn error] failed to set read deadline: %v", err)
		return
	}

	// Reset deadline after receiving pong signal.
	c.conn.SetPongHandler(func(appData string) error {
		return c.conn.SetReadDeadline(time.Now().UTC().Add(pongWait))
	})

	for {
		_, p, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				log.Printf("[conn error] %v", err)
			}
			break
		}

		// We need to unmarshal the JSON sent from the client side. HTMX's ws-send
		// attribute will also send a HEADERS field along with the client message.
		// Also, set CreatedAt to the current time.
		message := chat.Message{
			Userid:    c.Userid,
			Username:  c.Username,
			CreatedAt: time.Now().UTC(),
		}
		err = json.Unmarshal(p, &message)
		if err != nil {
			log.Printf("[server error] failed to process payload from client: %v", err)
			break
		}

		c.Hub.accept <- message
	}
}
