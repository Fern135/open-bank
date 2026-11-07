"use client";

import { useRouter } from "next/navigation";
import { useState, type FormEvent } from "react";
import { api, errorMessage, type Account } from "../../../lib/api";
import forms from "../../../components/forms.module.css";
import { useBank } from "../../DashboardShell";
import styles from "../../dashboard.module.css";

export default function NewAccount() {
  const router = useRouter();
  const { reloadAccounts } = useBank();
  const [type, setType] = useState("savings");
  const [name, setName] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  async function submit(event: FormEvent) {
    event.preventDefault();
    setBusy(true);
    setError("");
    try {
      const { data } = await api.post<Account>("accounts", { type, name });
      reloadAccounts();
      router.push(`/dashboard/accounts/${data.id}`);
    } catch (failure) {
      setError(errorMessage(failure));
      setBusy(false);
    }
  }

  return (
    <>
      <h1 className={styles.heading}>Open an account</h1>
      <form className={forms.form} onSubmit={submit}>
        <fieldset className={forms.field}>
          <legend className={forms.legend}>Account type</legend>
          <div className={forms.choices}>
            {["savings", "checking"].map((option) => (
              <label key={option}>
                <input
                  type="radio"
                  name="type"
                  value={option}
                  checked={type === option}
                  onChange={() => setType(option)}
                />
                {option === "savings" ? "Savings" : "Checking"}
              </label>
            ))}
          </div>
        </fieldset>
        <div className={forms.field}>
          <label htmlFor="name">Account name</label>
          <input
            id="name"
            required
            maxLength={50}
            placeholder="Rainy day fund"
            value={name}
            onChange={(event) => setName(event.target.value)}
          />
        </div>
        <p className={forms.error} role="alert">{error}</p>
        <div className={forms.buttons}>
          <button className={forms.button} type="submit" disabled={busy}>Open account</button>
        </div>
      </form>
    </>
  );
}
