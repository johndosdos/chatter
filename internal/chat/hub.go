package chat

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
	BrokerMsg  chan model.Message
	ClientMsg  chan model.Message
	sanitizer  sanitizer
}

// Run manages incoming and outgoing hub traffic.
func (h *Hub) Run(ctx context.Context, js jetstream.Stream) {
	err := broker.Subscriber(ctx, js, h.BrokerMsg)
	if err != nil {
		log.Printf("could not connect to the postgresql database: %v", err)
	}

	for {
		select {
		case reg := <-h.Register:
			c := reg.Client
			h.clients[c.UserID] = c
			close(reg.Done)

		case client := <-h.Unregister:
			delete(h.clients, client.UserID)
			close(client.MessageCh)

		case message := <-h.ClientMsg:
			sanitizedMsg := h.sanitizer.Sanitize(message.Content)
			message.Content = sanitizedMsg
			messageParams := database.CreateMessageParams{
				UserID:  pgtype.UUID{Bytes: [16]byte(message.UserID), Valid: true},
				Content: message.Content,
				CreatedAt: pgtype.Timestamptz{
					Time:             message.CreatedAt,
					InfinityModifier: 0,
					Valid:            true,
				},
			}
			_, err = h.db.CreateMessage(ctx, messageParams)
			if err != nil {
				log.Printf("failed to store payload to database: %v", err)
				continue
			}

			err = broker.Publisher(ctx, h.jetstream, message)
			if err != nil {
				log.Printf("%v", err)
				continue
			}

		case payload := <-h.BrokerMsg:
			for _, c := range h.clients {
				select {
				case c.MessageCh <- payload:
				default:
					log.Println("skipping message payload...")
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
		BrokerMsg:  make(chan model.Message, 64),
		ClientMsg:  make(chan model.Message, 64),
		sanitizer:  bluemonday.StrictPolicy(),
	}
}
