import test from "node:test";
import assert from "node:assert/strict";
import { createRequire } from "node:module";

const require = createRequire(import.meta.url);
const { toDiagnostics, severityFor, toCodeActions } = require("../diagnostics.js");

test("severityFor maps gwc severities to VS Code severity numbers", () => {
  assert.equal(severityFor("error"), 0);
  assert.equal(severityFor("warning"), 1);
  assert.equal(severityFor("info"), 2);
  assert.equal(severityFor("hint"), 3);
  assert.equal(severityFor(undefined), 1); // default
  assert.equal(severityFor("mystery"), 1); // unknown -> warning
});

test("toDiagnostics converts 1-based gwc positions to 0-based and tags the source", () => {
  const diags = toDiagnostics({
    issues: [
      { linter: "hookcheck", severity: "error", path: "ui/app.go", line: 10, column: 5, message: "conditional hook" },
    ],
  });
  assert.equal(diags.length, 1);
  assert.equal(diags[0].path, "ui/app.go");
  assert.equal(diags[0].range.startLine, 9);
  assert.equal(diags[0].range.startColumn, 4);
  assert.equal(diags[0].severity, 0);
  assert.equal(diags[0].source, "gwc:hookcheck");
  assert.equal(diags[0].message, "conditional hook");
});

test("toDiagnostics surfaces the symbol (e.g. hook name) when present", () => {
  const diags = toDiagnostics({
    issues: [
      { linter: "gwc-hook-rules", severity: "error", path: "ui/app.go", line: 5, column: 2, message: "called conditionally", symbol: "UseState" },
    ],
  });
  assert.equal(diags[0].symbol, "UseState");
  assert.ok(diags[0].message.startsWith("UseState: "), "message should be symbol-prefixed");
});

test("toDiagnostics tolerates missing fields and empty/empty-ish summaries", () => {
  assert.deepEqual(toDiagnostics({}), []);
  assert.deepEqual(toDiagnostics(null), []);

  const diags = toDiagnostics({ issues: [{ message: "bare" }] });
  assert.equal(diags[0].range.startLine, 0); // line defaults to 1 -> 0-based 0
  assert.equal(diags[0].range.startColumn, 0);
  assert.equal(diags[0].severity, 1); // default warning
  assert.equal(diags[0].source, "gwc"); // no linter
});

test("toCodeActions offers a quick-fix only for fixable issues, at 0-based positions", () => {
  const actions = toCodeActions({
    issues: [
      { linter: "gofmt", path: "ui/app.go", line: 10, column: 5, message: "needs gofmt", fixable: true },
      { linter: "hookcheck", path: "ui/app.go", line: 3, column: 1, message: "conditional hook" }, // not fixable
    ],
  });
  assert.equal(actions.length, 1, "only the fixable issue yields a code-action");
  assert.equal(actions[0].command, "gwc.fix");
  assert.equal(actions[0].path, "ui/app.go");
  assert.equal(actions[0].range.startLine, 9);
  assert.equal(actions[0].range.startColumn, 4);
  assert.equal(actions[0].isPreferred, true);
  assert.ok(actions[0].title.includes("gofmt"), "title names the offending linter");
});

test("toCodeActions ignores fixable issues with no path and tolerates empty summaries", () => {
  assert.deepEqual(toCodeActions({}), []);
  assert.deepEqual(toCodeActions(null), []);
  assert.deepEqual(toCodeActions({ issues: [{ fixable: true, message: "no path" }] }), []);
});
