// Package auth provides password hashing, session management and HTTP
// authentication middleware for mdtree.
package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword returns a bcrypt hash of the given plaintext password.
func HashPassword(plain string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(h), nil
}

// VerifyPassword reports whether plain matches the given bcrypt hash. The
// comparison is constant-time, as provided by the bcrypt package.
func VerifyPassword(hash, plain string) bool {
	if hash == "" {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

// GeneratePassword returns a cryptographically random, URL-safe password.
// nBytes of entropy are drawn from the operating system CSPRNG.
func GeneratePassword(nBytes int) (string, error) {
	return randomToken(nBytes)
}

// randomToken returns a URL-safe random token drawn from the system CSPRNG.
func randomToken(nBytes int) (string, error) {
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("read CSPRNG: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
