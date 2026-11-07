"use client";

import { useState } from "react";
import { formatCents } from "../../components/ledger";
import type { Account, Statement } from "../../lib/api";
import { useData } from "../../lib/useData";
import forms from "../../components/forms.module.css";
import { useBank } from "../DashboardShell";
import Ledger from "../Ledger";
import styles from "../dashboard.module.css";

const monthName = new Intl.DateTimeFormat("en-US", {
  month: "long",
  year: "numeric",
  timeZone: "UTC",
});

// Every month from the account opening to now, newest first, in UTC to match the server.
function monthsFor(account: Account) {
  const months: string[] = [];
  const opened = new Date(account.created_at);
  const now = new Date();
  let year = now.getUTCFullYear();
  let month = now.getUTCMonth();
  while (
    year > opened.getUTCFullYear() ||
    (year === opened.getUTCFullYear() && month >= opened.getUTCMonth())
  ) {
    months.push(`${year}-${String(month + 1).padStart(2, "0")}`);
    month -= 1;
    if (month < 0) {
      month = 11;
      year -= 1;
    }
  }
  return months;
}

const label = (month: string) => monthName.format(new Date(`${month}-01T00:00:00Z`));

function StatementSheet({ accountId, month }: { accountId: string; month: string }) {
  const { data, error } = useData<Statement>(`accounts/${accountId}/statements/${month}`);

  if (error) return <p className={styles.error} role="alert">{error}</p>;
  if (!data) return <p>Loading statement</p>;

  return (
    <div className={styles.sheet}>
      <h2 className={styles.subheading}>
        {data.account.name} statement for {label(month)}
      </h2>
      <p className={styles.accountMeta}>Account {data.account.number}</p>

      <Ledger caption={`Statement for ${label(month)}`} entries={data.entries}>
        <tfoot>
          <tr>
            <th scope="row" colSpan={2}>Totals</th>
            <td className={styles.amount}>{formatCents(data.debits_cents)}</td>
            <td className={styles.amount}>{formatCents(data.credits_cents)}</td>
            <td className={`${styles.amount} ${styles.balanceColumn}`}></td>
          </tr>
        </tfoot>
      </Ledger>
      {data.entries.length === 0 && <p className="mt-3">No entries this month.</p>}

      <dl className={styles.review}>
        <div><dt>Opening balance</dt><dd>{formatCents(data.opening_cents)}</dd></div>
        <div className={styles.reviewTotal}><dt>Closing balance</dt><dd>{formatCents(data.closing_cents)}</dd></div>
      </dl>
    </div>
  );
}

export default function Statements() {
  const { accounts } = useBank();
  const [accountId, setAccountId] = useState("");
  const [month, setMonth] = useState("");

  const account = accounts?.find((item) => item.id === accountId) ?? accounts?.[0];
  const months = account ? monthsFor(account) : [];
  const selectedMonth = months.includes(month) ? month : months[0];

  return (
    <>
      <h1 className={styles.heading}>Statements</h1>
      <div className={`${styles.statementControls} ${styles.noPrint}`}>
        <div className={forms.field}>
          <label htmlFor="account">Account</label>
          <select id="account" value={account?.id ?? ""} onChange={(event) => setAccountId(event.target.value)}>
            {accounts?.map((item) => (
              <option key={item.id} value={item.id}>{item.name}</option>
            ))}
          </select>
        </div>
        <div className={forms.field}>
          <label htmlFor="month">Month</label>
          <select id="month" value={selectedMonth ?? ""} onChange={(event) => setMonth(event.target.value)}>
            {months.map((item) => (
              <option key={item} value={item}>{label(item)}</option>
            ))}
          </select>
        </div>
        <button className={forms.quietButton} type="button" onClick={() => window.print()}>
          <i className="fa-solid fa-print" aria-hidden="true"></i> Print
        </button>
      </div>
      {account && selectedMonth && <StatementSheet accountId={account.id} month={selectedMonth} />}
    </>
  );
}
