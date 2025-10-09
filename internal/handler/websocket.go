package handler

import (
	"context"
	"log"
	"net/http"
	"time"

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
		go KeepaliveConn(conn)

		// Run these goroutines to listen and process messages from other
		// clients.
		go c.WriteMessage()
		go c.ReadMessage()
	}
}

func KeepaliveConn(conn *websocket.Conn) {
	// Ping client every 60s.
	pongWait := 60 * time.Second

	// The default connection behavior is to wait indefinitely for incoming data.
	// Firewalls, proxies, and other services have their own system to invalidate
	// a stale connection. Therefore, we must keep the connection alive by sending
	// ping pong signals between the server and the client (to simulate network traffic)
	// within a set deadline.
	err := conn.SetReadDeadline(time.Now().Add(pongWait))
	if err != nil {
		log.Printf("[error] failed to set read deadline: %v", err)
		return
	}

	// Reset deadline after receiving pong signal.
	conn.SetPongHandler(func(appData string) error {
		return conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	ticker := time.NewTicker((pongWait * 9) / 10)

	for range ticker.C {
		err := conn.WriteMessage(websocket.PingMessage, nil)
		if err != nil {
			log.Printf("[error] failed to send ping signal: %v", err)
			break
		}
	}
}
