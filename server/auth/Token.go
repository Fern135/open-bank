package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	AccessTTL  = 15 * time.Minute
	SessionTTL = 12 * time.Hour // absolute limit, refreshing doesn't extend it

	issuer = "open-tally"
)

// Claims is the payload of an access token.
type Claims struct {
	SessionID string `json:"sid"`
	jwt.RegisteredClaims
}

// IssueAccessToken signs a short-lived JWT for a session.
func IssueAccessToken(secret []byte, userID, sessionID string) (string, error) {
	now := time.Now()
	return jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			Issuer:    issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(AccessTTL)),
		},
	}).SignedString(secret)
}

// ParseAccessToken verifies a JWT. Pinning the method stops a forged token
// from choosing its own algorithm.
func ParseAccessToken(secret []byte, token string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(token, claims,
		func(*jwt.Token) (any, error) { return secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(issuer),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return nil, err
	}
	return claims, nil
}

// NewRefreshSecret returns a random secret and the hash to store. Only the
// hash reaches the database, so a database leak can't be used to sign in.
func NewRefreshSecret() (secret string, hash []byte) {
	raw := make([]byte, 32)
	rand.Read(raw) // never returns an error, it crashes the program instead
	secret = base64.RawURLEncoding.EncodeToString(raw)
	return secret, HashRefreshSecret(secret)
}

// HashRefreshSecret hashes a refresh secret. SHA-256 is enough here because
// the secret is 256 random bits, not a password someone could guess.
func HashRefreshSecret(secret string) []byte {
	sum := sha256.Sum256([]byte(secret))
	return sum[:]
}

// HashesEqual compares two hashes in constant time.
func HashesEqual(a, b []byte) bool {
	return subtle.ConstantTimeCompare(a, b) == 1
}

// FormatRefreshToken joins a session ID and secret into the cookie value.
func FormatRefreshToken(sessionID, secret string) string {
	return sessionID + "." + secret
}

// SplitRefreshToken is the reverse of FormatRefreshToken.
func SplitRefreshToken(token string) (sessionID, secret string, err error) {
	sessionID, secret, ok := strings.Cut(token, ".")
	if !ok || sessionID == "" || secret == "" {
		return "", "", errors.New("malformed refresh token")
	}
	return sessionID, secret, nil
}
