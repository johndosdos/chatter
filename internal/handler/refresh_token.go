package handler

import (
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/johndosdos/chatter/internal/auth"
	"github.com/johndosdos/chatter/internal/database"
)

// RefreshToken handles issuance of JWT and refresh token.
func RefreshToken(db *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		refreshTokCookie, err := r.Cookie("refresh_token")
		if errors.Is(err, http.ErrNoCookie) {
			w.Header().Set("HX-Redirect", "/account/login")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		refreshTok, err := db.GetUserFromRefreshTok(r.Context(), refreshTokCookie.Value)
		if err != nil {
			log.Printf("handler/refresh token: failed to retrieve user: %v", err)
			return
		}

		jwtString, err := auth.MakeJWT(refreshTok.UserID.Bytes, os.Getenv("JWT_SECRET"), 5*time.Minute)
		if err != nil {
			log.Printf("handler/refresh token: failed to create JWT: %v", err)
			return
		}

		// Set cookie for access token. Expires in 5 minutes.
		http.SetCookie(w, &http.Cookie{
			Name:        "jwt",
			Value:       jwtString,
			Quoted:      false,
			Path:        "/",
			Domain:      "",
			Expires:     time.Time{},
			RawExpires:  "",
			MaxAge:      5 * 60,
			Secure:      true,
			HttpOnly:    true,
			SameSite:    http.SameSiteLaxMode,
			Partitioned: false,
			Raw:         "",
			Unparsed:    []string{},
		})

		w.WriteHeader(http.StatusOK)
	}
}
