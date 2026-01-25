package handler

import (
	"context"
	"log"
	"net/http"

	"github.com/a-h/templ"
	"github.com/johndosdos/chatter/internal/auth"
	"github.com/johndosdos/chatter/internal/database"

	viewChat "github.com/johndosdos/chatter/components/chat"
)

// ServeMessages handles client message rendering. It will load recent
// chat history to current client.
func ServeMessages(db *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Parse JWT and get UserID.
		userID, err := auth.GetUserFromContext(ctx)
		if err != nil {
			log.Printf("%v", err)
			return
		}

		// Fetch latest 50 messages
		dbMessageList, err := db.ListMessages(ctx, 50)
		if err != nil {
			log.Printf("%v", err)
			return
		}

		var prevMsg database.ListMessagesRow

		// Iterate in reverse to show oldest messages first (chronological order)
		for i := len(dbMessageList) - 1; i >= 0; i-- {
			message := dbMessageList[i]

			// Check if current and previous messages have the same UserID.
			sameUser := false
			if message.UserID == prevMsg.UserID {
				sameUser = true
			}

			w.Header().Set("Content-Type", "text/html")

			// Render message as sender or receiver.
			var content templ.Component
			if message.UserID.Bytes == userID {
				content = viewChat.SenderBubble(message.Username, message.Content, sameUser, message.ID)
			} else {
				content = viewChat.ReceiverBubble(message.Username, message.Content, sameUser, message.ID)
			}
			if err := content.Render(context.Background(), w); err != nil {
				log.Printf("failed to render component: %v", err)
				return
			}

			prevMsg = message
		}
	}
}
