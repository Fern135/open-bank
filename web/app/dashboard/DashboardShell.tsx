"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { createContext, use, useMemo, type ReactNode } from "react";
import Config from "@config";
import { formatCents } from "../components/ledger";
import { api, type Account, type User } from "../lib/api";
import { useData } from "../lib/useData";
import styles from "./dashboard.module.css";

type Bank = {
  user?: User;
  accounts?: Account[];
  reloadUser: () => void;
  reloadAccounts: () => void;
};

const BankContext = createContext<Bank | null>(null);

export function useBank() {
  const bank = use(BankContext);
  if (!bank) throw new Error("useBank must be used inside DashboardShell");
  return bank;
}

const MOVE_LINKS = [
  { href: "/dashboard/transfer", label: "Transfer money", icon: "fa-right-left" },
  { href: "/dashboard/deposit", label: "Add a deposit", icon: "fa-money-check-dollar" },
  { href: "/dashboard/statements", label: "Statements", icon: "fa-file-lines" },
];

const SETTINGS_LINKS = [
  { href: "/dashboard/profile", label: "Profile", icon: "fa-user" },
  { href: "/dashboard/security", label: "Security", icon: "fa-shield-halved" },
];

async function signOut() {
  // Sign-out should still leave the page if the request fails.
  await api.post("auth/logout").catch(() => {});
  // A full load on purpose: it drops any account data held in memory.
  // eslint-disable-next-line @next/next/no-location-assign-relative-destination
  window.location.assign("/");
}

export default function DashboardShell({ children }: { children: ReactNode }) {
  const pathname = usePathname();
  const me = useData<User>("me");
  const accounts = useData<Account[]>("accounts");

  const bank = useMemo(
    () => ({
      user: me.data,
      accounts: accounts.data,
      reloadUser: me.reload,
      reloadAccounts: accounts.reload,
    }),
    [me.data, accounts.data, me.reload, accounts.reload],
  );

  const current = (href: string) => (pathname === href ? "page" : undefined);

  return (
    <BankContext value={bank}>
      <div className={styles.shell}>
        <nav className={styles.rail} aria-label="Dashboard">
          <Link className={styles.wordmark} href="/dashboard">
            {Config.getAppName()}
          </Link>

          <h2 className={styles.railHeading}>
            <Link href="/dashboard" aria-current={current("/dashboard")}>
              All accounts
            </Link>
          </h2>
          <ul className={styles.tabs}>
            {accounts.data?.map((account) => {
              const href = `/dashboard/accounts/${account.id}`;
              return (
                <li key={account.id}>
                  <Link className={styles.tab} href={href} aria-current={current(href)}>
                    <span className={styles.tabName}>{account.name}</span>
                    <span className={styles.tabNumber}>
                      Ending {account.number.slice(-4)}
                    </span>
                    <span className={styles.tabBalance}>
                      {formatCents(account.balance_cents)}
                    </span>
                  </Link>
                </li>
              );
            })}
            <li>
              <Link
                className={styles.railLink}
                href="/dashboard/accounts/new"
                aria-current={current("/dashboard/accounts/new")}
              >
                <i className="fa-solid fa-plus" aria-hidden="true"></i> Open an account
              </Link>
            </li>
          </ul>
          {accounts.error && <p className={styles.railError}>{accounts.error}</p>}

          <ul className={styles.railLinks}>
            {MOVE_LINKS.map((link) => (
              <li key={link.href}>
                <Link className={styles.railLink} href={link.href} aria-current={current(link.href)}>
                  <i className={`fa-solid ${link.icon}`} aria-hidden="true"></i> {link.label}
                </Link>
              </li>
            ))}
          </ul>

          <ul className={`${styles.railLinks} ${styles.railEnd}`}>
            {SETTINGS_LINKS.map((link) => (
              <li key={link.href}>
                <Link className={styles.railLink} href={link.href} aria-current={current(link.href)}>
                  <i className={`fa-solid ${link.icon}`} aria-hidden="true"></i> {link.label}
                </Link>
              </li>
            ))}
            <li>
              <button className={styles.railLink} type="button" onClick={signOut}>
                <i className="fa-solid fa-right-from-bracket" aria-hidden="true"></i> Sign out
              </button>
            </li>
          </ul>
        </nav>

        <main id="main" className={styles.page}>
          {children}
        </main>
      </div>
    </BankContext>
  );
}
