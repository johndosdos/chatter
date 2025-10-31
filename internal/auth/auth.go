package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/johndosdos/chatter/internal/database"
)

type ContextKey string

const UserIdKey ContextKey = "userId"

func HashPassword(password string) (string, error) {
	hashed_pw, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil {
		return "", fmt.Errorf("internal/auth: pw hash failed: %w", err)
	}

	return hashed_pw, nil
}

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

func MakeJWT(userId uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	now := time.Now().UTC()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    os.Getenv("JWT_ISS"),
		Subject:   userId.String(),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(expiresIn)),
	})

	return token.SignedString([]byte(tokenSecret))
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	claims := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(t *jwt.Token) (any, error) { return []byte(tokenSecret), nil },
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

func MakeRefreshToken(ctx context.Context, db *database.Queries) (string, error) {
	rnd := make([]byte, 32)

	// rand.Read() never returns an error.
	_, _ = rand.Read(rnd)
	rndStr := hex.EncodeToString(rnd)

	userId := ctx.Value(UserIdKey).(uuid.UUID)
	refreshTokenExp := time.Now().UTC().AddDate(0, 0, 7)
	refreshToken, err := db.CreateRefreshToken(ctx, database.CreateRefreshTokenParams{
		Token:     rndStr,
		CreatedAt: pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
		UserID:    pgtype.UUID{Bytes: userId, Valid: true},
		ExpiresAt: pgtype.Timestamptz{Time: refreshTokenExp, Valid: true},
	})
	if err != nil {
		return "", fmt.Errorf("database error: %w", err)
	}

	return refreshToken.Token, nil
}
