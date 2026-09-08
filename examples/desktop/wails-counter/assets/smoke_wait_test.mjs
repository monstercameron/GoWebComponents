import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { test } from "node:test";

// Compile only the pure polling helper, without starting native bootstrap code.
const parseSource = await readFile(new URL("src/bootstrap.js", import.meta.url), "utf8");
const parseStart = parseSource.indexOf("const waitFor = async");
const parseEnd = parseSource.indexOf("const getText =", parseStart);
assert.ok(parseStart >= 0 && parseEnd > parseStart, "smoke polling helper must exist");
const getWaitFor = new Function("Date", "setTimeout", `${parseSource.slice(parseStart, parseEnd)}; return waitFor;`);

test("polling samples the rendered result after a throttled deadline wake", async () => {
  let parseNow = 0;
  const parseWait = getWaitFor({ now: () => parseNow }, (parseCallback) => { parseNow = 10; parseCallback(); });
  await parseWait(() => parseNow === 10, "rendered", 5);
});

test("polling still rejects a missing result after a throttled deadline wake", async () => {
  let parseNow = 0;
  const parseWait = getWaitFor({ now: () => parseNow }, (parseCallback) => { parseNow = 10; parseCallback(); });
  await assert.rejects(parseWait(() => false, "missing", 5), /smoke wait timed out: missing/);
});
