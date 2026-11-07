import type { Metadata } from "next";
import Link from "next/link";

export const metadata: Metadata = { title: "Electronic consent" };

export default function ElectronicConsent() {
  return (
    <>
      <h1>Electronic consent</h1>
      <p>Last updated November 7, 2026</p>
      <p>
        This consent will apply when you open an account. It lets us give you
        documents electronically, as the federal E-SIGN Act allows.
      </p>
      <h2>What you&apos;d receive electronically</h2>
      <p>Account agreements, statements, notices, disclosures and tax forms.</p>
      <h2>What you need</h2>
      <p>
        A current web browser, a PDF reader, an email address, and enough
        storage to save or print documents.
      </p>
      <h2>Paper copies</h2>
      <p>
        You can ask for a paper copy of any document. The{" "}
        <Link href="/fees">fee schedule</Link> lists any charge for this.
      </p>
      <h2>Withdrawing consent</h2>
      <p>
        You can withdraw your consent at any time by contacting us. We&apos;ll
        then send documents on paper, or we may close your account if we
        can&apos;t serve it that way.
      </p>
      <h2>Updating your email</h2>
      <p>Keep your email address current so that our notices reach you.</p>
    </>
  );
}
