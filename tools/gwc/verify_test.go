package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunVerifyJSONRunsTestsAndCIBuild(t *testing.T) {
	tempApp := t.TempDir()
	goModPath := filepath.Join(tempApp, "go.mod")
	mainPath := filepath.Join(tempApp, "main.go")
	testPath := filepath.Join(tempApp, "main_test.go")
	if err := os.WriteFile(goModPath, []byte("module example.com/gwcverifytest\n\ngo 1.25.0\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(mainPath, []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}
	if err := os.WriteFile(testPath, []byte("package main\nimport \"testing\"\nfunc TestSmoke(t *testing.T) {}\n"), 0644); err != nil {
		t.Fatalf("write main_test.go: %v", err)
	}

	stdout, restoreStdout, err := captureExamplesStdout()
	if err != nil {
		t.Fatalf("capture stdout: %v", err)
	}
	defer restoreStdout()

	launcher := launcher{}
	if err := launcher.run([]string{"verify", "-app", mainPath, "-root", tempApp, "-json"}); err != nil {
		t.Fatalf("run verify: %v", err)
	}

	output, err := stdout()
	if err != nil {
		t.Fatalf("read captured stdout: %v", err)
	}
	var summary verifySummary
	if err := json.Unmarshal([]byte(output), &summary); err != nil {
		t.Fatalf("unmarshal verify summary: %v\n%s", err, output)
	}
	if !summary.OK {
		t.Fatalf("expected successful verify summary, got %#v", summary)
	}
	if !summary.Tests.Ran || summary.Tests.Skipped {
		t.Fatalf("expected verify to run go tests, got %#v", summary)
	}
	if summary.Build.Profile.Name != "ci" {
		t.Fatalf("expected verify build profile ci, got %#v", summary)
	}
	if info, err := os.Stat(summary.Build.OutputPath); err != nil || info.Size() <= 0 {
		t.Fatalf("expected verify build artifact at %q, stat err=%v size=%v", summary.Build.OutputPath, err, info)
	}
}

func TestRunVerifyJSONSkipsTestsWhenProjectHasNoGoTests(t *testing.T) {
	tempApp := t.TempDir()
	goModPath := filepath.Join(tempApp, "go.mod")
	mainPath := filepath.Join(tempApp, "main.go")
	if err := os.WriteFile(goModPath, []byte("module example.com/gwcverifyskip\n\ngo 1.25.0\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(mainPath, []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}

	stdout, restoreStdout, err := captureExamplesStdout()
	if err != nil {
		t.Fatalf("capture stdout: %v", err)
	}
	defer restoreStdout()

	launcher := launcher{}
	if err := launcher.run([]string{"verify", "-app", mainPath, "-root", tempApp, "-json"}); err != nil {
		t.Fatalf("run verify: %v", err)
	}

	output, err := stdout()
	if err != nil {
		t.Fatalf("read captured stdout: %v", err)
	}
	var summary verifySummary
	if err := json.Unmarshal([]byte(output), &summary); err != nil {
		t.Fatalf("unmarshal verify summary: %v\n%s", err, output)
	}
	if !summary.OK {
		t.Fatalf("expected successful verify summary, got %#v", summary)
	}
	if summary.Tests.Ran || !summary.Tests.Skipped {
		t.Fatalf("expected verify to skip tests when no _test.go files exist, got %#v", summary)
	}
	if summary.Build.Profile.Name != "ci" {
		t.Fatalf("expected verify build profile ci, got %#v", summary)
	}
}

func TestRunVerifyPropagatesGoTestFailure(t *testing.T) {
	tempApp := t.TempDir()
	goModPath := filepath.Join(tempApp, "go.mod")
	mainPath := filepath.Join(tempApp, "main.go")
	testPath := filepath.Join(tempApp, "main_test.go")
	if err := os.WriteFile(goModPath, []byte("module example.com/gwcverifyfail\n\ngo 1.25.0\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(mainPath, []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}
	if err := os.WriteFile(testPath, []byte("package main\nimport \"testing\"\nfunc TestFail(t *testing.T) { t.Fatal(\"boom\") }\n"), 0644); err != nil {
		t.Fatalf("write main_test.go: %v", err)
	}

	launcher := launcher{}
	err := launcher.run([]string{"verify", "-app", mainPath, "-root", tempApp})
	if err == nil {
		t.Fatal("expected verify to fail when go test fails")
	}
	if !strings.Contains(err.Error(), "go test failed") || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected go test failure to be surfaced, got %v", err)
	}
}

func TestRunVerifySkipTestsSkipsEvenWhenTestsExist(t *testing.T) {
	tempApp := t.TempDir()
	goModPath := filepath.Join(tempApp, "go.mod")
	mainPath := filepath.Join(tempApp, "main.go")
	testPath := filepath.Join(tempApp, "main_test.go")
	if err := os.WriteFile(goModPath, []byte("module example.com/gwcverifyskipflag\n\ngo 1.25.0\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(mainPath, []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}
	if err := os.WriteFile(testPath, []byte("package main\nimport \"testing\"\nfunc TestSmoke(t *testing.T) {}\n"), 0644); err != nil {
		t.Fatalf("write main_test.go: %v", err)
	}

	stdout, restoreStdout, err := captureExamplesStdout()
	if err != nil {
		t.Fatalf("capture stdout: %v", err)
	}
	defer restoreStdout()

	launcher := launcher{}
	if err := launcher.run([]string{"verify", "-app", mainPath, "-root", tempApp, "-skip-tests", "-json"}); err != nil {
		t.Fatalf("run verify with skip-tests: %v", err)
	}

	output, err := stdout()
	if err != nil {
		t.Fatalf("read captured stdout: %v", err)
	}
	var summary verifySummary
	if err := json.Unmarshal([]byte(output), &summary); err != nil {
		t.Fatalf("unmarshal verify summary: %v\n%s", err, output)
	}
	if summary.Tests.Ran || !summary.Tests.Skipped {
		t.Fatalf("expected skip-tests to skip go tests even when tests exist, got %#v", summary)
	}
}

func TestRunVerifyAuditJSONIncludesAuditReport(t *testing.T) {
	tempApp := t.TempDir()
	goModPath := filepath.Join(tempApp, "go.mod")
	mainPath := filepath.Join(tempApp, "main.go")
	htmlPath := filepath.Join(tempApp, "index.html")
	if err := os.WriteFile(goModPath, []byte("module example.com/gwcverifyaudit\n\ngo 1.25.0\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(mainPath, []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}
	if err := os.WriteFile(htmlPath, []byte("<!doctype html><html><body><div id=\"app\"></div></body></html>\n"), 0644); err != nil {
		t.Fatalf("write index.html: %v", err)
	}

	stdout, restoreStdout, err := captureExamplesStdout()
	if err != nil {
		t.Fatalf("capture stdout: %v", err)
	}
	defer restoreStdout()

	launcher := launcher{}
	if err := launcher.run([]string{"verify", "-app", mainPath, "-root", tempApp, "-skip-tests", "-audit", "-audit-policy", "advisory", "-json"}); err != nil {
		t.Fatalf("run verify with audit: %v", err)
	}

	output, err := stdout()
	if err != nil {
		t.Fatalf("read captured stdout: %v", err)
	}
	var summary verifySummary
	if err := json.Unmarshal([]byte(output), &summary); err != nil {
		t.Fatalf("unmarshal verify summary: %v\n%s", err, output)
	}
	if !summary.OK {
		t.Fatalf("expected successful verify summary, got %#v", summary)
	}
	if summary.Audit == nil {
		t.Fatalf("expected audit report in verify summary, got %#v", summary)
	}
	if summary.Audit.Mode != "golden-path" || summary.Audit.Policy != "advisory" || !summary.Audit.OK {
		t.Fatalf("expected advisory golden-path audit report, got %#v", summary.Audit)
	}
}

func TestRunVerifyAuditStrictFailureReturnsError(t *testing.T) {
	tempApp := t.TempDir()
	goModPath := filepath.Join(tempApp, "go.mod")
	mainPath := filepath.Join(tempApp, "main.go")
	badClientPath := filepath.Join(tempApp, "client", "boundary.go")
	if err := os.WriteFile(goModPath, []byte("module example.com/gwcverifyauditfail\n\ngo 1.25.0\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(mainPath, []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(badClientPath), 0755); err != nil {
		t.Fatalf("mkdir client dir: %v", err)
	}
	if err := os.WriteFile(badClientPath, []byte("//go:build js && wasm\n// +build js,wasm\n\npackage client\n\nimport _ \"database/sql\"\n"), 0644); err != nil {
		t.Fatalf("write failing audit fixture: %v", err)
	}

	stdout, restoreStdout, err := captureExamplesStdout()
	if err != nil {
		t.Fatalf("capture stdout: %v", err)
	}
	defer restoreStdout()

	err = (launcher{}).run([]string{"verify", "-app", mainPath, "-root", tempApp, "-skip-tests", "-audit", "-json"})
	if err == nil || !strings.Contains(err.Error(), "verify audit found error-severity findings") {
		t.Fatalf("expected strict audit verify failure, got %v", err)
	}

	output, readErr := stdout()
	if readErr != nil {
		t.Fatalf("read captured stdout: %v", readErr)
	}
	var summary verifySummary
	if err := json.Unmarshal([]byte(output), &summary); err != nil {
		t.Fatalf("unmarshal verify summary: %v\n%s", err, output)
	}
	if summary.OK {
		t.Fatalf("expected failing verify summary, got %#v", summary)
	}
	if summary.Audit == nil || summary.Audit.Policy != "strict" || summary.Audit.OK {
		t.Fatalf("expected failing strict audit report, got %#v", summary.Audit)
	}
}

func TestRunVerifyPropagatesProjectScanFailure(t *testing.T) {
	tempApp := t.TempDir()
	mainPath := filepath.Join(tempApp, "main.go")
	if err := os.WriteFile(filepath.Join(tempApp, "go.mod"), []byte("module example.com/gwcverifyscanfail\n\ngo 1.25.0\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(mainPath, []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}
	missingRoot := filepath.Join(t.TempDir(), "missing-root")

	err := (launcher{}).run([]string{"verify", "-app", mainPath, "-root", missingRoot})
	if err == nil || !strings.Contains(err.Error(), "scan project tests") {
		t.Fatalf("expected project scan failure, got %v", err)
	}
}

func TestRunVerifyHelpReturnsNil(t *testing.T) {
	launcher := launcher{}
	if err := launcher.run([]string{"verify", "-help"}); err != nil {
		t.Fatalf("expected verify help to succeed, got %v", err)
	}
}

func TestRunVerifyHandlesInvalidFlags(t *testing.T) {
	if err := (launcher{}).runVerify([]string{"-definitely-invalid"}); err == nil || !strings.Contains(err.Error(), "flag provided but not defined") {
		t.Fatalf("expected invalid verify flag error, got %v", err)
	}
}

func TestRunVerifyEnforcesEnterpriseRequiredLanes(t *testing.T) {
	tempApp := t.TempDir()
	mainPath := filepath.Join(tempApp, "main.go")
	if err := os.WriteFile(filepath.Join(tempApp, "go.mod"), []byte("module example.com/gwcverifyrequiredlane\n\ngo 1.25.0\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(mainPath, []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}

	originalPolicy := launcherActiveEnterpriseConfig
	t.Cleanup(func() { launcherActiveEnterpriseConfig = originalPolicy })
	launcherActiveEnterpriseConfig = launcherEnterpriseConfig{
		Policy: launcherEnterprisePolicy{
			RequiredTestLanes: []string{"unit"},
		},
	}

	err := (launcher{}).runVerify([]string{"-app", mainPath, "-root", tempApp, "-skip-tests"})
	if err == nil || !strings.Contains(err.Error(), "verify does not satisfy enterprise lane policy") {
		t.Fatalf("expected verify lane policy failure, got %v", err)
	}
}

func TestRunVerifyEnforcesApprovedGoToolchainPolicy(t *testing.T) {
	tempApp := t.TempDir()
	mainPath := filepath.Join(tempApp, "main.go")
	if err := os.WriteFile(filepath.Join(tempApp, "go.mod"), []byte("module example.com/gwcverifytoolchain\n\ngo 1.25.0\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(mainPath, []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}

	originalPolicy := launcherActiveEnterpriseConfig
	originalRunCommand := launcherRunCommand
	t.Cleanup(func() {
		launcherActiveEnterpriseConfig = originalPolicy
		launcherRunCommand = originalRunCommand
	})
	launcherActiveEnterpriseConfig = launcherEnterpriseConfig{
		Policy: launcherEnterprisePolicy{
			ApprovedGoToolchains: []string{"go1.26.x"},
		},
	}
	launcherRunCommand = func(command string, args []string, cwd string, env []string) (string, error) {
		if command == "go" && len(args) == 2 && args[0] == "env" && args[1] == "GOVERSION" {
			return "go1.25.3", nil
		}
		return "", nil
	}

	err := (launcher{}).runVerify([]string{"-app", mainPath, "-root", tempApp, "-skip-tests"})
	if err == nil || !strings.Contains(err.Error(), "not approved") {
		t.Fatalf("expected approved-toolchain policy failure, got %v", err)
	}
}

func TestRunVerifyPrintsSummaryWithoutJSON(t *testing.T) {
	tempApp := t.TempDir()
	mainPath := filepath.Join(tempApp, "main.go")
	if err := os.WriteFile(filepath.Join(tempApp, "go.mod"), []byte("module example.com/gwcverifytext\n\ngo 1.25.0\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(mainPath, []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}

	stdout, restoreStdout, err := captureExamplesStdout()
	if err != nil {
		t.Fatalf("capture stdout: %v", err)
	}
	defer restoreStdout()

	if err := (launcher{}).run([]string{"verify", "-app", mainPath, "-root", tempApp, "-skip-tests", "-audit", "-audit-policy", "advisory"}); err != nil {
		t.Fatalf("run verify: %v", err)
	}
	output, err := stdout()
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	for _, expected := range []string{"GWC verify", "tests:        skipped", "build:        ci -> ", "audit[golden-path]: PASS (policy: advisory)"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("expected verify output to contain %q, got:\n%s", expected, output)
		}
	}
}
