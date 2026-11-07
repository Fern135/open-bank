"use client";

import { useRef, useState, type FormEvent } from "react";
import { formatCents } from "../../components/ledger";
import { api, errorMessage } from "../../lib/api";
import { parseCents } from "../../lib/money";
import forms from "../../components/forms.module.css";
import { useBank } from "../DashboardShell";
import styles from "../dashboard.module.css";

export default function Deposit() {
  const { accounts, reloadAccounts } = useBank();
  const [accountId, setAccountId] = useState("");
  const [amount, setAmount] = useState("");
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [busy, setBusy] = useState(false);

  // Kept across retries of the same form, so a lost response can't deposit twice.
  const key = useRef<string | null>(null);

  const selected = accountId || accounts?.[0]?.id || "";

  async function submit(event: FormEvent) {
    event.preventDefault();
    setError("");
    setSuccess("");

    const cents = parseCents(amount);
    if (cents === null) {
      setError("Enter an amount like 250.00.");
      return;
    }

    setBusy(true);
    key.current ??= crypto.randomUUID();
    try {
      await api.post(
        "deposits",
        { account_id: selected, amount_cents: cents },
        { headers: { "Idempotency-Key": key.current } },
      );
      key.current = null;
      setAmount("");
      setSuccess(`Posted a deposit of ${formatCents(cents)}.`);
      reloadAccounts();
    } catch (failure) {
      setError(errorMessage(failure));
    }
    setBusy(false);
  }

  return (
    <>
      <h1 className={styles.heading}>Add a deposit</h1>
      <p className={styles.intro}>
        Open Tally is a demo, so this adds simulated money. It posts to the
        ledger like a real deposit, with Open Tally&apos;s funding account on the
        other side, and the most you can add at once is $10,000.
      </p>
      <form className={forms.form} onSubmit={submit}>
        <div className={forms.field}>
          <label htmlFor="account">Deposit to</label>
          <select
            id="account"
            value={selected}
            onChange={(event) => {
              setAccountId(event.target.value);
              key.current = null;
            }}
          >
            {accounts?.map((account) => (
              <option key={account.id} value={account.id}>
                {account.name} ({formatCents(account.balance_cents)})
              </option>
            ))}
          </select>
        </div>
        <div className={forms.field}>
          <label htmlFor="amount">Amount in dollars</label>
          <input
            id="amount"
            inputMode="decimal"
            autoComplete="off"
            required
            placeholder="250.00"
            value={amount}
            onChange={(event) => {
              setAmount(event.target.value);
              key.current = null;
            }}
          />
        </div>
        <p className={forms.error} role="alert">{error}</p>
        <p className={forms.success} role="status">{success}</p>
        <div className={forms.buttons}>
          <button className={forms.button} type="submit" disabled={busy || !selected}>
            Post deposit
          </button>
        </div>
      </form>
    </>
  );
}
