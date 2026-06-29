// extension.js is the thin VS Code host glue for the GoWebComponents extension: it runs
// `gwc lint --json` on save and surfaces the results through a DiagnosticCollection. All the
// real mapping logic is in the host-independent, unit-tested diagnostics.js.
const vscode = require("vscode");
const childProcess = require("node:child_process");
const path = require("node:path");
const { toDiagnostics, toCodeActions } = require("./diagnostics.js");

// lastSummary caches the most recent `gwc lint --json` summary so the code-action provider can
// offer quick-fixes for the fixable issues without re-running the linter on every cursor move.
let lastSummary = null;

function activate(parseContext) {
  const parseCollection = vscode.languages.createDiagnosticCollection("gwc");
  parseContext.subscriptions.push(parseCollection);

  const parseRun = () => refresh(parseCollection);
  parseContext.subscriptions.push(vscode.commands.registerCommand("gwc.lint", parseRun));

  // gwc.fix delegates to golangci's own verified `gwc lint --fix`, then re-lints. The editor
  // never computes the edit itself, so a quick-fix applies exactly what the CLI would.
  parseContext.subscriptions.push(
    vscode.commands.registerCommand("gwc.fix", (parsePath) => {
      const parseRoot = workspaceRoot();
      if (!parseRoot) {
        return;
      }
      // When invoked from a quick-fix the offending file's path is passed, so the fix is
      // scoped to that file (-path); the bare command (palette) fixes the whole workspace.
      const parseArgs = parsePath ? ["lint", "--fix", "--path", parsePath] : ["lint", "--fix"];
      childProcess.execFile("gwc", parseArgs, { cwd: parseRoot, maxBuffer: 16 * 1024 * 1024 }, () => {
        parseRun();
      });
    }),
  );

  // Quick-fix provider: surface a code-action for each fixable issue at its range.
  parseContext.subscriptions.push(
    vscode.languages.registerCodeActionsProvider(
      { language: "go" },
      {
        provideCodeActions(parseDoc, parseRange) {
          return codeActionsFor(parseDoc, parseRange);
        },
      },
      { providedCodeActionKinds: [vscode.CodeActionKind.QuickFix] },
    ),
  );

  parseContext.subscriptions.push(
    vscode.workspace.onDidSaveTextDocument((parseDoc) => {
      if (parseDoc.languageId === "go") {
        parseRun();
      }
    }),
  );
  parseRun();
}

function workspaceRoot() {
  return (
    (vscode.workspace.workspaceFolders &&
      vscode.workspace.workspaceFolders[0] &&
      vscode.workspace.workspaceFolders[0].uri.fsPath) ||
    null
  );
}

// codeActionsFor builds VS Code quick-fix actions for the cached summary's fixable issues that
// fall on the requested document and range. The mapping logic lives in toCodeActions (tested).
function codeActionsFor(parseDoc, parseRange) {
  const parseRoot = workspaceRoot();
  if (!parseRoot || !lastSummary) {
    return [];
  }
  const parseActions = [];
  for (const parseDescriptor of toCodeActions(lastSummary)) {
    const parseURI = vscode.Uri.file(path.resolve(parseRoot, parseDescriptor.path));
    if (parseURI.toString() !== parseDoc.uri.toString()) {
      continue;
    }
    const parsePos = new vscode.Position(parseDescriptor.range.startLine, parseDescriptor.range.startColumn);
    if (!parseRange.contains(parsePos) && parseRange.start.line !== parsePos.line) {
      continue;
    }
    const parseAction = new vscode.CodeAction(parseDescriptor.title, vscode.CodeActionKind.QuickFix);
    parseAction.command = {
      command: parseDescriptor.command,
      title: parseDescriptor.title,
      arguments: [parseDescriptor.path],
    };
    parseAction.isPreferred = Boolean(parseDescriptor.isPreferred);
    parseActions.push(parseAction);
  }
  return parseActions;
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
      lastSummary = parseSummary;
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
