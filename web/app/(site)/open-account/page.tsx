import type { Metadata } from "next";
import Link from "next/link";
import OpenAccountForm from "./OpenAccountForm";

export const metadata: Metadata = { title: "Open an account" };

export default function OpenAccount() {
  return (
    <div className="container py-5" style={{ maxWidth: "40rem" }}>
      <h1 className="mb-4">Open an account</h1>
      <p>
        Your first checking account opens as soon as you finish this form. Open
        Tally is a demo, so use a made-up name and never put real money or
        details here.
      </p>
      <OpenAccountForm />
      <p className="mt-4 mb-0">
        Read how your account works in our{" "}
        <Link href="/deposit-agreement">deposit account agreement</Link>.
        Already have an account? <Link href="/sign-in">Sign in</Link>.
      </p>
    </div>
  );
}
