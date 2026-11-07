import Link from "next/link";
import styles from "./site.module.css";

export default function SiteHeader() {
  return (
    <header>
      <nav className={`container ${styles.nav}`} aria-label="Main">
        <Link className={styles.wordmark} href="/">
          Open Bank
        </Link>
        <div className={styles.navLinks}>
          <Link href="/#accounts">Accounts</Link>
          <Link href="/#payments">Payments</Link>
          <Link href="/sign-in">
            <i className="fa-solid fa-lock" aria-hidden="true"></i> Sign in
          </Link>
          <Link className={styles.navCta} href="/open-account">
            Open an account
          </Link>
        </div>
      </nav>
    </header>
  );
}
