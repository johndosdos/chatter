package handler

import (
	"context"
	"net/http"

	viewChat "github.com/johndosdos/chatter/components/chat"
)

func ServeChat(ctx context.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			return
		}

		viewChat.ChatLayout().Render(ctx, w)
	}
}
