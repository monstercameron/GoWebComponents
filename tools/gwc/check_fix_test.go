package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestCheckFixFormatsProject proves `gwc check --fix` applies gofmt before checking — the
// post-edit hook AGENTS.md documents, which previously errored because the flag did not
// exist.
func TestCheckFixFormatsProject(parseT *testing.T) {
	parseDir := parseT.TempDir()
	if parseErr := os.WriteFile(filepath.Join(parseDir, "go.mod"), []byte("module tmpcheckfix\n\ngo 1.26\n"), 0644); parseErr != nil {
		parseT.Fatalf("write go.mod: %v", parseErr)
	}
	parseBad := filepath.Join(parseDir, "main.go")
	if parseErr := os.WriteFile(parseBad, []byte("package main\nfunc main(){x:=1;_=x}\n"), 0644); parseErr != nil {
		parseT.Fatalf("write main.go: %v", parseErr)
	}

	// Sanity: gofmt flags the file before the fix.
	if !gofmtFlags(parseT, parseBad) {
		parseT.Fatal("fixture should be mis-formatted before --fix")
	}

	parseLauncher := launcher{}
	if parseErr := parseLauncher.runCheck([]string{"--fix", "-root", parseDir, "-skip-tests", "-skip-hooks", "-skip-conventions"}); parseErr != nil {
		parseT.Fatalf("gwc check --fix returned error: %v", parseErr)
	}

	// After --fix the file must be gofmt-clean.
	if gofmtFlags(parseT, parseBad) {
		parseData, _ := os.ReadFile(parseBad)
		parseT.Fatalf("--fix should have gofmt-ed the file; still flagged:\n%s", parseData)
	}
}

// gofmtFlags reports whether gofmt -l flags the file (i.e. it needs formatting).
func gofmtFlags(parseT *testing.T, parsePath string) bool {
	parseT.Helper()
	parseOut, parseErr := exec.Command("gofmt", "-l", parsePath).Output()
	if parseErr != nil {
		parseT.Fatalf("gofmt -l: %v", parseErr)
	}
	return len(parseOut) > 0
}
