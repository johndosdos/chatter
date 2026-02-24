package handler

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/coder/websocket"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/johndosdos/chatter/internal/auth"
	"github.com/johndosdos/chatter/internal/database"
	ws "github.com/johndosdos/chatter/internal/websocket"
)

// ServeWs handles the client's websocket connection upgrade.
func ServeWs(h *ws.Hub, db *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Validate user from request context and check if user is in DB before
		// websocket upgrade request. We want to fail early if one of the checks
		// fail.
		userID, err := auth.GetUserFromContext(ctx)
		if err != nil {
			slog.WarnContext(ctx, "unable to get user info from request context",
				"error", err,
				"userID", userID)

			w.Header().Add("HX-Redirect", "/account/login")
			w.WriteHeader(http.StatusOK)
			return
		}

		user, err := db.GetUserById(ctx, pgtype.UUID{Bytes: userID, Valid: true})
		if err != nil {
			slog.ErrorContext(ctx, "failed to get user from DB",
				"error", err)
			return
		}

		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			slog.WarnContext(ctx, "WS handshake failed",
				"error", err)
			return
		}

		slog.InfoContext(ctx, "user connection upgrade",
			slog.String("username", user.Username))

		// We'll register our new client to the central hub.
		c := ws.NewClient(conn, user.UserID.Bytes, user.Username)
		reg := ws.Registration{
			Client: c,
			Done:   make(chan struct{}),
		}

		messageReq := 30
		typingReq := 30

		c.SetMessageLimiter(messageReq, time.Minute)
		c.SetTypingLimiter(typingReq, time.Minute)

		h.Register <- reg

		// Wait for registration to complete
		<-reg.Done

		// Run these goroutines to listen and process messages from other
		// clients.
		//
		// We block on c.ReadMessage() because the request context will be canceled as soon
		// we return from the ServeWs() handler.
		go c.WriteMessage(ctx)
		c.ReadMessage(ctx)
	}
}
