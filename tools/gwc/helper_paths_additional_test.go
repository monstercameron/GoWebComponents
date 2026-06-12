package main

import (
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestDoctorAuditHelpersCoverMappings verifies doctor audit rule and location helper branches.
func TestDoctorAuditHelpersCoverMappings(parseT *testing.T) {
	parseCases := map[string]string{
		"Audit policy":                        "audit.policy",
		"Audit baseline":                      "audit.baseline",
		"Audit target":                        "audit.target",
		"App entrypoint":                      "audit.app_entrypoint",
		"HTML shell":                          "audit.html_shell",
		"Starter metadata anchor":             "audit.metadata_anchor",
		"State and ownership boundaries":      "audit.state_boundaries",
		"Local versus shared state ownership": "audit.state_ownership",
		"Route shape and delivery":            "audit.route_delivery",
		"Mutation and resilience":             "audit.mutation_resilience",
		"Startup cost and ownership evidence": "audit.startup_evidence",
		"Runtime evidence":                    "audit.runtime_evidence",
		"Unknown":                             "",
	}
	for parseName, parseWant := range parseCases {
		if parseGot := doctorAuditRuleIDForName(parseName); parseGot != parseWant {
			parseT.Fatalf("doctorAuditRuleIDForName(%q) = %q, want %q", parseName, parseGot, parseWant)
		}
	}

	parseLocations := extractDoctorAuditLocations("See ui/form.go:12, ui/form.go:12, config/app.json, and app.wasm.")
	if len(parseLocations) != 3 || parseLocations[0] != "ui/form.go" || parseLocations[1] != "config/app.json" || parseLocations[2] != "app.wasm" {
		parseT.Fatalf("unexpected audit locations %#v", parseLocations)
	}
	if parseEmpty := extractDoctorAuditLocations("no file references here"); parseEmpty != nil {
		parseT.Fatalf("expected nil locations for unmatched summary, got %#v", parseEmpty)
	}
}

// TestLintPathHelpersNormalizeInputs verifies lint path and linter list normalization.
func TestLintPathHelpersNormalizeInputs(parseT *testing.T) {
	if parseGot := parseLintPaths(nil); len(parseGot) != 1 || parseGot[0] != "./..." {
		parseT.Fatalf("expected default lint path, got %#v", parseGot)
	}
	if parseGot := parseLintPaths([]string{" ./ui ", "", "./ui", "./internal/runtime", "   ", "./internal/runtime"}); len(parseGot) != 2 || parseGot[0] != "./ui" || parseGot[1] != "./internal/runtime" {
		parseT.Fatalf("unexpected normalized lint paths %#v", parseGot)
	}
	if parseGot := parseLintPaths([]string{" ", "\t"}); len(parseGot) != 1 || parseGot[0] != "./..." {
		parseT.Fatalf("expected fallback lint path after trimming empties, got %#v", parseGot)
	}
	if parseGot := parseLintList([]string{" govet ", "", "govet", "staticcheck", "staticcheck"}); len(parseGot) != 2 || parseGot[0] != "govet" || parseGot[1] != "staticcheck" {
		parseT.Fatalf("unexpected normalized linter list %#v", parseGot)
	}
}

// TestRenderLifecycleMigrateSummaryEmitsReadableOutput verifies the human-readable migrate summary printer.
func TestRenderLifecycleMigrateSummaryEmitsReadableOutput(parseT *testing.T) {
	parseStdout, parseRestoreStdout, parseErr := captureExamplesStdout()
	if parseErr != nil {
		parseT.Fatalf("capture stdout: %v", parseErr)
	}
	defer parseRestoreStdout()

	renderLifecycleMigrateSummary(lifecycleMigrateSummary{
		Root:             "./examples/app",
		MetadataPath:     "./examples/app/gwc-start.json",
		ReportPath:       "./examples/app/docs/migrate-report.md",
		FindingCount:     3,
		RuntimeAssetPath: "./examples/app/public/wasm_exec.js",
	})

	parseOutput, parseErr := parseStdout()
	if parseErr != nil {
		parseT.Fatalf("read captured stdout: %v", parseErr)
	}
	for _, parseWant := range []string{
		"GWC migrate",
		"root:           ./examples/app",
		"metadata:       ./examples/app/gwc-start.json",
		"report:         ./examples/app/docs/migrate-report.md",
		"findings:       3",
		"wasm_exec.js:   ./examples/app/public/wasm_exec.js",
	} {
		if !strings.Contains(parseOutput, parseWant) {
			parseT.Fatalf("expected migrate summary to contain %q, got %q", parseWant, parseOutput)
		}
	}
}

// TestProcessControlHelpersManageDisposableProcess verifies live PID checks and termination helpers.
func TestProcessControlHelpersManageDisposableProcess(parseT *testing.T) {
	if parseErr := terminateLauncherPIDTree(0); parseErr != nil {
		parseT.Fatalf("terminateLauncherPIDTree(0): %v", parseErr)
	}
	if checkLauncherPIDRunning(0) {
		parseT.Fatal("expected pid 0 to report not running")
	}

	parseCmd := buildProcessControlSleepCommand()
	if parseErr := parseCmd.Start(); parseErr != nil {
		parseT.Fatalf("start disposable process: %v", parseErr)
	}

	parsePID := parseCmd.Process.Pid
	if !checkLauncherPIDRunning(parsePID) {
		_ = parseCmd.Process.Kill()
		_, _ = parseCmd.Process.Wait()
		parseT.Fatalf("expected disposable process %d to report running", parsePID)
	}
	if parseErr := terminateLauncherPIDTree(parsePID); parseErr != nil {
		_ = parseCmd.Process.Kill()
		_, _ = parseCmd.Process.Wait()
		parseT.Fatalf("terminateLauncherPIDTree(%d): %v", parsePID, parseErr)
	}

	_, _ = parseCmd.Process.Wait()
	for range 20 {
		if !checkLauncherPIDRunning(parsePID) {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if checkLauncherPIDRunning(parsePID) {
		parseT.Fatalf("expected disposable process %d to stop running", parsePID)
	}

	parseCmd2 := buildProcessControlSleepCommand()
	if parseErr2 := parseCmd2.Start(); parseErr2 != nil {
		parseT.Fatalf("start disposable process for command-tree termination: %v", parseErr2)
	}
	terminateLauncherProcessTree(parseCmd2)
	_, _ = parseCmd2.Process.Wait()
}

// buildProcessControlSleepCommand builds one disposable process that stays alive long enough for termination tests.
func buildProcessControlSleepCommand() *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.Command("cmd", "/c", "ping -n 30 127.0.0.1 > nul")
	}
	return exec.Command("sleep", "30")
}
