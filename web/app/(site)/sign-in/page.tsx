import type { Metadata } from "next";
import Link from "next/link";
import SignInForm from "./SignInForm";

export const metadata: Metadata = { title: "Sign in" };

export default function SignIn() {
  return (
    <div className="container py-5" style={{ maxWidth: "40rem" }}>
      <h1 className="mb-4">Sign in</h1>
      <SignInForm />
      <p className="mt-4">
        New here? <Link href="/open-account">Open an account</Link>.
      </p>
      <p className="mb-0">
        Never enter your password or card details on a page you reached from an
        unexpected message.
      </p>
    </div>
  );
}
