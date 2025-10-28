package handler

import (
	"errors"
	"log"
	"net/http"
)

func ServeRoot() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get refresh token and check if valid.
		// If invalid, redirect to "/account/login".
		// If valid, proceed to "/chat" and let them handle the access token.

		jwtCookie, err := r.Cookie("jwt")
		switch {
		case errors.Is(err, http.ErrNoCookie):
			http.Redirect(w, r, "/account/login", http.StatusSeeOther)
			return
		case jwtCookie.MaxAge < 0:
			log.Printf("handler/root: expired JWT: %v", jwtCookie.MaxAge)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		http.Redirect(w, r, "/chat", http.StatusSeeOther)
	}
}
