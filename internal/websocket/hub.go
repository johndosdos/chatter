package websocket

import (
	"context"
	"log"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/johndosdos/chatter/internal/broker"
	"github.com/johndosdos/chatter/internal/database"
	"github.com/johndosdos/chatter/internal/model"
	"github.com/microcosm-cc/bluemonday"
	"github.com/nats-io/nats.go/jetstream"
)

type sanitizer interface {
	Sanitize(s string) string
	SanitizeBytes(p []byte) []byte
}

// Hub contains functions needed for thee app state management.
type Hub struct {
	db         *database.Queries
	jetstream  jetstream.JetStream
	clients    map[uuid.UUID]*Client
	Register   chan *Client
	Unregister chan *Client
	fromClient chan model.Message
	FromWorker chan model.Message
	sanitizer  sanitizer
	Ok         chan bool
}

// Run manages incoming and outgoing hub traffic.
func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case client := <-h.Register:
			h.clients[client.UserID] = client
			client.Hub = h
			h.Ok <- true

		case client := <-h.Unregister:
			delete(h.clients, client.UserID)
			close(client.FromHub)

		case payload := <-h.fromClient:
			// We need to sanitize incoming messages to prevent XSS.
			sanitized := h.sanitizer.Sanitize(payload.Content)
			payload.Content = sanitized

			message := database.CreateMessageParams{
				UserID:  pgtype.UUID{Bytes: [16]byte(payload.UserID), Valid: true},
				Content: string(payload.Content),
				CreatedAt: pgtype.Timestamptz{
					Time:             payload.CreatedAt,
					InfinityModifier: 0,
					Valid:            true,
				},
			}

			_, err := h.db.CreateMessage(ctx, message)
			if err != nil {
				log.Printf("worker/database: failed to store payload to database: %v", err)
				continue
			}

			err = broker.Publisher(ctx, h.jetstream, payload)
			if err != nil {
				log.Printf("worker/database: %v", err)
				continue
			}

		case payload := <-h.FromWorker:
			for _, client := range h.clients {
				client.FromHub <- payload
			}

		case <-ctx.Done():
			log.Printf("websocket/hub: context cancelled: %v", ctx.Err())
			return
		}
	}
}

// NewHub returns a new instance of Hub.
func NewHub(js jetstream.JetStream, db *database.Queries) *Hub {
	return &Hub{
		db:         db,
		jetstream:  js,
		clients:    make(map[uuid.UUID]*Client),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		fromClient: make(chan model.Message, 1024),
		FromWorker: make(chan model.Message, 1024),
		sanitizer:  bluemonday.StrictPolicy(),
		Ok:         make(chan bool, 64),
	}
}
