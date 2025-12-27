package chat

import (
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/johndosdos/chatter/internal/auth"
	"github.com/johndosdos/chatter/internal/database"
	"github.com/johndosdos/chatter/internal/model"
)

// Client contains client connection information.
type Client struct {
	UserID    uuid.UUID
	Username  string
	Hub       *Hub
	MessageCh chan model.Message
}

// NewClient returns a new instance of Client.
func NewClient(userID uuid.UUID, username string, hub *Hub) *Client {
	return &Client{
		UserID:    userID,
		Username:  username,
		Hub:       hub,
		MessageCh: make(chan model.Message, 64),
	}
}

func Send(hub *Hub, db *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if r.Method != http.MethodPost {
			return
		}

		err := r.ParseForm()
		if err != nil {
			log.Printf("invalid form data: %v", err)
			http.Error(w, "Invalid form data", http.StatusBadRequest)
			return
		}

		content := r.FormValue("content")
		userID, err := auth.GetUserFromContext(ctx)
		if err != nil {
			log.Printf("%v", err)
			return
		}

		userDB, err := db.GetUserById(ctx, pgtype.UUID{
			Bytes: userID,
			Valid: true,
		})
		if err != nil {
			log.Printf("failed to get user by ID: %v", err)
			return
		}

		var message model.Message

		message.Content = content
		message.CreatedAt = time.Now().UTC()
		message.UserID = userID
		message.Username = userDB.Username

		hub.ClientMsg <- message
	}
}
