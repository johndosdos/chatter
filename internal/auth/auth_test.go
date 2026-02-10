package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/johndosdos/chatter/internal/database"
	"github.com/johndosdos/chatter/internal/testutil"
)

func TestHashPassword(t *testing.T) {
	t.Run("unique hashes", func(t *testing.T) {
		pw := "password1234"
		hash, err := HashPassword(pw)
		if err != nil {
			t.Fatalf("password hash fail #1: %+v", err)
		}

		hash2, err := HashPassword(pw)
		if err != nil {
			t.Fatalf("password hash fail #2: %+v", err)
		}

		if hash == hash2 {
			t.Fatalf("hash and hash2 are the same hashes; should be different: %s, %s", hash, hash2)
		}
	})

	t.Run("empty password", func(t *testing.T) {
		_, err := HashPassword("")
		if err != nil {
			t.Errorf("HashPassword() failed on empty string: %+v", err)
		}
	})
}

func TestCheckPasswordHash(t *testing.T) {
	tests := []struct {
		name      string
		password  string
		checkPw   string
		hash      string
		wantErr   bool
		wantMatch bool
	}{
		{"correct pw", "mypassword1234", "mypassword1234", "", false, true},
		{"incorrect pw", "mypassword1234", "passwordDD1234", "", false, false},
		{"wrong hash", "mypassword1234", "passwordDD1234", "not-a-hash", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var hash string
			var err error

			if tt.hash != "" {
				hash = tt.hash
			} else {
				hash, err = HashPassword(tt.password)
				if err != nil {
					t.Fatalf("%+v", err)
				}
			}

			isMatch, err := CheckPasswordHash(tt.checkPw, hash)
			if (err != nil) != tt.wantErr {
				t.Fatalf("CheckPasswordHash() error = %+v", err)
			}
			if isMatch != tt.wantMatch {
				t.Errorf("password and hash don't match")
			}
		})
	}

}

func TestJWT(t *testing.T) {
	t.Run("Valid_JWT", func(t *testing.T) {
		userID := uuid.New()
		tokenSecret := "validtokensecret"
		expiration := 15 * time.Second
		tokenString, err := MakeJWT(userID, tokenSecret, expiration)
		if err != nil {
			t.Fatalf("MakeJWT() error = %+v", err)
		}
		gotUserID, err := ValidateJWT(tokenString, tokenSecret)
		if err != nil {
			t.Fatalf("ValidateJWT() error = %+v", err)
		}
		if gotUserID != userID {
			t.Errorf("want = %+v, got = %+v", userID, gotUserID)
		}
	})

	t.Run("Incorrect_secret", func(t *testing.T) {
		userID := uuid.New()
		tokenSecret := "validtokensecret"
		expiration := 15 * time.Second
		tokenString, err := MakeJWT(userID, tokenSecret, expiration)
		if err != nil {
			t.Fatalf("MakeJWT() error = %+v", err)
		}
		fakeSecret := "fakesecret"
		_, err = ValidateJWT(tokenString, fakeSecret)
		if err == nil {
			t.Fatalf("ValidateJWT() error = %+v", err)
		}
	})

	t.Run("Expired_token", func(t *testing.T) {
		userID := uuid.New()
		tokenSecret := "validtokensecret"
		expiration := -1 * time.Second
		tokenString, err := MakeJWT(userID, tokenSecret, expiration)
		if err != nil {
			t.Fatalf("MakeJWT() error = %+v", err)
		}
		_, err = ValidateJWT(tokenString, tokenSecret)
		if err == nil {
			t.Fatalf("ValidateJWT() error = %+v", err)
		}
	})

	t.Run("Corrupt_token", func(t *testing.T) {
		tokenSecret := "validtokensecret"
		tokenString := "corrupttoken"
		_, err := ValidateJWT(tokenString, tokenSecret)
		if err == nil {
			t.Fatalf("ValidateJWT() error = %+v", err)
		}
	})
}

func TestGetUserFromContext(t *testing.T) {
	t.Run("is_valid_UUID", func(t *testing.T) {
		wantUserID := uuid.New()
		ctx := context.WithValue(context.Background(), UserIDKey, wantUserID)
		gotUserID, err := GetUserFromContext(ctx)
		if err != nil {
			t.Fatalf("GetUserFromContext(): expected userID but got error = %+v", err)
		}
		if gotUserID.String() != wantUserID.String() {
			t.Errorf("want %+v but got %+v", wantUserID, gotUserID)
		}
	})

	t.Run("invalid_UUID", func(t *testing.T) {
		wantUserID := "not-UUID"
		ctx := context.WithValue(context.Background(), UserIDKey, wantUserID)
		_, err := GetUserFromContext(ctx)
		if err == nil {
			t.Fatal("GetUserFromContext(): expected error but got none")
		}
	})

	t.Run("empty_context_value", func(t *testing.T) {
		wantUserID := ""
		ctx := context.WithValue(context.Background(), UserIDKey, wantUserID)
		_, err := GetUserFromContext(ctx)
		if err == nil {
			t.Fatal("GetUserFromContext(): expected error but got none")
		}
	})

	t.Run("no_context", func(t *testing.T) {
		ctx := context.Background()
		_, err := GetUserFromContext(ctx)
		if err == nil {
			t.Fatal("GetUserFromContext(): expected error but got none")
		}
	})
}

func TestMakeRefreshToken(t *testing.T) {
	db, dbForGoose, migDir := testutil.DbInit()
	testutil.DbGooseUp(dbForGoose, migDir)
	defer testutil.DbCleanup(db, migDir)

	queries := database.New(db)

	user, err := queries.CreateUser(context.Background(), database.CreateUserParams{
		UserID: pgtype.UUID{
			Bytes: uuid.New(),
			Valid: true,
		},
		Username: "dummy",
		Email:    "dummy@test.com",
	})
	if err != nil {
		log.Fatalf("failed to create user: %+v", err)
	}

	t.Run("valid_refresh_token", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)

		refreshTokenExp := 7 * 24 * time.Hour
		tokenString, err := MakeRefreshToken(ctx, queries, user.UserID.Bytes, refreshTokenExp)
		if err != nil {
			t.Fatalf("MakeRefreshToken() unexpected error = %+v", err)
		}

		tokenFromDB, err := queries.GetRefreshToken(ctx, tokenString)
		if err != nil {
			t.Fatalf("dbQueries.GetRefreshToken() unexpected error = %+v", err)
		}

		if tokenFromDB.Token != tokenString {
			t.Errorf("got = %s, want = %s", tokenFromDB.Token, tokenString)
		}

		cancel()
	})

	t.Run("token_not_found_in_DB", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)

		tokenString := "invalid-refresh-token"
		tokenFromDB, err := queries.GetRefreshToken(ctx, tokenString)
		if err == nil {
			t.Fatalf("dbQueries.GetRefreshToken() unexpected error = %+v", err)
		}

		if tokenFromDB.Token != tokenString {
			t.Logf("refresh token not found in the db: %s", tokenString)
		}

		cancel()
	})

	t.Run("expired_token", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)

		refreshToken, err := MakeRefreshToken(ctx,
			queries,
			user.UserID.Bytes,
			-1*time.Millisecond)
		if err != nil {
			t.Fatalf("%+v", err)
		}

		tokenFromDB, err := queries.GetRefreshToken(ctx, refreshToken)
		if err == nil {
			if !tokenFromDB.Valid.Bool {
				t.Logf("expired refresh token: created at %v, expired at %v",
					tokenFromDB.CreatedAt.Time,
					tokenFromDB.ExpiresAt.Time)
			}
		}

		cancel()
	})

	t.Run("tampered_token_userID", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)

		rnd := make([]byte, 32)

		// rand.Read() never returns an error.
		_, _ = rand.Read(rnd)
		rndStr := hex.EncodeToString(rnd)

		token, err := queries.CreateRefreshToken(ctx, database.CreateRefreshTokenParams{
			Token:     rndStr,
			CreatedAt: pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
			UserID:    pgtype.UUID{Bytes: user.UserID.Bytes, Valid: true},
			ExpiresAt: pgtype.Timestamptz{Time: time.Now().UTC().Add(1 * time.Millisecond), Valid: true},
		})
		if err != nil {
			t.Fatalf("database error: %v", err)
		}

		token.UserID.Bytes = uuid.New()

		_, err = queries.GetRefreshToken(ctx, token.Token)
		if err == nil {
			t.Logf("tampred refresh token: userID at creation = %s, got = %s",
				user.UserID.String(),
				token.UserID.String())
		}

		cancel()
	})
}
