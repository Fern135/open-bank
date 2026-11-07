# open-tally

A "for fun" banking system to see how a bank works behind the scenes. Every deposit and transfer
is a matching pair of ledger entries you can inspect yourself.

> **USE AT YOUR OWN RISK.** Open Tally is a demo, not a bank. All money in it is simulated, so
> never enter real personal or financial details.

## Features

- Checking and savings accounts, with balances calculated from an append-only double-entry ledger
- Transfers between your own accounts or to other people, by email or account number
- Simulated deposits and monthly statements you can print
- Sign-in with JWTs in `HttpOnly` cookies, plus two-factor codes from an authenticator app
- A list of signed-in devices you can sign out, and a log of security events

## Tech stack

| Part | Built with |
|---|---|
| `web/` | Next.js 16, React 19, TypeScript, Bootstrap |
| `server/` | Go 1.26, standard library `net/http` |
| Database | PostgreSQL 17 |
| Cache and rate limits | Redis 8 |

## Run with Docker

This is the easiest way, and you only need [Docker](https://docs.docker.com/get-docker/).

1. Create your settings file:

   ```bash
   cp .env.example .env
   ```

2. Fill in each empty value in `.env`, using the command shown beside it.

3. Start everything:

   ```bash
   docker compose up --build
   ```

4. Open <http://localhost:3000> and choose **Open an account**.

| Command | What it does |
|---|---|
| `docker compose stop` | Stops the containers |
| `docker compose down` | Stops and removes the containers, and keeps your data |
| `docker compose down -v` | Stops everything and **permanently deletes the database** |
| `docker compose logs -f server` | Follows the API logs |

To use a different port, set `WEB_PORT` in `.env`.

## Run for development

Use this to get hot reloading while you work on the code. You'll need Docker, Go 1.26 and
Node.js 22. Stop the Docker setup above first, because both use port 3000.

1. Start Postgres and Redis:

   ```bash
   docker run -d --rm --name open-tally-dev-db \
     -e POSTGRES_USER=opentally -e POSTGRES_PASSWORD=dev -e POSTGRES_DB=opentally \
     -p 127.0.0.1:5432:5432 postgres:17-alpine

   docker run -d --rm --name open-tally-dev-redis -p 127.0.0.1:6379:6379 redis:8-alpine
   ```

2. Start the API in one terminal. It listens on port 3001 and creates its tables on startup:

   ```bash
   cd server
   export DATABASE_URL='postgres://opentally:dev@localhost:5432/opentally?sslmode=disable'
   export REDIS_URL='redis://localhost:6379/0'
   export JWT_SECRET="$(openssl rand -hex 32)"
   export TOTP_ENCRYPTION_KEY="$(openssl rand -hex 32)"
   export COOKIE_SECURE=false
   go run .
   ```

   These commands make new secrets each time, so restarting the API signs you out. Two-factor
   sign-in also stops working for any account that set it up before the restart. To avoid that,
   save the two values and reuse them.

3. Start the website in a second terminal:

   ```bash
   cd web
   npm install
   npm run dev
   ```

4. Open <http://localhost:3000>.

5. When you're done, stop the API and website with `Ctrl+C`, then remove the containers. This
   deletes the development database:

   ```bash
   docker rm -f open-tally-dev-db open-tally-dev-redis
   ```

## Run the tests

```bash
cd server && go test ./...
cd web && node --test app/lib/money.test.mjs app/components/ledger.test.mjs
```

`go test` skips the API tests in `server/routes/` unless Postgres and Redis are available, and it
still prints `ok`. The [backend guide](server/docs/README.md#tests) shows how to run them.

## Project layout

```
open-tally/
├── docker-compose.yml   Runs web, server, db and redis
├── .env.example         Settings to copy into .env
├── server/              Go API
│   └── docs/            Backend, API and security docs
└── web/                 Next.js website
    └── app/
        ├── (site)/      Public pages
        └── dashboard/   Signed-in banking pages
```

## Documentation

- [Backend guide](server/docs/README.md): how the system and ledger work
- [API reference](server/docs/API.md): every route, limit and error code
- [Security](server/docs/SECURITY.md): the protections, and what's missing before real use

## License

[MIT](LICENSE)
