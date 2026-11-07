import axios, { isAxiosError, type InternalAxiosRequestConfig } from "axios";
import Config from "@config";

export type User = {
  id: string;
  name: string;
  email: string;
  two_factor_enabled: boolean;
};

export type Account = {
  id: string;
  number: string;
  type: "checking" | "savings";
  name: string;
  balance_cents: number;
  created_at: string;
};

export type Entry = {
  id: number;
  transaction_id: string;
  account_id: string;
  account_name: string;
  amount_cents: number;
  balance_after_cents: number;
  kind: "deposit" | "internal_transfer" | "external_transfer";
  memo: string;
  counterparty: string;
  created_at: string;
};

export type Statement = {
  account: Account;
  month: string;
  opening_cents: number;
  closing_cents: number;
  credits_cents: number;
  debits_cents: number;
  entries: Entry[];
};

export type Session = {
  id: string;
  user_agent: string;
  ip: string;
  created_at: string;
  last_used_at: string;
  current: boolean;
};

export type Activity = { action: string; ip: string; created_at: string };

type RetriableConfig = InternalAxiosRequestConfig & { retried?: boolean };

export const api = axios.create({ baseURL: Config.getApiBaseUrl() });

let refreshing: Promise<unknown> | null = null;

// The access token lasts 15 minutes, so renew it once and repeat the request.
// Sharing one promise stops parallel requests from each trying to refresh.
api.interceptors.response.use(undefined, async (error) => {
  const original: RetriableConfig | undefined = error.config;
  const expired =
    error.response?.status === 401 &&
    error.response?.data?.code === "unauthorized";

  if (!expired || !original || original.retried || original.url === "auth/refresh") {
    throw error;
  }
  original.retried = true;

  refreshing ??= api.post("auth/refresh").finally(() => {
    refreshing = null;
  });
  try {
    await refreshing;
  } catch {
    // A full load on purpose: it drops any account data held in memory.
    // eslint-disable-next-line @next/next/no-location-assign-relative-destination
    window.location.assign("/sign-in");
    throw error;
  }
  return api(original);
});

export function errorCode(error: unknown): string | undefined {
  return isAxiosError(error) ? error.response?.data?.code : undefined;
}

export function errorMessage(error: unknown): string {
  if (isAxiosError(error) && error.response?.data?.error) {
    return error.response.data.error;
  }
  return "Open Tally can't be reached. Check your connection and try again.";
}
