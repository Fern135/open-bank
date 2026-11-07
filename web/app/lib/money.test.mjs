import assert from "node:assert/strict";
import test from "node:test";
import { parseCents } from "./money.ts";

test("reads amounts into exact cents", () => {
  assert.equal(parseCents("19.99"), 1999);
  assert.equal(parseCents("1.1"), 110);
  assert.equal(parseCents("5"), 500);
  assert.equal(parseCents("0.01"), 1);
  assert.equal(parseCents(" $1,250.50 "), 125050);
  assert.equal(parseCents("4.35"), 435);
});

test("refuses anything that isn't a positive amount", () => {
  for (const input of ["", "0", "0.00", "-5", "abc", "1.999", "1e5", "1,23", ".5", "1.", "99999999999999999"]) {
    assert.equal(parseCents(input), null, input);
  }
});
