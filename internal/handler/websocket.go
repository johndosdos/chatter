package handler

import (
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/johndosdos/chatter/internal/auth"
	"github.com/johndosdos/chatter/internal/database"
	ws "github.com/johndosdos/chatter/internal/websocket"
)

// ServeWs handles the client's websocket connection upgrade.
func ServeWs(h *ws.Hub, db *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		upgrader := websocket.Upgrader{
			CheckOrigin: func(_ *http.Request) bool { return true },
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("handler/websocket: failed to upgrade connection to WebSocket: %v", err)
			return
		}

		userID := ctx.Value(auth.UserIDKey).(uuid.UUID)

		user, _ := db.GetUserById(ctx, pgtype.UUID{Bytes: userID, Valid: true})

		// We'll register our new client to the central hub.
		c := ws.NewClient(conn)
		c.UserID = user.UserID.Bytes
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
