package handler

import (
	"context"
	"log"
	"net/http"

	"github.com/a-h/templ"
	"github.com/google/uuid"
	"github.com/johndosdos/chatter/internal/chat"
	"github.com/johndosdos/chatter/internal/database"

	viewChat "github.com/johndosdos/chatter/components/chat"
)

// Load recent chat history to current client.
func ServeMessages(ctx context.Context, db *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			return
		}

		dbMessageList, err := db.ListMessages(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("[error] failed to load messages from database: %v", err)
			return
		}

		userid, _ := uuid.Parse(r.URL.Query().Get("userid"))

		var prevMsg chat.Message
		for _, msg := range dbMessageList {
			message := chat.Message{
				Userid:    msg.UserID.Bytes,
				Username:  msg.Username,
				Content:   msg.Content,
				CreatedAt: msg.CreatedAt.Time,
			}

			// Check if current and previous messages have the same userid.
			sameUser := false
			if message.Userid == prevMsg.Userid {
				sameUser = true
			}

			w.Header().Set("Content-Type", "text/html")

			// Render message as sender or receiver.
			var content templ.Component
			if message.Userid == userid {
				content = viewChat.SenderBubble(message.Username, message.Content, sameUser, message.CreatedAt)
			} else {
				content = viewChat.ReceiverBubble(message.Username, message.Content, sameUser, message.CreatedAt)
			}
			content.Render(context.Background(), w)

			prevMsg = message
		}
	}
}
