package handler

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	viewChat "github.com/johndosdos/chatter/components/chat"
)

func ServeChat(ctx context.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			return
		}

		userid, _ := uuid.Parse(r.URL.Query().Get("userid"))

		viewChat.ChatLayout(userid.String()).Render(ctx, w)
	}
}
