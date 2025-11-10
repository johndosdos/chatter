package handler

import (
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	viewAuth "github.com/johndosdos/chatter/components/auth"
	"github.com/johndosdos/chatter/internal/auth"
	"github.com/johndosdos/chatter/internal/database"
)

// ServeLogin handles user login.
func ServeLogin(db *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if r.Method != http.MethodPost {
			if err := viewAuth.Login().Render(ctx, w); err != nil {
				log.Printf("handler/account/login: failed to render component: %v", err)
			}
			return
		}

		err := r.ParseForm()
		if err != nil {
			http.Error(w, "Invalid form data.", http.StatusBadRequest)
			log.Printf("handler/account/login: failed to parse form values: %v", err)
			return
		}

		email := r.PostFormValue("email")
		password := r.PostFormValue("password")

		user, err := db.GetUserWithPasswordByEmail(ctx, email)
		if err != nil {
			if err := viewAuth.Error("Invalid email or password.").Render(ctx, w); err != nil {
				log.Printf("handler/account/login: failed to render component: %v", err)
				return
			}
			log.Printf("handler/account/login: failed to retrieve user from db: %v", err)
			return
		}

		ok, err := auth.CheckPasswordHash(password, user.HashedPassword)
		if err != nil {
			http.Error(w, "Server error.", http.StatusInternalServerError)
			log.Printf("handler/account/login: cannot verify password — hash may be corrupted: %v", err)
			return
		}
		if !ok {
			if err := viewAuth.Error("Invalid email or password.").Render(ctx, w); err != nil {
				log.Printf("handler/account/login: failed to render component: %v", err)
			}
			return
		}

		err = auth.SetTokensAndCookies(w, r, db, user.UserID.Bytes)
		if err != nil {
			log.Printf("handler/account/login: %v", err)
			return
		}

		w.Header().Set("HX-Redirect", "/chat")
		w.WriteHeader(http.StatusOK)
	}
}

// ServeSignup handles user account creation.
func ServeSignup(db *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if r.Method != http.MethodPost {
			if err := viewAuth.Signup().Render(ctx, w); err != nil {
				log.Printf("handler/account/signup: failed to close connection: %v", err)
				return
			}
			return
		}

		err := r.ParseForm()
		if err != nil {
			http.Error(w, "Invalid form data.", http.StatusBadRequest)
			log.Printf("handler/account/signup: failed to parse form values: %v", err)
			return
		}

		password := r.PostFormValue("password")
		confirmPw := r.PostFormValue("confirm_password")

		// Validate password by comparing main and confirm.
		if password != confirmPw {
			if err := viewAuth.Error("Passwords do not match!").Render(ctx, w); err != nil {
				log.Printf("handler/account/signup: failed to close connection: %v", err)
				return
			}
			return
		}

		username := r.PostFormValue("username")
		email := r.PostFormValue("email")
		user, err := db.CreateUser(ctx, database.CreateUserParams{
			UserID:   pgtype.UUID{Bytes: uuid.New(), Valid: true},
			Username: username,
			Email:    email,
		})
		if err != nil {
			http.Error(w, "Database error.", http.StatusInternalServerError)
			log.Printf("handler/account/signup: failed to create user entry in database: %v", err)
			return
		}

		// Hash and compare password before storing to database.
		hashedPw, err := auth.HashPassword(password)
		if err != nil {
			http.Error(w, "Server error.", http.StatusInternalServerError)
			log.Printf("handler/account/signup: argon2id hash creation failed: %v", err)
			return
		}

		ok, err := auth.CheckPasswordHash(password, hashedPw)
		if err != nil {
			http.Error(w, "Server error.", http.StatusInternalServerError)
			log.Printf("handler/account/signup: cannot verify password — hash may be corrupted: %v", err)
			return
		}

		if ok {
			_, err := db.CreatePassword(ctx, database.CreatePasswordParams{
				UserID:         user.UserID,
				HashedPassword: hashedPw,
				CreatedAt:      pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
			})
			if err != nil {
				http.Error(w, "Database error.", http.StatusInternalServerError)
				log.Printf("handler/account/signup: failed to create password entry in database: %v", err)
				return
			}
		}

		w.Header().Set("HX-Redirect", "/account/login")
		w.WriteHeader(http.StatusOK)
	}
}

// ServeLogout deletes the user's assigned refresh token, and redirects
// the user to the login page.
func ServeLogout(db *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if r.Method != http.MethodPost {
			if err := viewAuth.Login().Render(ctx, w); err != nil {
				log.Printf("handler/account/logout: failed to render component: %v", err)
			}
			return
		}

		refreshTok, err := r.Cookie("refresh_token")
		if err == nil {
			err = db.RevokeRefreshToken(ctx, refreshTok.Value)
			if err != nil {
				log.Printf("handler/account/logout: failed to process token deletion: %v", err)
				return
			}

			refreshTok.MaxAge = -1
			http.SetCookie(w, refreshTok)
			w.Header().Set("HX-Redirect", "/account/login")
			w.WriteHeader(http.StatusOK)
			return
		}
	}
}
