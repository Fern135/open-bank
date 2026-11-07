// Parses what someone typed into whole cents, or null if it isn't an amount.
// Works on the digits as text, because 19.99 * 100 is 1998.9999999999998 in floats.
export function parseCents(input: string): number | null {
  const match = /^\$?(\d{1,3}(?:,\d{3})+|\d+)(?:\.(\d{1,2}))?$/.exec(input.trim());
  if (!match) return null;

  const dollars = match[1].replaceAll(",", "");
  const cents = (match[2] ?? "").padEnd(2, "0");
  const total = Number(dollars + cents);
  return Number.isSafeInteger(total) && total > 0 ? total : null;
}
