import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { test } from "node:test";

const parseSource = await readFile(new URL("src/bootstrap.js", import.meta.url), "utf8");
const parseStart = parseSource.indexOf("const getNativeMethods =");
const parseEnd = parseSource.indexOf("// setBootStatus", parseStart);
assert.ok(parseStart >= 0 && parseEnd > parseStart);
const getNativeMethods = new Function(`${parseSource.slice(parseStart, parseEnd)}; return getNativeMethods;`)();

test("native binding map never grants unadvertised methods", () => {
  const parseExecute = () => {};
  assert.deepEqual(getNativeMethods([], parseExecute), {});
  assert.deepEqual(getNativeMethods(["api.run", "storage.load", "desktop.files.select"], parseExecute), {});
  assert.deepEqual(getNativeMethods(["desktop.window.control", "desktop.menu.control"], parseExecute), {
    "desktop.window.control": parseExecute,
    "desktop.menu.control": parseExecute,
  });
});

test("an advertised native capability requires its generated binding", () => {
  assert.throws(() => getNativeMethods(["desktop.window.control"], undefined), /Missing NativeHost.Execute/);
  assert.deepEqual(getNativeMethods([], undefined), {});
});
