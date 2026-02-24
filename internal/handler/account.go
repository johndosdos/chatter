package handler

import (
	"log"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	viewAuth "github.com/johndosdos/chatter/components/auth"
	"github.com/johndosdos/chatter/internal/auth"
	"github.com/johndosdos/chatter/internal/database"
)

func ServeLoginPage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := viewAuth.Login().Render(r.Context(), w); err != nil {
			log.Printf("failed to render component: %v", err)
		}
	}
}

// SubmitLoginForm handles user login.
func SubmitLoginForm(db *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		err := r.ParseForm()
		if err != nil {
			http.Error(w, "Invalid form data.", http.StatusBadRequest)
			log.Printf("failed to parse form values: %v", err)
			return
		}

		email := r.PostFormValue("email")
		password := r.PostFormValue("password")

		user, err := db.GetUserWithPasswordByEmail(ctx, email)
		if err != nil {
			if err := viewAuth.ErrorMsgAuth("Invalid email or password.").Render(ctx, w); err != nil {
				log.Printf("failed to render component: %v", err)
				return
			}
			log.Printf("failed to retrieve user from db: %v", err)
			return
		}

		ok, err := auth.CheckPasswordHash(password, user.HashedPassword)
		if err != nil {
			http.Error(w, "Server error.", http.StatusInternalServerError)
			log.Printf("cannot verify password — hash may be corrupted: %v", err)
			return
		}
		if !ok {
			if err := viewAuth.ErrorMsgAuth("Invalid email or password.").Render(ctx, w); err != nil {
				log.Printf("failed to render component: %v", err)
			}
			return
		}

		refreshTokenExp := 7 * 24 * time.Hour
		jwtExp := 5 * time.Minute
		err = auth.SetTokensAndCookies(w, r, db,
			user.UserID.Bytes, refreshTokenExp, jwtExp)
		if err != nil {
			log.Printf("%v", err)
			return
		}

		w.Header().Set("HX-Redirect", "/chat")
		w.WriteHeader(http.StatusOK)

		slog.InfoContext(ctx, "user logged in",
			slog.String("username", user.Username))
	}
}

func ServeSignupPage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := viewAuth.Signup().Render(r.Context(), w); err != nil {
			log.Printf("failed to render component: %v", err)
		}
	}
}

// SubmitSignupForm handles user account creation.
func SubmitSignupForm(db *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		err := r.ParseForm()
		if err != nil {
			http.Error(w, "Invalid form data.", http.StatusBadRequest)
			log.Printf("failed to parse form values: %v", err)
			return
		}

		password := r.PostFormValue("password")
		confirmPw := r.PostFormValue("confirm_password")

		// Validate password by comparing main and confirm.
		if password != confirmPw {
			if err := viewAuth.ErrorMsgAuth("Passwords do not match!").Render(ctx, w); err != nil {
				log.Printf("failed to close connection: %v", err)
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
			log.Printf("failed to create user entry in database: %v", err)
			return
		}

		// Hash and compare password before storing to database.
		hashedPw, err := auth.HashPassword(password)
		if err != nil {
			http.Error(w, "Server error.", http.StatusInternalServerError)
			log.Printf("argon2id hash creation failed: %v", err)
			return
		}

		ok, err := auth.CheckPasswordHash(password, hashedPw)
		if err != nil {
			http.Error(w, "Server error.", http.StatusInternalServerError)
			log.Printf("cannot verify password — hash may be corrupted: %v", err)
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
				log.Printf("failed to create password entry in database: %v", err)
				return
			}
		}

		w.Header().Set("HX-Redirect", "/account/login")
		w.WriteHeader(http.StatusOK)

		slog.InfoContext(ctx, "user signed up",
			slog.String("username", user.Username))
	}
}

// SubmitLogoutReq deletes the user's assigned refresh token, and redirects
// the user to the login page.
func SubmitLogoutReq(db *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		refreshTok, err := r.Cookie("refresh_token")
		if err == nil {
			err = db.RevokeRefreshToken(ctx, refreshTok.Value)
			if err != nil {
				log.Printf("failed to process token deletion: %v", err)
			}
		}

		clearCookie := func(w http.ResponseWriter, name string) {
			http.SetCookie(w, &http.Cookie{
				Name:        name,
				Value:       "",
				Quoted:      false,
				Path:        "/",
				Domain:      "",
				Expires:     time.Time{},
				RawExpires:  "",
				MaxAge:      -1,
				Secure:      true,
				HttpOnly:    true,
				SameSite:    http.SameSiteLaxMode,
				Partitioned: false,
				Raw:         "",
				Unparsed:    []string{},
			})
		}

		clearCookie(w, "jwt")
		clearCookie(w, "refresh_token")
		w.Header().Set("HX-Redirect", "/account/login")
		w.WriteHeader(http.StatusOK)

		log.Printf("user logged out")
	}
}
