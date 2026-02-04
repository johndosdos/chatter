package websocket

import (
	"context"
	"log"
	"strconv"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/johndosdos/chatter/internal/database"
	"github.com/johndosdos/chatter/internal/model"
	"github.com/microcosm-cc/bluemonday"
)

type sanitizer interface {
	Sanitize(s string) string
	SanitizeBytes(p []byte) []byte
}

type Registration struct {
	Client *Client
	Done   chan struct{}
}

type Hub struct {
	db *database.Queries
	// jetstream  jetstream.JetStream
	clients    map[uuid.UUID]*Client
	Register   chan Registration
	Unregister chan *Client
	ClientMsg  chan model.ChatMessage
	// BrokerMsg  chan model.ChatMessage
	sanitizer sanitizer
}

func (h *Hub) Run(ctx context.Context) {
	/* 	err := broker.Subscriber(ctx, stream, h.BrokerMsg)
	   	if err != nil {
	   		log.Printf("failed to subscribe to broker: %v", err)
	   	} */

	for {
		select {
		case reg := <-h.Register:
			client := reg.Client
			h.clients[client.UserID] = client
			client.Hub = h
			h.connectedUsers()
			close(reg.Done)

		case client := <-h.Unregister:
			delete(h.clients, client.UserID)
			h.connectedUsers()
			close(client.MessageCh)

		case payload := <-h.ClientMsg:
			// We need to sanitize incoming messages to prevent XSS.
			sanitized := h.sanitizer.Sanitize(payload.Content)
			payload.Content = sanitized

			/* 			// If the message is a typing indicator, we don't need to persist it.
			   			if payload.Type == "typing" {
			   				err := broker.Publisher(ctx, h.jetstream, payload)
			   				if err != nil {
			   					log.Printf("%v", err)
			   				}
			   				continue
			   			} */

			message := database.CreateMessageParams{
				UserID:  pgtype.UUID{Bytes: [16]byte(payload.UserID), Valid: true},
				Content: string(payload.Content),
				CreatedAt: pgtype.Timestamptz{
					Time:             payload.CreatedAt,
					InfinityModifier: 0,
					Valid:            true,
				},
			}

			// If payload.Type is 'typing', try not to create an entry in the database.
			if payload.Type == "message" {
				createdMsg, err := h.db.CreateMessage(ctx, message)
				if err != nil {
					log.Printf("failed to store payload to database: %v", err)
					continue
				}
				payload.ID = createdMsg.ID
				payload.CreatedAt = createdMsg.CreatedAt.Time
			}

			/* 			err = broker.Publisher(ctx, h.jetstream, payload)
			   			if err != nil {
			   				log.Printf("%v", err)
			   				continue
			   			} */

			for _, client := range h.clients {
				client.MessageCh <- payload
			}

		case <-ctx.Done():
			log.Printf("context cancelled: %v", ctx.Err())
			return
		}
	}
}

func (h *Hub) connectedUsers() {
	// Retrieve connected users through the clients table.
	// Send HTML fragment to client through websockets and do OOB swap thereafter.
	// Remember to send the data through the client.MessageCh. DO NOT CREATE A WRITER.
	userSize := len(h.clients)
	for _, client := range h.clients {
		client.MessageCh <- model.ChatMessage{
			Content: strconv.Itoa(userSize),
			Type:    "presenceCount",
		}
	}
}

func NewHub(db *database.Queries) *Hub {
	return &Hub{
		db: db,
		// jetstream:  js,
		clients:    make(map[uuid.UUID]*Client),
		Register:   make(chan Registration),
		Unregister: make(chan *Client),
		ClientMsg:  make(chan model.ChatMessage, 1024),
		// BrokerMsg:  make(chan model.ChatMessage, 1024),
		sanitizer: bluemonday.StrictPolicy(),
	}
}
