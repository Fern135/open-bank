"use client";

import { useState, type FormEvent } from "react";
import { api, errorMessage } from "../../lib/api";
import forms from "../../components/forms.module.css";
import { useBank } from "../DashboardShell";
import styles from "../dashboard.module.css";

function NameForm({ initial }: { initial: string }) {
  const { reloadUser } = useBank();
  const [name, setName] = useState(initial);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");

  async function submit(event: FormEvent) {
    event.preventDefault();
    setError("");
    setSuccess("");
    try {
      await api.patch("me", { name });
      setSuccess("Saved your name.");
      reloadUser();
    } catch (failure) {
      setError(errorMessage(failure));
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
      <p className={forms.error} role="alert">{error}</p>
      <p className={forms.success} role="status">{success}</p>
      <div className={forms.buttons}>
        <button className={forms.button} type="submit">Save name</button>
      </div>
    </form>
  );
}

function PasswordForm() {
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");

  async function submit(event: FormEvent) {
    event.preventDefault();
    setError("");
    setSuccess("");
    try {
      await api.post("me/password", { current_password: current, new_password: next });
      setCurrent("");
      setNext("");
      setSuccess("Changed your password and signed out your other devices.");
    } catch (failure) {
      setError(errorMessage(failure));
    }
  }

  return (
    <form className={forms.form} onSubmit={submit}>
      <div className={forms.field}>
        <label htmlFor="current-password">Current password</label>
        <input
          id="current-password"
          type="password"
          autoComplete="current-password"
          required
          value={current}
          onChange={(event) => setCurrent(event.target.value)}
        />
      </div>
      <div className={forms.field}>
        <label htmlFor="new-password">New password</label>
        <input
          id="new-password"
          type="password"
          autoComplete="new-password"
          required
          minLength={12}
          maxLength={128}
          aria-describedby="new-password-hint"
          value={next}
          onChange={(event) => setNext(event.target.value)}
        />
        <p className={forms.hint} id="new-password-hint">
          At least 12 characters. Changing it signs out your other devices.
        </p>
      </div>
      <p className={forms.error} role="alert">{error}</p>
      <p className={forms.success} role="status">{success}</p>
      <div className={forms.buttons}>
        <button className={forms.button} type="submit">Change password</button>
      </div>
    </form>
  );
}

export default function Profile() {
  const { user } = useBank();

  return (
    <>
      <h1 className={styles.heading}>Profile</h1>
      {user && (
        <>
          <p>You sign in as {user.email}.</p>
          <NameForm key={user.id} initial={user.name} />
        </>
      )}
      <section className={styles.section}>
        <h2 className={styles.subheading}>Password</h2>
        <PasswordForm />
      </section>
    </>
  );
}
