package handler

import (
	"net/http"

	viewChat "github.com/johndosdos/chatter/components/chat"
)

func ServeChat() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if r.Method != http.MethodGet {
			return
		}

		viewChat.ChatLayout().Render(ctx, w)
	}
}
