package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCheckFixRewritesServerLeakConstraint proves `gwc check --fix`'s structured remediation:
// a browser (js && wasm) file that imports a server-only package has its build constraint
// rewritten to `//go:build !js || !wasm`, moving it server-side — not merely reformatted. After
// the fix the leak is gone, and a second pass is a no-op (idempotent).
func TestCheckFixRewritesServerLeakConstraint(parseT *testing.T) {
	parseDir := parseT.TempDir()
	mustWrite(parseT, parseDir, "go.mod", "module example.com/app\n\ngo 1.26.0\n")
	mustWrite(parseT, parseDir, "client.go", "//go:build js && wasm\n\npackage app\n\nimport _ \"os/exec\"\n")

	parseDiags := collectServerLeakDiagnostics(parseDir)
	if len(parseDiags) == 0 {
		parseT.Fatal("setup: expected a server leak before fixing")
	}

	parseApplied, parseManual, parseErr := applyAgenticEdits(parseDir, parseDiags)
	if parseErr != nil {
		parseT.Fatalf("applyAgenticEdits: %v", parseErr)
	}
	if len(parseApplied) != 1 {
		parseT.Fatalf("expected exactly one applied fix, got %d: %#v", len(parseApplied), parseApplied)
	}
	if parseManual[checkHookContextCode] != 0 {
		parseT.Fatalf("server leak should not be counted as manual: %#v", parseManual)
	}

	parseContent, _ := os.ReadFile(filepath.Join(parseDir, "client.go"))
	if !strings.Contains(string(parseContent), "//go:build !js || !wasm") {
		parseT.Fatalf("constraint should be rewritten to server-only, got:\n%s", parseContent)
	}
	if strings.Contains(string(parseContent), "//go:build js && wasm") {
		parseT.Fatalf("old client constraint should be gone, got:\n%s", parseContent)
	}

	// The leak is resolved...
	if parseLeaks := collectServerLeakDiagnostics(parseDir); len(parseLeaks) != 0 {
		parseT.Fatalf("expected no leaks after fix, got %#v", parseLeaks)
	}
	// ...and re-applying the (now empty) edit set changes nothing.
	parseAppliedAgain, _, _ := applyAgenticEdits(parseDir, collectServerLeakDiagnostics(parseDir))
	if len(parseAppliedAgain) != 0 {
		parseT.Fatalf("fix should be idempotent, re-applied %d", len(parseAppliedAgain))
	}
}

// TestCheckFixReportsManualOnlyDiagnostics proves a diagnostic that carries no deterministic
// edit (a hook called outside a component) is reported as manual, not silently swallowed or
// falsely "fixed".
func TestCheckFixReportsManualOnlyDiagnostics(parseT *testing.T) {
	parseManual := map[string]int{}
	parseDiags := []agenticDiagnostic{
		{Code: checkHookContextCode, Severity: "error", Message: "hook outside component"},
		{Code: "GWC-CHECK-SERVER-LEAK", Edits: []agenticTextEdit{{File: "missing.go", OldText: "x", NewText: "y"}}},
	}
	parseApplied, parseManual, parseErr := applyAgenticEdits(parseT.TempDir(), parseDiags)
	if parseErr != nil {
		parseT.Fatalf("applyAgenticEdits: %v", parseErr)
	}
	// The server-leak edit targets a missing file → idempotent no-op (not applied, not manual).
	if len(parseApplied) != 0 {
		parseT.Fatalf("no edits should apply against a missing file, got %#v", parseApplied)
	}
	if parseManual[checkHookContextCode] != 1 {
		parseT.Fatalf("the hook diagnostic should be reported as 1 manual fix, got %#v", parseManual)
	}
	if parseRemediation := manualRemediationFor(checkHookContextCode); !strings.Contains(parseRemediation, "top level") {
		parseT.Fatalf("manual remediation should guide hoisting the hook, got %q", parseRemediation)
	}
}
