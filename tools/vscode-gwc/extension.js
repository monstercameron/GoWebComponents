// extension.js is the thin VS Code host glue for the GoWebComponents extension: it runs
// `gwc lint --json` on save and surfaces the results through a DiagnosticCollection. All the
// real mapping logic is in the host-independent, unit-tested diagnostics.js.
const vscode = require("vscode");
const childProcess = require("node:child_process");
const path = require("node:path");
const { toDiagnostics } = require("./diagnostics.js");

function activate(parseContext) {
  const parseCollection = vscode.languages.createDiagnosticCollection("gwc");
  parseContext.subscriptions.push(parseCollection);

  const parseRun = () => refresh(parseCollection);
  parseContext.subscriptions.push(vscode.commands.registerCommand("gwc.lint", parseRun));
  parseContext.subscriptions.push(
    vscode.workspace.onDidSaveTextDocument((parseDoc) => {
      if (parseDoc.languageId === "go") {
        parseRun();
      }
    }),
  );
  parseRun();
}

function refresh(parseCollection) {
  const parseRoot =
    vscode.workspace.workspaceFolders &&
    vscode.workspace.workspaceFolders[0] &&
    vscode.workspace.workspaceFolders[0].uri.fsPath;
  if (!parseRoot) {
    return;
  }
  childProcess.execFile(
    "gwc",
    ["lint", "--json"],
    { cwd: parseRoot, maxBuffer: 16 * 1024 * 1024 },
    (parseErr, parseStdout) => {
      let parseSummary;
      try {
        parseSummary = JSON.parse(parseStdout);
      } catch (_parse) {
        return;
      }
      parseCollection.clear();
      const parseByFile = new Map();
      for (const parseDiag of toDiagnostics(parseSummary)) {
        const parseURI = vscode.Uri.file(path.resolve(parseRoot, parseDiag.path));
        const parseRange = new vscode.Range(
          parseDiag.range.startLine,
          parseDiag.range.startColumn,
          parseDiag.range.startLine,
          parseDiag.range.startColumn + 1,
        );
        const parseEntry = new vscode.Diagnostic(parseRange, parseDiag.message, parseDiag.severity);
        parseEntry.source = parseDiag.source;
        const parseKey = parseURI.toString();
        if (!parseByFile.has(parseKey)) {
          parseByFile.set(parseKey, { uri: parseURI, diags: [] });
        }
        parseByFile.get(parseKey).diags.push(parseEntry);
      }
      for (const { uri, diags } of parseByFile.values()) {
        parseCollection.set(uri, diags);
      }
    },
  );
}

module.exports = { activate, deactivate() {} };
