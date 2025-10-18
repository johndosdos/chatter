package handler

import (
	"context"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgtype"

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

		userid, _ := uuid.Parse(r.URL.Query().Get("userid"))

		user, _ := db.GetUserById(ctx, pgtype.UUID{Bytes: userid, Valid: true})

		// We'll register our new client to the central hub.
		c := ws.NewClient(conn)
		c.Userid = user.UserID.Bytes
		c.Username = user.Username

		h.Register <- c
		// Ok is a signalling channel from our hub, indicating if register was
		// successful.
		<-h.Ok

		// Run these goroutines to listen and process messages from other
		// clients.
		go c.WriteMessage()
		go c.ReadMessage()
	}
}
