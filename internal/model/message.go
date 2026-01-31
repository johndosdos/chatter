package model

import (
	"time"

	"github.com/google/uuid"
)

// ChatMessage represents a message for the chat application,
// used for both NATS payloads and WebSocket communication.
type ChatMessage struct {
	ID        int64     `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Username  string    `json:"username"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`

	// The HTTP request headers sent during websocket transmission
	// used for typing indicator information.
	Headers map[string]string `json:"HEADERS"`
	Type    string            `json:"type,omitempty"`
}
