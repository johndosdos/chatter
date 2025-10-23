package handler

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/a-h/templ"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/johndosdos/chatter/internal/chat"
	"github.com/johndosdos/chatter/internal/database"

	viewChat "github.com/johndosdos/chatter/components/chat"
)

// Load recent chat history to current client.
func ServeMessages(db *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if r.Method != http.MethodGet {
			return
		}

		userid, _ := uuid.Parse(r.URL.Query().Get("userid"))
		since := r.URL.Query().Get("since")

		var dbMessageList []database.ListMessagesRow
		var err error

		if since == "" {
			dbMessageList, err = db.ListMessages(ctx, pgtype.Timestamptz{Valid: false})
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Printf("[error] failed to load messages from database: %v", err)
				return
			}
		} else {
			t, err := time.Parse(time.RFC3339Nano, since)
			if err != nil {
				log.Println(err.Error())
				return
			}

			dbMessageList, err = db.ListMessages(ctx, pgtype.Timestamptz{Time: t, Valid: true})
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Printf("[error] failed to load messages from database: %v", err)
				return
			}
		}

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
