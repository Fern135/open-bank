"use client";

import Link from "next/link";
import { formatCents } from "../components/ledger";
import type { Entry } from "../lib/api";
import { useData } from "../lib/useData";
import { useBank } from "./DashboardShell";
import Ledger from "./Ledger";
import styles from "./dashboard.module.css";

export default function Overview() {
  const { user, accounts } = useBank();
  const recent = useData<Entry[]>("entries?limit=10");
  const total = accounts?.reduce((sum, account) => sum + account.balance_cents, 0);

  return (
    <>
      <h1 className={styles.heading}>{user ? `${user.name}'s books` : "Your books"}</h1>

      <dl className={styles.total}>
        <dt>Total across your accounts</dt>
        <dd>{total === undefined ? "Loading" : formatCents(total)}</dd>
      </dl>

      <div className={styles.actions}>
        <Link className={styles.action} href="/dashboard/transfer">Transfer money</Link>
        <Link className={styles.action} href="/dashboard/deposit">Add a deposit</Link>
      </div>

      <section className={styles.section}>
        <h2 className={styles.subheading}>Latest entries</h2>
        {recent.error && <p className={styles.error} role="alert">{recent.error}</p>}
        {recent.data?.length === 0 && (
          <p>
            Your ledger is empty. <Link href="/dashboard/deposit">Add a deposit</Link> to
            post your first entry.
          </p>
        )}
        {!!recent.data?.length && (
          <div className={styles.sheet}>
            <Ledger caption="Latest entries across all accounts" entries={recent.data} showBalance={false} />
          </div>
        )}
      </section>
    </>
  );
}
