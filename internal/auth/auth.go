package auth

import (
	"fmt"

	"github.com/alexedwards/argon2id"
)

func HashPassword(password string) (string, error) {
	hashed_pw, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil {
		return "", fmt.Errorf("[error] failed to hash password: %v", err)
	}

	return hashed_pw, nil
}

func CheckPasswordHash(password, hash string) (bool, error) {
	isMatch, err := argon2id.ComparePasswordAndHash(password, hash)
	if err != nil {
		return false, fmt.Errorf("[error] original and hashed password don't match: %v", err)
	}

	return isMatch, nil
}
