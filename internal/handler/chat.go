package handler

import (
	"log"
	"net/http"

	viewChat "github.com/johndosdos/chatter/components/chat"
)

// ServeChat handles the chat interface.
func ServeChat() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if r.Method != http.MethodGet {
			return
		}

		if err := viewChat.ChatLayout().Render(ctx, w); err != nil {
			log.Printf("handler/chat: failed to close connection: %v", err)
			return
		}
	}
}
