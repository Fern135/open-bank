import type { Metadata } from "next";

export const metadata: Metadata = { title: "Sign in" };

export default function SignIn() {
  return (
    <div className="container py-5" style={{ maxWidth: "40rem" }}>
      <h1 className="mb-4">Sign in</h1>
      <p>
        Online banking isn&apos;t open yet, so there is nothing to sign in to.
        When it opens, you&apos;ll sign in here.
      </p>
      <p className="mb-0">
        Never enter your password or card details on a page you reached from an
        unexpected message.
      </p>
    </div>
  );
}
