"use client";

import { useRef, useState, type FormEvent } from "react";
import { formatCents } from "../../components/ledger";
import { api, errorMessage } from "../../lib/api";
import { parseCents } from "../../lib/money";
import forms from "../../components/forms.module.css";
import { useBank } from "../DashboardShell";
import styles from "../dashboard.module.css";

type Review = { cents: number; recipientLabel: string; body: Record<string, unknown> };

export default function Transfer() {
  const { accounts, reloadAccounts } = useBank();
  const [target, setTarget] = useState<"person" | "own">("person");
  const [from, setFrom] = useState("");
  const [toOwn, setToOwn] = useState("");
  const [recipient, setRecipient] = useState("");
  const [amount, setAmount] = useState("");
  const [memo, setMemo] = useState("");
  const [review, setReview] = useState<Review | null>(null);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [busy, setBusy] = useState(false);

  // Kept across retries of the same review, so a lost response can't pay twice.
  const key = useRef<string | null>(null);

  const fromId = from || accounts?.[0]?.id || "";
  const others = accounts?.filter((account) => account.id !== fromId) ?? [];
  const toOwnId = others.some((account) => account.id === toOwn)
    ? toOwn
    : (others[0]?.id ?? "");
  const fromAccount = accounts?.find((account) => account.id === fromId);

  function prepare(event: FormEvent) {
    event.preventDefault();
    setError("");
    setSuccess("");

    const cents = parseCents(amount);
    if (cents === null) {
      setError("Enter an amount like 25.00.");
      return;
    }

    const body: Record<string, unknown> = {
      from_account_id: fromId,
      amount_cents: cents,
      memo,
    };

    let recipientLabel: string;
    if (target === "own") {
      if (!toOwnId) {
        setError("Open a second account to move money between your own accounts.");
        return;
      }
      body.to_account_id = toOwnId;
      recipientLabel = others.find((account) => account.id === toOwnId)?.name ?? "";
    } else {
      const value = recipient.trim();
      body[value.includes("@") ? "to_email" : "to_account_number"] = value;
      recipientLabel = value;
    }

    key.current = crypto.randomUUID();
    setReview({ cents, recipientLabel, body });
  }

  async function send() {
    if (!review || !key.current) return;
    setBusy(true);
    setError("");
    try {
      await api.post("transfers", review.body, {
        headers: { "Idempotency-Key": key.current },
      });
      setSuccess(`Sent ${formatCents(review.cents)} to ${review.recipientLabel}.`);
      setReview(null);
      setAmount("");
      setMemo("");
      setRecipient("");
      reloadAccounts();
    } catch (failure) {
      setError(errorMessage(failure));
    }
    setBusy(false);
  }

  if (review) {
    return (
      <>
        <h1 className={styles.heading}>Check this transfer</h1>
        <div className={styles.sheet}>
          <dl className={styles.review}>
            <div><dt>From</dt><dd>{fromAccount?.name}</dd></div>
            <div><dt>To</dt><dd>{review.recipientLabel}</dd></div>
            {memo && <div><dt>Memo</dt><dd>{memo}</dd></div>}
            <div className={styles.reviewTotal}><dt>Amount</dt><dd>{formatCents(review.cents)}</dd></div>
          </dl>
        </div>
        <p className="mt-3">Transfers post right away and can&apos;t be undone.</p>
        <p className={forms.error} role="alert">{error}</p>
        <div className={`${forms.buttons} mt-3`}>
          <button className={forms.button} type="button" disabled={busy} onClick={send}>
            Send {formatCents(review.cents)}
          </button>
          <button
            className={forms.quietButton}
            type="button"
            disabled={busy}
            onClick={() => {
              setReview(null);
              setError("");
            }}
          >
            Change details
          </button>
        </div>
      </>
    );
  }

  return (
    <>
      <h1 className={styles.heading}>Transfer money</h1>
      <form className={forms.form} onSubmit={prepare}>
        <fieldset className={forms.field}>
          <legend className={forms.legend}>Send to</legend>
          <div className={forms.choices}>
            <label>
              <input
                type="radio"
                name="target"
                checked={target === "person"}
                onChange={() => setTarget("person")}
              />
              Another person
            </label>
            <label>
              <input
                type="radio"
                name="target"
                checked={target === "own"}
                onChange={() => setTarget("own")}
              />
              My own account
            </label>
          </div>
        </fieldset>

        <div className={forms.field}>
          <label htmlFor="from">From</label>
          <select id="from" value={fromId} onChange={(event) => setFrom(event.target.value)}>
            {accounts?.map((account) => (
              <option key={account.id} value={account.id}>
                {account.name} ({formatCents(account.balance_cents)})
              </option>
            ))}
          </select>
        </div>

        {target === "own" ? (
          <div className={forms.field}>
            <label htmlFor="to-own">To</label>
            <select id="to-own" value={toOwnId} onChange={(event) => setToOwn(event.target.value)}>
              {others.map((account) => (
                <option key={account.id} value={account.id}>
                  {account.name} ({formatCents(account.balance_cents)})
                </option>
              ))}
            </select>
          </div>
        ) : (
          <div className={forms.field}>
            <label htmlFor="recipient">Their email or account number</label>
            <input
              id="recipient"
              autoComplete="off"
              required
              aria-describedby="recipient-hint"
              value={recipient}
              onChange={(event) => setRecipient(event.target.value)}
            />
            <p className={forms.hint} id="recipient-hint">
              You can send up to $25,000 a day from each account.
            </p>
          </div>
        )}

        <div className={forms.field}>
          <label htmlFor="amount">Amount in dollars</label>
          <input
            id="amount"
            inputMode="decimal"
            autoComplete="off"
            required
            placeholder="25.00"
            value={amount}
            onChange={(event) => setAmount(event.target.value)}
          />
        </div>

        <div className={forms.field}>
          <label htmlFor="memo">Memo (optional)</label>
          <input
            id="memo"
            autoComplete="off"
            maxLength={140}
            value={memo}
            onChange={(event) => setMemo(event.target.value)}
          />
        </div>

        <p className={forms.error} role="alert">{error}</p>
        <p className={forms.success} role="status">{success}</p>
        <div className={forms.buttons}>
          <button className={forms.button} type="submit" disabled={!fromId}>
            Review transfer
          </button>
        </div>
      </form>
    </>
  );
}
