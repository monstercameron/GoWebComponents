// diagnostics.js is the pure, host-independent core of the GoWebComponents VS Code
// extension: it maps a `gwc lint --json` summary into VS Code-shaped diagnostic descriptors.
// It does not import the `vscode` module, so it is unit-testable under plain Node — the
// extension's only non-trivial logic lives here, verified; extension.js is thin host glue.

// VS Code DiagnosticSeverity: Error=0, Warning=1, Information=2, Hint=3.
const SEVERITY = {
  error: 0,
  warning: 1,
  warn: 1,
  info: 2,
  information: 2,
  hint: 3,
};

// severityFor maps a gwc severity string to a VS Code severity number (defaulting to
// Warning for anything unrecognized).
function severityFor(parseSeverity) {
  const parseKey = String(parseSeverity || "warning").toLowerCase();
  return Object.prototype.hasOwnProperty.call(SEVERITY, parseKey) ? SEVERITY[parseKey] : 1;
}

// toDiagnostics maps a `gwc lint --json` summary into plain diagnostic descriptors. gwc
// reports 1-based line/column; VS Code positions are 0-based, so they are converted here.
function toDiagnostics(parseSummary) {
  const parseIssues = (parseSummary && parseSummary.issues) || [];
  return parseIssues.map((parseIssue) => ({
    path: parseIssue.path || "",
    range: {
      startLine: Math.max(0, (parseIssue.line || 1) - 1),
      startColumn: Math.max(0, (parseIssue.column || 1) - 1),
    },
    severity: severityFor(parseIssue.severity),
    // Prefix the symbol (e.g. the offending hook) when present, so the editor shows a
    // symbol-named diagnostic rather than only free text.
    message: parseIssue.symbol ? `${parseIssue.symbol}: ${parseIssue.message || ""}` : parseIssue.message || "",
    symbol: parseIssue.symbol || "",
    source: parseIssue.linter ? `gwc:${parseIssue.linter}` : "gwc",
  }));
}

// toCodeActions maps a `gwc lint --json` summary into plain quick-fix descriptors for the
// issues the underlying linter can autofix (issue.fixable). Each descriptor names the command
// the host runs (gwc.fix) and the file to fix; the actual edit is applied by golangci's own
// verified `gwc lint --fix`, never computed in the editor. Descriptors are host-independent so
// this is unit-testable under plain Node; extension.js turns them into vscode.CodeAction.
function toCodeActions(parseSummary) {
  const parseIssues = (parseSummary && parseSummary.issues) || [];
  return parseIssues
    .filter((parseIssue) => parseIssue && parseIssue.fixable && parseIssue.path)
    .map((parseIssue) => ({
      title: `GWC: fix ${parseIssue.linter || "lint"} issue (gwc lint --fix)`,
      command: "gwc.fix",
      path: parseIssue.path,
      range: {
        startLine: Math.max(0, (parseIssue.line || 1) - 1),
        startColumn: Math.max(0, (parseIssue.column || 1) - 1),
      },
      isPreferred: true,
    }));
}

module.exports = { toDiagnostics, severityFor, toCodeActions };
