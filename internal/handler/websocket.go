package handler

import (
	"context"
	"log"
	"net/http"

	"github.com/a-h/templ"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgtype"

	components "github.com/johndosdos/chatter/components/chat"
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

		// Create a new user entity in the database.
		// If user creation in the database should fail, it doesn't make
		// sense if we proceed to hub registration.
		err = db.CreateUser(ctx, database.CreateUserParams{
			UserID:   pgtype.UUID{Bytes: [16]byte(c.Userid), Valid: true},
			Username: c.Username,
		})
		if err != nil {
			log.Printf("[DB error] failed to create user: %v", err)
			return
		}

		h.Register <- c
		// Ok is a signalling channel from our hub, indicating if register was
		// successful.
		<-h.Ok

		// Try to keep the connection alive.
		go chat.KeepaliveConn(conn)

		// Run these goroutines to listen and process messages from other
		// clients.
		go c.WriteMessage()
		go c.ReadMessage()
	}
}

// Load recent chat history to current client.
func ServeMessages(ctx context.Context, db *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbMessageList, err := db.ListMessages(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("[error] failed to load messages from database: %v", err)
			return
		}

		w.Header().Set("content-type", "text/html")

		userid, _ := uuid.Parse(r.URL.Query().Get("userid"))

		var prevMsg chat.Message
		for _, msg := range dbMessageList {
			message := chat.Message{
				Userid:    msg.UserID.Bytes,
				Username:  msg.Username,
				Content:   msg.Content,
				CreatedAt: msg.CreatedAt.Time,
			}

			// Check if current and previous messages have the same userid.
			sameUser := false
			if message.Userid == prevMsg.Userid {
				sameUser = true
			}

			// Render message as sender or receiver.
			var content templ.Component
			if message.Userid == userid {
				content = components.SenderBubble(&message, sameUser, message.CreatedAt)
			} else {
				content = components.ReceiverBubble(&message, sameUser, message.CreatedAt)
			}
			content.Render(context.Background(), w)

			prevMsg = message
		}
	}
}
