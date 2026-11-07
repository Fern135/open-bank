package ledger

import (
	"context"
	"errors"
	"sort"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// FundingAccountID is the bank's own account, seeded by the first migration.
// It's the other side of every simulated deposit, so it's allowed to go negative.
const FundingAccountID = "00000000-0000-0000-0000-000000000001"

// DailyExternalLimitCents caps what one account can send to other people per UTC day.
const DailyExternalLimitCents = 2_500_000

const (
	KindDeposit          = "deposit"
	KindInternalTransfer = "internal_transfer"
	KindExternalTransfer = "external_transfer"
)

var (
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrDailyLimit        = errors.New("daily transfer limit reached")
)

// Posting is one movement of money from one account to another.
type Posting struct {
	Kind           string
	Memo           string
	InitiatedBy    string
	IdempotencyKey string
	FromAccountID  string
	ToAccountID    string
	AmountCents    int64
}

// Post writes a Posting as a transaction with two matching entries, and
// returns the transaction ID. Repeating an IdempotencyKey returns the original
// transaction and moves nothing, so a retried request can't pay twice.
func Post(ctx context.Context, pool *pgxpool.Pool, p Posting) (string, error) {
	if p.AmountCents <= 0 {
		return "", errors.New("amount must be positive")
	}
	if p.FromAccountID == p.ToAccountID {
		return "", errors.New("accounts must differ")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	// A concurrent retry waits here until this transaction finishes.
	var id string
	err = tx.QueryRow(ctx, `
		INSERT INTO transactions (kind, memo, initiated_by, idempotency_key)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (initiated_by, idempotency_key) DO NOTHING
		RETURNING id::text`,
		p.Kind, p.Memo, p.InitiatedBy, p.IdempotencyKey).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(ctx, `
			SELECT id::text FROM transactions
			WHERE initiated_by = $1 AND idempotency_key = $2`,
			p.InitiatedBy, p.IdempotencyKey).Scan(&id)
		return id, err
	}
	if err != nil {
		return "", err
	}

	// Locking in a fixed order stops two opposite transfers from deadlocking.
	// Holding the lock is what makes the balance check below safe: nothing
	// else can spend from this account until we commit.
	ids := []string{p.FromAccountID, p.ToAccountID}
	sort.Strings(ids)
	for _, accountID := range ids {
		if _, err := tx.Exec(ctx, "SELECT 1 FROM accounts WHERE id = $1 FOR UPDATE", accountID); err != nil {
			return "", err
		}
	}

	if p.FromAccountID != FundingAccountID {
		// shortcut: sums every entry on each posting, add balance snapshots if accounts grow to many thousands of entries.
		var balance int64
		if err := tx.QueryRow(ctx,
			"SELECT COALESCE(sum(amount_cents), 0)::bigint FROM entries WHERE account_id = $1",
			p.FromAccountID).Scan(&balance); err != nil {
			return "", err
		}
		if balance < p.AmountCents {
			return "", ErrInsufficientFunds
		}
	}

	if p.Kind == KindExternalTransfer {
		var sentToday int64
		if err := tx.QueryRow(ctx, `
			SELECT COALESCE(-sum(e.amount_cents), 0)::bigint
			FROM entries e
			JOIN transactions t ON t.id = e.transaction_id
			WHERE e.account_id = $1
			  AND e.amount_cents < 0
			  AND t.kind = $2
			  AND e.created_at >= date_trunc('day', now() AT TIME ZONE 'UTC') AT TIME ZONE 'UTC'`,
			p.FromAccountID, KindExternalTransfer).Scan(&sentToday); err != nil {
			return "", err
		}
		if sentToday+p.AmountCents > DailyExternalLimitCents {
			return "", ErrDailyLimit
		}
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO entries (transaction_id, account_id, amount_cents)
		VALUES ($1, $2, $3), ($1, $4, $5)`,
		id, p.FromAccountID, -p.AmountCents, p.ToAccountID, p.AmountCents); err != nil {
		return "", err
	}

	return id, tx.Commit(ctx)
}
