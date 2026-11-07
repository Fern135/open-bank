"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { LEGAL_LINKS } from "../../components/legalLinks";
import styles from "./legal.module.css";

export default function LegalNav() {
  const pathname = usePathname();

  return (
    <nav aria-label="Legal documents">
      <ul className={styles.nav}>
        {LEGAL_LINKS.map((link) => (
          <li key={link.href}>
            <Link
              href={link.href}
              aria-current={pathname === link.href ? "page" : undefined}
            >
              {link.label}
            </Link>
          </li>
        ))}
      </ul>
    </nav>
  );
}
