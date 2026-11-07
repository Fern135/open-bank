export type Account = "Vault cash" | "Your checking" | "Landlord's checking";

export type Entry = {
  memo: string;
  debit: Account;
  credit: Account;
  cents: number;
};

export const OPENING_ENTRY: Entry = {
  memo: "Opening deposit",
  debit: "Vault cash",
  credit: "Your checking",
  cents: 50000,
};

export const ACTIONS = [
  {
    label: "Deposit paycheck",
    icon: "fa-money-check-dollar",
    entry: {
      memo: "Paycheck deposited",
      debit: "Vault cash",
      credit: "Your checking",
      cents: 240000,
    },
  },
  {
    label: "Pay rent",
    icon: "fa-house",
    entry: {
      memo: "Rent paid to landlord",
      debit: "Your checking",
      credit: "Landlord's checking",
      cents: 120000,
    },
  },
  {
    label: "Withdraw cash",
    icon: "fa-money-bill-wave",
    entry: {
      memo: "Cash withdrawn",
      debit: "Your checking",
      credit: "Vault cash",
      cents: 6000,
    },
  },
] as const satisfies readonly { label: string; icon: string; entry: Entry }[];

const usd = new Intl.NumberFormat("en-US", {
  style: "currency",
  currency: "USD",
});

export const formatCents = (cents: number) => usd.format(cents / 100);

export function balance(entries: Entry[], account: Account) {
  // Vault cash is an asset, so debits raise it. Deposits are debts the bank owes, so credits raise them.
  const sign = account === "Vault cash" ? 1 : -1;
  return entries.reduce((sum, entry) => {
    if (entry.debit === account) return sum + sign * entry.cents;
    if (entry.credit === account) return sum - sign * entry.cents;
    return sum;
  }, 0);
}

export function post(entries: Entry[], entry: Entry) {
  const available = balance(entries, "Your checking");
  if (entry.debit === "Your checking" && available < entry.cents) {
    return {
      entries,
      declined: `Declined. Your checking has ${formatCents(available)}, and this needs ${formatCents(entry.cents)}. Nothing was posted.`,
    };
  }
  return { entries: [...entries, entry] };
}
