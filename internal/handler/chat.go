package handler

import (
	"net/http"

	"github.com/google/uuid"
	viewChat "github.com/johndosdos/chatter/components/chat"
)

func ServeChat() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if r.Method != http.MethodGet {
			return
		}

		userid, _ := uuid.Parse(r.URL.Query().Get("userid"))

		viewChat.ChatLayout(userid.String()).Render(ctx, w)
	}
}
