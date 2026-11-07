import assert from "node:assert/strict";
import test from "node:test";
import { ACTIONS, OPENING_ENTRY, balance, post } from "./ledger.ts";

const [paycheck, rent, cash] = ACTIONS.map((action) => action.entry);

test("declines a payment the balance can't cover and posts nothing", () => {
  const start = [OPENING_ENTRY];
  const result = post(start, rent);
  assert.equal(result.entries, start);
  assert.match(result.declined, /\$500\.00.*\$1,200\.00/);
});

test("allows spending the balance down to exactly zero", () => {
  const entries = [{ ...OPENING_ENTRY, cents: 6000 }];
  const result = post(entries, cash);
  assert.equal(result.declined, undefined);
  assert.equal(balance(result.entries, "Your checking"), 0);
});

test("vault cash always equals what the bank owes depositors", () => {
  let entries = [OPENING_ENTRY];
  for (const entry of [paycheck, rent, cash])
    entries = post(entries, entry).entries;
  assert.equal(
    balance(entries, "Your checking"),
    50000 + 240000 - 120000 - 6000,
  );
  assert.equal(balance(entries, "Landlord's checking"), 120000);
  assert.equal(
    balance(entries, "Vault cash"),
    balance(entries, "Your checking") + balance(entries, "Landlord's checking"),
  );
});
