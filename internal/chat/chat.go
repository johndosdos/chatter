package chat

import (
	"context"
	"log"

	"github.com/google/uuid"
	"github.com/johndosdos/chat-app/internal/database"
)

type Message struct {
	Content  string `json:"content"`
	Username string
	Userid   uuid.UUID
}

func DbLoadChatHistory(ctx context.Context, recv chan Message, db *database.Queries) {
	// Send the last 50 messages to the client on new connection.
	dbMessageList, err := db.ListMessages(ctx)
	if err != nil {
		log.Printf("[error] failed to load messages from database: %v", err)
		return
	}

	// Use *Client recv channel.
	for _, msg := range dbMessageList {
		recv <- Message{
			Userid:   msg.UserID.Bytes,
			Username: msg.Username,
			Content:  msg.Content,
		}
	}
}
