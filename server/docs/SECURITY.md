# Security

Open Tally is a demo, not a licensed bank. This page explains how it's protected and what's
still missing before it could hold anything real.

## Sign-in

- **Passwords** are hashed with Argon2id.
- **Access tokens** are JWTs that last 15 minutes, signed with HS256. The algorithm is pinned, so a
  forged token can't choose its own.
- **Sessions** are checked on every request as well as the JWT, so signing out takes effect
  immediately rather than when the token expires. The cost is one small database query per request.
- **Refresh tokens** change on every use and only their hash is stored. If an old one comes back
  more than 30 seconds later, it's treated as stolen and the session ends. The 30 seconds lets two
  browser tabs refresh at the same moment.
- **Sessions end after 12 hours**, however often they're refreshed.
- **Cookies** are `HttpOnly`, `SameSite=Strict` and `Secure`, so page scripts can't read them.
- **Two-factor secrets** are encrypted with AES-256-GCM, and each code works only once.
- **Wrong email and wrong password** get the same response and take the same time.
- **Changing a password** signs out every other device.

## Money

- **Overdrafts:** the account row is locked before its balance is checked, so parallel payments
  can't both spend the same money.
- **Deadlocks:** accounts are always locked in the same order.
- **Double payments:** every payment needs an idempotency key.
- **Tampering:** database triggers block edits to the ledger and audit log, and reject any
  transaction that doesn't balance.
- **Access:** every query is filtered by the signed-in user. New routes need sign-in by default.
- **Redis** is only used for display and rate limits. Nothing that moves money reads the cache.
  Cached balances last up to 30 seconds and are cleared after each payment.

## Web and network

- `http.CrossOriginProtection` rejects cross-site writes, behind the `SameSite` cookies.
- API responses are marked `no-store`.
- `next.config.ts` sets a Content Security Policy, and pages can't be framed.
- Logs record the route and status only, never bodies or cookies.
- Containers run as non-root users, and the API's filesystem is read-only.
- Secrets come from `.env`, which git ignores. Compose won't start without them.

## Known limits

Fix these before this holds anything real.

| Limit | Why it matters | What to do |
|---|---|---|
| **No https** | Local Docker serves plain http, so cookies aren't `Secure` | Put an https proxy in front and set `COOKIE_SECURE=true` |
| **The client IP can be forged** | Next keeps a client-sent `X-Forwarded-For`, so per-IP limits are best effort. Per-email and per-user limits aren't affected | Use a proxy that overwrites the header |
| **Email isn't verified** | Anyone can sign up with someone else's address | Send a confirmation email |
| **Sign-up reveals registered emails** | The `409` and transfer errors show that an address has an account | Answer the same way either way, once email verification exists |
| **No password reset** | A forgotten password locks the account for good | Add a reset flow, which needs email |
| **No two-factor recovery codes** | A lost phone locks the account for good | Issue one-time recovery codes at setup |
| **Inline scripts are allowed by the CSP** | The pages are prerendered, so they can't carry a nonce | Move to nonces if the pages become dynamic |
| **Sign-in lockout can be abused** | Anyone can use up an email's 10 attempts and block its owner for 15 minutes | Add a challenge rather than a hard block |
| **Transfers settle instantly** | There are no holds, reviews or fraud checks | Add them as separate posting states |
| **Database traffic is unencrypted** | It's fine on the private Docker network, but not across machines | Use TLS if the database moves |

## Scan results

`gosec` reports six items, all reviewed and left as they are:

| Rule | Where | Why it's fine |
|---|---|---|
| G115 | `auth/Totp.go` | The counter comes from the current time, so it's never negative |
| G124 (two) | `controllers/Http.go` | `Secure` comes from `COOKIE_SECURE` and defaults to on |
| G104 (three) | Response writers and `cache/Cache.go` | The response has already started, or the connection is being discarded |
