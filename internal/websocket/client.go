package websocket

import (
	"context"
	"log"
	"strconv"
	"time"

	"github.com/a-h/templ"
	"github.com/coder/websocket"
	"github.com/google/uuid"
	"github.com/johndosdos/chatter/components/chat"
	components "github.com/johndosdos/chatter/components/chat"
	"github.com/johndosdos/chatter/internal/model"
)

type Client struct {
	UserID    uuid.UUID
	Username  string
	conn      *websocket.Conn
	Hub       *Hub
	MessageCh chan model.ChatMessage
}

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
		case payload, ok := <-c.MessageCh:
			// We don't want to continue processing when the channel has already been
			// closed.
			if !ok {
				c.conn.Close(websocket.StatusNormalClosure, "channel closed")
				return
			}

			sameUser := payload.UserID == prevMsg.UserID

			var content templ.Component
			switch payload.Type {
			case "typing":
				if payload.UserID == c.UserID {
					continue
				}
				content = chat.TypingIndicator(payload.Username)

			case "presenceCount":
				// We expect a string that contain the count of currently connected users.
				s, err := strconv.Atoi(payload.Content)
				if err != nil {
					log.Printf("failed to convert string to int: %+v", err)
					continue
				}
				content = chat.PresenceCount(s)

			case "message":
				if payload.UserID == c.UserID {
					content = components.SenderBubble(payload.Username, payload.Content, sameUser, payload.ID)
				} else {
					content = components.ReceiverBubble(payload.Username, payload.Content, sameUser, payload.ID)
				}
			}

			writeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			w, err := c.conn.Writer(writeCtx, websocket.MessageText)
			if err != nil {
				log.Printf("%+v", err)
				cancel()
				continue
			}

			if err := content.Render(context.Background(), w); err != nil {
				log.Printf("failed to render component: %v", err)
				w.Close()
				cancel()
				continue
			}

			w.Close()
			cancel()

			// Only update prevMsg for regular messages, not typing indicators.
			prevMsg = payload

		case <-ctx.Done():
			c.conn.Close(websocket.StatusGoingAway, "context cancelled")
			return
		}
	}
}
