"use client";

import { useState, type FormEvent } from "react";
import { api, errorMessage, type Activity, type Session } from "../../lib/api";
import { useData } from "../../lib/useData";
import forms from "../../components/forms.module.css";
import { useBank } from "../DashboardShell";
import { formatDate } from "../Ledger";
import styles from "../dashboard.module.css";

const ACTIVITY_LABELS: Record<string, string> = {
  account_created: "Created your profile",
  bank_account_opened: "Opened a bank account",
  signed_in: "Signed in",
  sign_in_failed: "Failed sign-in, wrong password",
  two_factor_failed: "Failed sign-in, wrong code",
  password_changed: "Changed your password",
  password_change_failed: "Failed password change",
  profile_updated: "Updated your profile",
  session_revoked: "Signed out a device",
  refresh_token_reused: "Ended a session after a reused token",
  two_factor_enabled: "Turned on two-factor sign-in",
  two_factor_disabled: "Turned off two-factor sign-in",
  two_factor_disable_failed: "Failed to turn off two-factor sign-in",
};

type Setup = { secret: string; qr_code: string };

function EnableTwoFactor({ onDone }: { onDone: () => void }) {
  const [setup, setSetup] = useState<Setup | null>(null);
  const [code, setCode] = useState("");
  const [error, setError] = useState("");

  async function start() {
    setError("");
    try {
      const { data } = await api.post<Setup>("2fa/setup");
      setSetup(data);
    } catch (failure) {
      setError(errorMessage(failure));
    }
  }

  async function confirm(event: FormEvent) {
    event.preventDefault();
    setError("");
    try {
      await api.post("2fa/enable", { code });
      onDone();
    } catch (failure) {
      setError(errorMessage(failure));
    }
  }

  if (!setup) {
    return (
      <>
        <p>Two-factor sign-in is off. Turn it on to ask for a code from an authenticator app each time you sign in.</p>
        <p className={forms.error} role="alert">{error}</p>
        <button className={forms.button} type="button" onClick={start}>
          Set up two-factor sign-in
        </button>
      </>
    );
  }

  return (
    <form className={forms.form} onSubmit={confirm}>
      <p className="mb-0">Scan this code with your authenticator app, then enter the 6 digits it shows.</p>
      {/* eslint-disable-next-line @next/next/no-img-element -- a data URI, next/image can't optimize it */}
      <img className={styles.qr} src={setup.qr_code} width={200} height={200} alt="QR code for your authenticator app" />
      <p className={forms.hint}>
        Can&apos;t scan it? Enter this key by hand: <code>{setup.secret}</code>
      </p>
      <div className={forms.field}>
        <label htmlFor="enable-code">6-digit code</label>
        <input
          id="enable-code"
          inputMode="numeric"
          autoComplete="one-time-code"
          pattern="[0-9]{6}"
          maxLength={6}
          required
          value={code}
          onChange={(event) => setCode(event.target.value)}
        />
      </div>
      <p className={forms.error} role="alert">{error}</p>
      <div className={forms.buttons}>
        <button className={forms.button} type="submit">Turn on two-factor sign-in</button>
      </div>
    </form>
  );
}

function DisableTwoFactor({ onDone }: { onDone: () => void }) {
  const [password, setPassword] = useState("");
  const [code, setCode] = useState("");
  const [error, setError] = useState("");

  async function submit(event: FormEvent) {
    event.preventDefault();
    setError("");
    try {
      await api.post("2fa/disable", { password, code });
      onDone();
    } catch (failure) {
      setError(errorMessage(failure));
    }
  }

  return (
    <form className={forms.form} onSubmit={submit}>
      <p className="mb-0">Two-factor sign-in is on. To turn it off, confirm your password and a current code.</p>
      <div className={forms.field}>
        <label htmlFor="disable-password">Password</label>
        <input
          id="disable-password"
          type="password"
          autoComplete="current-password"
          required
          value={password}
          onChange={(event) => setPassword(event.target.value)}
        />
      </div>
      <div className={forms.field}>
        <label htmlFor="disable-code">6-digit code</label>
        <input
          id="disable-code"
          inputMode="numeric"
          autoComplete="one-time-code"
          pattern="[0-9]{6}"
          maxLength={6}
          required
          value={code}
          onChange={(event) => setCode(event.target.value)}
        />
      </div>
      <p className={forms.error} role="alert">{error}</p>
      <div className={forms.buttons}>
        <button className={forms.quietButton} type="submit">Turn off two-factor sign-in</button>
      </div>
    </form>
  );
}

export default function Security() {
  const { user, reloadUser } = useBank();
  const sessions = useData<Session[]>("sessions");
  const activity = useData<Activity[]>("activity");
  const [error, setError] = useState("");

  async function revoke(id: string) {
    setError("");
    try {
      await api.delete(`sessions/${id}`);
      sessions.reload();
      activity.reload();
    } catch (failure) {
      setError(errorMessage(failure));
    }
  }

  function twoFactorChanged() {
    reloadUser();
    activity.reload();
  }

  return (
    <>
      <h1 className={styles.heading}>Security</h1>

      <section>
        <h2 className={styles.subheading}>Two-factor sign-in</h2>
        {user &&
          (user.two_factor_enabled ? (
            <DisableTwoFactor onDone={twoFactorChanged} />
          ) : (
            <EnableTwoFactor onDone={twoFactorChanged} />
          ))}
      </section>

      <section className={styles.section}>
        <h2 className={styles.subheading}>Signed-in devices</h2>
        {sessions.error && <p className={styles.error} role="alert">{sessions.error}</p>}
        <p className={forms.error} role="alert">{error}</p>
        <ul className={styles.rows}>
          {sessions.data?.map((session) => (
            <li key={session.id}>
              <div>
                <span className={styles.rowTitle}>{session.current ? "This device" : "Another device"}</span>
                <span className={styles.sides}>
                  {session.user_agent || "Unknown browser"}, last used {formatDate(session.last_used_at)}, from {session.ip}
                </span>
              </div>
              {!session.current && (
                <button className={forms.quietButton} type="button" onClick={() => revoke(session.id)}>
                  Sign out device
                </button>
              )}
            </li>
          ))}
        </ul>
      </section>

      <section className={styles.section}>
        <h2 className={styles.subheading}>Recent activity</h2>
        {activity.error && <p className={styles.error} role="alert">{activity.error}</p>}
        <ul className={styles.rows}>
          {activity.data?.map((event, index) => (
            <li key={index}>
              <div>
                <span className={styles.rowTitle}>{ACTIVITY_LABELS[event.action] ?? event.action}</span>
                <span className={styles.sides}>{formatDate(event.created_at)}, from {event.ip}</span>
              </div>
            </li>
          ))}
        </ul>
      </section>
    </>
  );
}
