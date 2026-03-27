package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestReleaseWriteDiffReportCoversSuccessAndFailures verifies diff report generation plus marshal and write failures.
func TestReleaseWriteDiffReportCoversSuccessAndFailures(parseT *testing.T) {
	parseOriginalMarshalIndent := releaseMarshalIndent
	parseT.Cleanup(func() {
		releaseMarshalIndent = parseOriginalMarshalIndent
	})

	parseT.Run("success", func(parseT2 *testing.T) {
		parseBaselineDir := parseT2.TempDir()
		parseCurrentDir := parseT2.TempDir()

		buildReleasePackageAttributionSnapshot(parseT2, filepath.Join(parseBaselineDir, "packages.json"), []releasePackageSizeRecord{
			{ImportPath: "example.com/core", ArchiveBytes: 100, SourceBytes: 40},
		})
		buildReleasePackageAttributionSnapshot(parseT2, filepath.Join(parseCurrentDir, "packages.json"), []releasePackageSizeRecord{
			{ImportPath: "example.com/core", ArchiveBytes: 180, SourceBytes: 45},
			{ImportPath: "example.com/new", ArchiveBytes: 12, SourceBytes: 3},
		})
		parseBaselineManifestPath := buildReleaseManifestSnapshotForDiff(parseT2, parseBaselineDir, map[string]releaseArtifactRecord{
			"wasm": {Path: "app.wasm", Bytes: 100},
		}, &releaseAttributionRecord{Path: "packages.json", Mode: "all"})

		parseDiff, parseErr := releaseWriteDiffReport(
			parseBaselineManifestPath,
			filepath.Join(parseCurrentDir, "manifest.json"),
			&releaseAttributionRecord{Path: "packages.json", Mode: "all"},
			map[string]releaseArtifactRecord{
				"wasm": {Path: "app.wasm", Bytes: 140},
			},
			parseCurrentDir,
		)
		if parseErr != nil {
			parseT2.Fatalf("releaseWriteDiffReport(success): %v", parseErr)
		}
		if parseDiff == nil || parseDiff.Path != "wasm-release-size-diff.json" || len(parseDiff.ArtifactChanges) == 0 || len(parseDiff.LikelyCulprits) == 0 {
			parseT2.Fatalf("unexpected diff report %#v", parseDiff)
		}
		if _, parseErr2 := os.Stat(filepath.Join(parseCurrentDir, parseDiff.Path)); parseErr2 != nil {
			parseT2.Fatalf("expected diff report file to exist: %v", parseErr2)
		}
	})

	parseT.Run("baseline read failure", func(parseT2 *testing.T) {
		parseDiff, parseErr := releaseWriteDiffReport(
			filepath.Join(parseT2.TempDir(), "missing-manifest.json"),
			"manifest.json",
			nil,
			nil,
			parseT2.TempDir(),
		)
		if parseErr == nil || !strings.Contains(parseErr.Error(), "read compare manifest") {
			parseT2.Fatalf("expected baseline read failure, got diff=%#v err=%v", parseDiff, parseErr)
		}
	})

	parseT.Run("marshal failure", func(parseT2 *testing.T) {
		parseBaselineDir := parseT2.TempDir()
		parseCurrentDir := parseT2.TempDir()
		buildReleasePackageAttributionSnapshot(parseT2, filepath.Join(parseBaselineDir, "packages.json"), []releasePackageSizeRecord{{ImportPath: "example.com/core", ArchiveBytes: 100}})
		buildReleasePackageAttributionSnapshot(parseT2, filepath.Join(parseCurrentDir, "packages.json"), []releasePackageSizeRecord{{ImportPath: "example.com/core", ArchiveBytes: 120}})
		parseBaselineManifestPath := buildReleaseManifestSnapshotForDiff(parseT2, parseBaselineDir, map[string]releaseArtifactRecord{
			"wasm": {Path: "app.wasm", Bytes: 100},
		}, &releaseAttributionRecord{Path: "packages.json", Mode: "all"})

		releaseMarshalIndent = func(parseValue interface{}, parsePrefix string, parseIndent string) ([]byte, error) {
			return nil, errors.New("marshal failed")
		}
		parseDiff, parseErr := releaseWriteDiffReport(
			parseBaselineManifestPath,
			filepath.Join(parseCurrentDir, "manifest.json"),
			&releaseAttributionRecord{Path: "packages.json", Mode: "all"},
			map[string]releaseArtifactRecord{
				"wasm": {Path: "app.wasm", Bytes: 120},
			},
			parseCurrentDir,
		)
		if parseErr == nil || !strings.Contains(parseErr.Error(), "encode release diff report") {
			parseT2.Fatalf("expected marshal failure, got diff=%#v err=%v", parseDiff, parseErr)
		}
		releaseMarshalIndent = parseOriginalMarshalIndent
	})

	parseT.Run("write failure", func(parseT2 *testing.T) {
		parseBaselineDir := parseT2.TempDir()
		parseCurrentDir := parseT2.TempDir()
		buildReleasePackageAttributionSnapshot(parseT2, filepath.Join(parseBaselineDir, "packages.json"), []releasePackageSizeRecord{{ImportPath: "example.com/core", ArchiveBytes: 100}})
		buildReleasePackageAttributionSnapshot(parseT2, filepath.Join(parseCurrentDir, "packages.json"), []releasePackageSizeRecord{{ImportPath: "example.com/core", ArchiveBytes: 120}})
		parseBaselineManifestPath := buildReleaseManifestSnapshotForDiff(parseT2, parseBaselineDir, map[string]releaseArtifactRecord{
			"wasm": {Path: "app.wasm", Bytes: 100},
		}, &releaseAttributionRecord{Path: "packages.json", Mode: "all"})
		if parseErr := os.MkdirAll(filepath.Join(parseCurrentDir, "wasm-release-size-diff.json"), 0o755); parseErr != nil {
			parseT2.Fatalf("mkdir diff collision path: %v", parseErr)
		}

		parseDiff, parseErr := releaseWriteDiffReport(
			parseBaselineManifestPath,
			filepath.Join(parseCurrentDir, "manifest.json"),
			&releaseAttributionRecord{Path: "packages.json", Mode: "all"},
			map[string]releaseArtifactRecord{
				"wasm": {Path: "app.wasm", Bytes: 120},
			},
			parseCurrentDir,
		)
		if parseErr == nil || !strings.Contains(parseErr.Error(), "write release diff report") {
			parseT2.Fatalf("expected write failure, got diff=%#v err=%v", parseDiff, parseErr)
		}
	})
}

// TestReleaseSnapshotReadAndSummaryPrinterCoversBranches verifies attribution read failures and optional release summary output.
func TestReleaseSnapshotReadAndSummaryPrinterCoversBranches(parseT *testing.T) {
	if _, parseErr := releaseReadPackageAttributionSnapshot(filepath.Join(parseT.TempDir(), "missing-packages.json")); parseErr == nil || !strings.Contains(parseErr.Error(), "read package attribution artifact") {
		parseT.Fatalf("expected package attribution read failure, got %v", parseErr)
	}

	parseStdout, parseRestoreStdout, parseErr := captureExamplesStdout()
	if parseErr != nil {
		parseT.Fatalf("capture stdout: %v", parseErr)
	}
	defer parseRestoreStdout()

	printReleaseSummary(releaseSummary{
		Profile:      buildProfile{Name: "release"},
		AppPath:      `C:\repo\app\main.go`,
		ProjectRoot:  `C:\repo\app`,
		PackageDir:   `C:\repo\app`,
		OutDir:       `C:\repo\bin`,
		ManifestPath: `C:\repo\bin\manifest.json`,
		Optimizer:    &releaseOptimizerRecord{Mode: "wasm-opt", Tool: "wasm-opt"},
		Attribution:  &releaseAttributionRecord{Path: "packages.json", Mode: "all", PackageCount: 2},
		Startup:      &releaseStartupRecord{Path: "wasm-startup-report.json", Mode: "browser", TransportEncoding: "gzip"},
		Validation:   &releaseValidationRecord{Path: "wasm-release-validation.json", WasmContentType: "application/wasm"},
		Diff:         &releaseDiffArtifactRecord{Path: "wasm-release-size-diff.json", BaselineManifestPath: `C:\repo\baseline\manifest.json`},
		Artifacts: map[string]releaseArtifactRecord{
			"gzip": {Path: "app.wasm.gz", Bytes: 10},
			"wasm": {Path: "app.wasm", Bytes: 20},
		},
	})

	parseOutput, parseErr := parseStdout()
	if parseErr != nil {
		parseT.Fatalf("read stdout: %v", parseErr)
	}
	for _, parseSnippet := range []string{
		"optimizer:    wasm-opt (wasm-opt)",
		"attribution:  packages.json (all, 2 packages)",
		"startup:      wasm-startup-report.json (browser via gzip)",
		"validation:   wasm-release-validation.json (application/wasm)",
		"diff:         wasm-release-size-diff.json (baseline C:\\repo\\baseline\\manifest.json)",
	} {
		if !strings.Contains(parseOutput, parseSnippet) {
			parseT.Fatalf("expected release summary output to contain %q\n%s", parseSnippet, parseOutput)
		}
	}
}

// buildReleaseManifestSnapshotForDiff writes one release manifest snapshot for diff-report tests.
func buildReleaseManifestSnapshotForDiff(parseT *testing.T, parseDir string, parseArtifacts map[string]releaseArtifactRecord, parseAttribution *releaseAttributionRecord) string {
	parseT.Helper()

	parseManifestPath := filepath.Join(parseDir, "manifest.json")
	parseEncoded, parseErr := json.Marshal(releaseManifestSnapshot{
		Package:     "example.com/app",
		Profile:     "release",
		GOOS:        "js",
		GOARCH:      "wasm",
		Artifacts:   parseArtifacts,
		Attribution: parseAttribution,
	})
	if parseErr != nil {
		parseT.Fatalf("marshal release manifest snapshot: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(parseManifestPath, parseEncoded, 0o644); parseErr2 != nil {
		parseT.Fatalf("write release manifest snapshot: %v", parseErr2)
	}
	return parseManifestPath
}

// buildReleasePackageAttributionSnapshot writes one package-attribution snapshot for diff-report tests.
func buildReleasePackageAttributionSnapshot(parseT *testing.T, parsePath string, parsePackages []releasePackageSizeRecord) {
	parseT.Helper()

	parseEncoded, parseErr := json.Marshal(releasePackageAttributionSnapshot{Packages: parsePackages})
	if parseErr != nil {
		parseT.Fatalf("marshal package attribution snapshot: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(parsePath, parseEncoded, 0o644); parseErr2 != nil {
		parseT.Fatalf("write package attribution snapshot: %v", parseErr2)
	}
}
