import type { Metadata } from "next";
import { REPO_URL } from "../../../components/legalLinks";

export const metadata: Metadata = { title: "Security" };

export default function Security() {
  return (
    <>
      <h1>Security</h1>
      <p>Last updated November 7, 2026</p>
      <h2>What we&apos;ll never ask for</h2>
      <p>
        We&apos;ll never call, text or email to ask for your password, your
        one-time code or your full card number. If someone does, it isn&apos;t
        us. Stop responding and contact us.
      </p>
      <h2>Protect your account</h2>
      <ul>
        <li>Use a unique password and a password manager.</li>
        <li>
          Type our address yourself, instead of following links in messages.
        </li>
        <li>Keep your phone, computer and browser up to date.</li>
        <li>
          Check your ledger often, and report anything you don&apos;t recognize.
        </li>
      </ul>
      <h2>Report a problem</h2>
      <p>
        If you think you&apos;ve found a vulnerability, contact us through our{" "}
        <a href={`${REPO_URL}/issues`}>support page</a> without posting the
        details publicly. Don&apos;t access anyone else&apos;s data while
        testing.
      </p>
    </>
  );
}
