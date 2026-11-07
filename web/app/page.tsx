import Link from "next/link";
import LedgerDemo from "./components/LedgerDemo";
import { REPO_URL } from "./components/legalLinks";
import styles from "./page.module.css";

const ACCOUNTS = [
  {
    icon: "fa-wallet",
    name: "Checking",
    text: "For everyday spending. Each payment shows the entries behind it, so your balance is never a mystery.",
  },
  {
    icon: "fa-piggy-bank",
    name: "Savings",
    text: "For money you want to set aside. Move it to and from checking, and see both sides of every move.",
  },
  {
    icon: "fa-right-left",
    name: "Transfers",
    text: "Send money to another person or bank, and follow the payment from request to settlement.",
  },
];

const PAYMENT_STEPS = [
  {
    name: "Request",
    text: "You send a payment, and we receive the amount and the recipient.",
  },
  {
    name: "Authorize",
    text: "We confirm it's you, then check your balance and limits.",
  },
  {
    name: "Hold",
    text: "We set the money aside so it can't be spent twice. Your available balance drops first.",
  },
  {
    name: "Post",
    text: "Two matching entries go in the ledger, one debit and one credit. Entries are never edited. A mistake gets a new entry that reverses it.",
  },
  {
    name: "Settle",
    text: "Banks square up with each other, often in batches. That's why some payments take a day or more.",
  },
];

export default function Home() {
  return (
    <>
      <div className={`container ${styles.hero}`}>
        <h1 className={styles.headline}>Banking with the books open.</h1>
        <div className="row gy-5">
          <div className="col-lg-4">
            <p className={styles.lead}>
              Open Bank shows you the ledger behind your money. Every deposit,
              payment and withdrawal is a matching pair of entries you can check
              yourself.
            </p>
            <p>
              Our code is open too, so you can{" "}
              <a href={REPO_URL}>read how it all works</a>.
            </p>
            <p>Try it with this example account, which starts with $500.</p>
            <Link className={styles.cta} href="/open-account">
              Open an account
            </Link>
          </div>
          <div className="col-lg-8">
            <LedgerDemo />
          </div>
        </div>
      </div>

      <section id="accounts" className={`container ${styles.section}`}>
        <h2 className={styles.heading}>Accounts</h2>
        <div className="row gy-4">
          {ACCOUNTS.map((account) => (
            <div className="col-md-4" key={account.name}>
              <h3 className={styles.subheading}>
                <i
                  className={`fa-solid ${account.icon}`}
                  aria-hidden="true"
                ></i>{" "}
                {account.name}
              </h3>
              <p className="mb-0">{account.text}</p>
            </div>
          ))}
        </div>
      </section>

      <section id="payments" className={`container ${styles.section}`}>
        <h2 className={styles.heading}>How a payment moves</h2>
        <ol className={styles.steps}>
          {PAYMENT_STEPS.map((step) => (
            <li key={step.name}>
              <h3 className={styles.subheading}>{step.name}</h3>
              <p className="mb-0">{step.text}</p>
            </li>
          ))}
        </ol>
      </section>
    </>
  );
}
