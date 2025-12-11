// Package model defines data structure.
package model

import (
	"time"

	"github.com/google/uuid"
)

// Message holds information about a single message.
type Message struct {
	Content   string `json:"content"`
	Username  string
	UserID    uuid.UUID
	CreatedAt time.Time
}
