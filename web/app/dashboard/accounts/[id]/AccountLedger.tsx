"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useState } from "react";
import { formatCents } from "../../../components/ledger";
import { api, errorMessage, type Account, type Entry } from "../../../lib/api";
import { useData } from "../../../lib/useData";
import forms from "../../../components/forms.module.css";
import Ledger from "../../Ledger";
import styles from "../../dashboard.module.css";

const PAGE_SIZE = 25;

export default function AccountLedger() {
  const { id } = useParams<{ id: string }>();
  const account = useData<Account>(`accounts/${id}`);
  const firstPage = useData<Entry[]>(`accounts/${id}/entries?limit=${PAGE_SIZE}`);

  const [older, setOlder] = useState<{ id: string; entries: Entry[]; finished: boolean }>({
    id,
    entries: [],
    finished: false,
  });
  const [loadError, setLoadError] = useState("");

  if (account.error) {
    return (
      <>
        <h1 className={styles.heading}>Account not found</h1>
        <p className={styles.error} role="alert">{account.error}</p>
        <p><Link href="/dashboard">Back to all accounts</Link></p>
      </>
    );
  }

  const earlier = older.id === id ? older : { entries: [], finished: false };
  const entries = [...(firstPage.data ?? []), ...earlier.entries];
  const hasMore =
    !earlier.finished && (firstPage.data?.length ?? 0) === PAGE_SIZE;

  async function loadOlder() {
    setLoadError("");
    try {
      const before = entries[entries.length - 1].id;
      const { data } = await api.get<Entry[]>(
        `accounts/${id}/entries?limit=${PAGE_SIZE}&before=${before}`,
      );
      setOlder({
        id,
        entries: [...earlier.entries, ...data],
        finished: data.length < PAGE_SIZE,
      });
    } catch (error) {
      setLoadError(errorMessage(error));
    }
  }

  return (
    <>
      <h1 className={styles.heading}>{account.data?.name ?? "Account"}</h1>
      {account.data && (
        <>
          <dl className={styles.total}>
            <dt>Balance</dt>
            <dd>{formatCents(account.data.balance_cents)}</dd>
          </dl>
          <p className={styles.accountMeta}>
            {account.data.type === "savings" ? "Savings" : "Checking"} account{" "}
            {account.data.number}
          </p>
        </>
      )}

      <section className={styles.section}>
        <h2 className={styles.subheading}>Ledger</h2>
        {firstPage.error && <p className={styles.error} role="alert">{firstPage.error}</p>}
        {firstPage.data?.length === 0 && (
          <p>
            No entries yet. <Link href="/dashboard/deposit">Add a deposit</Link> or{" "}
            <Link href="/dashboard/transfer">transfer money</Link> into this account.
          </p>
        )}
        {entries.length > 0 && (
          <div className={styles.sheet}>
            <Ledger caption={`Ledger for ${account.data?.name ?? "this account"}`} entries={entries} />
          </div>
        )}
        {loadError && <p className={styles.error} role="alert">{loadError}</p>}
        {hasMore && (
          <button className={`${forms.quietButton} mt-3`} type="button" onClick={loadOlder}>
            Show older entries
          </button>
        )}
      </section>
    </>
  );
}
