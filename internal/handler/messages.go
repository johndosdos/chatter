package handler

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/a-h/templ"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/johndosdos/chatter/internal/auth"
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

		// Parse JWT and get userId.
		userId := ctx.Value(auth.UserIdKey).(uuid.UUID)
		since := r.URL.Query().Get("since")

		dbMessageList, err := filterMessages(ctx, since, db)
		if err != nil {
			log.Printf("handler/messages: %v", err)
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
			if message.Userid == userId {
				content = viewChat.SenderBubble(message.Username, message.Content, sameUser, message.CreatedAt)
			} else {
				content = viewChat.ReceiverBubble(message.Username, message.Content, sameUser, message.CreatedAt)
			}
			if err := content.Render(context.Background(), w); err != nil {
				log.Printf("handler/messages: failed to render component: %v", err)
				return
			}

			prevMsg = message
		}
	}
}

func filterMessages(ctx context.Context, since string, db *database.Queries) ([]database.ListMessagesRow, error) {
	var dbMessageList []database.ListMessagesRow
	var err error

	if since != "" {
		t, err := time.Parse(time.RFC3339Nano, since)
		if err != nil {
			return nil, fmt.Errorf("handler/messages: failed to parse time in specified format: %w", err)
		}

		dbMessageList, err = db.ListMessages(ctx, pgtype.Timestamptz{Time: t, Valid: true})
		if err != nil {
			return nil, fmt.Errorf("handler/messages: failed to load messages from database: %w", err)
		}
	} else {
		dbMessageList, err = db.ListMessages(ctx, pgtype.Timestamptz{Time: time.Time{}, Valid: false})
		if err != nil {
			return nil, fmt.Errorf("handler/messages: failed to load messages from database: %w", err)
		}
	}

	return dbMessageList, nil
}
