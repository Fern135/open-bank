import Link from "next/link";
import { LEGAL_LINKS, REPO_URL } from "./legalLinks";
import styles from "./site.module.css";
import Config from "@config";

export default function SiteFooter() {
  return (
    <footer className={styles.footer}>
      <div className="container">
        <div className="row gy-4">
          <div className="col-lg-5">
            <p className={styles.footerName}>{Config.getAppName()}</p>
            <p className={styles.footerNote}>
              {Config.getAppName()} is not a chartered bank, and it is not FDIC insured.
              Read our <Link href="/disclosures">disclosures</Link>.
            </p>
            <p className={`${styles.footerNote} mt-3`}>
              <i className="fa-brands fa-github" aria-hidden="true"></i>{" "}
              <a href={REPO_URL}>Open source</a> under the{" "}
              <a href={`${REPO_URL}/blob/main/LICENSE`}>MIT License</a>.
            </p>
          </div>
          <nav className="col-lg-6 offset-lg-1" aria-label="Legal">
            <ul className={styles.footerLinks}>
              {LEGAL_LINKS.map((link) => (
                <li key={link.href}>
                  <Link href={link.href}>{link.label}</Link>
                </li>
              ))}
            </ul>
          </nav>
        </div>
      </div>
    </footer>
  );
}
