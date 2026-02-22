package websocket

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/coder/websocket"
	"github.com/johndosdos/chatter/internal/model"
)

// ReadMessage reads the incoming data from the websocket stream.
func (c *Client) ReadMessage(ctx context.Context) {
	defer func() {
		c.Hub.Unregister <- c
		c.conn.CloseNow()
	}()

	for {
		msgType, p, err := c.conn.Read(ctx)
		if err != nil {
			status := websocket.CloseStatus(err)
			if status != websocket.StatusNormalClosure &&
				status != websocket.StatusGoingAway &&
				status != -1 {
				log.Printf("%v", err)
			}
			return
		}

		log.Printf("received message type %v payload: %s", msgType, string(p))

		// The app only supports text format for now...
		if msgType != websocket.MessageText {
			continue
		}

		// We need to unmarshal the JSON sent from the client side. HTMX's ws-send
		// attribute also sends a HEADERS field along with the client message.
		//
		// Also, set CreatedAt to the current time.
		// Set message.Type to 'message' as default. Override as needed.
		var payload model.ChatMessage
		err = json.Unmarshal(p, &payload)
		if err != nil {
			log.Printf("failed to process payload from client: %v", err)
			continue
		}
		// Reassign user info after deserializing the payload. The payload could be hijacked during
		// transmission and we don't want to assign the incorrect info.
		payload.UserID = c.UserID
		payload.Username = c.Username
		payload.CreatedAt = time.Now().UTC()
		payload.Type = "message"

		// Check if the message is a typing indicator.
		// Typing rate limit
		if trigger, ok := payload.Headers["HX-Trigger"]; ok && trigger == "user-input" {
			payload.Type = "typing"

			if !c.typingLim.Allow() {
				continue
			}
		}

		// Message rate limit
		if payload.Type == "message" {
			limitWindow := 10 * time.Second // 10s penalty when burst sending 30 messages/min
			if !c.timeWarned.IsZero() && time.Since(c.timeWarned) < limitWindow {
				continue
			}

			if !c.messageLim.Allow() {
				c.timeWarned = time.Now()
				c.MessageCh <- model.ChatMessage{Type: "rateLimitMessage"}
				continue
			}
		}

		c.Hub.ClientMsg <- payload
	}
}
