package handler

import (
	"context"
	"net/http"

	"github.com/johndosdos/chatter/components/auth"
)

func ServeLogin(ctx context.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth.Login().Render(ctx, w)
	}
}
