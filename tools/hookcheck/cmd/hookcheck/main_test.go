package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestCmdEndToEnd builds and runs the hookcheck binary against a fixture tree,
// asserting the exit code and report — the analyzer working as a real CLI.
func TestCmdEndToEnd(t *testing.T) {
	parseDir := t.TempDir()

	// Build the binary.
	parseBin := filepath.Join(parseDir, "hookcheck.exe")
	parseBuild := exec.Command("go", "build", "-o", parseBin, ".")
	if parseOut, parseErr := parseBuild.CombinedOutput(); parseErr != nil {
		t.Fatalf("build cmd: %v\n%s", parseErr, parseOut)
	}

	// Fixture: a clean file and a violating file, plus one ignored violation.
	parseSrc := parseDir
	parseClean := "package p\nfunc C() { _ = UseState(0) }\n"
	parseBad := "package p\nfunc L(xs []int) {\n\tfor range xs {\n\t\t_ = UseRef(0)\n\t\t_ = UseMemo(0) //hookcheck:ignore\n\t}\n}\n"
	if parseErr := os.WriteFile(filepath.Join(parseSrc, "clean.go"), []byte(parseClean), 0o644); parseErr != nil {
		t.Fatal(parseErr)
	}
	if parseErr := os.WriteFile(filepath.Join(parseSrc, "bad.go"), []byte(parseBad), 0o644); parseErr != nil {
		t.Fatal(parseErr)
	}

	// Run on the violating tree → exit 1, one finding (UseRef; UseMemo ignored).
	parseRun := exec.Command(parseBin, parseSrc)
	parseOut, parseErr := parseRun.CombinedOutput()
	parseExit := 0
	if parseExitErr, parseOk := parseErr.(*exec.ExitError); parseOk {
		parseExit = parseExitErr.ExitCode()
	} else if parseErr != nil {
		t.Fatalf("run: %v\n%s", parseErr, parseOut)
	}
	if parseExit != 1 {
		t.Fatalf("expected exit 1 for violations, got %d\n%s", parseExit, parseOut)
	}
	parseText := string(parseOut)
	if !strings.Contains(parseText, "UseRef") || strings.Contains(parseText, "UseMemo") {
		t.Fatalf("expected only UseRef reported (UseMemo ignored):\n%s", parseText)
	}
	if !strings.Contains(parseText, "1 hook-in-loop violation") {
		t.Fatalf("expected the count summary:\n%s", parseText)
	}

	// Run on a clean-only subdir → exit 0.
	parseCleanDir := filepath.Join(parseDir, "cleanonly")
	if parseErr := os.MkdirAll(parseCleanDir, 0o755); parseErr != nil {
		t.Fatal(parseErr)
	}
	if parseErr := os.WriteFile(filepath.Join(parseCleanDir, "a.go"), []byte(parseClean), 0o644); parseErr != nil {
		t.Fatal(parseErr)
	}
	parseCleanRun := exec.Command(parseBin, parseCleanDir)
	if parseOut2, parseErr2 := parseCleanRun.CombinedOutput(); parseErr2 != nil {
		t.Fatalf("clean run should exit 0, got %v\n%s", parseErr2, parseOut2)
	}
}
