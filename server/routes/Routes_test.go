package routes

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"api/cache"
	"api/config"
	"api/controllers"
	"api/database"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pquerna/otp/totp"
)

// These tests run the whole API against real Postgres and Redis, because the
// guarantees that matter (locks, triggers, constraints) live there. They skip
// unless TEST_DATABASE_URL and TEST_REDIS_URL are set.

var (
	server *httptest.Server
	pool   *pgxpool.Pool
)

func TestMain(m *testing.M) {
	dbURL, redisURL := os.Getenv("TEST_DATABASE_URL"), os.Getenv("TEST_REDIS_URL")
	if dbURL == "" || redisURL == "" {
		fmt.Println("skipping integration tests: TEST_DATABASE_URL and TEST_REDIS_URL are not set")
		os.Exit(0)
	}

	ctx := context.Background()
	var err error
	if pool, err = database.Connect(ctx, dbURL); err != nil {
		panic(err)
	}
	if err = database.Migrate(ctx, pool); err != nil {
		panic(err)
	}
	redis, err := cache.Connect(ctx, redisURL)
	if err != nil {
		panic(err)
	}

	handler, err := Router(&controllers.Deps{DB: pool, Cache: redis, Cfg: &config.Config{
		JWTSecret:  []byte("test-secret-test-secret-test-secret"),
		TOTPKey:    make([]byte, 32),
		TrustProxy: true,
	}})
	if err != nil {
		panic(err)
	}
	server = httptest.NewServer(handler)
	os.Exit(m.Run())
}

// client is one browser: its own cookie jar and its own address, so rate limits
// can't leak between tests or from one run into the next.
type client struct {
	t    *testing.T
	http *http.Client
	ip   string
}

func newClient(t *testing.T) *client {
	jar, _ := cookiejar.New(nil)
	return &client{t: t, http: &http.Client{Jar: jar}, ip: "client-" + newKey()}
}

func (c *client) do(method, path string, body any, headers map[string]string) (int, map[string]any) {
	var reader bytes.Buffer
	if body != nil {
		json.NewEncoder(&reader).Encode(body)
	}
	req, err := http.NewRequest(method, server.URL+"/api/v1"+path, &reader)
	if err != nil {
		c.t.Fatal(err)
	}
	req.Header.Set("X-Forwarded-For", c.ip)
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	res, err := c.http.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer res.Body.Close()

	var decoded any
	json.NewDecoder(res.Body).Decode(&decoded)
	if object, ok := decoded.(map[string]any); ok {
		return res.StatusCode, object
	}
	return res.StatusCode, map[string]any{"list": decoded}
}

func (c *client) expect(want int, method, path string, body any) map[string]any {
	c.t.Helper()
	status, data := c.do(method, path, body, nil)
	if status != want {
		c.t.Fatalf("%s %s: got %d, want %d: %v", method, path, status, want, data)
	}
	return data
}

func newKey() string {
	raw := make([]byte, 16)
	rand.Read(raw)
	return hex.EncodeToString(raw)
}

func uniqueEmail() string {
	return "user-" + newKey() + "@example.com"
}

const password = "correct horse battery"

// signUp creates a user and returns the client with their checking account ID.
func signUp(t *testing.T) (*client, string, string) {
	t.Helper()
	c := newClient(t)
	email := uniqueEmail()
	c.expect(201, "POST", "/users", map[string]string{"name": "Ana", "email": email, "password": password})
	accounts := c.expect(200, "GET", "/accounts", nil)["list"].([]any)
	if len(accounts) != 1 {
		t.Fatalf("got %d accounts, want 1", len(accounts))
	}
	return c, accounts[0].(map[string]any)["id"].(string), email
}

func (c *client) deposit(accountID string, cents int64) {
	c.t.Helper()
	status, data := c.do("POST", "/deposits", map[string]any{"account_id": accountID, "amount_cents": cents},
		map[string]string{"Idempotency-Key": newKey()})
	if status != 201 {
		c.t.Fatalf("deposit: got %d: %v", status, data)
	}
}

func (c *client) balance(accountID string) int64 {
	c.t.Helper()
	return int64(c.expect(200, "GET", "/accounts/"+accountID, nil)["balance_cents"].(float64))
}

func TestSignUpOpensCheckingAndRejectsDuplicates(t *testing.T) {
	c, _, email := signUp(t)
	if got := c.expect(200, "GET", "/me", nil)["email"]; got != email {
		t.Errorf("got %v", got)
	}

	other := newClient(t)
	other.expect(409, "POST", "/users", map[string]string{"name": "Ana", "email": strings.ToUpper(email), "password": password})
}

func TestEveryProtectedRouteNeedsSignIn(t *testing.T) {
	anonymous := newClient(t)
	for _, path := range []string{"/me", "/accounts", "/entries", "/sessions", "/activity", "/anything-new"} {
		anonymous.expect(401, "GET", path, nil)
	}
	anonymous.expect(401, "POST", "/transfers", map[string]any{})
	anonymous.expect(401, "POST", "/deposits", map[string]any{})
}

func TestCannotTouchSomeoneElsesAccount(t *testing.T) {
	alice, aliceAccount, _ := signUp(t)
	bob, bobAccount, _ := signUp(t)
	alice.deposit(aliceAccount, 10_000)

	bob.expect(404, "GET", "/accounts/"+aliceAccount, nil)
	bob.expect(404, "GET", "/accounts/"+aliceAccount+"/entries", nil)
	bob.expect(404, "GET", "/accounts/"+aliceAccount+"/statements/2026-11", nil)

	status, _ := bob.do("POST", "/deposits", map[string]any{"account_id": aliceAccount, "amount_cents": 100},
		map[string]string{"Idempotency-Key": newKey()})
	if status != 404 {
		t.Errorf("deposit into another account: got %d", status)
	}

	status, _ = bob.do("POST", "/transfers", map[string]any{
		"from_account_id": aliceAccount, "to_account_id": bobAccount, "amount_cents": 100,
	}, map[string]string{"Idempotency-Key": newKey()})
	if status != 404 {
		t.Errorf("transfer out of another account: got %d", status)
	}

	// to_account_id is for your own accounts, others go by number or email
	bob.deposit(bobAccount, 500)
	status, _ = bob.do("POST", "/transfers", map[string]any{
		"from_account_id": bobAccount, "to_account_id": aliceAccount, "amount_cents": 100,
	}, map[string]string{"Idempotency-Key": newKey()})
	if status != 422 {
		t.Errorf("transfer by another user's account ID: got %d", status)
	}

	if got := alice.balance(aliceAccount); got != 10_000 {
		t.Errorf("alice's balance changed to %d", got)
	}
}

func TestCannotSendToBankFundingAccount(t *testing.T) {
	c, account, _ := signUp(t)
	c.deposit(account, 1000)
	status, _ := c.do("POST", "/transfers", map[string]any{
		"from_account_id": account, "to_account_number": "0000000000", "amount_cents": 100,
	}, map[string]string{"Idempotency-Key": newKey()})
	if status != 422 {
		t.Errorf("got %d", status)
	}
}

func TestTransfersMoveMoneyAndShowBothSides(t *testing.T) {
	alice, aliceAccount, _ := signUp(t)
	bob, bobAccount, bobEmail := signUp(t)
	alice.deposit(aliceAccount, 10_000)

	// reading first means the recipient's balance is cached and must be invalidated
	bob.expect(200, "GET", "/accounts", nil)

	status, data := alice.do("POST", "/transfers", map[string]any{
		"from_account_id": aliceAccount, "to_email": bobEmail, "amount_cents": 2_500, "memo": "Lunch",
	}, map[string]string{"Idempotency-Key": newKey()})
	if status != 201 {
		t.Fatalf("got %d: %v", status, data)
	}

	if got := alice.balance(aliceAccount); got != 7_500 {
		t.Errorf("alice has %d", got)
	}
	accounts := bob.expect(200, "GET", "/accounts", nil)["list"].([]any)
	if got := accounts[0].(map[string]any)["balance_cents"].(float64); got != 2_500 {
		t.Errorf("bob's cached balance is %v", got)
	}

	entries := bob.expect(200, "GET", "/accounts/"+bobAccount+"/entries", nil)["list"].([]any)
	if len(entries) != 1 {
		t.Fatalf("got %d entries", len(entries))
	}
	entry := entries[0].(map[string]any)
	if entry["memo"] != "Lunch" || entry["counterparty"] != "Ana" || entry["balance_after_cents"].(float64) != 2_500 {
		t.Errorf("got %v", entry)
	}
}

func TestRepeatedIdempotencyKeyPaysOnce(t *testing.T) {
	c, account, _ := signUp(t)
	key := newKey()
	body := map[string]any{"account_id": account, "amount_cents": 1_000}

	var ids sync.Map
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			status, data := c.do("POST", "/deposits", body, map[string]string{"Idempotency-Key": key})
			if status != 201 {
				t.Errorf("got %d: %v", status, data)
			}
			ids.Store(data["transaction_id"], true)
		})
	}
	wg.Wait()

	count := 0
	ids.Range(func(_, _ any) bool { count++; return true })
	if count != 1 {
		t.Errorf("got %d different transactions", count)
	}
	if got := c.balance(account); got != 1_000 {
		t.Errorf("balance is %d", got)
	}
}

func TestMoneyRequestsNeedAnIdempotencyKey(t *testing.T) {
	c, account, _ := signUp(t)
	c.expect(400, "POST", "/deposits", map[string]any{"account_id": account, "amount_cents": 100})
}

func TestAmountsAreValidated(t *testing.T) {
	c, account, _ := signUp(t)
	for _, cents := range []int64{0, -100, 1_000_001} {
		status, _ := c.do("POST", "/deposits", map[string]any{"account_id": account, "amount_cents": cents},
			map[string]string{"Idempotency-Key": newKey()})
		if status != 400 {
			t.Errorf("deposit of %d: got %d", cents, status)
		}
	}

	// fractions of a cent and text must not coerce into an amount
	for _, raw := range []string{`10.5`, `"100"`} {
		req, _ := http.NewRequest("POST", server.URL+"/api/v1/deposits",
			strings.NewReader(`{"account_id":"`+account+`","amount_cents":`+raw+`}`))
		req.Header.Set("Idempotency-Key", newKey())
		req.Header.Set("X-Forwarded-For", c.ip)
		res, err := c.http.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != 400 {
			t.Errorf("amount %s: got %d", raw, res.StatusCode)
		}
	}
}

func TestParallelTransfersCannotOverdraw(t *testing.T) {
	alice, aliceAccount, _ := signUp(t)
	_, _, bobEmail := signUp(t)
	alice.deposit(aliceAccount, 500)

	var accepted, declined atomic.Int64
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			status, data := alice.do("POST", "/transfers", map[string]any{
				"from_account_id": aliceAccount, "to_email": bobEmail, "amount_cents": 100,
			}, map[string]string{"Idempotency-Key": newKey()})
			switch status {
			case 201:
				accepted.Add(1)
			case 422:
				declined.Add(1)
			default:
				t.Errorf("got %d: %v", status, data)
			}
		})
	}
	wg.Wait()

	if accepted.Load() != 5 || declined.Load() != 15 {
		t.Errorf("accepted %d, declined %d", accepted.Load(), declined.Load())
	}
	if got := alice.balance(aliceAccount); got != 0 {
		t.Errorf("balance is %d", got)
	}
}

func TestOppositeTransfersDoNotDeadlock(t *testing.T) {
	c, first, _ := signUp(t)
	second := c.expect(201, "POST", "/accounts", map[string]string{"type": "savings", "name": "Rainy day"})["id"].(string)
	c.deposit(first, 10_000)
	c.deposit(second, 10_000)

	var wg sync.WaitGroup
	for i := range 30 {
		from, to := first, second
		if i%2 == 1 {
			from, to = second, first
		}
		wg.Go(func() {
			status, data := c.do("POST", "/transfers", map[string]any{
				"from_account_id": from, "to_account_id": to, "amount_cents": 100,
			}, map[string]string{"Idempotency-Key": newKey()})
			if status != 201 {
				t.Errorf("got %d: %v", status, data)
			}
		})
	}
	wg.Wait()

	if a, b := c.balance(first), c.balance(second); a+b != 20_000 {
		t.Errorf("money changed: %d + %d", a, b)
	}
}

func TestDailyLimitAppliesToOtherPeopleOnly(t *testing.T) {
	c, account, _ := signUp(t)
	savings := c.expect(201, "POST", "/accounts", map[string]string{"type": "savings", "name": "Savings"})["id"].(string)
	_, _, recipient := signUp(t)
	for range 3 {
		c.deposit(account, 1_000_000)
	}

	send := func(body map[string]any) (int, map[string]any) {
		body["from_account_id"] = account
		return c.do("POST", "/transfers", body, map[string]string{"Idempotency-Key": newKey()})
	}

	if status, data := send(map[string]any{"to_email": recipient, "amount_cents": 2_500_000}); status != 201 {
		t.Fatalf("got %d: %v", status, data)
	}
	status, data := send(map[string]any{"to_email": recipient, "amount_cents": 1})
	if status != 422 || data["code"] != "daily_limit" {
		t.Errorf("got %d: %v", status, data)
	}
	if status, data := send(map[string]any{"to_account_id": savings, "amount_cents": 100}); status != 201 {
		t.Errorf("own transfer: got %d: %v", status, data)
	}
}

func TestLedgerCannotBeTamperedWith(t *testing.T) {
	c, account, _ := signUp(t)
	c.deposit(account, 1_000)
	ctx := context.Background()

	for name, sql := range map[string]string{
		"update entry":       "UPDATE entries SET amount_cents = 999999 WHERE account_id = '" + account + "'",
		"delete entry":       "DELETE FROM entries WHERE account_id = '" + account + "'",
		"truncate entries":   "TRUNCATE entries CASCADE",
		"update transaction": "UPDATE transactions SET memo = 'edited'",
		"delete audit":       "DELETE FROM audit_log",
	} {
		if _, err := pool.Exec(ctx, sql); err == nil {
			t.Errorf("%s succeeded", name)
		}
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	var transactionID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO transactions (kind, initiated_by, idempotency_key)
		SELECT 'deposit', user_id, $1 FROM accounts WHERE id = $2
		RETURNING id::text`, newKey(), account).Scan(&transactionID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, "INSERT INTO entries (transaction_id, account_id, amount_cents) VALUES ($1, $2, 500)", transactionID, account); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err == nil {
		t.Error("a one-sided transaction was committed")
	}
}

func TestBooksAlwaysBalance(t *testing.T) {
	var total int64
	if err := pool.QueryRow(context.Background(), "SELECT COALESCE(sum(amount_cents), 0)::bigint FROM entries").Scan(&total); err != nil {
		t.Fatal(err)
	}
	if total != 0 {
		t.Errorf("the ledger is out by %d cents", total)
	}
}

func TestStatementTotals(t *testing.T) {
	c, account, _ := signUp(t)
	c.deposit(account, 1_000)
	c.deposit(account, 500)
	savings := c.expect(201, "POST", "/accounts", map[string]string{"type": "savings", "name": "Savings"})["id"].(string)
	status, _ := c.do("POST", "/transfers", map[string]any{
		"from_account_id": account, "to_account_id": savings, "amount_cents": 300,
	}, map[string]string{"Idempotency-Key": newKey()})
	if status != 201 {
		t.Fatalf("got %d", status)
	}

	month := time.Now().UTC().Format("2006-01")
	statement := c.expect(200, "GET", "/accounts/"+account+"/statements/"+month, nil)
	for field, want := range map[string]float64{"opening_cents": 0, "credits_cents": 1_500, "debits_cents": 300, "closing_cents": 1_200} {
		if statement[field] != want {
			t.Errorf("%s: got %v, want %v", field, statement[field], want)
		}
	}

	next := time.Now().UTC().AddDate(0, 1, 0).Format("2006-01")
	if got := c.expect(200, "GET", "/accounts/"+account+"/statements/"+next, nil)["opening_cents"]; got != 1_200.0 {
		t.Errorf("next month opens at %v", got)
	}
	c.expect(400, "GET", "/accounts/"+account+"/statements/nope", nil)
}

func TestLoginGivesTheSameErrorForUnknownEmailAndWrongPassword(t *testing.T) {
	_, _, email := signUp(t)
	c := newClient(t)

	wrong := c.expect(401, "POST", "/auth/login", map[string]string{"email": email, "password": "wrong password here"})
	unknown := c.expect(401, "POST", "/auth/login", map[string]string{"email": uniqueEmail(), "password": "wrong password here"})
	if wrong["error"] != unknown["error"] || wrong["code"] != unknown["code"] {
		t.Errorf("responses differ: %v and %v", wrong, unknown)
	}
	c.expect(204, "POST", "/auth/login", map[string]string{"email": email, "password": password})
	c.expect(200, "GET", "/me", nil)
}

func TestLoginIsRateLimitedPerEmailAcrossAddresses(t *testing.T) {
	_, _, email := signUp(t)
	for range 10 {
		newClient(t).expect(401, "POST", "/auth/login", map[string]string{"email": email, "password": "wrong password here"})
	}
	// the right password from a new address is still refused while limited
	newClient(t).expect(429, "POST", "/auth/login", map[string]string{"email": email, "password": password})
}

func TestLoginIsRateLimitedPerAddress(t *testing.T) {
	c := newClient(t)
	for range 20 {
		c.expect(401, "POST", "/auth/login", map[string]string{"email": uniqueEmail(), "password": "wrong password here"})
	}
	c.expect(429, "POST", "/auth/login", map[string]string{"email": uniqueEmail(), "password": "wrong password here"})
}

func TestMoneyLimitFollowsTheUserNotTheAddress(t *testing.T) {
	c, account, _ := signUp(t)
	body := map[string]any{"account_id": account, "amount_cents": 1}

	for range 60 {
		c.ip = "client-" + newKey() // a forged address must not reset the limit
		if status, data := c.do("POST", "/deposits", body, map[string]string{"Idempotency-Key": newKey()}); status != 201 {
			t.Fatalf("got %d: %v", status, data)
		}
	}
	c.ip = "client-" + newKey()
	if status, _ := c.do("POST", "/deposits", body, map[string]string{"Idempotency-Key": newKey()}); status != 429 {
		t.Errorf("got %d", status)
	}

	// someone else on the same address isn't affected
	other, otherAccount, _ := signUp(t)
	other.ip = c.ip
	other.deposit(otherAccount, 1)
}

func cookieValue(c *client, name string) string {
	base, _ := url.Parse(server.URL)
	for _, cookie := range c.http.Jar.Cookies(base) {
		if cookie.Name == name {
			return cookie.Value
		}
	}
	return ""
}

func setCookie(c *client, name, value string) {
	base, _ := url.Parse(server.URL)
	c.http.Jar.SetCookies(base, []*http.Cookie{{Name: name, Value: value, Path: "/"}})
}

func TestRefreshRotatesAndDetectsTheft(t *testing.T) {
	c, _, _ := signUp(t)
	stolen := cookieValue(c, "refresh_token")

	c.expect(204, "POST", "/auth/refresh", nil)
	if cookieValue(c, "refresh_token") == stolen {
		t.Fatal("refresh token did not rotate")
	}

	// within the grace period a second tab's old token is tolerated
	tab := newClient(t)
	setCookie(tab, "refresh_token", stolen)
	tab.expect(204, "POST", "/auth/refresh", nil)

	sessionID := strings.Split(stolen, ".")[0]
	if _, err := pool.Exec(context.Background(), "UPDATE sessions SET rotated_at = now() - interval '1 minute' WHERE id = $1", sessionID); err != nil {
		t.Fatal(err)
	}

	// after it, the old token means theft and ends the session for everyone
	thief := newClient(t)
	setCookie(thief, "refresh_token", stolen)
	thief.expect(401, "POST", "/auth/refresh", nil)
	c.expect(401, "GET", "/me", nil)
	c.expect(401, "POST", "/auth/refresh", nil)
}

func TestForgedRefreshTokenCannotEndASession(t *testing.T) {
	c, _, _ := signUp(t)
	sessionID := strings.Split(cookieValue(c, "refresh_token"), ".")[0]

	attacker := newClient(t)
	setCookie(attacker, "refresh_token", sessionID+".not-the-secret")
	attacker.expect(204, "POST", "/auth/logout", nil)
	c.expect(200, "GET", "/me", nil)
}

func TestLogoutEndsAccessImmediately(t *testing.T) {
	c, _, _ := signUp(t)
	access := cookieValue(c, "access_token")
	c.expect(204, "POST", "/auth/logout", nil)

	// the JWT is still unexpired, but its session is gone
	replay := newClient(t)
	setCookie(replay, "access_token", access)
	replay.expect(401, "GET", "/me", nil)
}

func TestSessionsCanBeSeenAndRevoked(t *testing.T) {
	phone, _, email := signUp(t)
	laptop := newClient(t)
	laptop.expect(204, "POST", "/auth/login", map[string]string{"email": email, "password": password})

	sessions := laptop.expect(200, "GET", "/sessions", nil)["list"].([]any)
	if len(sessions) != 2 {
		t.Fatalf("got %d sessions", len(sessions))
	}
	var phoneSession string
	for _, raw := range sessions {
		if s := raw.(map[string]any); s["current"] != true {
			phoneSession = s["id"].(string)
		}
	}

	stranger, _, _ := signUp(t)
	stranger.expect(404, "DELETE", "/sessions/"+phoneSession, nil)
	phone.expect(200, "GET", "/me", nil)

	laptop.expect(204, "DELETE", "/sessions/"+phoneSession, nil)
	phone.expect(401, "GET", "/me", nil)
}

func TestChangingPasswordSignsOutOtherSessions(t *testing.T) {
	phone, _, email := signUp(t)
	laptop := newClient(t)
	laptop.expect(204, "POST", "/auth/login", map[string]string{"email": email, "password": password})

	laptop.expect(403, "POST", "/me/password", map[string]string{"current_password": "not my password", "new_password": "a brand new password"})
	laptop.expect(400, "POST", "/me/password", map[string]string{"current_password": password, "new_password": "short"})
	laptop.expect(204, "POST", "/me/password", map[string]string{"current_password": password, "new_password": "a brand new password"})

	phone.expect(401, "GET", "/me", nil)
	laptop.expect(200, "GET", "/me", nil)
	newClient(t).expect(401, "POST", "/auth/login", map[string]string{"email": email, "password": password})
	newClient(t).expect(204, "POST", "/auth/login", map[string]string{"email": email, "password": "a brand new password"})
}

func TestTwoFactor(t *testing.T) {
	c, _, email := signUp(t)

	setup := c.expect(200, "POST", "/2fa/setup", nil)
	secret := setup["secret"].(string)
	if !strings.HasPrefix(setup["qr_code"].(string), "data:image/png;base64,") {
		t.Error("no QR code")
	}

	var stored []byte
	if err := pool.QueryRow(context.Background(), "SELECT totp_secret FROM users WHERE email = $1", email).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(stored, []byte(secret)) {
		t.Error("two-factor secret is stored in the clear")
	}

	c.expect(400, "POST", "/2fa/enable", map[string]string{"code": "000000"})
	// not on until a code proves the app has the secret
	newClient(t).expect(204, "POST", "/auth/login", map[string]string{"email": email, "password": password})

	code, _ := totp.GenerateCode(secret, time.Now())
	c.expect(204, "POST", "/2fa/enable", map[string]string{"code": code})

	login := newClient(t)
	data := login.expect(401, "POST", "/auth/login", map[string]string{"email": email, "password": password})
	if data["code"] != "two_factor_required" {
		t.Errorf("got %v", data)
	}
	login.expect(401, "GET", "/me", nil)

	// the code that enabled two-factor has been spent, so it can't sign in
	data = login.expect(401, "POST", "/auth/login", map[string]string{"email": email, "password": password, "code": code})
	if data["code"] != "invalid_code" {
		t.Errorf("got %v", data)
	}

	// a valid code doesn't help without the password
	future, _ := totp.GenerateCode(secret, time.Now().Add(30*time.Second))
	login.expect(401, "POST", "/auth/login", map[string]string{"email": email, "password": "wrong password here", "code": future})

	login.expect(204, "POST", "/auth/login", map[string]string{"email": email, "password": password, "code": future})
	login.expect(200, "GET", "/me", nil)

	c.expect(403, "POST", "/2fa/disable", map[string]string{"password": "wrong password here", "code": future})
}

func TestCrossSiteWritesAreRejected(t *testing.T) {
	c, _, _ := signUp(t)
	status, _ := c.do("POST", "/accounts", map[string]string{"type": "savings", "name": "x"}, map[string]string{"Sec-Fetch-Site": "cross-site"})
	if status != 403 {
		t.Errorf("got %d", status)
	}
}

func TestResponsesAreNotCacheable(t *testing.T) {
	c, _, _ := signUp(t)
	req, _ := http.NewRequest("GET", server.URL+"/api/v1/accounts", nil)
	res, err := c.http.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if got := res.Header.Get("Cache-Control"); got != "no-store" {
		t.Errorf("got %q", got)
	}
}
