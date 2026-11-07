import { formatCents } from "../components/ledger";
import type { Entry } from "../lib/api";
import styles from "./dashboard.module.css";

const dateFormat = new Intl.DateTimeFormat("en-US", {
  month: "short",
  day: "numeric",
  year: "numeric",
});

export const formatDate = (iso: string) => dateFormat.format(new Date(iso));

function title(entry: Entry) {
  if (entry.memo) return entry.memo;
  return entry.amount_cents < 0
    ? `To ${entry.counterparty}`
    : `From ${entry.counterparty}`;
}

// The landing page promises both sides of every move, so each row states them.
function bothSides(entry: Entry) {
  const [debit, credit] =
    entry.amount_cents < 0
      ? [entry.account_name, entry.counterparty]
      : [entry.counterparty, entry.account_name];
  return `Debit ${debit}, credit ${credit}`;
}

type Props = {
  caption: string;
  entries: Entry[];
  showBalance?: boolean;
  children?: React.ReactNode; // footer rows
};

export default function Ledger({ caption, entries, showBalance = true, children }: Props) {
  return (
    <table className={styles.ledger}>
      <caption className="visually-hidden">{caption}</caption>
      <thead>
        <tr>
          <th scope="col">Date</th>
          <th scope="col">Entry</th>
          <th scope="col" className={styles.amount}>Debit</th>
          <th scope="col" className={styles.amount}>Credit</th>
          {showBalance && (
            <th scope="col" className={`${styles.amount} ${styles.balanceColumn}`}>Balance</th>
          )}
        </tr>
      </thead>
      <tbody>
        {entries.map((entry) => (
          <tr key={entry.id}>
            <td className={styles.date}>{formatDate(entry.created_at)}</td>
            <td>
              {title(entry)}
              <span className={styles.sides}>{bothSides(entry)}</span>
            </td>
            <td className={styles.amount}>
              {entry.amount_cents < 0 && formatCents(-entry.amount_cents)}
            </td>
            <td className={styles.amount}>
              {entry.amount_cents > 0 && formatCents(entry.amount_cents)}
            </td>
            {showBalance && (
              <td className={`${styles.amount} ${styles.balanceColumn}`}>
                {formatCents(entry.balance_after_cents)}
              </td>
            )}
          </tr>
        ))}
      </tbody>
      {children}
    </table>
  );
}
