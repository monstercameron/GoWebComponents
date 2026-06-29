package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunVerifyJSONRunsTestsAndCIBuild(parseT *testing.T) {
	parseTempApp := parseT.TempDir()
	parseGoModPath := filepath.Join(parseTempApp, "go.mod")
	parseMainPath := filepath.Join(parseTempApp, "main.go")
	parseTestPath := filepath.Join(parseTempApp, "main_test.go")
	if parseErr := os.WriteFile(parseGoModPath, []byte("module example.com/gwcverifytest\n\ngo 1.25.0\n"), 0644); parseErr != nil {
		parseT.Fatalf("write go.mod: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(parseMainPath, []byte("package main\nfunc main() {}\n"), 0644); parseErr2 != nil {
		parseT.Fatalf("write main.go: %v", parseErr2)
	}
	if parseErr3 := os.WriteFile(parseTestPath, []byte("package main\nimport \"testing\"\nfunc TestSmoke(t *testing.T) {}\n"), 0644); parseErr3 != nil {
		parseT.Fatalf("write main_test.go: %v", parseErr3)
	}

	parseStdout, parseRestoreStdout, parseErr4 := captureExamplesStdout()
	if parseErr4 != nil {
		parseT.Fatalf("capture stdout: %v", parseErr4)
	}
	defer parseRestoreStdout()

	parseLauncher := launcher{}
	if parseErr5 := parseLauncher.run([]string{"verify", "-app", parseMainPath, "-root", parseTempApp, "-json"}); parseErr5 != nil {
		parseT.Fatalf("run verify: %v", parseErr5)
	}

	parseOutput, parseErr4 := parseStdout()
	if parseErr4 != nil {
		parseT.Fatalf("read captured stdout: %v", parseErr4)
	}
	var parseSummary verifySummary
	if parseErr6 := json.Unmarshal([]byte(parseOutput), &parseSummary); parseErr6 != nil {
		parseT.Fatalf("unmarshal verify summary: %v\n%s", parseErr6, parseOutput)
	}
	if !parseSummary.OK {
		parseT.Fatalf("expected successful verify summary, got %#v", parseSummary)
	}
	if !parseSummary.Tests.Ran || parseSummary.Tests.Skipped {
		parseT.Fatalf("expected verify to run go tests, got %#v", parseSummary)
	}
	if parseSummary.Build.Profile.Name != "ci" {
		parseT.Fatalf("expected verify build profile ci, got %#v", parseSummary)
	}
	if parseInfo, parseErr7 := os.Stat(parseSummary.Build.OutputPath); parseErr7 != nil || parseInfo.Size() <= 0 {
		parseT.Fatalf("expected verify build artifact at %q, stat err=%v size=%v", parseSummary.Build.OutputPath, parseErr7, parseInfo)
	}
}

func TestRunVerifyJSONSkipsTestsWhenProjectHasNoGoTests(parseT *testing.T) {
	parseTempApp := parseT.TempDir()
	parseGoModPath := filepath.Join(parseTempApp, "go.mod")
	parseMainPath := filepath.Join(parseTempApp, "main.go")
	if parseErr := os.WriteFile(parseGoModPath, []byte("module example.com/gwcverifyskip\n\ngo 1.25.0\n"), 0644); parseErr != nil {
		parseT.Fatalf("write go.mod: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(parseMainPath, []byte("package main\nfunc main() {}\n"), 0644); parseErr2 != nil {
		parseT.Fatalf("write main.go: %v", parseErr2)
	}

	parseStdout, parseRestoreStdout, parseErr3 := captureExamplesStdout()
	if parseErr3 != nil {
		parseT.Fatalf("capture stdout: %v", parseErr3)
	}
	defer parseRestoreStdout()

	parseLauncher := launcher{}
	if parseErr4 := parseLauncher.run([]string{"verify", "-app", parseMainPath, "-root", parseTempApp, "-json"}); parseErr4 != nil {
		parseT.Fatalf("run verify: %v", parseErr4)
	}

	parseOutput, parseErr3 := parseStdout()
	if parseErr3 != nil {
		parseT.Fatalf("read captured stdout: %v", parseErr3)
	}
	var parseSummary verifySummary
	if parseErr5 := json.Unmarshal([]byte(parseOutput), &parseSummary); parseErr5 != nil {
		parseT.Fatalf("unmarshal verify summary: %v\n%s", parseErr5, parseOutput)
	}
	if !parseSummary.OK {
		parseT.Fatalf("expected successful verify summary, got %#v", parseSummary)
	}
	if parseSummary.Tests.Ran || !parseSummary.Tests.Skipped {
		parseT.Fatalf("expected verify to skip tests when no _test.go files exist, got %#v", parseSummary)
	}
	if parseSummary.Build.Profile.Name != "ci" {
		parseT.Fatalf("expected verify build profile ci, got %#v", parseSummary)
	}
}

func TestRunVerifyPropagatesGoTestFailure(parseT *testing.T) {
	parseTempApp := parseT.TempDir()
	parseGoModPath := filepath.Join(parseTempApp, "go.mod")
	parseMainPath := filepath.Join(parseTempApp, "main.go")
	parseTestPath := filepath.Join(parseTempApp, "main_test.go")
	if parseErr := os.WriteFile(parseGoModPath, []byte("module example.com/gwcverifyfail\n\ngo 1.25.0\n"), 0644); parseErr != nil {
		parseT.Fatalf("write go.mod: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(parseMainPath, []byte("package main\nfunc main() {}\n"), 0644); parseErr2 != nil {
		parseT.Fatalf("write main.go: %v", parseErr2)
	}
	if parseErr3 := os.WriteFile(parseTestPath, []byte("package main\nimport \"testing\"\nfunc TestFail(t *testing.T) { t.Fatal(\"boom\") }\n"), 0644); parseErr3 != nil {
		parseT.Fatalf("write main_test.go: %v", parseErr3)
	}

	parseLauncher := launcher{}
	parseErr4 := parseLauncher.run([]string{"verify", "-app", parseMainPath, "-root", parseTempApp})
	if parseErr4 == nil {
		parseT.Fatal("expected verify to fail when go test fails")
	}
	if !strings.Contains(parseErr4.Error(), "go test failed") || !strings.Contains(parseErr4.Error(), "boom") {
		parseT.Fatalf("expected go test failure to be surfaced, got %v", parseErr4)
	}
}

func TestRunVerifySkipTestsSkipsEvenWhenTestsExist(parseT *testing.T) {
	parseTempApp := parseT.TempDir()
	parseGoModPath := filepath.Join(parseTempApp, "go.mod")
	parseMainPath := filepath.Join(parseTempApp, "main.go")
	parseTestPath := filepath.Join(parseTempApp, "main_test.go")
	if parseErr := os.WriteFile(parseGoModPath, []byte("module example.com/gwcverifyskipflag\n\ngo 1.25.0\n"), 0644); parseErr != nil {
		parseT.Fatalf("write go.mod: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(parseMainPath, []byte("package main\nfunc main() {}\n"), 0644); parseErr2 != nil {
		parseT.Fatalf("write main.go: %v", parseErr2)
	}
	if parseErr3 := os.WriteFile(parseTestPath, []byte("package main\nimport \"testing\"\nfunc TestSmoke(t *testing.T) {}\n"), 0644); parseErr3 != nil {
		parseT.Fatalf("write main_test.go: %v", parseErr3)
	}

	parseStdout, parseRestoreStdout, parseErr4 := captureExamplesStdout()
	if parseErr4 != nil {
		parseT.Fatalf("capture stdout: %v", parseErr4)
	}
	defer parseRestoreStdout()

	parseLauncher := launcher{}
	if parseErr5 := parseLauncher.run([]string{"verify", "-app", parseMainPath, "-root", parseTempApp, "-skip-tests", "-json"}); parseErr5 != nil {
		parseT.Fatalf("run verify with skip-tests: %v", parseErr5)
	}

	parseOutput, parseErr4 := parseStdout()
	if parseErr4 != nil {
		parseT.Fatalf("read captured stdout: %v", parseErr4)
	}
	var parseSummary verifySummary
	if parseErr6 := json.Unmarshal([]byte(parseOutput), &parseSummary); parseErr6 != nil {
		parseT.Fatalf("unmarshal verify summary: %v\n%s", parseErr6, parseOutput)
	}
	if parseSummary.Tests.Ran || !parseSummary.Tests.Skipped {
		parseT.Fatalf("expected skip-tests to skip go tests even when tests exist, got %#v", parseSummary)
	}
}

func TestRunVerifyAuditJSONIncludesAuditReport(parseT *testing.T) {
	parseTempApp := parseT.TempDir()
	parseGoModPath := filepath.Join(parseTempApp, "go.mod")
	parseMainPath := filepath.Join(parseTempApp, "main.go")
	parseHtmlPath := filepath.Join(parseTempApp, "index.html")
	if parseErr := os.WriteFile(parseGoModPath, []byte("module example.com/gwcverifyaudit\n\ngo 1.25.0\n"), 0644); parseErr != nil {
		parseT.Fatalf("write go.mod: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(parseMainPath, []byte("package main\nfunc main() {}\n"), 0644); parseErr2 != nil {
		parseT.Fatalf("write main.go: %v", parseErr2)
	}
	if parseErr3 := os.WriteFile(parseHtmlPath, []byte("<!doctype html><html><body><div id=\"app\"></div></body></html>\n"), 0644); parseErr3 != nil {
		parseT.Fatalf("write index.html: %v", parseErr3)
	}

	parseStdout, parseRestoreStdout, parseErr4 := captureExamplesStdout()
	if parseErr4 != nil {
		parseT.Fatalf("capture stdout: %v", parseErr4)
	}
	defer parseRestoreStdout()

	parseLauncher := launcher{}
	if parseErr5 := parseLauncher.run([]string{"verify", "-app", parseMainPath, "-root", parseTempApp, "-skip-tests", "-audit", "-audit-policy", "advisory", "-json"}); parseErr5 != nil {
		parseT.Fatalf("run verify with audit: %v", parseErr5)
	}

	parseOutput, parseErr4 := parseStdout()
	if parseErr4 != nil {
		parseT.Fatalf("read captured stdout: %v", parseErr4)
	}
	var parseSummary verifySummary
	if parseErr6 := json.Unmarshal([]byte(parseOutput), &parseSummary); parseErr6 != nil {
		parseT.Fatalf("unmarshal verify summary: %v\n%s", parseErr6, parseOutput)
	}
	if !parseSummary.OK {
		parseT.Fatalf("expected successful verify summary, got %#v", parseSummary)
	}
	if parseSummary.Audit == nil {
		parseT.Fatalf("expected audit report in verify summary, got %#v", parseSummary)
	}
	if parseSummary.Audit.Mode != "golden-path" || parseSummary.Audit.Policy != "advisory" || !parseSummary.Audit.OK {
		parseT.Fatalf("expected advisory golden-path audit report, got %#v", parseSummary.Audit)
	}
}

func TestRunVerifyAuditStrictFailureReturnsError(parseT *testing.T) {
	parseTempApp := parseT.TempDir()
	parseGoModPath := filepath.Join(parseTempApp, "go.mod")
	parseMainPath := filepath.Join(parseTempApp, "main.go")
	parseBadClientPath := filepath.Join(parseTempApp, "client", "boundary.go")
	if parseErr := os.WriteFile(parseGoModPath, []byte("module example.com/gwcverifyauditfail\n\ngo 1.25.0\n"), 0644); parseErr != nil {
		parseT.Fatalf("write go.mod: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(parseMainPath, []byte("package main\nfunc main() {}\n"), 0644); parseErr2 != nil {
		parseT.Fatalf("write main.go: %v", parseErr2)
	}
	if parseErr3 := os.MkdirAll(filepath.Dir(parseBadClientPath), 0755); parseErr3 != nil {
		parseT.Fatalf("mkdir client dir: %v", parseErr3)
	}
	if parseErr4 := os.WriteFile(parseBadClientPath, []byte("//go:build js && wasm\n// +build js,wasm\n\npackage client\n\nimport _ \"database/sql\"\n"), 0644); parseErr4 != nil {
		parseT.Fatalf("write failing audit fixture: %v", parseErr4)
	}

	parseStdout, parseRestoreStdout, parseErr5 := captureExamplesStdout()
	if parseErr5 != nil {
		parseT.Fatalf("capture stdout: %v", parseErr5)
	}
	defer parseRestoreStdout()

	parseErr5 = (launcher{}).run([]string{"verify", "-app", parseMainPath, "-root", parseTempApp, "-skip-tests", "-audit", "-json"})
	if parseErr5 == nil || !strings.Contains(parseErr5.Error(), "verify audit found error-severity findings") {
		parseT.Fatalf("expected strict audit verify failure, got %v", parseErr5)
	}

	parseOutput, parseReadErr := parseStdout()
	if parseReadErr != nil {
		parseT.Fatalf("read captured stdout: %v", parseReadErr)
	}
	var parseSummary verifySummary
	if parseErr6 := json.Unmarshal([]byte(parseOutput), &parseSummary); parseErr6 != nil {
		parseT.Fatalf("unmarshal verify summary: %v\n%s", parseErr6, parseOutput)
	}
	if parseSummary.OK {
		parseT.Fatalf("expected failing verify summary, got %#v", parseSummary)
	}
	if parseSummary.Audit == nil || parseSummary.Audit.Policy != "strict" || parseSummary.Audit.OK {
		parseT.Fatalf("expected failing strict audit report, got %#v", parseSummary.Audit)
	}
}

func TestRunVerifyPropagatesProjectScanFailure(parseT *testing.T) {
	parseTempApp := parseT.TempDir()
	parseMainPath := filepath.Join(parseTempApp, "main.go")
	if parseErr := os.WriteFile(filepath.Join(parseTempApp, "go.mod"), []byte("module example.com/gwcverifyscanfail\n\ngo 1.25.0\n"), 0644); parseErr != nil {
		parseT.Fatalf("write go.mod: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(parseMainPath, []byte("package main\nfunc main() {}\n"), 0644); parseErr2 != nil {
		parseT.Fatalf("write main.go: %v", parseErr2)
	}
	parseMissingRoot := filepath.Join(parseT.TempDir(), "missing-root")

	parseErr3 := (launcher{}).run([]string{"verify", "-app", parseMainPath, "-root", parseMissingRoot})
	if parseErr3 == nil || !strings.Contains(parseErr3.Error(), "scan project tests") {
		parseT.Fatalf("expected project scan failure, got %v", parseErr3)
	}
}

func TestRunVerifyHelpReturnsNil(parseT *testing.T) {
	parseLauncher := launcher{}
	if parseErr := parseLauncher.run([]string{"verify", "-help"}); parseErr != nil {
		parseT.Fatalf("expected verify help to succeed, got %v", parseErr)
	}
}

func TestRunVerifyHandlesInvalidFlags(parseT *testing.T) {
	if parseErr := (launcher{}).runVerify([]string{"-definitely-invalid"}); parseErr == nil || !strings.Contains(parseErr.Error(), "flag provided but not defined") {
		parseT.Fatalf("expected invalid verify flag error, got %v", parseErr)
	}
}

func TestRunVerifyEnforcesEnterpriseRequiredLanes(parseT *testing.T) {
	parseTempApp := parseT.TempDir()
	parseMainPath := filepath.Join(parseTempApp, "main.go")
	if parseErr := os.WriteFile(filepath.Join(parseTempApp, "go.mod"), []byte("module example.com/gwcverifyrequiredlane\n\ngo 1.25.0\n"), 0644); parseErr != nil {
		parseT.Fatalf("write go.mod: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(parseMainPath, []byte("package main\nfunc main() {}\n"), 0644); parseErr2 != nil {
		parseT.Fatalf("write main.go: %v", parseErr2)
	}

	parseOriginalPolicy := launcherActiveEnterpriseConfig
	parseT.Cleanup(func() { launcherActiveEnterpriseConfig = parseOriginalPolicy })
	launcherActiveEnterpriseConfig = launcherEnterpriseConfig{
		Policy: launcherEnterprisePolicy{
			RequiredTestLanes: []string{"unit"},
		},
	}

	parseErr3 := (launcher{}).runVerify([]string{"-app", parseMainPath, "-root", parseTempApp, "-skip-tests"})
	if parseErr3 == nil || !strings.Contains(parseErr3.Error(), "verify does not satisfy enterprise lane policy") {
		parseT.Fatalf("expected verify lane policy failure, got %v", parseErr3)
	}
}

func TestRunVerifyEnforcesApprovedGoToolchainPolicy(parseT *testing.T) {
	parseTempApp := parseT.TempDir()
	parseMainPath := filepath.Join(parseTempApp, "main.go")
	if parseErr := os.WriteFile(filepath.Join(parseTempApp, "go.mod"), []byte("module example.com/gwcverifytoolchain\n\ngo 1.25.0\n"), 0644); parseErr != nil {
		parseT.Fatalf("write go.mod: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(parseMainPath, []byte("package main\nfunc main() {}\n"), 0644); parseErr2 != nil {
		parseT.Fatalf("write main.go: %v", parseErr2)
	}

	parseOriginalPolicy := launcherActiveEnterpriseConfig
	parseOriginalRunCommand := launcherRunCommand
	parseT.Cleanup(func() {
		launcherActiveEnterpriseConfig = parseOriginalPolicy
		launcherRunCommand = parseOriginalRunCommand
	})
	launcherActiveEnterpriseConfig = launcherEnterpriseConfig{
		Policy: launcherEnterprisePolicy{
			ApprovedGoToolchains: []string{"go1.26.x"},
		},
	}
	launcherRunCommand = func(parseCommand string, parseArgs []string, parseCwd string, parseEnv []string) (string, error) {
		if parseCommand == "go" && len(parseArgs) == 2 && parseArgs[0] == "env" && parseArgs[1] == "GOVERSION" {
			return "go1.25.3", nil
		}
		return "", nil
	}

	parseErr3 := (launcher{}).runVerify([]string{"-app", parseMainPath, "-root", parseTempApp, "-skip-tests"})
	if parseErr3 == nil || !strings.Contains(parseErr3.Error(), "not approved") {
		parseT.Fatalf("expected approved-toolchain policy failure, got %v", parseErr3)
	}
}

func TestRunVerifyPrintsSummaryWithoutJSON(parseT *testing.T) {
	parseTempApp := parseT.TempDir()
	parseMainPath := filepath.Join(parseTempApp, "main.go")
	if parseErr := os.WriteFile(filepath.Join(parseTempApp, "go.mod"), []byte("module example.com/gwcverifytext\n\ngo 1.25.0\n"), 0644); parseErr != nil {
		parseT.Fatalf("write go.mod: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(parseMainPath, []byte("package main\nfunc main() {}\n"), 0644); parseErr2 != nil {
		parseT.Fatalf("write main.go: %v", parseErr2)
	}

	parseStdout, parseRestoreStdout, parseErr3 := captureExamplesStdout()
	if parseErr3 != nil {
		parseT.Fatalf("capture stdout: %v", parseErr3)
	}
	defer parseRestoreStdout()

	if parseErr4 := (launcher{}).run([]string{"verify", "-app", parseMainPath, "-root", parseTempApp, "-skip-tests", "-audit", "-audit-policy", "advisory"}); parseErr4 != nil {
		parseT.Fatalf("run verify: %v", parseErr4)
	}
	parseOutput, parseErr3 := parseStdout()
	if parseErr3 != nil {
		parseT.Fatalf("read stdout: %v", parseErr3)
	}
	for _, parseExpected := range []string{"GWC verify", "tests:        skipped", "build:        ci -> ", "audit[golden-path]: PASS (policy: advisory)"} {
		if !strings.Contains(parseOutput, parseExpected) {
			parseT.Fatalf("expected verify output to contain %q, got:\n%s", parseExpected, parseOutput)
		}
	}
}
