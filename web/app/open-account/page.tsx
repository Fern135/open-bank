import type { Metadata } from "next";

export const metadata: Metadata = { title: "Open an account" };

export default function OpenAccount() {
  return (
    <div className="container py-5" style={{ maxWidth: "40rem" }}>
      <h1 className="mb-4">Open an account</h1>
      <p>
        We aren&apos;t accepting new accounts yet. When we are, you&apos;ll be
        able to apply here.
      </p>
      <p className="mb-0">
        Until then, read how your account would work in our deposit account
        agreement.
      </p>
    </div>
  );
}
