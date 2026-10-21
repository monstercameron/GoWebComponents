package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// useRetryingDir creates a scratch directory whose removal retries with backoff, because
// on Windows a freshly built and run *.exe stays locked for a moment (file scanner) and
// the stock t.TempDir cleanup fails with "Access is denied". A lock that outlives the
// window is logged rather than failed; it does not affect what the test asserts.
func useRetryingDir(t *testing.T) string {
	t.Helper()
	parseDir, parseErr := os.MkdirTemp("", "hookcheck-e2e-")
	if parseErr != nil {
		t.Fatal(parseErr)
	}
	t.Cleanup(func() {
		parseDeadline := time.Now().Add(10 * time.Second)
		parseDelay := 50 * time.Millisecond
		for {
			parseRemoveErr := os.RemoveAll(parseDir)
			if parseRemoveErr == nil {
				return
			}
			if time.Now().Add(parseDelay).After(parseDeadline) {
				t.Logf("scratch dir %s not fully removed: %v", parseDir, parseRemoveErr)
				return
			}
			time.Sleep(parseDelay)
			if parseDelay < time.Second {
				parseDelay *= 2
			}
		}
	})
	return parseDir
}

// TestCmdEndToEnd builds and runs the hookcheck binary against a fixture tree,
// asserting the exit code and report — the analyzer working as a real CLI.
func TestCmdEndToEnd(t *testing.T) {
	parseDir := useRetryingDir(t)
	t.Setenv("GOTMPDIR", useRetryingDir(t))

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
