import type { Metadata } from "next";

export const metadata: Metadata = { title: "Deposit account agreement" };

export default function DepositAgreement() {
  return (
    <>
      <h1>Deposit account agreement</h1>
      <p>Last updated November 7, 2026</p>
      <p>
        This agreement describes how checking and savings accounts will work
        once we accept them. We aren&apos;t opening accounts yet, and no one is
        bound by it today.
      </p>
      <h2>Opening an account</h2>
      <p>
        You must be at least 18 and a US resident. Federal law requires us to
        verify your name, address, date of birth and identification number
        before we open an account.
      </p>
      <h2>Deposits and available funds</h2>
      <p>
        Your available balance is your ledger balance minus any holds. A deposit
        may not be available right away. We&apos;ll tell you when the funds will
        be available.
      </p>
      <h2>Insufficient funds</h2>
      <p>
        We decline a payment if your available balance can&apos;t cover it.
        Nothing is posted to your account, and your balance doesn&apos;t change.
      </p>
      <h2>Statements and your ledger</h2>
      <p>
        You can see every entry on your account. Entries aren&apos;t edited. We
        fix a mistake by posting a new entry that reverses it, and both stay
        visible.
      </p>
      <h2>Errors and unauthorized transfers</h2>
      <p>
        Contact us right away if you think a transfer is wrong or wasn&apos;t
        authorized. Tell us no later than 60 days after we send the statement
        where the problem first appeared. We&apos;ll investigate and give you
        the result within 10 business days, or within 45 days if we need more
        time and provisionally credit your account.
      </p>
      <p>
        If you tell us within 2 business days of learning your access was lost
        or stolen, you could lose up to $50. After that, you could lose up to
        $500. If you don&apos;t report an unauthorized transfer within 60 days,
        you could lose more.
      </p>
      <h2>Closing an account</h2>
      <p>
        You can close your account at any time. We can close it with notice, or
        without notice where the law allows. We&apos;ll return any remaining
        balance to you.
      </p>
      <h2>Inactive accounts</h2>
      <p>
        State law may require us to turn over funds from an inactive account to
        the state.
      </p>
    </>
  );
}
