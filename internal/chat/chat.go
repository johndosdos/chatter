package chat

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/johndosdos/chatter/internal/database"
)

type Message struct {
	Content   string `json:"content"`
	Username  string
	Userid    uuid.UUID
	CreatedAt time.Time
}

func DbLoadChatHistory(ctx context.Context, recv chan Message, db *database.Queries) {
	// Send the last 50 messages to the client on new connection.
	dbMessageList, err := db.ListMessages(ctx)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		log.Printf("[error] failed to load messages from database: %v", err)
		return
	}

	// Use *Client recv channel.
	for _, msg := range dbMessageList {
		select {
		case recv <- Message{
			Userid:    msg.UserID.Bytes,
			Username:  msg.Username,
			Content:   msg.Content,
			CreatedAt: msg.CreatedAt.Time,
		}:
		case <-ctx.Done():
			return
		}
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
