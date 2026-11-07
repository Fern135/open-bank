"use client";

import { useState } from "react";
import {
  ACTIONS,
  OPENING_ENTRY,
  type Entry,
  balance,
  formatCents,
  post,
} from "./ledger";
import styles from "./ledger.module.css";

const VISIBLE_ENTRIES = 3;
const START_MESSAGE = "Choose a transaction to post it to the ledger.";

export default function LedgerDemo() {
  const [entries, setEntries] = useState<Entry[]>([OPENING_ENTRY]);
  const [message, setMessage] = useState(START_MESSAGE);
  const [declined, setDeclined] = useState(false);

  const firstVisible = Math.max(0, entries.length - VISIBLE_ENTRIES);
  const visible = entries.slice(firstVisible);
  const blankLines = (VISIBLE_ENTRIES - visible.length) * 3;
  const total = entries.reduce((sum, entry) => sum + entry.cents, 0);

  function handlePost(entry: Entry) {
    const result = post(entries, entry);
    setEntries(result.entries);
    setDeclined(Boolean(result.declined));
    setMessage(
      result.declined ?? `Posted: ${entry.memo}. Debits still equal credits.`,
    );
  }

  function handleReset() {
    setEntries([OPENING_ENTRY]);
    setDeclined(false);
    setMessage(START_MESSAGE);
  }

  return (
    <section className={styles.sheet} aria-label="Example account ledger">
      <div className={styles.actions}>
        {ACTIONS.map((action) => (
          <button
            key={action.label}
            type="button"
            className={styles.action}
            onClick={() => handlePost(action.entry)}
          >
            <i className={`fa-solid ${action.icon}`} aria-hidden="true"></i>{" "}
            {action.label}
          </button>
        ))}
      </div>

      <p role="status" className={declined ? styles.declined : styles.status}>
        {message}
      </p>

      <table className={styles.ledger}>
        <caption className="visually-hidden">
          Latest ledger entries for an example account
        </caption>
        <thead>
          <tr>
            <th scope="col">Account</th>
            <th scope="col" className={styles.amount}>
              Debit
            </th>
            <th scope="col" className={styles.amount}>
              Credit
            </th>
          </tr>
        </thead>
        {visible.map((entry, index) => {
          const position = firstVisible + index;
          const isNew = position > 0 && position === entries.length - 1;
          return (
            <tbody key={position} className={isNew ? styles.inked : undefined}>
              <tr>
                <td>{entry.debit}</td>
                <td className={styles.amount}>{formatCents(entry.cents)}</td>
                <td className={styles.amount}></td>
              </tr>
              <tr>
                <td className={styles.creditLine}>{entry.credit}</td>
                <td className={styles.amount}></td>
                <td className={styles.amount}>{formatCents(entry.cents)}</td>
              </tr>
              <tr>
                <td className={styles.memo}>{entry.memo}</td>
                <td className={styles.amount}></td>
                <td className={styles.amount}></td>
              </tr>
            </tbody>
          );
        })}
        {blankLines > 0 && (
          <tbody aria-hidden="true">
            {Array.from({ length: blankLines }, (_, line) => (
              <tr key={line}>
                <td>&nbsp;</td>
                <td className={styles.amount}></td>
                <td className={styles.amount}></td>
              </tr>
            ))}
          </tbody>
        )}
        <tfoot>
          <tr>
            <th scope="row">Total, all entries</th>
            <td className={styles.amount}>{formatCents(total)}</td>
            <td className={styles.amount}>{formatCents(total)}</td>
          </tr>
        </tfoot>
      </table>

      <dl className={styles.position}>
        <div>
          <dt>Your checking</dt>
          <dd>{formatCents(balance(entries, "Your checking"))}</dd>
        </div>
        <div>
          <dt>Landlord&apos;s checking</dt>
          <dd>{formatCents(balance(entries, "Landlord's checking"))}</dd>
        </div>
        <div>
          <dt>Vault cash</dt>
          <dd>{formatCents(balance(entries, "Vault cash"))}</dd>
        </div>
      </dl>

      <button type="button" className={styles.reset} onClick={handleReset}>
        Start over
      </button>
    </section>
  );
}
