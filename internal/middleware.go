package internal

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/johndosdos/chatter/internal/auth"
	"github.com/johndosdos/chatter/internal/database"
)

// Middleware validates the client's JWT.
func Middleware(next http.Handler, db *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if refresh token cookie exists in the database. If it doesn't,
		// redirect to the login page. We don't want unauthorized access to
		// our /chat endpoint.
		refreshTokCookie, err := r.Cookie("refresh_token")
		if err == nil {
			_, err = db.DoesRefreshTokenExist(r.Context(), refreshTokCookie.Value)
			if err != nil {
				http.Redirect(w, r, "/account/login", http.StatusSeeOther)
				return
			}
		}

		jwtCookie, err := r.Cookie("jwt")
		// Check JWT if it exists. If it does, validate the JWT. If valid,
		// append user ID to context and serve the next handler.
		if err == nil {
			uuid, err := auth.ValidateJWT(jwtCookie.Value, os.Getenv("JWT_SECRET"))
			if err == nil {
				r = r.WithContext(context.WithValue(r.Context(), auth.UserIDKey, uuid))
				next.ServeHTTP(w, r)
				return
			}
		}

		// If JWT does not exist or is not valid, check refresh token if it
		// exists. If it does, create a new JWT, append user ID to context
		// and serve the next handler.
		userID, err := auth.RefreshSession(w, r, db)
		if err != nil {
			log.Printf("middleware: %v", err)
			return
		}

		r = r.WithContext(context.WithValue(r.Context(), auth.UserIDKey, userID))
		next.ServeHTTP(w, r)
	}
}
