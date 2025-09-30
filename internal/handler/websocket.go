package handler

import (
	"context"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/johndosdos/chatter/internal/chat"
	"github.com/johndosdos/chatter/internal/database"
	ws "github.com/johndosdos/chatter/internal/websocket"
)

func ServeWs(ctx context.Context, h *ws.Hub, db *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("[error] failed to upgrade connection to WebSocket: %v", err)
			return
		}

		username := r.URL.Query().Get("username")
		userid, _ := uuid.Parse(r.URL.Query().Get("userid"))

		// We'll register our new client to the central hub.
		c := ws.NewClient(conn)
		c.Username = username
		c.Userid = userid
		h.Register <- c

		// Ok is a signalling channel from our hub, indicating if register was
		// successful.
		<-h.Ok

		// Run these goroutines to listen and process messages from other
		// clients.
		go c.WriteMessage()
		go c.ReadMessage()

		// Load recent chat history to current client. We must initialize
		// WriteMessage() and ReadMessage().
		go chat.DbLoadChatHistory(ctx, c.Recv, db)
	}
}
