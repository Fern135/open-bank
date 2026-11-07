package controllers

import (
	"errors"
	"net/http"
	"net/mail"
	"strings"
	"unicode/utf8"

	"api/middleware"

	"github.com/alexedwards/argon2id"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	minPasswordLength = 12
	maxPasswordLength = 128 // hashing is deliberately slow, so bound the input
	maxNameLength     = 100
)

const uniqueViolation = "23505"

// UserController groups the handlers for sign-up and the signed-in user.
type UserController struct {
	*Deps
}

// User is what the API sends back. Never add a password field here without
// the `json:"-"` tag, or it will appear in responses.
type User struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Email            string `json:"email"`
	TwoFactorEnabled bool   `json:"two_factor_enabled"`
}

// createUserRequest is what clients may send. It's separate from User so a
// client can't choose its own ID.
type createUserRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// NewUserController returns a ready to use UserController.
func NewUserController(deps *Deps) *UserController {
	return &UserController{deps}
}

func validPassword(password string) bool {
	length := utf8.RuneCountInString(password)
	return length >= minPasswordLength && length <= maxPasswordLength
}

func validName(name string) bool {
	return name != "" && utf8.RuneCountInString(name) <= maxNameLength
}

// NewUser handles POST /users: it creates the user with a first checking
// account, signs them in and returns the user with a 201.
func (uc *UserController) NewUser(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Email = normalizeEmail(req.Email)

	if !validName(req.Name) {
		writeError(w, http.StatusBadRequest, "invalid_name", "Enter your name, up to 100 characters.")
		return
	}
	if address, err := mail.ParseAddress(req.Email); err != nil || address.Address != req.Email {
		writeError(w, http.StatusBadRequest, "invalid_email", "Enter a valid email address.")
		return
	}
	if !validPassword(req.Password) {
		writeError(w, http.StatusBadRequest, "invalid_password", "Use a password of 12 to 128 characters.")
		return
	}

	hash, err := argon2id.CreateHash(req.Password, argon2id.DefaultParams)
	if err != nil {
		writeServerError(w, r, err)
		return
	}

	tx, err := uc.DB.Begin(r.Context())
	if err != nil {
		writeServerError(w, r, err)
		return
	}
	defer tx.Rollback(r.Context())

	user := User{Name: req.Name, Email: req.Email}
	err = tx.QueryRow(r.Context(), `
		INSERT INTO users (name, email, password_hash) VALUES ($1, $2, $3)
		RETURNING id::text`, req.Name, req.Email, hash).Scan(&user.ID)

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
		writeError(w, http.StatusConflict, "email_taken", "An account with this email already exists. Sign in instead.")
		return
	}
	if err != nil {
		writeServerError(w, r, err)
		return
	}

	if _, err := insertAccount(r.Context(), tx, user.ID, "checking", "Checking"); err != nil {
		writeServerError(w, r, err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeServerError(w, r, err)
		return
	}

	if err := uc.startSession(w, r, user.ID); err != nil {
		writeServerError(w, r, err)
		return
	}
	uc.audit(r.Context(), user.ID, "account_created")
	writeJSON(w, http.StatusCreated, user)
}

// Me handles GET /me.
func (uc *UserController) Me(w http.ResponseWriter, r *http.Request) {
	var user User
	err := uc.DB.QueryRow(r.Context(), `
		SELECT id::text, name, email, totp_enabled FROM users WHERE id = $1`,
		middleware.UserID(r.Context())).Scan(&user.ID, &user.Name, &user.Email, &user.TwoFactorEnabled)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Sign in to continue.")
		return
	}
	if err != nil {
		writeServerError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, user)
}

// UpdateMe handles PATCH /me. Email can't change here, it's the sign-in name.
func (uc *UserController) UpdateMe(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if !validName(req.Name) {
		writeError(w, http.StatusBadRequest, "invalid_name", "Enter your name, up to 100 characters.")
		return
	}

	userID := middleware.UserID(r.Context())
	if _, err := uc.DB.Exec(r.Context(), "UPDATE users SET name = $1 WHERE id = $2", req.Name, userID); err != nil {
		writeServerError(w, r, err)
		return
	}
	uc.audit(r.Context(), userID, "profile_updated")
	uc.Me(w, r)
}

// ChangePassword handles POST /me/password. It signs out every other session,
// because someone changing their password may be locking out an intruder.
func (uc *UserController) ChangePassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}

	if !validPassword(req.NewPassword) {
		writeError(w, http.StatusBadRequest, "invalid_password", "Use a password of 12 to 128 characters.")
		return
	}

	userID := middleware.UserID(r.Context())
	var currentHash string
	if err := uc.DB.QueryRow(r.Context(), "SELECT password_hash FROM users WHERE id = $1", userID).Scan(&currentHash); err != nil {
		writeServerError(w, r, err)
		return
	}
	if len(req.CurrentPassword) > maxPasswordLength*4 {
		writeError(w, http.StatusForbidden, "wrong_password", "Your current password is incorrect.")
		return
	}
	match, err := argon2id.ComparePasswordAndHash(req.CurrentPassword, currentHash)
	if err != nil {
		writeServerError(w, r, err)
		return
	}
	if !match {
		uc.audit(r.Context(), userID, "password_change_failed")
		writeError(w, http.StatusForbidden, "wrong_password", "Your current password is incorrect.")
		return
	}

	hash, err := argon2id.CreateHash(req.NewPassword, argon2id.DefaultParams)
	if err != nil {
		writeServerError(w, r, err)
		return
	}

	tx, err := uc.DB.Begin(r.Context())
	if err != nil {
		writeServerError(w, r, err)
		return
	}
	defer tx.Rollback(r.Context())

	if _, err := tx.Exec(r.Context(), "UPDATE users SET password_hash = $1 WHERE id = $2", hash, userID); err != nil {
		writeServerError(w, r, err)
		return
	}
	if _, err := tx.Exec(r.Context(), `
		UPDATE sessions SET revoked_at = now()
		WHERE user_id = $1 AND id <> $2 AND revoked_at IS NULL`,
		userID, middleware.SessionID(r.Context())); err != nil {
		writeServerError(w, r, err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeServerError(w, r, err)
		return
	}

	uc.audit(r.Context(), userID, "password_changed")
	w.WriteHeader(http.StatusNoContent)
}
