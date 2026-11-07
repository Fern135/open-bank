package controllers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"

	"api/auth"
	"api/cache"
	"api/config"
	"api/middleware"

	"github.com/jackc/pgx/v5/pgxpool"
)

// maxBodyBytes caps request bodies so a huge upload can't exhaust memory.
const maxBodyBytes = 1 << 20 // 1 MB

const refreshCookie = "refresh_token"

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// Deps is what every controller needs.
type Deps struct {
	DB    *pgxpool.Pool
	Cache *cache.Cache
	Cfg   *config.Config
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]string{"code": code, "error": message})
}

// writeServerError logs the real cause and tells the client nothing about it.
func writeServerError(w http.ResponseWriter, r *http.Request, err error) {
	slog.Error("request failed", "path", r.URL.Path, "error", err)
	writeError(w, http.StatusInternalServerError, "server_error", "Something went wrong on our side. Nothing was changed.")
}

// decodeJSON reads the body into dest, and writes the error response if it can't.
func decodeJSON(w http.ResponseWriter, r *http.Request, dest any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields() // reject fields we don't expect, like "id"
	if err := dec.Decode(dest); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "The request body is not valid.")
		return false
	}
	return true
}

// audit records a security event. A failed write is logged, not fatal: the
// action itself has already happened.
func (d *Deps) audit(ctx context.Context, userID, action string) {
	_, err := d.DB.Exec(ctx, "INSERT INTO audit_log (user_id, action, ip) VALUES ($1, $2, $3)",
		userID, action, middleware.ClientIP(ctx))
	if err != nil {
		slog.Error("audit write failed", "action", action, "error", err)
	}
}

// startSession creates a session and sets the sign-in cookies.
func (d *Deps) startSession(w http.ResponseWriter, r *http.Request, userID string) error {
	secret, hash := auth.NewRefreshSecret()

	var sessionID string
	err := d.DB.QueryRow(r.Context(), `
		INSERT INTO sessions (user_id, refresh_hash, user_agent, ip, expires_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id::text`,
		userID, hash, truncate(r.UserAgent(), 256), middleware.ClientIP(r.Context()),
		time.Now().Add(auth.SessionTTL)).Scan(&sessionID)
	if err != nil {
		return err
	}

	if err := d.setAccessCookie(w, userID, sessionID); err != nil {
		return err
	}
	d.setCookie(w, refreshCookie, auth.FormatRefreshToken(sessionID, secret), auth.SessionTTL)
	return nil
}

func (d *Deps) setAccessCookie(w http.ResponseWriter, userID, sessionID string) error {
	token, err := auth.IssueAccessToken(d.Cfg.JWTSecret, userID, sessionID)
	if err != nil {
		return err
	}
	d.setCookie(w, middleware.AccessCookie, token, auth.AccessTTL)
	return nil
}

func (d *Deps) setCookie(w http.ResponseWriter, name, value string, ttl time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		MaxAge:   int(ttl.Seconds()),
		HttpOnly: true,
		Secure:   d.Cfg.CookieSecure,
		SameSite: http.SameSiteStrictMode,
	})
}

func (d *Deps) clearCookies(w http.ResponseWriter) {
	for _, name := range []string{middleware.AccessCookie, refreshCookie} {
		http.SetCookie(w, &http.Cookie{
			Name:     name,
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
			Secure:   d.Cfg.CookieSecure,
			SameSite: http.SameSiteStrictMode,
		})
	}
}

func truncate(value string, length int) string {
	runes := []rune(value)
	if len(runes) > length {
		return string(runes[:length])
	}
	return value
}

// normalizeEmail makes addresses comparable, so one person can't sign up twice
// by changing the capitalisation.
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
