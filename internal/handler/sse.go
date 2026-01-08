package handler

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/a-h/templ"
	"github.com/jackc/pgx/v5/pgtype"
	components "github.com/johndosdos/chatter/components/chat"
	"github.com/johndosdos/chatter/internal/auth"
	"github.com/johndosdos/chatter/internal/chat"
	"github.com/johndosdos/chatter/internal/database"
	"github.com/johndosdos/chatter/internal/model"
)

func StreamSSE(hub *chat.Hub, db *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			return
		}

		w.Header().Set("X-Accel-Buffering", "no")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		rc := http.NewResponseController(w)
		err := rc.Flush()
		if err != nil {
			log.Printf("%v", err)
			return
		}

		ctx := r.Context()

		userID, err := auth.GetUserFromContext(ctx)
		if err != nil {
			log.Printf("%v", err)
			return
		}

		user, _ := db.GetUserById(ctx, pgtype.UUID{Bytes: userID, Valid: true})

		// We'll register our new client to the central hub.
		c := chat.NewClient(user.UserID.Bytes, user.Username, hub)
		reg := chat.Registration{
			Client: c,
			Done:   make(chan struct{}),
		}
		hub.Register <- reg
		<-reg.Done
		log.Printf("client [%s] connected\n", c.Username)

		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		var prevMsg model.Message
		for {
			select {
			case message, ok := <-c.MessageCh:
				if !ok {
					continue
				}

				// Check if current and previous messages have the same userid.
				sameUser := false
				if message.UserID == prevMsg.UserID {
					sameUser = true
				}

				// Render message as sender or receiver.
				var comp templ.Component
				if message.UserID == c.UserID {
					comp = components.SenderBubble(message.Username, message.Content, sameUser, message.CreatedAt)
				} else {
					comp = components.ReceiverBubble(message.Username, message.Content, sameUser, message.CreatedAt)
				}
				prevMsg = message

				var dataBuf bytes.Buffer

				if err := comp.Render(ctx, &dataBuf); err != nil {
					log.Printf("failed to render component: %v", err)
					return
				}

				data := bytes.ReplaceAll(dataBuf.Bytes(), []byte("\n"), []byte(" "))

				// fmt.Fprintf(w, "id: %v\n", message.SequenceID)
				fmt.Fprint(w, "event: message\n")    //nolint:errcheck
				fmt.Fprintf(w, "data: %s\n\n", data) //nolint:errcheck

				if err := rc.Flush(); err != nil {
					log.Printf("could not flush buffer to writer: %+v", err)
				}

			case <-ticker.C:
				fmt.Fprint(w, ": \n\n") //nolint:errcheck
				if err := rc.Flush(); err != nil {
					log.Printf("could not flush buffer to writer: %+v", err)
				}

			case <-ctx.Done():
				log.Printf("%v", ctx.Err())
				return
			}
		}
	}
}
