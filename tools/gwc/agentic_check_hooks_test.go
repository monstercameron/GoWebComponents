package main

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// hookContextFixture is a single-file GWC consumer exercising every hook-context
// case: two valid (component, Use* hook) and three invalid (ordinary helper,
// effect closure, package-level initializer).
const hookContextFixture = `package sample

import (
	"github.com/monstercameron/GoWebComponents/v5/state"
	"github.com/monstercameron/GoWebComponents/v5/ui"
)

// valid: a Use* hook function may call hooks.
func UsePrefs() state.Atom[int] {
	return state.UseAtom("prefs", 0)
}

// valid: a component (returns ui.Node) may call hooks at its top level.
func widget(struct{}) ui.Node {
	a := UsePrefs()
	ui.UseEffect(func() func() {
		recordSnapshot() // not a hook call; fine
		return nil
	}, "")
	_ = a
	return nil
}

// invalid: ordinary helper, neither component nor Use*, calls a hook.
func recordSnapshot() {
	state.UseAtom("snap", 0)
}

// invalid: hook inside an effect closure.
func widgetBad(struct{}) ui.Node {
	ui.UseEffect(func() func() {
		state.UseAtom("inside-effect", 0)
		return nil
	}, "")
	return nil
}

// invalid: package-level initializer calls a hook.
var bootAtom = state.UseAtom("boot", 0)
`

func TestCollectCheckHookContextDiagnosticsFlagsMisuse(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	if parseErr := os.WriteFile(filepath.Join(parseRoot, "sample.go"), []byte(hookContextFixture), 0644); parseErr != nil {
		parseT.Fatalf("write fixture: %v", parseErr)
	}

	parseDiagnostics := collectCheckHookContextDiagnostics(parseRoot)
	if len(parseDiagnostics) != 3 {
		parseT.Fatalf("expected 3 hook-context diagnostics, got %d: %+v", len(parseDiagnostics), parseDiagnostics)
	}

	parseFunctions := []string{}
	for _, parseDiagnostic := range parseDiagnostics {
		if parseDiagnostic.Code != checkHookContextCode {
			parseT.Fatalf("unexpected diagnostic code %q", parseDiagnostic.Code)
		}
		if parseDiagnostic.Severity != "error" {
			parseT.Fatalf("expected error severity, got %q", parseDiagnostic.Severity)
		}
		if parseDiagnostic.Attributes["docs"] != checkHookContextDocs {
			parseT.Fatalf("expected docs anchor attribute, got %q", parseDiagnostic.Attributes["docs"])
		}
		if parseFn := parseDiagnostic.Attributes["function"]; parseFn != "" {
			parseFunctions = append(parseFunctions, parseFn)
		}
	}

	sort.Strings(parseFunctions)
	// Only the ordinary-helper case carries a function name; the closure and
	// package-level cases do not.
	if len(parseFunctions) != 1 || parseFunctions[0] != "recordSnapshot" {
		parseT.Fatalf("expected the ordinary helper recordSnapshot to be named, got %v", parseFunctions)
	}
}

func TestCollectCheckHookContextDiagnosticsIgnoresValidContexts(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseValid := `package sample

import (
	"github.com/monstercameron/GoWebComponents/v5/state"
	"github.com/monstercameron/GoWebComponents/v5/ui"
)

func UseThing() state.Atom[int] { return state.UseAtom("thing", 0) }

func screen(struct{}) ui.Node {
	_ = UseThing()
	return nil
}
`
	if parseErr := os.WriteFile(filepath.Join(parseRoot, "valid.go"), []byte(parseValid), 0644); parseErr != nil {
		parseT.Fatalf("write fixture: %v", parseErr)
	}
	if parseDiagnostics := collectCheckHookContextDiagnostics(parseRoot); len(parseDiagnostics) != 0 {
		parseT.Fatalf("expected no diagnostics for valid hook usage, got %+v", parseDiagnostics)
	}
}

func TestCollectCheckHookContextDiagnosticsSkipsNonHookFiles(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parsePlain := "package sample\n\nfunc Add(a, b int) int { return a + b }\n"
	if parseErr := os.WriteFile(filepath.Join(parseRoot, "plain.go"), []byte(parsePlain), 0644); parseErr != nil {
		parseT.Fatalf("write fixture: %v", parseErr)
	}
	if parseDiagnostics := collectCheckHookContextDiagnostics(parseRoot); len(parseDiagnostics) != 0 {
		parseT.Fatalf("expected no diagnostics for a file without GWC hook imports, got %+v", parseDiagnostics)
	}
}
