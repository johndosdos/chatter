package handler

import (
	"log"
	"net/http"

	"github.com/coder/websocket"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/johndosdos/chatter/internal/auth"
	"github.com/johndosdos/chatter/internal/database"
	ws "github.com/johndosdos/chatter/internal/websocket"
)

// ServeWs handles the client's websocket connection upgrade.
func ServeWs(h *ws.Hub, db *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			InsecureSkipVerify: true,
		})

		userID, err := auth.GetUserFromContext(ctx)
		if err != nil {
			log.Printf("%v", err)
			return
		}

		user, _ := db.GetUserById(ctx, pgtype.UUID{Bytes: userID, Valid: true})
		log.Printf("upgraded connection for user %s", user.Username)

		// We'll register our new client to the central hub.
		c := ws.NewClient(conn, user.UserID.Bytes, user.Username)
		reg := ws.Registration{
			Client: c,
			Done:   make(chan struct{}),
		}

		h.Register <- reg

		// Wait for registration to complete
		<-reg.Done

		// Run these goroutines to listen and process messages from other
		// clients.
		//
		// We block on c.ReadMessage() because the request context will be canceled as soon
		// we return from the ServeWs() handler.
		go c.WriteMessage(ctx)
		c.ReadMessage(ctx)
	}
}
