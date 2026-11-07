# Open Tally backend

Open Tally is a demo bank. The Go API in `server/` keeps a double-entry ledger and handles
sign-in. The Next.js app in `web/` is the only thing the browser talks to.

- [API.md](API.md) lists every route.
- [SECURITY.md](SECURITY.md) explains the protections and their known limits.

## How it fits together

```
browser ──▶ web (Next.js, :3000) ──/api/v1/*──▶ server (Go, :3001) ──▶ db (Postgres)
                                                                  └──▶ redis
```

The browser never calls the API directly. `web/next.config.ts` rewrites `/api/v1/*` to the
server, so both share one origin. That's what lets the auth cookies be `HttpOnly` and
`SameSite=Strict`, and why the API needs no CORS setup.

Only `web` publishes a port. `db` and `redis` sit on an internal network that `web` isn't part of.

## Run it

```bash
cp .env.example .env    # then fill in each value with the command shown beside it
docker compose up --build
```

Open <http://localhost:3000> and choose **Open an account**.

| Command | What it does |
|---|---|
| `docker compose down` | Stops everything and keeps your data |
| `docker compose down -v` | Stops everything and **deletes the database** |
| `docker compose logs -f server` | Follows the API logs |

Migrations run automatically when the server starts.

## Configuration

| Variable | Required | Purpose |
|---|---|---|
| `DATABASE_URL` | Yes | Postgres connection URL |
| `REDIS_URL` | Yes | Redis connection URL |
| `JWT_SECRET` | Yes | Signs access tokens, at least 32 characters |
| `TOTP_ENCRYPTION_KEY` | Yes | 64 hex characters, encrypts two-factor secrets |
| `PORT` | No | Defaults to `3001` |
| `COOKIE_SECURE` | No | Defaults to `true`. Set `false` only for plain http on localhost |
| `TRUST_PROXY` | No | Set `true` only behind a proxy you control |

The server refuses to start if a required value is missing. Docker Compose builds the two URLs
from `POSTGRES_PASSWORD` and `REDIS_PASSWORD` in `.env`.

Don't change `TOTP_ENCRYPTION_KEY` after people have turned on two-factor sign-in: their saved
secrets can't be decrypted without the old key, and they won't be able to sign in.

## Code layout

| Folder | Holds |
|---|---|
| `routes/` | Every route, and which ones need sign-in |
| `controllers/` | Request handlers |
| `ledger/` | The only code that posts money |
| `auth/` | JWTs, refresh tokens and two-factor codes |
| `middleware/` | Sign-in checks, rate limits and logging |
| `database/` | The connection and SQL migrations |
| `cache/` | Redis caching and rate-limit counters |

### Adding a route

Register it on the `protected` router in `routes/Routes.go`. Routes there need sign-in by
default, and only routes added to `api` are public. Any query that reads an account must filter
on the signed-in user.

### Adding a migration

Add a file to `database/migrations/` with the next number, such as `0002_add_payees.sql`. Never
edit a migration that has already run.

## How the ledger works

Every money movement is one row in `transactions` and two rows in `entries`, one negative and one
positive. A positive amount is a credit to the account.

- **Balances are never stored.** A balance is always the sum of the account's entries.
- **Simulated deposits** come from the bank's funding account, which goes negative, so the whole
  ledger always sums to zero.
- **Entries can't be edited.** Database triggers reject updates and deletes on `entries`,
  `transactions` and `audit_log`. A mistake needs a new entry that reverses it.
- **A one-sided transaction can't be saved.** A trigger checks that it sums to zero at commit.

Because the ledger is append-only, test data can't be deleted row by row. Reset a local database
with `docker compose down -v`.

## Tests

Unit tests need nothing else:

```bash
cd server && go test ./...
cd web && node --test app/lib/money.test.mjs app/components/ledger.test.mjs
```

`go test` alone **skips** the tests in `routes/`, and still prints `ok`. They need real services,
because the guarantees they check live in Postgres:

```bash
docker run -d --rm --name ot-test-db -e POSTGRES_PASSWORD=test -e POSTGRES_DB=test \
  -p 127.0.0.1:55432:5432 postgres:17-alpine
docker run -d --rm --name ot-test-redis -p 127.0.0.1:56379:6379 redis:8-alpine

TEST_DATABASE_URL='postgres://postgres:test@127.0.0.1:55432/test?sslmode=disable' \
TEST_REDIS_URL='redis://127.0.0.1:56379/0' \
go test -race -count=1 ./...

docker rm -f ot-test-db ot-test-redis
```

Scan for known vulnerabilities before a release:

```bash
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```
