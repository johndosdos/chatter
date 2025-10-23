package handler

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	viewAuth "github.com/johndosdos/chatter/components/auth"
	"github.com/johndosdos/chatter/internal/auth"
	"github.com/johndosdos/chatter/internal/database"
)

func ServeLogin(db *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if r.Method != http.MethodPost {
			viewAuth.Login().Render(ctx, w)
			return
		}

		err := r.ParseForm()
		if err != nil {
			http.Error(w, "Invalid form data.", http.StatusBadRequest)
			log.Printf("[error] failed to parse form values: %v", err)
			return
		}

		email := r.PostFormValue("email")
		password := r.PostFormValue("password")

		user, err := db.GetUserWithPasswordByEmail(ctx, email)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			viewAuth.Error("Invalid email or password.").Render(ctx, w)
			return
		case err != nil:
			http.Error(w, "Server error.", http.StatusInternalServerError)
			log.Printf("[error] failed to retrieve user: %v", err)
			return
		}

		ok, err := auth.CheckPasswordHash(password, user.HashedPassword)
		if err != nil {
			http.Error(w, "Server error.", http.StatusInternalServerError)
			log.Printf("[error] cannot verify password — hash may be corrupted: %v", err)
			return
		}
		if !ok {
			viewAuth.Error("Invalid email or password.").Render(ctx, w)
			return
		}

		w.Header().Set("HX-Redirect", fmt.Sprintf("/chat?userid=%s", user.UserID))
		w.WriteHeader(http.StatusOK)
	}
}

func ServeSignup(db *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
ctx := r.Context()

		if r.Method != http.MethodPost {
			viewAuth.Signup().Render(ctx, w)
			return
		}

		err := r.ParseForm()
		if err != nil {
			http.Error(w, "Invalid form data.", http.StatusBadRequest)
			log.Printf("[error] failed to parse form values: %v", err)
			return
		}

		password := r.PostFormValue("password")
		confirm_pw := r.PostFormValue("confirm_password")

		// Validate password by comparing main and confirm.
		if password != confirm_pw {
			viewAuth.Error("Passwords do not match!").Render(ctx, w)
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
			log.Printf("[error] failed to create user entry in database: %v", err)
			return
		}

		// Hash and compare password before storing to database.
		hashed_pw, err := auth.HashPassword(password)
		if err != nil {
			http.Error(w, "Server error.", http.StatusInternalServerError)
			log.Printf("[error] argon2id hash creation failed: %v", err)
			return
		}

		ok, err := auth.CheckPasswordHash(password, hashed_pw)
		if err != nil {
			http.Error(w, "Server error.", http.StatusInternalServerError)
			log.Printf("[error] cannot verify password — hash may be corrupted: %v", err)
			return
		}

		if ok {
			_, err := db.CreatePassword(ctx, database.CreatePasswordParams{
				UserID:         user.UserID,
				HashedPassword: hashed_pw,
				CreatedAt:      pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
			})
			if err != nil {
				http.Error(w, "Database error.", http.StatusInternalServerError)
				log.Printf("[error] failed to create password entry in database: %v", err)
				return
			}
		}

		w.Header().Set("HX-Redirect", "/account/login")
		w.WriteHeader(http.StatusOK)
	}
}
