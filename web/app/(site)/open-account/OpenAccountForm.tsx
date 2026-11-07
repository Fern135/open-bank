"use client";

import { useRouter } from "next/navigation";
import { useState, type FormEvent } from "react";
import { api, errorMessage } from "../../lib/api";
import forms from "../../components/forms.module.css";

export default function OpenAccountForm() {
  const router = useRouter();
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  async function submit(event: FormEvent) {
    event.preventDefault();
    setBusy(true);
    setError("");
    try {
      await api.post("users", { name, email, password });
      router.push("/dashboard");
    } catch (failure) {
      setError(errorMessage(failure));
      setBusy(false);
    }
  }

  return (
    <form className={forms.form} onSubmit={submit}>
      <div className={forms.field}>
        <label htmlFor="name">Full name</label>
        <input
          id="name"
          autoComplete="name"
          required
          maxLength={100}
          value={name}
          onChange={(event) => setName(event.target.value)}
        />
      </div>
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
          autoComplete="new-password"
          required
          minLength={12}
          maxLength={128}
          aria-describedby="password-hint"
          value={password}
          onChange={(event) => setPassword(event.target.value)}
        />
        <p className={forms.hint} id="password-hint">
          At least 12 characters. A few random words work well.
        </p>
      </div>
      <p className={forms.error} role="alert">
        {error}
      </p>
      <div className={forms.buttons}>
        <button className={forms.button} type="submit" disabled={busy}>
          {busy ? "Opening your account" : "Open my account"}
        </button>
      </div>
    </form>
  );
}
