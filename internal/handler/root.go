package handler

import (
	"errors"
	"net/http"
)

func ServeRoot() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get refresh token and check if valid.
		// If invalid, redirect to "/account/login".
		// If valid, proceed to "/chat" and let them handle the access token.

		jwtCookie, err := r.Cookie("jwt")
		if errors.Is(err, http.ErrNoCookie) || jwtCookie.MaxAge < 0 {
			http.Redirect(w, r, "/account/login", http.StatusSeeOther)
			return
		}

		http.Redirect(w, r, "/chat", http.StatusSeeOther)
	}
}
