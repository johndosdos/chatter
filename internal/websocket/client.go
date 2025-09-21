package websocket

import (
	"context"
	"encoding/json"
	"log"

	"github.com/a-h/templ"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	components "github.com/johndosdos/chat-app/components/chat"
	"github.com/johndosdos/chat-app/internal/chat"
)

type Client struct {
	userid   uuid.UUID
	username string
	conn     *websocket.Conn
	Hub      *Hub
	Recv     chan chat.Message
}

func NewClient(conn *websocket.Conn) *Client {
	return &Client{
		conn: conn,
		Recv: make(chan chat.Message),
	}
}

func (c *Client) WriteMessage() {
	for {
		message, ok := <-c.Recv
		if !ok {
			c.conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		}

		// Invoke a new writer from the current connection.
		w, err := c.conn.NextWriter(websocket.TextMessage)
		if err != nil {
			log.Printf("[error] %v", err)
			break
		}

		// Render message as sender or receiver.
		var content templ.Component
		if message.Username == c.username && message.Userid != c.userid {
			content = components.ReceiverBubble(message.Content, message.Username)
		} else {
			content = components.SenderBubble(message.Content, c.username)
		}
		content.Render(context.Background(), w)

		w.Close()
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
		message := chat.Message{
			Userid: c.userid,
		}
		err = json.Unmarshal(p, &message)
		if err != nil {
			log.Printf("[error] failed to process payload from client: %v", err)
			break
		}

		// Set client's username on the server using the payload sent from
		// client's browser.
		c.username = message.Username

		c.Hub.accept <- message
	}
}
