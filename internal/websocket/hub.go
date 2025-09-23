package websocket

import (
	"context"
	"log"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/johndosdos/chat-app/internal/chat"
	"github.com/johndosdos/chat-app/internal/database"
	"github.com/microcosm-cc/bluemonday"
)

type Hub struct {
	clients    map[uuid.UUID]*Client
	Register   chan *Client
	Unregister chan *Client
	accept     chan chat.Message
	sendToDb   chan chat.Message
	sanitizer  sanitizer
	Ok         chan bool
}

type sanitizer interface {
	Sanitize(s string) string
	SanitizeBytes(p []byte) []byte
}

func (h *Hub) Run(ctx context.Context, db *database.Queries) {
	for {
		select {
		case client := <-h.Register:
			h.clients[client.Userid] = client
			client.Hub = h
			h.Ok <- true
		case client := <-h.Unregister:
			delete(h.clients, client.Userid)
		case message := <-h.accept:
			// We need to sanitize incoming messages to prevent XSS.
			sanitized := h.sanitizer.Sanitize(message.Content)
			message.Content = sanitized
			h.DbStoreMessage(ctx, db, message)
			for _, client := range h.clients {
				client.Recv <- message
			}
		case <-ctx.Done():
			log.Printf("[error] context cancelled: %v", ctx.Err().Error())
			return
		}
	}
}

func (h *Hub) DbStoreMessage(ctx context.Context, db *database.Queries, message chat.Message) {
	_, err := db.CreateMessage(ctx, database.CreateMessageParams{
		UserID:   pgtype.UUID{Bytes: [16]byte(message.Userid), Valid: true},
		Username: message.Username,
		Content:  string(message.Content),
	})
	if err != nil {
		log.Printf("[DB error] failed to store message to database: %v", err)
		return
	}
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[uuid.UUID]*Client),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		accept:     make(chan chat.Message),
		sendToDb:   make(chan chat.Message),
		sanitizer:  bluemonday.StrictPolicy(),
		Ok:         make(chan bool),
	}
}
