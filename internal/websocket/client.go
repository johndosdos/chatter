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

func NewClient(conn *websocket.Conn) *Client {
	return &Client{
		conn: conn,
		Recv: make(chan chat.Message, 64),
	}
}

func (c *Client) WriteMessage() {
	// In order to group messages by sender, we need to reference the
	// previous message. We can achieve this by setting the current
	// message as the previous after processing.
	var prevMsg chat.Message
	for {
		for message := range c.Recv {
			// Invoke a new writer from the current connection.
			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				log.Printf("[error] %v", err)
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
		}
	}
}

func (c *Client) ReadMessage() {
	defer func() {
		c.Hub.Unregister <- c
		c.conn.Close()
	}()

	for {
		_, p, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				log.Printf("[error] %v", err)
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
			log.Printf("[error] failed to process payload from client: %v", err)
			break
		}

		c.Hub.accept <- message
	}
}
