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

type Registration struct {
	Client *Client
	Done   chan struct{}
}

// Hub contains functions needed for thee app state management.
type Hub struct {
	db         *database.Queries
	jetstream  jetstream.JetStream
	clients    map[uuid.UUID]*Client
	Register   chan Registration
	Unregister chan *Client
	ClientMsg  chan model.ChatMessage
	BrokerMsg  chan model.ChatMessage
	sanitizer  sanitizer
}

// Run manages incoming and outgoing hub traffic.
func (h *Hub) Run(ctx context.Context, stream jetstream.Stream) {
	err := broker.Subscriber(ctx, stream, h.BrokerMsg)
	if err != nil {
		log.Printf("failed to subscribe to broker: %v", err)
	}

	for {
		select {
		case reg := <-h.Register:
			client := reg.Client
			h.clients[client.UserID] = client
			client.Hub = h
			close(reg.Done)

		case client := <-h.Unregister:
			delete(h.clients, client.UserID)
			close(client.MessageCh)

		case payload := <-h.ClientMsg:
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

			// We persist the message to DB. Note: DB generates the ID.
			// The ID in payload is currently 0 or temp, but for broadcast we might want real ID?
			// Ideally we get the ID back from DB and update payload before broadcasting.
			createdMsg, err := h.db.CreateMessage(ctx, message)
			if err != nil {
				log.Printf("failed to store payload to database: %v", err)
				continue
			}

			// Update payload with DB-generated ID and created_at
			payload.ID = createdMsg.ID
			payload.CreatedAt = createdMsg.CreatedAt.Time

			err = broker.Publisher(ctx, h.jetstream, payload)
			if err != nil {
				log.Printf("%v", err)
				continue
			}

		case payload := <-h.BrokerMsg:
			for _, client := range h.clients {
				select {
				case client.MessageCh <- payload:
				default:
					log.Println("skipping message payload - channel full or client slow")
				}
			}

		case <-ctx.Done():
			log.Printf("context cancelled: %v", ctx.Err())
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
		Register:   make(chan Registration),
		Unregister: make(chan *Client),
		ClientMsg:  make(chan model.ChatMessage, 1024),
		BrokerMsg:  make(chan model.ChatMessage, 1024),
		sanitizer:  bluemonday.StrictPolicy(),
	}
}
