// Package auth provides functions related to password hashing and session
// tokens.
package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/johndosdos/chatter/internal/database"
)

// The ContextKey type is meant for passing userID as key for
// context.WithValue.
type ContextKey string

// UserIDKey implements the ContextKey type.
const UserIDKey ContextKey = "userId"

// HashPassword returns the hashed password created using the argon2id
// package.
func HashPassword(password string) (string, error) {
	hashedPw, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil {
		return "", fmt.Errorf("internal/auth: pw hash failed: %w", err)
	}

	return hashedPw, nil
}

// CheckPasswordHash compares the password and the hash. It returns true
// when they match, otherwise it returns false.
func CheckPasswordHash(password, hash string) (bool, error) {
	isMatch, err := argon2id.ComparePasswordAndHash(password, hash)
	if err != nil {
		return false, fmt.Errorf("internal/auth: pw and hash comparison failed: %w", err)
	}
	if !isMatch {
		return false, errors.New("internal/auth: pw and hash do not match")
	}

	return isMatch, nil
}

// MakeJWT returns a JSON Web Token string to be used as an acess token
// for client session.
func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	now := time.Now().UTC()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    os.Getenv("JWT_ISS"),
		Subject:   userID.String(),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(expiresIn)),
	})

	return token.SignedString([]byte(tokenSecret))
}

// ValidateJWT tries to validate the access token. It returns the user id
// as a uuid.UUID type. The returned error is a uuid.Parse error.
func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	claims := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(*jwt.Token) (any, error) { return []byte(tokenSecret), nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("internal/auth: failed to parse token: %w", err)
	}

	if !token.Valid {
		return uuid.UUID{}, errors.New("internal/auth: token is invalid")
	}

	if claims.Subject == "" {
		return uuid.UUID{}, errors.New("subject claim is missing")
	}

	userid, _ := token.Claims.GetSubject()
	return uuid.Parse(userid)
}

// MakeRefreshToken returns a refresh token string, while also storing the
// token to the database.
func MakeRefreshToken(ctx context.Context, db *database.Queries, userID uuid.UUID) (string, error) {
	rnd := make([]byte, 32)

	// rand.Read() never returns an error.
	_, _ = rand.Read(rnd)
	rndStr := hex.EncodeToString(rnd)

	refreshTokenExp := time.Now().UTC().AddDate(0, 0, 7)
	refreshToken, err := db.CreateRefreshToken(ctx, database.CreateRefreshTokenParams{
		Token:     rndStr,
		CreatedAt: pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
		UserID:    pgtype.UUID{Bytes: userID, Valid: true},
		ExpiresAt: pgtype.Timestamptz{Time: refreshTokenExp, Valid: true},
	})
	if err != nil {
		return "", fmt.Errorf("database error: %w", err)
	}

	return refreshToken.Token, nil
}

// GetUserFromContext validates r.Context.Value if it exists and returns
// the user's uuid, otherwise it returns an error.
func GetUserFromContext(ctx context.Context) (uuid.UUID, error) {
	userID, ok := ctx.Value(UserIDKey).(uuid.UUID)
	if !ok {
		return uuid.UUID{}, errors.New("failed to assert UserIDKey to UUID")
	}

	return userID, nil
}

// SetTokensAndCookies creates new JWTs and refresh tokens, and set the HTTP
// response cookies.
func SetTokensAndCookies(w http.ResponseWriter, r *http.Request, db *database.Queries, userID uuid.UUID) error {
	refreshToken, err := MakeRefreshToken(r.Context(), db, userID)
	if err != nil {
		return fmt.Errorf("internal/auth: failed to create refresh token: %v", err)
	}

	jwt, err := MakeJWT(userID, os.Getenv("JWT_SECRET"), 5*time.Minute)
	if err != nil {
		return fmt.Errorf("internal/auth: failed to make JWT: %v", err)
	}

	// Set cookie for access token. Expires in 5 minutes.
	http.SetCookie(w, &http.Cookie{
		Name:        "jwt",
		Value:       jwt,
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

	// Set another cookie for refresh tokens. Expires in 7 days.
	http.SetCookie(w, &http.Cookie{
		Name:        "refresh_token",
		Value:       refreshToken,
		Quoted:      false,
		Path:        "/",
		Domain:      "",
		Expires:     time.Time{},
		RawExpires:  "",
		MaxAge:      7 * 24 * 60 * 60,
		Secure:      true,
		HttpOnly:    true,
		SameSite:    http.SameSiteStrictMode,
		Partitioned: false,
		Raw:         "",
		Unparsed:    []string{},
	})

	return nil
}
