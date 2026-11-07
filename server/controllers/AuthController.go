package controllers

import (
	"bytes"
	"encoding/base64"
	"errors"
	"image/png"
	"log/slog"
	"net/http"
	"time"

	"api/auth"
	"api/middleware"

	"github.com/alexedwards/argon2id"
	"github.com/jackc/pgx/v5"
	"github.com/pquerna/otp/totp"
)

// reuseGrace lets two browser tabs refresh at the same moment without the
// slower one looking like a stolen token.
const reuseGrace = 30 * time.Second

const (
	loginLimitPerEmail = 10
	loginLimitWindow   = 15 * time.Minute
)

// AuthController groups sign-in, sessions and two-factor.
type AuthController struct {
	*Deps
	dummyHash string
}

// NewAuthController returns a ready to use AuthController.
func NewAuthController(deps *Deps) (*AuthController, error) {
	hash, err := argon2id.CreateHash("not a real password", argon2id.DefaultParams)
	if err != nil {
		return nil, err
	}
	return &AuthController{Deps: deps, dummyHash: hash}, nil
}

// Login handles POST /auth/login. A wrong email and a wrong password get the
// same response and take the same time, so it can't reveal who has an account.
func (ac *AuthController) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Code     string `json:"code"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	req.Email = normalizeEmail(req.Email)

	// The per-IP limit can be dodged by changing address, this one can't.
	allowed, err := ac.Cache.Allow(r.Context(), "rate:login-email:"+req.Email, loginLimitPerEmail, loginLimitWindow)
	if err != nil {
		slog.Error("rate limit check failed", "error", err)
		writeError(w, http.StatusServiceUnavailable, "unavailable", "Try again in a moment.")
		return
	}
	if !allowed {
		writeError(w, http.StatusTooManyRequests, "rate_limited", "Too many attempts. Wait a few minutes and try again.")
		return
	}

	var (
		userID, hash string
		totpEnabled  bool
		sealed       []byte
		lastCounter  int64
	)
	err = ac.DB.QueryRow(r.Context(), `
		SELECT id::text, password_hash, totp_enabled, totp_secret, totp_last_counter
		FROM users WHERE email = $1`, req.Email).Scan(&userID, &hash, &totpEnabled, &sealed, &lastCounter)

	known := err == nil
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		writeServerError(w, r, err)
		return
	}
	if !known {
		hash = ac.dummyHash
	}

	if len(req.Password) > maxPasswordLength*4 {
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "That email or password is incorrect.")
		return
	}
	match, err := argon2id.ComparePasswordAndHash(req.Password, hash)
	if err != nil {
		writeServerError(w, r, err)
		return
	}
	if !known || !match {
		if known {
			ac.audit(r.Context(), userID, "sign_in_failed")
		}
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "That email or password is incorrect.")
		return
	}

	if totpEnabled {
		// only reached with the right password, so this reveals nothing to a guesser
		if req.Code == "" {
			writeError(w, http.StatusUnauthorized, "two_factor_required", "Enter the 6-digit code from your authenticator app.")
			return
		}
		ok, err := ac.consumeCode(r, userID, sealed, req.Code, lastCounter)
		if err != nil {
			writeServerError(w, r, err)
			return
		}
		if !ok {
			ac.audit(r.Context(), userID, "two_factor_failed")
			writeError(w, http.StatusUnauthorized, "invalid_code", "That code is incorrect or has already been used.")
			return
		}
	}

	if err := ac.startSession(w, r, userID); err != nil {
		writeServerError(w, r, err)
		return
	}
	ac.audit(r.Context(), userID, "signed_in")
	w.WriteHeader(http.StatusNoContent)
}

// consumeCode checks a two-factor code and marks it used. The conditional
// update means two requests racing with one code can't both succeed.
func (ac *AuthController) consumeCode(r *http.Request, userID string, sealed []byte, code string, lastCounter int64) (bool, error) {
	secret, err := auth.DecryptSecret(ac.Cfg.TOTPKey, sealed)
	if err != nil {
		return false, err
	}
	counter, ok := auth.VerifyTOTP(secret, code, lastCounter, time.Now())
	if !ok {
		return false, nil
	}
	tag, err := ac.DB.Exec(r.Context(), `
		UPDATE users SET totp_last_counter = $1 WHERE id = $2 AND totp_last_counter < $1`, counter, userID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// Refresh handles POST /auth/refresh. The refresh token changes on every use,
// so a stolen copy shows up as a reused token and the session is ended.
func (ac *AuthController) Refresh(w http.ResponseWriter, r *http.Request) {
	reject := func() {
		ac.clearCookies(w)
		writeError(w, http.StatusUnauthorized, "unauthorized", "Sign in to continue.")
	}

	cookie, err := r.Cookie(refreshCookie)
	if err != nil {
		reject()
		return
	}
	sessionID, secret, err := auth.SplitRefreshToken(cookie.Value)
	if err != nil || !uuidPattern.MatchString(sessionID) {
		reject()
		return
	}

	tx, err := ac.DB.Begin(r.Context())
	if err != nil {
		writeServerError(w, r, err)
		return
	}
	defer tx.Rollback(r.Context())

	var (
		userID            string
		current, previous []byte
		rotatedAt         *time.Time
		expiresAt         time.Time
		revokedAt         *time.Time
	)
	err = tx.QueryRow(r.Context(), `
		SELECT user_id::text, refresh_hash, previous_refresh_hash, rotated_at, expires_at, revoked_at
		FROM sessions WHERE id = $1 FOR UPDATE`, sessionID).
		Scan(&userID, &current, &previous, &rotatedAt, &expiresAt, &revokedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		reject()
		return
	}
	if err != nil {
		writeServerError(w, r, err)
		return
	}
	if revokedAt != nil || time.Now().After(expiresAt) {
		reject()
		return
	}

	presented := auth.HashRefreshSecret(secret)
	switch {
	case auth.HashesEqual(presented, current):
		newSecret, newHash := auth.NewRefreshSecret()
		if _, err := tx.Exec(r.Context(), `
			UPDATE sessions
			SET refresh_hash = $1, previous_refresh_hash = refresh_hash, rotated_at = now(), last_used_at = now()
			WHERE id = $2`, newHash, sessionID); err != nil {
			writeServerError(w, r, err)
			return
		}
		if err := tx.Commit(r.Context()); err != nil {
			writeServerError(w, r, err)
			return
		}
		if err := ac.setAccessCookie(w, userID, sessionID); err != nil {
			writeServerError(w, r, err)
			return
		}
		ac.setCookie(w, refreshCookie, auth.FormatRefreshToken(sessionID, newSecret), time.Until(expiresAt))

	case previous != nil && auth.HashesEqual(presented, previous) && rotatedAt != nil && time.Since(*rotatedAt) < reuseGrace:
		// The other tab already set the new refresh cookie, so only renew access.
		if err := ac.setAccessCookie(w, userID, sessionID); err != nil {
			writeServerError(w, r, err)
			return
		}

	default:
		if _, err := tx.Exec(r.Context(), "UPDATE sessions SET revoked_at = now() WHERE id = $1", sessionID); err != nil {
			writeServerError(w, r, err)
			return
		}
		if err := tx.Commit(r.Context()); err != nil {
			writeServerError(w, r, err)
			return
		}
		ac.audit(r.Context(), userID, "refresh_token_reused")
		reject()
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Logout handles POST /auth/logout. It works from the refresh cookie, so an
// expired access token can still sign out.
func (ac *AuthController) Logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(refreshCookie); err == nil {
		sessionID, secret, err := auth.SplitRefreshToken(cookie.Value)
		if err == nil && uuidPattern.MatchString(sessionID) {
			// the hash check stops anyone ending a session they only know the ID of
			if _, err := ac.DB.Exec(r.Context(), `
				UPDATE sessions SET revoked_at = now()
				WHERE id = $1 AND refresh_hash = $2 AND revoked_at IS NULL`,
				sessionID, auth.HashRefreshSecret(secret)); err != nil {
				writeServerError(w, r, err)
				return
			}
		}
	}
	ac.clearCookies(w)
	w.WriteHeader(http.StatusNoContent)
}

// Session is one signed-in device.
type Session struct {
	ID         string    `json:"id"`
	UserAgent  string    `json:"user_agent"`
	IP         string    `json:"ip"`
	CreatedAt  time.Time `json:"created_at"`
	LastUsedAt time.Time `json:"last_used_at"`
	Current    bool      `json:"current"`
}

// ListSessions handles GET /sessions.
func (ac *AuthController) ListSessions(w http.ResponseWriter, r *http.Request) {
	rows, err := ac.DB.Query(r.Context(), `
		SELECT id::text, user_agent, ip, created_at, last_used_at FROM sessions
		WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > now()
		ORDER BY last_used_at DESC`, middleware.UserID(r.Context()))
	if err != nil {
		writeServerError(w, r, err)
		return
	}

	sessions, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Session, error) {
		var s Session
		err := row.Scan(&s.ID, &s.UserAgent, &s.IP, &s.CreatedAt, &s.LastUsedAt)
		s.Current = s.ID == middleware.SessionID(r.Context())
		return s, err
	})
	if err != nil {
		writeServerError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, sessions)
}

// RevokeSession handles DELETE /sessions/{id}. Filtering on user_id means
// someone else's session looks the same as one that doesn't exist.
func (ac *AuthController) RevokeSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !uuidPattern.MatchString(id) {
		writeError(w, http.StatusNotFound, "not_found", "That session doesn't exist.")
		return
	}

	userID := middleware.UserID(r.Context())
	tag, err := ac.DB.Exec(r.Context(), `
		UPDATE sessions SET revoked_at = now()
		WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL`, id, userID)
	if err != nil {
		writeServerError(w, r, err)
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "not_found", "That session doesn't exist.")
		return
	}
	ac.audit(r.Context(), userID, "session_revoked")
	w.WriteHeader(http.StatusNoContent)
}

// Activity handles GET /activity: the latest security events for this user.
func (ac *AuthController) Activity(w http.ResponseWriter, r *http.Request) {
	type event struct {
		Action    string    `json:"action"`
		IP        string    `json:"ip"`
		CreatedAt time.Time `json:"created_at"`
	}

	rows, err := ac.DB.Query(r.Context(), `
		SELECT action, ip, created_at FROM audit_log
		WHERE user_id = $1 ORDER BY id DESC LIMIT 20`, middleware.UserID(r.Context()))
	if err != nil {
		writeServerError(w, r, err)
		return
	}
	events, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (event, error) {
		var e event
		return e, row.Scan(&e.Action, &e.IP, &e.CreatedAt)
	})
	if err != nil {
		writeServerError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, events)
}

// SetupTwoFactor handles POST /2fa/setup. It stores a new secret but leaves
// two-factor off until EnableTwoFactor proves the app has it.
func (ac *AuthController) SetupTwoFactor(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r.Context())

	var email string
	var enabled bool
	if err := ac.DB.QueryRow(r.Context(), "SELECT email, totp_enabled FROM users WHERE id = $1", userID).Scan(&email, &enabled); err != nil {
		writeServerError(w, r, err)
		return
	}
	if enabled {
		writeError(w, http.StatusConflict, "already_enabled", "Two-factor sign-in is already on.")
		return
	}

	key, err := totp.Generate(totp.GenerateOpts{Issuer: "Open Tally", AccountName: email})
	if err != nil {
		writeServerError(w, r, err)
		return
	}
	sealed, err := auth.EncryptSecret(ac.Cfg.TOTPKey, key.Secret())
	if err != nil {
		writeServerError(w, r, err)
		return
	}
	if _, err := ac.DB.Exec(r.Context(), "UPDATE users SET totp_secret = $1 WHERE id = $2", sealed, userID); err != nil {
		writeServerError(w, r, err)
		return
	}

	image, err := key.Image(240, 240)
	if err != nil {
		writeServerError(w, r, err)
		return
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, image); err != nil {
		writeServerError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"secret":  key.Secret(),
		"qr_code": "data:image/png;base64," + base64.StdEncoding.EncodeToString(encoded.Bytes()),
	})
}

// EnableTwoFactor handles POST /2fa/enable.
func (ac *AuthController) EnableTwoFactor(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Code string `json:"code"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}

	userID := middleware.UserID(r.Context())
	var sealed []byte
	var lastCounter int64
	if err := ac.DB.QueryRow(r.Context(), "SELECT totp_secret, totp_last_counter FROM users WHERE id = $1", userID).Scan(&sealed, &lastCounter); err != nil {
		writeServerError(w, r, err)
		return
	}
	if sealed == nil {
		writeError(w, http.StatusConflict, "not_set_up", "Start two-factor setup first.")
		return
	}

	ok, err := ac.consumeCode(r, userID, sealed, req.Code, lastCounter)
	if err != nil {
		writeServerError(w, r, err)
		return
	}
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid_code", "That code is incorrect or has already been used.")
		return
	}

	if _, err := ac.DB.Exec(r.Context(), "UPDATE users SET totp_enabled = true WHERE id = $1", userID); err != nil {
		writeServerError(w, r, err)
		return
	}
	ac.audit(r.Context(), userID, "two_factor_enabled")
	w.WriteHeader(http.StatusNoContent)
}

// DisableTwoFactor handles POST /2fa/disable. It needs the password and a code
// so a stolen session alone can't remove the protection.
func (ac *AuthController) DisableTwoFactor(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Password string `json:"password"`
		Code     string `json:"code"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}

	userID := middleware.UserID(r.Context())
	var hash string
	var sealed []byte
	var enabled bool
	var lastCounter int64
	if err := ac.DB.QueryRow(r.Context(), `
		SELECT password_hash, totp_secret, totp_enabled, totp_last_counter FROM users WHERE id = $1`,
		userID).Scan(&hash, &sealed, &enabled, &lastCounter); err != nil {
		writeServerError(w, r, err)
		return
	}
	if !enabled {
		writeError(w, http.StatusConflict, "not_enabled", "Two-factor sign-in is already off.")
		return
	}

	if len(req.Password) > maxPasswordLength*4 {
		writeError(w, http.StatusForbidden, "wrong_password", "Your password is incorrect.")
		return
	}
	match, err := argon2id.ComparePasswordAndHash(req.Password, hash)
	if err != nil {
		writeServerError(w, r, err)
		return
	}
	if !match {
		ac.audit(r.Context(), userID, "two_factor_disable_failed")
		writeError(w, http.StatusForbidden, "wrong_password", "Your password is incorrect.")
		return
	}

	ok, err := ac.consumeCode(r, userID, sealed, req.Code, lastCounter)
	if err != nil {
		writeServerError(w, r, err)
		return
	}
	if !ok {
		ac.audit(r.Context(), userID, "two_factor_disable_failed")
		writeError(w, http.StatusBadRequest, "invalid_code", "That code is incorrect or has already been used.")
		return
	}

	if _, err := ac.DB.Exec(r.Context(), `
		UPDATE users SET totp_enabled = false, totp_secret = NULL WHERE id = $1`, userID); err != nil {
		writeServerError(w, r, err)
		return
	}
	ac.audit(r.Context(), userID, "two_factor_disabled")
	w.WriteHeader(http.StatusNoContent)
}
