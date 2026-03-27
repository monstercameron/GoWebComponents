package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLauncherDiagnosticHelpersCoverFailureCategories verifies launcher command parsing and diagnostic classification branches.
func TestLauncherDiagnosticHelpersCoverFailureCategories(parseT *testing.T) {
	parseCommand, parseArgs := launcherCommandAndArgs([]string{"-policy-pack", "policy.json", "test", "-json"})
	if parseCommand != "test" || len(parseArgs) != 1 || parseArgs[0] != "-json" {
		parseT.Fatalf("launcherCommandAndArgs(valid flags) = %q %#v", parseCommand, parseArgs)
	}
	parseCommand, parseArgs = launcherCommandAndArgs([]string{"-unknown", "test"})
	if parseCommand != "-unknown" || len(parseArgs) != 1 || parseArgs[0] != "test" {
		parseT.Fatalf("launcherCommandAndArgs(fallback) = %q %#v", parseCommand, parseArgs)
	}

	parseCases := []struct {
		name         string
		args         []string
		message      string
		wantPhase    string
		wantCategory string
		wantCode     string
		wantOverride string
		wantCommand  string
	}{
		{
			name:         "override",
			args:         []string{"build", "-json"},
			message:      "resolve browserworkspace override failed",
			wantPhase:    "configuration",
			wantCategory: "configuration",
			wantCode:     "invalid_runner_override",
			wantOverride: "browserWorkspace",
			wantCommand:  "build",
		},
		{
			name:         "policy",
			args:         []string{"verify"},
			message:      "blocked by security boundary",
			wantPhase:    "policy",
			wantCategory: "policy",
			wantCode:     "policy_violation",
			wantCommand:  "verify",
		},
		{
			name:         "configuration",
			args:         []string{"release"},
			message:      "configured output path does not exist",
			wantPhase:    "configuration",
			wantCategory: "configuration",
			wantCode:     "invalid_configuration",
			wantCommand:  "release",
		},
		{
			name:         "code",
			args:         []string{"build"},
			message:      "go build failed: compile error",
			wantPhase:    "execution",
			wantCategory: "code",
			wantCode:     "code_failure",
			wantCommand:  "build",
		},
		{
			name:         "audit",
			args:         []string{"verify"},
			message:      "verify audit found high findings",
			wantPhase:    "validation",
			wantCategory: "validation",
			wantCode:     "audit_failed",
			wantCommand:  "verify",
		},
		{
			name:         "checks",
			args:         []string{"doctor"},
			message:      "doctor found required checks",
			wantPhase:    "configuration",
			wantCategory: "configuration",
			wantCode:     "invalid_configuration",
			wantCommand:  "doctor",
		},
		{
			name:         "smoke",
			args:         []string{"release"},
			message:      "smoke validation failed",
			wantPhase:    "validation",
			wantCategory: "validation",
			wantCode:     "smoke_failed",
			wantCommand:  "release",
		},
		{
			name:         "runtime",
			args:         []string{"dev"},
			message:      "listen tcp: bind: address already in use",
			wantPhase:    "runtime",
			wantCategory: "runtime",
			wantCode:     "startup_failed",
			wantCommand:  "dev",
		},
	}

	for _, parseCase := range parseCases {
		parseDiagnostic := buildLauncherFailureDiagnostic(parseCase.args, errors.New(parseCase.message))
		if parseDiagnostic.Command != parseCase.wantCommand ||
			parseDiagnostic.Phase != parseCase.wantPhase ||
			parseDiagnostic.Category != parseCase.wantCategory ||
			parseDiagnostic.Code != parseCase.wantCode ||
			parseDiagnostic.Override != parseCase.wantOverride ||
			parseDiagnostic.Message != parseCase.message ||
			parseDiagnostic.OK {
			parseT.Fatalf("%s: unexpected diagnostic: %#v", parseCase.name, parseDiagnostic)
		}
	}
}

// TestEnterprisePolicyHelpersCoverToolchainBranches verifies enterprise lane and Go toolchain policy helpers.
func TestEnterprisePolicyHelpersCoverToolchainBranches(parseT *testing.T) {
	if parseErr := enforceEnterpriseRequiredTestLanes([]string{"unit"}, launcherEnterprisePolicy{}); parseErr != nil {
		parseT.Fatalf("enforceEnterpriseRequiredTestLanes(empty): %v", parseErr)
	}
	if parseErr := enforceEnterpriseRequiredTestLanes([]string{"unit", "wasm"}, launcherEnterprisePolicy{RequiredTestLanes: []string{"unit", "browser"}}); parseErr == nil || !strings.Contains(parseErr.Error(), "missing required test lanes: browser") {
		parseT.Fatalf("expected missing test lane error, got %v", parseErr)
	}

	if parseGot := normalizeGoToolchainPolicyValue(""); parseGot != "" {
		parseT.Fatalf("normalizeGoToolchainPolicyValue(blank) = %q", parseGot)
	}
	if parseGot := normalizeGoToolchainPolicyValue("1.25.3"); parseGot != "go1.25.3" {
		parseT.Fatalf("normalizeGoToolchainPolicyValue(version) = %q", parseGot)
	}
	if !goToolchainApprovedByPolicy("go1.25.3", "1.25.x") {
		parseT.Fatal("expected wildcard toolchain approval")
	}
	if !goToolchainApprovedByPolicy("go1.25.3", "go1.25") {
		parseT.Fatal("expected prefix toolchain approval")
	}
	if goToolchainApprovedByPolicy("go1.25.3", "") {
		parseT.Fatal("expected blank policy approval to fail")
	}
	if goToolchainApprovedByPolicy("go1.25.3", "1.24.x") {
		parseT.Fatal("expected mismatched wildcard approval to fail")
	}

	parseOriginalRunCommand := launcherRunCommand
	parseT.Cleanup(func() {
		launcherRunCommand = parseOriginalRunCommand
	})

	launcherRunCommand = func(string, []string, string, []string) (string, error) {
		return "", errors.New("go env failed")
	}
	if _, parseErr := resolveActiveGoToolchain(parseT.TempDir()); parseErr == nil || !strings.Contains(parseErr.Error(), "resolve active Go toolchain") {
		parseT.Fatalf("expected go env failure, got %v", parseErr)
	}

	launcherRunCommand = func(string, []string, string, []string) (string, error) {
		return "\n", nil
	}
	if _, parseErr := resolveActiveGoToolchain(parseT.TempDir()); parseErr == nil || !strings.Contains(parseErr.Error(), "returned empty output") {
		parseT.Fatalf("expected empty output error, got %v", parseErr)
	}

	launcherRunCommand = func(string, []string, string, []string) (string, error) {
		return "go1.25.3\nignored", nil
	}
	parseToolchain, parseErr := resolveActiveGoToolchain(parseT.TempDir())
	if parseErr != nil || parseToolchain != "go1.25.3" {
		parseT.Fatalf("resolveActiveGoToolchain = %q err=%v", parseToolchain, parseErr)
	}
	if parseErr2 := enforceEnterpriseGoToolchainPolicy(parseT.TempDir(), launcherEnterprisePolicy{ApprovedGoToolchains: []string{"1.25.x"}}); parseErr2 != nil {
		parseT.Fatalf("enforceEnterpriseGoToolchainPolicy(approved): %v", parseErr2)
	}
	if parseErr3 := enforceEnterpriseGoToolchainPolicy(parseT.TempDir(), launcherEnterprisePolicy{ApprovedGoToolchains: []string{"1.24.x"}}); parseErr3 == nil || !strings.Contains(parseErr3.Error(), `active Go toolchain "go1.25.3" is not approved`) {
		parseT.Fatalf("expected unapproved toolchain error, got %v", parseErr3)
	}
}

// TestEnterpriseReleasePolicyHelpersCoverValidationBranches verifies enterprise release policy checks.
func TestEnterpriseReleasePolicyHelpersCoverValidationBranches(parseT *testing.T) {
	parseTrue := true
	if parseErr := enforceEnterpriseReleasePolicy(releaseConfig{}, launcherEnterprisePolicy{RequireReleaseBudgets: &parseTrue}); parseErr == nil || !strings.Contains(parseErr.Error(), "requires release budgets") {
		parseT.Fatalf("expected required budgets error, got %v", parseErr)
	}
	if parseErr := enforceEnterpriseReleasePolicy(releaseConfig{compression: "gzip"}, launcherEnterprisePolicy{RequiredReleaseCompression: "zip"}); parseErr == nil || !strings.Contains(parseErr.Error(), "normalize enterprise required release compression") {
		parseT.Fatalf("expected compression normalization error, got %v", parseErr)
	}
	if parseErr := enforceEnterpriseReleasePolicy(releaseConfig{compression: "gzip"}, launcherEnterprisePolicy{RequiredReleaseCompression: "brotli"}); parseErr == nil || !strings.Contains(parseErr.Error(), `release compression policy "gzip" does not satisfy enterprise requirement "brotli"`) {
		parseT.Fatalf("expected compression mismatch error, got %v", parseErr)
	}
	if parseErr := enforceEnterpriseReleasePolicy(releaseConfig{compression: "gzip", binaryName: "app"}, launcherEnterprisePolicy{ReleaseBinaryPattern: "("}); parseErr == nil || !strings.Contains(parseErr.Error(), "compile enterprise release binary pattern") {
		parseT.Fatalf("expected invalid binary regex error, got %v", parseErr)
	}
	if parseErr := enforceEnterpriseReleasePolicy(releaseConfig{compression: "gzip", binaryName: "site"}, launcherEnterprisePolicy{ReleaseBinaryPattern: "^app$"}); parseErr == nil || !strings.Contains(parseErr.Error(), `release binary name "site" does not satisfy enterprise pattern "^app$"`) {
		parseT.Fatalf("expected binary mismatch error, got %v", parseErr)
	}
	if parseErr := enforceEnterpriseReleasePolicy(releaseConfig{compression: "gzip", binaryName: "app", manifestName: "manifest.json"}, launcherEnterprisePolicy{ReleaseManifestPattern: "("}); parseErr == nil || !strings.Contains(parseErr.Error(), "compile enterprise release manifest pattern") {
		parseT.Fatalf("expected invalid manifest regex error, got %v", parseErr)
	}
	if parseErr := enforceEnterpriseReleasePolicy(releaseConfig{compression: "gzip", binaryName: "app", manifestName: "report.json"}, launcherEnterprisePolicy{ReleaseManifestPattern: "^manifest\\.json$"}); parseErr == nil || !strings.Contains(parseErr.Error(), `release manifest name "report.json" does not satisfy enterprise pattern`) {
		parseT.Fatalf("expected manifest mismatch error, got %v", parseErr)
	}
	if parseErr := enforceEnterpriseReleasePolicy(
		releaseConfig{budgetsPath: "budgets.json", compression: "gzip+brotli", binaryName: "app", manifestName: "manifest.json"},
		launcherEnterprisePolicy{
			RequireReleaseBudgets:      &parseTrue,
			RequiredReleaseCompression: "both",
			ReleaseBinaryPattern:       "^app$",
			ReleaseManifestPattern:     "^manifest\\.json$",
		},
	); parseErr != nil {
		parseT.Fatalf("expected enterprise release policy success, got %v", parseErr)
	}
}

// TestReleaseSnapshotHelpersCoverReadAndDiffBranches verifies release snapshot readers and package attribution diffs.
func TestReleaseSnapshotHelpersCoverReadAndDiffBranches(parseT *testing.T) {
	parseRootPath := parseT.TempDir()

	if _, parseErr := releaseReadManifestSnapshot(filepath.Join(parseRootPath, "missing.json")); parseErr == nil || !strings.Contains(parseErr.Error(), "read compare manifest") {
		parseT.Fatalf("expected manifest read error, got %v", parseErr)
	}

	parseWriteManifest := func(parseName string, parsePayload any) string {
		parseT.Helper()
		parsePath := filepath.Join(parseRootPath, parseName)
		parseBytes, parseErr := json.Marshal(parsePayload)
		if parseErr != nil {
			parseT.Fatalf("json.Marshal(%s): %v", parseName, parseErr)
		}
		if parseErr2 := os.WriteFile(parsePath, parseBytes, 0o644); parseErr2 != nil {
			parseT.Fatalf("WriteFile(%s): %v", parseName, parseErr2)
		}
		return parsePath
	}

	parseInvalidJSONPath := filepath.Join(parseRootPath, "invalid.json")
	if parseErr := os.WriteFile(parseInvalidJSONPath, []byte("{"), 0o644); parseErr != nil {
		parseT.Fatalf("WriteFile invalid manifest: %v", parseErr)
	}
	if _, parseErr := releaseReadManifestSnapshot(parseInvalidJSONPath); parseErr == nil || !strings.Contains(parseErr.Error(), "parse compare manifest") {
		parseT.Fatalf("expected invalid manifest parse error, got %v", parseErr)
	}
	if _, parseErr := releaseReadManifestSnapshot(parseWriteManifest("missing-package.json", releaseManifestSnapshot{GOOS: "js", GOARCH: "wasm", Artifacts: map[string]releaseArtifactRecord{"wasm": {Path: "app.wasm", Bytes: 1}}})); parseErr == nil || !strings.Contains(parseErr.Error(), "missing package") {
		parseT.Fatalf("expected missing package error, got %v", parseErr)
	}
	if _, parseErr := releaseReadManifestSnapshot(parseWriteManifest("wrong-platform.json", releaseManifestSnapshot{Package: "example.com/app", GOOS: "linux", GOARCH: "amd64", Artifacts: map[string]releaseArtifactRecord{"wasm": {Path: "app.wasm", Bytes: 1}}})); parseErr == nil || !strings.Contains(parseErr.Error(), "must describe a js/wasm release") {
		parseT.Fatalf("expected platform mismatch error, got %v", parseErr)
	}
	if _, parseErr := releaseReadManifestSnapshot(parseWriteManifest("missing-artifacts.json", releaseManifestSnapshot{Package: "example.com/app", GOOS: "js", GOARCH: "wasm"})); parseErr == nil || !strings.Contains(parseErr.Error(), "missing artifacts") {
		parseT.Fatalf("expected missing artifacts error, got %v", parseErr)
	}

	parseManifestPath := parseWriteManifest("manifest.json", releaseManifestSnapshot{
		Package:   "example.com/app",
		GOOS:      "js",
		GOARCH:    "wasm",
		Artifacts: map[string]releaseArtifactRecord{"wasm": {Path: "app.wasm", Bytes: 42}},
	})
	parseManifest, parseErr := releaseReadManifestSnapshot(parseManifestPath)
	if parseErr != nil || parseManifest.Package != "example.com/app" || len(parseManifest.Artifacts) != 1 {
		parseT.Fatalf("releaseReadManifestSnapshot success = %#v err=%v", parseManifest, parseErr)
	}

	parseInvalidAttributionPath := filepath.Join(parseRootPath, "packages-invalid.json")
	if parseErr := os.WriteFile(parseInvalidAttributionPath, []byte("{"), 0o644); parseErr != nil {
		parseT.Fatalf("WriteFile invalid attribution: %v", parseErr)
	}
	if _, parseErr := releaseReadPackageAttributionSnapshot(parseInvalidAttributionPath); parseErr == nil || !strings.Contains(parseErr.Error(), "parse package attribution artifact") {
		parseT.Fatalf("expected invalid attribution parse error, got %v", parseErr)
	}

	if parsePercent := releasePercentDeltaPointer(0, 10); parsePercent != nil {
		parseT.Fatalf("expected nil percent delta for zero baseline, got %v", *parsePercent)
	}
	parsePercent := releasePercentDeltaPointer(100, 125)
	if parsePercent == nil || *parsePercent != 25 {
		parseT.Fatalf("releasePercentDeltaPointer = %v", parsePercent)
	}

	parseBaselineDir := filepath.Join(parseRootPath, "baseline")
	parseCurrentDir := filepath.Join(parseRootPath, "current")
	if parseErr := os.MkdirAll(parseBaselineDir, 0o755); parseErr != nil {
		parseT.Fatalf("MkdirAll baseline: %v", parseErr)
	}
	if parseErr := os.MkdirAll(parseCurrentDir, 0o755); parseErr != nil {
		parseT.Fatalf("MkdirAll current: %v", parseErr)
	}

	parseWritePackages := func(parseDirPath string, parseName string, parseRecords []releasePackageSizeRecord) string {
		parseT.Helper()
		parsePath := filepath.Join(parseDirPath, parseName)
		parseBytes, parseErr := json.Marshal(releasePackageAttributionSnapshot{Packages: parseRecords})
		if parseErr != nil {
			parseT.Fatalf("json.Marshal package snapshot: %v", parseErr)
		}
		if parseErr2 := os.WriteFile(parsePath, parseBytes, 0o644); parseErr2 != nil {
			parseT.Fatalf("WriteFile(%s): %v", parsePath, parseErr2)
		}
		return parsePath
	}

	if parseDiffs, parseErr2 := releaseComparePackageAttributionRecords(parseManifestPath, nil, parseCurrentDir, &releaseAttributionRecord{Path: "packages.json"}); parseErr2 != nil || parseDiffs != nil {
		parseT.Fatalf("expected nil diffs when baseline attribution is missing, got diffs=%#v err=%v", parseDiffs, parseErr2)
	}

	parseWritePackages(parseBaselineDir, "packages.json", []releasePackageSizeRecord{
		{ImportPath: "example.com/a", ArchiveBytes: 100, SourceBytes: 10},
		{ImportPath: "example.com/b", ArchiveBytes: 90, SourceBytes: 20},
		{ImportPath: "example.com/c", ArchiveBytes: 50, SourceBytes: 30},
	})
	parseWritePackages(parseCurrentDir, "packages.json", []releasePackageSizeRecord{
		{ImportPath: "example.com/a", ArchiveBytes: 180, SourceBytes: 15},
		{ImportPath: "example.com/b", ArchiveBytes: 80, SourceBytes: 25},
		{ImportPath: "example.com/c", ArchiveBytes: 50, SourceBytes: 35},
		{ImportPath: "example.com/d", ArchiveBytes: 12, SourceBytes: 3},
	})

	parseDiffs, parseErr2 := releaseComparePackageAttributionRecords(
		filepath.Join(parseBaselineDir, "manifest.json"),
		&releaseAttributionRecord{Path: "packages.json"},
		parseCurrentDir,
		&releaseAttributionRecord{Path: "packages.json"},
	)
	if parseErr2 != nil {
		parseT.Fatalf("releaseComparePackageAttributionRecords: %v", parseErr2)
	}
	if len(parseDiffs) != 2 {
		parseT.Fatalf("expected only positive archive deltas, got %#v", parseDiffs)
	}
	if parseDiffs[0].ImportPath != "example.com/a" || parseDiffs[0].Status != "grew" || derefInt64(parseDiffs[0].ArchiveDeltaBytes) != 80 {
		parseT.Fatalf("unexpected first diff: %#v", parseDiffs[0])
	}
	if parseDiffs[1].ImportPath != "example.com/d" || parseDiffs[1].Status != "added" || derefInt64(parseDiffs[1].ArchiveDeltaBytes) != 12 {
		parseT.Fatalf("unexpected second diff: %#v", parseDiffs[1])
	}

	parseLargeBaseline := make([]releasePackageSizeRecord, 0, 12)
	parseLargeCurrent := make([]releasePackageSizeRecord, 0, 12)
	for parseIndex := 0; parseIndex < 12; parseIndex++ {
		parseImportPath := "example.com/pkg" + string(rune('a'+parseIndex))
		parseLargeBaseline = append(parseLargeBaseline, releasePackageSizeRecord{ImportPath: parseImportPath, ArchiveBytes: 10, SourceBytes: 1})
		parseLargeCurrent = append(parseLargeCurrent, releasePackageSizeRecord{ImportPath: parseImportPath, ArchiveBytes: int64(20 + parseIndex), SourceBytes: 2})
	}
	parseWritePackages(parseBaselineDir, "packages-large.json", parseLargeBaseline)
	parseWritePackages(parseCurrentDir, "packages-large.json", parseLargeCurrent)

	parseLargeDiffs, parseErr3 := releaseComparePackageAttributionRecords(
		filepath.Join(parseBaselineDir, "manifest.json"),
		&releaseAttributionRecord{Path: "packages-large.json"},
		parseCurrentDir,
		&releaseAttributionRecord{Path: "packages-large.json"},
	)
	if parseErr3 != nil {
		parseT.Fatalf("releaseComparePackageAttributionRecords large set: %v", parseErr3)
	}
	if len(parseLargeDiffs) != 10 {
		parseT.Fatalf("expected diff truncation to top 10 entries, got %d", len(parseLargeDiffs))
	}
	if derefInt64(parseLargeDiffs[0].ArchiveDeltaBytes) < derefInt64(parseLargeDiffs[len(parseLargeDiffs)-1].ArchiveDeltaBytes) {
		parseT.Fatalf("expected diffs sorted by descending archive delta: %#v", parseLargeDiffs)
	}
}
