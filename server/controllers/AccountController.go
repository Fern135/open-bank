package controllers

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"api/middleware"

	"github.com/jackc/pgx/v5"
)

const (
	maxAccountsPerUser = 10
	maxAccountName     = 50
	defaultEntryLimit  = 25
	maxEntryLimit      = 100

	// Short because a read that races a transfer can re-cache an old balance.
	accountsCacheTTL = 30 * time.Second
)

// AccountController groups the handlers for accounts, entries and statements.
type AccountController struct {
	*Deps
}

// Account is a customer account with its balance.
type Account struct {
	ID           string    `json:"id"`
	Number       string    `json:"number"`
	Type         string    `json:"type"`
	Name         string    `json:"name"`
	BalanceCents int64     `json:"balance_cents"`
	CreatedAt    time.Time `json:"created_at"`
}

// Entry is one side of a transaction, as seen by the account's owner.
type Entry struct {
	ID                int64     `json:"id"`
	TransactionID     string    `json:"transaction_id"`
	AccountID         string    `json:"account_id"`
	AccountName       string    `json:"account_name"`
	AmountCents       int64     `json:"amount_cents"`
	BalanceAfterCents int64     `json:"balance_after_cents"`
	Kind              string    `json:"kind"`
	Memo              string    `json:"memo"`
	Counterparty      string    `json:"counterparty"`
	CreatedAt         time.Time `json:"created_at"`
}

// NewAccountController returns a ready to use AccountController.
func NewAccountController(deps *Deps) *AccountController {
	return &AccountController{deps}
}

func accountsCacheKey(userID string) string {
	return "accounts:" + userID
}

// queryer is the part of pgx that both a pool and a transaction provide.
type queryer interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// insertAccount creates an account with a random number, trying again in the
// unlikely case the number is already taken.
func insertAccount(ctx context.Context, q queryer, userID, accountType, name string) (Account, error) {
	var lastErr error
	for range 5 {
		number, err := newAccountNumber()
		if err != nil {
			return Account{}, err
		}

		// ON CONFLICT because a failed insert would abort the caller's transaction
		account := Account{Number: number, Type: accountType, Name: name}
		err = q.QueryRow(ctx, `
			INSERT INTO accounts (user_id, number, type, name) VALUES ($1, $2, $3, $4)
			ON CONFLICT (number) DO NOTHING
			RETURNING id::text, created_at`, userID, number, accountType, name).
			Scan(&account.ID, &account.CreatedAt)
		if err == nil {
			return account, nil
		}
		lastErr = err
		if !errors.Is(err, pgx.ErrNoRows) {
			return Account{}, err
		}
	}
	return Account{}, lastErr
}

func newAccountNumber() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(9_000_000_000))
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(n.Int64()+1_000_000_000, 10), nil
}

// List handles GET /accounts.
func (ac *AccountController) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r.Context())

	var accounts []Account
	if ac.Cache.Get(r.Context(), accountsCacheKey(userID), &accounts) {
		writeJSON(w, http.StatusOK, accounts)
		return
	}

	rows, err := ac.DB.Query(r.Context(), `
		SELECT a.id::text, a.number, a.type, a.name, COALESCE(sum(e.amount_cents), 0)::bigint, a.created_at
		FROM accounts a
		LEFT JOIN entries e ON e.account_id = a.id
		WHERE a.user_id = $1
		GROUP BY a.id
		ORDER BY a.created_at`, userID)
	if err != nil {
		writeServerError(w, r, err)
		return
	}
	accounts, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (Account, error) {
		var a Account
		return a, row.Scan(&a.ID, &a.Number, &a.Type, &a.Name, &a.BalanceCents, &a.CreatedAt)
	})
	if err != nil {
		writeServerError(w, r, err)
		return
	}

	ac.Cache.Set(r.Context(), accountsCacheKey(userID), accounts, accountsCacheTTL)
	writeJSON(w, http.StatusOK, accounts)
}

// Create handles POST /accounts.
func (ac *AccountController) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Type string `json:"type"`
		Name string `json:"name"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Type != "checking" && req.Type != "savings" {
		writeError(w, http.StatusBadRequest, "invalid_type", "Choose checking or savings.")
		return
	}
	if req.Name == "" || utf8.RuneCountInString(req.Name) > maxAccountName {
		writeError(w, http.StatusBadRequest, "invalid_name", "Name the account, up to 50 characters.")
		return
	}

	userID := middleware.UserID(r.Context())

	tx, err := ac.DB.Begin(r.Context())
	if err != nil {
		writeServerError(w, r, err)
		return
	}
	defer tx.Rollback(r.Context())

	// locking the user makes the count below safe against parallel requests
	if _, err := tx.Exec(r.Context(), "SELECT 1 FROM users WHERE id = $1 FOR UPDATE", userID); err != nil {
		writeServerError(w, r, err)
		return
	}
	var count int
	if err := tx.QueryRow(r.Context(), "SELECT count(*) FROM accounts WHERE user_id = $1", userID).Scan(&count); err != nil {
		writeServerError(w, r, err)
		return
	}
	if count >= maxAccountsPerUser {
		writeError(w, http.StatusUnprocessableEntity, "too_many_accounts", fmt.Sprintf("You can have up to %d accounts.", maxAccountsPerUser))
		return
	}

	account, err := insertAccount(r.Context(), tx, userID, req.Type, req.Name)
	if err != nil {
		writeServerError(w, r, err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeServerError(w, r, err)
		return
	}

	ac.Cache.Delete(r.Context(), accountsCacheKey(userID))
	ac.audit(r.Context(), userID, "bank_account_opened")
	writeJSON(w, http.StatusCreated, account)
}

// loadOwned fetches an account only if it belongs to the signed-in user. The
// response for someone else's account is the same as for one that doesn't
// exist, so IDs can't be probed.
func (ac *AccountController) loadOwned(w http.ResponseWriter, r *http.Request) (Account, bool) {
	notFound := func() (Account, bool) {
		writeError(w, http.StatusNotFound, "not_found", "That account doesn't exist.")
		return Account{}, false
	}

	id := r.PathValue("id")
	if !uuidPattern.MatchString(id) {
		return notFound()
	}

	var a Account
	err := ac.DB.QueryRow(r.Context(), `
		SELECT a.id::text, a.number, a.type, a.name,
		       (SELECT COALESCE(sum(amount_cents), 0)::bigint FROM entries WHERE account_id = a.id),
		       a.created_at
		FROM accounts a
		WHERE a.id = $1 AND a.user_id = $2`, id, middleware.UserID(r.Context())).
		Scan(&a.ID, &a.Number, &a.Type, &a.Name, &a.BalanceCents, &a.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return notFound()
	}
	if err != nil {
		writeServerError(w, r, err)
		return Account{}, false
	}
	return a, true
}

// Get handles GET /accounts/{id}.
func (ac *AccountController) Get(w http.ResponseWriter, r *http.Request) {
	if account, ok := ac.loadOwned(w, r); ok {
		writeJSON(w, http.StatusOK, account)
	}
}

// entryFilter narrows listEntries. Zero values mean no filter.
type entryFilter struct {
	accountID string
	beforeID  int64
	from, to  time.Time
	limit     int
	ascending bool
}

// listEntries returns the user's entries, always scoped to their own accounts.
// The running balance is computed before filtering so it stays correct on
// every page.
func (d *Deps) listEntries(ctx context.Context, userID string, f entryFilter) ([]Entry, error) {
	order := "DESC"
	if f.ascending {
		order = "ASC"
	}

	rows, err := d.DB.Query(ctx, `
		WITH owned AS (
			SELECT e.id, e.transaction_id, e.account_id, e.amount_cents, e.created_at,
			       sum(e.amount_cents) OVER (PARTITION BY e.account_id ORDER BY e.id)::bigint AS balance_after
			FROM entries e
			JOIN accounts a ON a.id = e.account_id
			WHERE a.user_id = $1
			  AND ($2 = '' OR e.account_id = NULLIF($2, '')::uuid)
		)
		SELECT o.id, o.transaction_id::text, o.account_id::text, a.name, o.amount_cents, o.balance_after,
		       t.kind, t.memo, o.created_at,
		       (SELECT CASE
		                 WHEN oa.user_id = $1 THEN oa.name
		                 WHEN oa.user_id IS NULL THEN oa.name
		                 ELSE ou.name
		               END
		        FROM entries oe
		        JOIN accounts oa ON oa.id = oe.account_id
		        LEFT JOIN users ou ON ou.id = oa.user_id
		        WHERE oe.transaction_id = o.transaction_id AND oe.account_id <> o.account_id
		        LIMIT 1)
		FROM owned o
		JOIN accounts a ON a.id = o.account_id
		JOIN transactions t ON t.id = o.transaction_id
		WHERE ($3 = 0 OR o.id < $3)
		  AND ($4::timestamptz IS NULL OR o.created_at >= $4)
		  AND ($5::timestamptz IS NULL OR o.created_at < $5)
		ORDER BY o.id `+order+`
		LIMIT $6`,
		userID, f.accountID, f.beforeID, nullableTime(f.from), nullableTime(f.to), f.limit)
	if err != nil {
		return nil, err
	}

	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Entry, error) {
		var e Entry
		return e, row.Scan(&e.ID, &e.TransactionID, &e.AccountID, &e.AccountName, &e.AmountCents,
			&e.BalanceAfterCents, &e.Kind, &e.Memo, &e.CreatedAt, &e.Counterparty)
	})
}

func nullableTime(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

// parsePage reads the limit and before query parameters.
func parsePage(r *http.Request) (limit int, before int64) {
	limit = defaultEntryLimit
	if n, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && n > 0 && n <= maxEntryLimit {
		limit = n
	}
	if n, err := strconv.ParseInt(r.URL.Query().Get("before"), 10, 64); err == nil && n > 0 {
		before = n
	}
	return limit, before
}

// RecentEntries handles GET /entries, newest first across all the user's accounts.
func (ac *AccountController) RecentEntries(w http.ResponseWriter, r *http.Request) {
	limit, before := parsePage(r)
	entries, err := ac.listEntries(r.Context(), middleware.UserID(r.Context()), entryFilter{limit: limit, beforeID: before})
	if err != nil {
		writeServerError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, entries)
}

// Entries handles GET /accounts/{id}/entries, newest first.
func (ac *AccountController) Entries(w http.ResponseWriter, r *http.Request) {
	account, ok := ac.loadOwned(w, r)
	if !ok {
		return
	}

	limit, before := parsePage(r)
	entries, err := ac.listEntries(r.Context(), middleware.UserID(r.Context()), entryFilter{
		accountID: account.ID, limit: limit, beforeID: before,
	})
	if err != nil {
		writeServerError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, entries)
}

// Statement handles GET /accounts/{id}/statements/{month}, month as 2026-11.
func (ac *AccountController) Statement(w http.ResponseWriter, r *http.Request) {
	account, ok := ac.loadOwned(w, r)
	if !ok {
		return
	}

	start, err := time.ParseInLocation("2006-01", r.PathValue("month"), time.UTC)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_month", "Use a month like 2026-11.")
		return
	}
	end := start.AddDate(0, 1, 0)

	var opening int64
	if err := ac.DB.QueryRow(r.Context(), `
		SELECT COALESCE(sum(amount_cents), 0)::bigint FROM entries
		WHERE account_id = $1 AND created_at < $2`, account.ID, start).Scan(&opening); err != nil {
		writeServerError(w, r, err)
		return
	}

	// shortcut: a statement shows at most 1000 entries, paginate it if an account ever exceeds that in a month.
	entries, err := ac.listEntries(r.Context(), middleware.UserID(r.Context()), entryFilter{
		accountID: account.ID, from: start, to: end, limit: 1000, ascending: true,
	})
	if err != nil {
		writeServerError(w, r, err)
		return
	}

	closing := opening
	var credits, debits int64
	for _, entry := range entries {
		closing += entry.AmountCents
		if entry.AmountCents > 0 {
			credits += entry.AmountCents
		} else {
			debits -= entry.AmountCents
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"account":       account,
		"month":         r.PathValue("month"),
		"opening_cents": opening,
		"closing_cents": closing,
		"credits_cents": credits,
		"debits_cents":  debits,
		"entries":       entries,
	})
}
