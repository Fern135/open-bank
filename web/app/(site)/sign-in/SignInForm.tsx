"use client";

import { useRouter } from "next/navigation";
import { useState, type FormEvent } from "react";
import { api, errorCode, errorMessage } from "../../lib/api";
import forms from "../../components/forms.module.css";

export default function SignInForm() {
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [code, setCode] = useState("");
  const [needsCode, setNeedsCode] = useState(false);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  async function submit(event: FormEvent) {
    event.preventDefault();
    setBusy(true);
    setError("");
    try {
      await api.post("auth/login", { email, password, code });
      router.push("/dashboard");
    } catch (failure) {
      if (errorCode(failure) === "two_factor_required") {
        setNeedsCode(true);
      } else {
        setError(errorMessage(failure));
      }
      setBusy(false);
    }
  }

  return (
    <form className={forms.form} onSubmit={submit}>
      <div className={forms.field}>
        <label htmlFor="email">Email</label>
        <input
          id="email"
          type="email"
          autoComplete="username"
          required
          value={email}
          onChange={(event) => setEmail(event.target.value)}
        />
      </div>
      <div className={forms.field}>
        <label htmlFor="password">Password</label>
        <input
          id="password"
          type="password"
          autoComplete="current-password"
          required
          value={password}
          onChange={(event) => setPassword(event.target.value)}
        />
      </div>
      {needsCode && (
        <div className={forms.field}>
          <label htmlFor="code">6-digit code from your authenticator app</label>
          <input
            id="code"
            inputMode="numeric"
            autoComplete="one-time-code"
            pattern="[0-9]{6}"
            maxLength={6}
            required
            autoFocus
            value={code}
            onChange={(event) => setCode(event.target.value)}
          />
        </div>
      )}
      <p className={forms.error} role="alert">
        {error}
      </p>
      <div className={forms.buttons}>
        <button className={forms.button} type="submit" disabled={busy}>
          {busy ? "Signing in" : "Sign in"}
        </button>
      </div>
    </form>
  );
}
