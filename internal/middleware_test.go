package internal

import (
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/johndosdos/chatter/internal/auth"
	"github.com/johndosdos/chatter/internal/database"
	"github.com/johndosdos/chatter/internal/testutil"
	"github.com/joho/godotenv"
)

func helper(t *testing.T,
	ctx context.Context,
	user database.User,
	queries *database.Queries,
	refreshTokenExp, jwtExp time.Duration,
	isCookieEmpty bool) (*http.Request, *httptest.ResponseRecorder) {

	req := httptest.NewRequest(http.MethodGet, "/chat", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	if isCookieEmpty {
		return req, rec
	}

	jwtStr, err := auth.MakeJWT(user.UserID.Bytes, os.Getenv("JWT_SECRET"), jwtExp)
	if err != nil {
		t.Fatalf("%+v", err)
	}

	refreshTokenStr, err := auth.MakeRefreshToken(ctx, queries,
		user.UserID.Bytes,
		refreshTokenExp)
	if err != nil {
		t.Fatalf("%+v", err)
	}

	req.AddCookie(&http.Cookie{Name: "jwt", Value: jwtStr})
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: refreshTokenStr})

	return req, rec
}

func TestMiddleware(t *testing.T) {
	if err := godotenv.Load(); err != nil {
		log.Printf("failed to load .env file: %+v", err)
	}

	db, dbForGoose, migDir := testutil.DbInit()
	testutil.DbGooseUp(dbForGoose, migDir)
	defer testutil.DbCleanup(db, migDir)

	queries := database.New(db)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	user, err := queries.CreateUser(ctx, database.CreateUserParams{
		UserID: pgtype.UUID{
			Bytes: uuid.New(),
			Valid: true,
		},
		Username: "dummy",
		Email:    "dummy@test.com",
	})
	if err != nil {
		t.Logf("%+v", err)
	}

	tests := []struct {
		Name              string
		jwtExp            time.Duration
		refreshTokenExp   time.Duration
		isCookieEmpty     bool
		wantHandlerCalled bool
		wantCode          int
	}{
		{"valid_JWT", 5 * time.Minute, 7 * 24 * time.Hour, false, true, http.StatusOK},
		{"expired_JWT", -1 * time.Second, 7 * 24 * time.Hour, false, true, http.StatusOK},
		{"exired_JWT_and_refresh_token", -1 * time.Second, -1 * time.Second, false, false, http.StatusSeeOther},
		{"empty_cookies", 0, 0, true, false, http.StatusSeeOther},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			req, rec := helper(t, ctx, user, queries, tt.refreshTokenExp, tt.jwtExp, tt.isCookieEmpty)

			isHandlerCalled := false
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				isHandlerCalled = true
				w.WriteHeader(http.StatusOK)
			})

			handler := Middleware(queries)(nextHandler)
			handler.ServeHTTP(rec, req)

			if isHandlerCalled != tt.wantHandlerCalled {
				t.Error("nextHandler was supposed to be called")
			}

			if rec.Code != tt.wantCode {
				t.Errorf("want %d, got %d", tt.wantCode, rec.Code)
			}
		})
	}
}
