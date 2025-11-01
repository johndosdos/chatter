package internal

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"

	"github.com/johndosdos/chatter/internal/auth"
)

func Middleware(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jwtCookie, err := r.Cookie("jwt")
		if errors.Is(err, http.ErrNoCookie) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		uuid, err := auth.ValidateJWT(jwtCookie.Value, os.Getenv("JWT_SECRET"))
		if err != nil {
			log.Printf("middleware: invalid access token: %v", err)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		r = r.WithContext(context.WithValue(r.Context(), auth.UserIDKey, uuid))
		next.ServeHTTP(w, r)
	}
}
