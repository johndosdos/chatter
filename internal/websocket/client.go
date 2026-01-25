package websocket

import (
	"context"
	"log"
	"time"

	"github.com/a-h/templ"
	"github.com/coder/websocket"
	"github.com/google/uuid"
	components "github.com/johndosdos/chatter/components/chat"
	"github.com/johndosdos/chatter/internal/model"
)

// Client contains client connection information.
type Client struct {
	UserID    uuid.UUID
	Username  string
	conn      *websocket.Conn
	Hub       *Hub
	MessageCh chan model.ChatMessage
}

// NewClient returns a new instance of Client.
func NewClient(conn *websocket.Conn, userID uuid.UUID, username string) *Client {
	return &Client{
		conn:      conn,
		MessageCh: make(chan model.ChatMessage, 64),
		UserID:    userID,
		Username:  username,
	}
}

// WriteMessage writes and renders to the outgoing websocket stream.
func (c *Client) WriteMessage(ctx context.Context) {
	// In order to group messages by sender, we need to reference the
	// previous message. We can achieve this by setting the current
	// message as the previous after processing.
	var prevMsg model.ChatMessage
	for {
		select {
		case message, ok := <-c.MessageCh:
			// Stop the process if the recv channel closed.
			if !ok {
				c.conn.Close(websocket.StatusNormalClosure, "channel closed")
				return
			}

			writeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			w, err := c.conn.Writer(writeCtx, websocket.MessageText)
			if err != nil {
				cancel()
				log.Printf("%+v", err)
				return
			}

			// Check if current and previous messages have the same userid.
			sameUser := false
			if message.UserID == prevMsg.UserID {
				sameUser = true
			}

			// Render message as sender or receiver.
			var content templ.Component
			if message.UserID == c.UserID {
				content = components.SenderBubble(message.Username, message.Content, sameUser, message.ID)
			} else {
				content = components.ReceiverBubble(message.Username, message.Content, sameUser, message.ID)
			}
			if err := content.Render(context.Background(), w); err != nil {
				log.Printf("failed to render component: %v", err)
				cancel()
				return
			}

			if err = w.Close(); err != nil {
				log.Printf("failed to close websocket writer: %+v", err)
			}
			cancel()
			prevMsg = message

		case <-ctx.Done():
			c.conn.Close(websocket.StatusGoingAway, "context cancelled")
			return
		}
	}
}
