package internal

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/google/uuid"
	"github.com/johndosdos/chatter/internal/auth"
	"github.com/johndosdos/chatter/internal/database"
)

// Middleware validates the client's JWT.
func Middleware(next http.Handler, db *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jwtCookie, err := r.Cookie("jwt")
		// Check JWT cookie if it exists. If it does, validate the JWT. If valid,
		// append user ID to context and serve the next handler.
		if err == nil {
			uuid, err := auth.ValidateJWT(jwtCookie.Value, os.Getenv("JWT_SECRET"))
			if err == nil {
				r = r.WithContext(context.WithValue(r.Context(), auth.UserIDKey, uuid))
				next.ServeHTTP(w, r)
				return
			}
		}

		// If JWT cookie does not exist or is not valid, check refresh token
		// cookie if it exists. If it doesn't, redirect user to the login page.
		// If it does, create a new JWT, append user ID to context and serve
		// the next handler.
		refreshTokCookie, err := r.Cookie("refresh_token")
		if err != nil {
			http.Redirect(w, r, "/account/login", http.StatusSeeOther)
			return
		}

		// Check if refresh token exists in the database. If it doesn't,
		// redirect user to the login page. We don't want unauthorized access
		// to our /chat endpoint.
		refreshTokenDB, err := db.GetRefreshToken(r.Context(), refreshTokCookie.Value)
		if err != nil {
			log.Printf("refresh token missing from the database: %v", err)
			http.Redirect(w, r, "/account/login", http.StatusSeeOther)
			return
		}

		uid := uuid.UUID(refreshTokenDB.UserID.Bytes)

		err = auth.SetTokensAndCookies(w, r, db, uid)
		if err != nil {
			log.Printf("%v", err)
			return
		}

		r = r.WithContext(context.WithValue(r.Context(), auth.UserIDKey, uid))
		next.ServeHTTP(w, r)
	}
}
