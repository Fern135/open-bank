# API reference

The base path is `/api/v1`. Requests and responses are JSON, and unknown fields in a request are
rejected. Amounts are whole cents, so `$12.50` is `1250`.

Errors look like this, and `code` is safe to check in code:

```json
{ "code": "insufficient_funds", "error": "This account doesn't have enough money for that. Nothing was posted." }
```

Requests from a signed-in browser use cookies, so there's no token to send. A `401` with the code
`unauthorized` means the access token expired: call `POST /auth/refresh`, then repeat the request.

Someone else's account returns `404`, the same as one that doesn't exist.

## Public routes

| Route | Body | Success |
|---|---|---|
| `POST /users` | `name`, `email`, `password` | `201` with the user. Opens a checking account and signs in |
| `POST /auth/login` | `email`, `password`, `code` | `204`. Sets the cookies |
| `POST /auth/refresh` | None | `204`. Renews the cookies |
| `POST /auth/logout` | None | `204`. Ends the session |

Passwords must be 12 to 128 characters.

`POST /auth/login` returns `401` with the code `two_factor_required` when the password is right and
the user has two-factor on. Send the same request again with `code` filled in.

## Profile and security

| Route | Body | Success |
|---|---|---|
| `GET /me` | None | The user |
| `PATCH /me` | `name` | The updated user |
| `POST /me/password` | `current_password`, `new_password` | `204`. Signs out other sessions |
| `GET /sessions` | None | Signed-in devices |
| `DELETE /sessions/{id}` | None | `204` |
| `GET /activity` | None | The latest 20 security events |
| `POST /2fa/setup` | None | `secret` and `qr_code`, a PNG data URI |
| `POST /2fa/enable` | `code` | `204` |
| `POST /2fa/disable` | `password`, `code` | `204` |

## Accounts

| Route | Body | Success |
|---|---|---|
| `GET /accounts` | None | The user's accounts with balances |
| `POST /accounts` | `type` (`checking` or `savings`), `name` | `201` with the account |
| `GET /accounts/{id}` | None | The account |
| `GET /accounts/{id}/entries` | None | Entries, newest first |
| `GET /accounts/{id}/statements/{month}` | None | Statement for a month such as `2026-11`, in UTC |
| `GET /entries` | None | Entries across all the user's accounts, newest first |

Each user can have up to 10 accounts.

The entry lists take `limit` (1 to 100, default 25) and `before`. To get the next page, pass the
`id` of the last entry you received as `before`.

## Moving money

Both routes need an `Idempotency-Key` header of up to 64 characters. Use a new key for each new
payment, and the same key when retrying one. A repeated key returns the original
`transaction_id` and moves nothing.

| Route | Body | Success |
|---|---|---|
| `POST /deposits` | `account_id`, `amount_cents` | `201` with `transaction_id` |
| `POST /transfers` | `from_account_id`, one recipient, `amount_cents`, `memo` | `201` with `transaction_id` |

A transfer takes exactly one of these recipients:

| Field | Sends to |
|---|---|
| `to_account_id` | One of your own accounts |
| `to_account_number` | Someone else's account |
| `to_email` | Someone else's oldest checking account |

| Limit | Value |
|---|---|
| Simulated deposit | $0.01 to $10,000 each |
| Transfer | $0.01 to $1,000,000 each |
| Sent to other people | $25,000 per account per UTC day |
| Memo | 140 characters |

Common error codes on these routes are `insufficient_funds`, `daily_limit`, `recipient_not_found`,
`same_account`, `invalid_amount` and `missing_idempotency_key`.

## Rate limits

Going over a limit returns `429` with the code `rate_limited`.

| Routes | Limit | Counted per |
|---|---|---|
| Sign-up and sign-in | 20 per 15 minutes | IP address |
| Sign-up and sign-in | 300 per minute | Whole site |
| Sign-in | 10 per 15 minutes | Email address |
| Password change and two-factor changes | 20 per 15 minutes | User |
| Deposits and transfers | 60 per minute | User |
| Token refresh | 120 per minute | IP address |

## Health check

`GET /healthz` is outside `/api/v1`. It returns `204` when the database is reachable.
