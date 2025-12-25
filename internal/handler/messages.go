package handler

import (
	"log"
	"net/http"

	"github.com/a-h/templ"
	"github.com/johndosdos/chatter/internal/auth"
	"github.com/johndosdos/chatter/internal/database"
	"github.com/johndosdos/chatter/internal/model"

	"github.com/johndosdos/chatter/components/chat"
)

// ServeMessages handles client message rendering. It will load recent
// chat history to current client.
func ServeMessages(db *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if r.Method != http.MethodGet {
			return
		}

		// Parse JWT and get UserID.
		userID, err := auth.GetUserFromContext(ctx)
		if err != nil {
			log.Printf("%v", err)
			return
		}

		msgLimit := 50
		dbMessageList, err := db.ListMessages(ctx, int32(msgLimit))
		if err != nil {
			log.Printf("%v", err)
			return
		}

		var prevMsg model.Message

		for i := len(dbMessageList) - 1; i >= 0; i-- {
			msg := dbMessageList[i]

			message := model.Message{
				UserID:    msg.UserID.Bytes,
				Username:  msg.Username,
				Content:   msg.Content,
				CreatedAt: msg.CreatedAt.Time,
			}

			// Check if current and previous messages have the same UserID.
			sameUser := false
			if message.UserID == prevMsg.UserID {
				sameUser = true
			}

			w.Header().Set("Content-Type", "text/html")

			// Render message as sender or receiver.
			var content templ.Component
			if message.UserID == userID {
				content = chat.SenderBubble(message.Username, message.Content, sameUser, message.CreatedAt)
			} else {
				content = chat.ReceiverBubble(message.Username, message.Content, sameUser, message.CreatedAt)
			}

			if err := content.Render(ctx, w); err != nil {
				log.Printf("failed to render component: %v", err)
				return
			}

			prevMsg = message
		}
	}
}
