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
		// attribute will also send a HEADERS field along with the client message.
		// Also, set CreatedAt to the current time.
		payload := model.ChatMessage{
			UserID:    c.UserID,
			Username:  c.Username,
			CreatedAt: time.Now().UTC(),
		}
		err = json.Unmarshal(p, &payload)
		if err != nil {
			log.Printf("failed to process payload from client: %v", err)
			continue
		}

		c.Hub.ClientMsg <- payload
	}
}
