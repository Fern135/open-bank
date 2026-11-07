package controllers

import (
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	"api/ledger"
	"api/middleware"

	"github.com/jackc/pgx/v5"
)

const (
	maxDepositCents  = 1_000_000   // $10,000 per simulated deposit
	maxTransferCents = 100_000_000 // $1,000,000
	maxMemoLength    = 140
)

// TransferController groups the handlers that move money.
type TransferController struct {
	*Deps
}

// NewTransferController returns a ready to use TransferController.
func NewTransferController(deps *Deps) *TransferController {
	return &TransferController{deps}
}

// idempotencyKey reads the header every money request must carry.
func idempotencyKey(w http.ResponseWriter, r *http.Request) (string, bool) {
	key := r.Header.Get("Idempotency-Key")
	if key == "" || len(key) > 64 {
		writeError(w, http.StatusBadRequest, "missing_idempotency_key", "Send an Idempotency-Key header of up to 64 characters.")
		return "", false
	}
	return key, true
}

func (tc *TransferController) ownsAccount(r *http.Request, accountID string) (bool, error) {
	var owned bool
	err := tc.DB.QueryRow(r.Context(), `
		SELECT EXISTS (SELECT 1 FROM accounts WHERE id = $1 AND user_id = $2)`,
		accountID, middleware.UserID(r.Context())).Scan(&owned)
	return owned, err
}

// post writes the posting, clears stale balances and sends the response.
func (tc *TransferController) post(w http.ResponseWriter, r *http.Request, p ledger.Posting, recipientID string) {
	id, err := ledger.Post(r.Context(), tc.DB, p)
	switch {
	case errors.Is(err, ledger.ErrInsufficientFunds):
		writeError(w, http.StatusUnprocessableEntity, "insufficient_funds", "This account doesn't have enough money for that. Nothing was posted.")
		return
	case errors.Is(err, ledger.ErrDailyLimit):
		writeError(w, http.StatusUnprocessableEntity, "daily_limit", "This would pass the $25,000 daily limit for sending to other people. Nothing was posted.")
		return
	case err != nil:
		writeServerError(w, r, err)
		return
	}

	tc.Cache.Delete(r.Context(), accountsCacheKey(p.InitiatedBy), accountsCacheKey(recipientID))
	writeJSON(w, http.StatusCreated, map[string]string{"transaction_id": id})
}

// Deposit handles POST /deposits: simulated money, posted against the bank's
// funding account so the ledger still balances.
func (tc *TransferController) Deposit(w http.ResponseWriter, r *http.Request) {
	key, ok := idempotencyKey(w, r)
	if !ok {
		return
	}

	var req struct {
		AccountID   string `json:"account_id"`
		AmountCents int64  `json:"amount_cents"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}

	if req.AmountCents <= 0 || req.AmountCents > maxDepositCents {
		writeError(w, http.StatusBadRequest, "invalid_amount", "Enter an amount from $0.01 to $10,000.00.")
		return
	}
	if !uuidPattern.MatchString(req.AccountID) {
		writeError(w, http.StatusNotFound, "not_found", "That account doesn't exist.")
		return
	}
	owned, err := tc.ownsAccount(r, req.AccountID)
	if err != nil {
		writeServerError(w, r, err)
		return
	}
	if !owned {
		writeError(w, http.StatusNotFound, "not_found", "That account doesn't exist.")
		return
	}

	userID := middleware.UserID(r.Context())
	tc.post(w, r, ledger.Posting{
		Kind:           ledger.KindDeposit,
		Memo:           "Simulated deposit",
		InitiatedBy:    userID,
		IdempotencyKey: key,
		FromAccountID:  ledger.FundingAccountID,
		ToAccountID:    req.AccountID,
		AmountCents:    req.AmountCents,
	}, userID)
}

type transferRequest struct {
	FromAccountID   string `json:"from_account_id"`
	ToAccountID     string `json:"to_account_id"`     // one of the sender's own accounts
	ToAccountNumber string `json:"to_account_number"` // someone else's account
	ToEmail         string `json:"to_email"`          // someone else's oldest checking account
	AmountCents     int64  `json:"amount_cents"`
	Memo            string `json:"memo"`
}

// countTargets reports how many of the three recipient fields are filled in.
func (t transferRequest) countTargets() int {
	count := 0
	for _, value := range []string{t.ToAccountID, t.ToAccountNumber, t.ToEmail} {
		if value != "" {
			count++
		}
	}
	return count
}

// resolveRecipient finds the destination account and its owner.
func (tc *TransferController) resolveRecipient(r *http.Request, t transferRequest) (accountID, ownerID string, err error) {
	switch {
	case t.ToAccountID != "":
		if !uuidPattern.MatchString(t.ToAccountID) {
			return "", "", pgx.ErrNoRows
		}
		// to_account_id is only for moving between your own accounts
		err = tc.DB.QueryRow(r.Context(), `
			SELECT id::text, user_id::text FROM accounts WHERE id = $1 AND user_id = $2`,
			t.ToAccountID, middleware.UserID(r.Context())).Scan(&accountID, &ownerID)
	case t.ToAccountNumber != "":
		err = tc.DB.QueryRow(r.Context(), `
			SELECT id::text, user_id::text FROM accounts
			WHERE number = $1 AND type <> 'funding'`,
			strings.TrimSpace(t.ToAccountNumber)).Scan(&accountID, &ownerID)
	default:
		err = tc.DB.QueryRow(r.Context(), `
			SELECT a.id::text, a.user_id::text FROM accounts a
			JOIN users u ON u.id = a.user_id
			WHERE u.email = $1 AND a.type = 'checking'
			ORDER BY a.created_at LIMIT 1`,
			normalizeEmail(t.ToEmail)).Scan(&accountID, &ownerID)
	}
	return accountID, ownerID, err
}

// Transfer handles POST /transfers.
func (tc *TransferController) Transfer(w http.ResponseWriter, r *http.Request) {
	key, ok := idempotencyKey(w, r)
	if !ok {
		return
	}

	var req transferRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	req.Memo = strings.TrimSpace(req.Memo)
	if req.AmountCents <= 0 || req.AmountCents > maxTransferCents {
		writeError(w, http.StatusBadRequest, "invalid_amount", "Enter an amount from $0.01 to $1,000,000.00.")
		return
	}
	if utf8.RuneCountInString(req.Memo) > maxMemoLength {
		writeError(w, http.StatusBadRequest, "invalid_memo", "Keep the memo to 140 characters.")
		return
	}
	if req.countTargets() != 1 {
		writeError(w, http.StatusBadRequest, "invalid_recipient", "Choose one recipient.")
		return
	}
	if !uuidPattern.MatchString(req.FromAccountID) {
		writeError(w, http.StatusNotFound, "not_found", "That account doesn't exist.")
		return
	}

	owned, err := tc.ownsAccount(r, req.FromAccountID)
	if err != nil {
		writeServerError(w, r, err)
		return
	}
	if !owned {
		writeError(w, http.StatusNotFound, "not_found", "That account doesn't exist.")
		return
	}

	toAccountID, recipientID, err := tc.resolveRecipient(r, req)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusUnprocessableEntity, "recipient_not_found", "No Open Tally account matches that recipient. Check it and try again.")
		return
	}
	if err != nil {
		writeServerError(w, r, err)
		return
	}
	if toAccountID == req.FromAccountID {
		writeError(w, http.StatusBadRequest, "same_account", "Choose two different accounts.")
		return
	}

	userID := middleware.UserID(r.Context())
	kind := ledger.KindExternalTransfer
	if recipientID == userID {
		kind = ledger.KindInternalTransfer
	}

	tc.post(w, r, ledger.Posting{
		Kind:           kind,
		Memo:           req.Memo,
		InitiatedBy:    userID,
		IdempotencyKey: key,
		FromAccountID:  req.FromAccountID,
		ToAccountID:    toAccountID,
		AmountCents:    req.AmountCents,
	}, recipientID)
}
