package chat

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
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
