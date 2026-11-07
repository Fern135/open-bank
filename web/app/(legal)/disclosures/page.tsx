import type { Metadata } from "next";

export const metadata: Metadata = { title: "Disclosures" };

export default function Disclosures() {
  return (
    <>
      <h1>Disclosures</h1>
      <p>Last updated November 7, 2026</p>
      <h2>Our status</h2>
      <p>
        Open Bank is not a chartered bank or credit union. It does not hold a
        banking or money transmitter license.
      </p>
      <h2>Deposit insurance</h2>
      <p>
        Open Bank is not a member of the FDIC or the NCUA. Nothing on this site
        is a federally insured deposit.
      </p>
      <h2>Accounts and availability</h2>
      <p>
        We aren&apos;t accepting deposits, making loans or moving money yet.
        Account descriptions on this site show how we plan for accounts to work,
        and they may change.
      </p>
      <h2>Example figures</h2>
      <p>
        The ledger on our home page is an example. Its balances and transactions
        aren&apos;t real.
      </p>
      <h2>Other companies</h2>
      <p>
        Open Bank isn&apos;t affiliated with any other bank or financial company
        that has a similar name.
      </p>
    </>
  );
}
