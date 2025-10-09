package chat

import (
	"time"

	"github.com/google/uuid"
)

type Message struct {
	Content   string `json:"content"`
	Username  string
	Userid    uuid.UUID
	CreatedAt time.Time
}
