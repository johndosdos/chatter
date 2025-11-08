package auth

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/johndosdos/chatter/internal/database"
)

// RefreshSession handles issuance of JWT.
func RefreshSession(w http.ResponseWriter, r *http.Request, db *database.Queries) (uuid.UUID, error) {
	refreshTokCookie, err := r.Cookie("refresh_token")
	if err != nil {
		w.Header().Set("HX-Redirect", "/account/login")
		w.WriteHeader(http.StatusOK)
		return uuid.UUID{}, nil
	}

	userID, err := db.GetUserFromRefreshTok(r.Context(), refreshTokCookie.Value)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("internal/auth: failed to retrieve user from refresh token: %v", err)
	}

	jwt, err := MakeJWT(userID.Bytes, os.Getenv("JWT_SECRET"), 5*time.Minute)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("internal/auth: failed to make JWT: %v", err)
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

	return userID.Bytes, nil
}
