package config

import (
	"encoding/hex"
	"errors"
	"fmt"
	"os"
)

// defaultPort is the port the web app proxies to in development.
const defaultPort = "3001"

// Config holds the settings the server reads at startup.
type Config struct {
	Port         string // listen address in the form ":3001"
	DatabaseURL  string
	RedisURL     string
	JWTSecret    []byte // signs access tokens
	TOTPKey      []byte // AES-256 key that encrypts two-factor secrets at rest
	CookieSecure bool   // false only for plain http on localhost
	TrustProxy   bool   // true only when a proxy we control sets X-Forwarded-For
}

// New builds the Config from environment variables. It fails instead of
// falling back to a default for anything secret.
func New() (*Config, error) {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	cfg := &Config{
		// net/http needs the leading colon, PORT is just the number
		Port:         ":" + port,
		DatabaseURL:  os.Getenv("DATABASE_URL"),
		RedisURL:     os.Getenv("REDIS_URL"),
		JWTSecret:    []byte(os.Getenv("JWT_SECRET")),
		CookieSecure: os.Getenv("COOKIE_SECURE") != "false",
		TrustProxy:   os.Getenv("TRUST_PROXY") == "true",
	}

	if cfg.DatabaseURL == "" {
		return nil, errors.New("DATABASE_URL is required")
	}
	if cfg.RedisURL == "" {
		return nil, errors.New("REDIS_URL is required")
	}
	if len(cfg.JWTSecret) < 32 {
		return nil, errors.New("JWT_SECRET must be at least 32 characters")
	}

	key, err := hex.DecodeString(os.Getenv("TOTP_ENCRYPTION_KEY"))
	if err != nil || len(key) != 32 {
		return nil, fmt.Errorf("TOTP_ENCRYPTION_KEY must be 64 hex characters")
	}
	cfg.TOTPKey = key

	return cfg, nil
}
