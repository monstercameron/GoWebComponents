# GoWebComponents — VS Code extension

Surfaces `gwc lint --json` diagnostics inline in the editor via a VS Code
`DiagnosticCollection`. On save of a Go file (or the **GWC: Lint workspace**
command) it runs the project's `gwc` CLI, parses the machine-readable lint
summary, and shows each issue at its source location with the right severity.

## Design

The extension is intentionally thin. Its only non-trivial logic — mapping the
`gwc lint --json` summary into VS Code diagnostic descriptors (severity mapping,
1-based → 0-based position conversion, source tagging) — lives in
[`diagnostics.js`](./diagnostics.js), which does **not** import the `vscode`
module and is therefore unit-tested under plain Node:

```sh
npm test   # node --test test/diagnostics.test.mjs
```

[`extension.js`](./extension.js) is the host glue (spawn `gwc`, publish the
mapped diagnostics) and depends on the live VS Code API.

## The contract

The extension consumes the stable CLI contract produced by `gwc lint --json`
(`tools/gwc/lint.go`): a summary whose `issues[]` each carry
`linter`, `severity`, `path`, `line`, `column`, and `message`. Because that
contract is owned and versioned in the Go CLI, the editor integration stays a
thin, language-agnostic consumer.

## Install (development)

Open this folder in VS Code and press **F5** to launch an Extension Development
Host, or package it with `vsce package`. Requires the `gwc` binary on `PATH`.
