package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/pwa"
	playwright "github.com/mxschmitt/playwright-go"
)

// TestMeasureReleaseStartupExercisesControlFlow verifies startup measurement success and failure branches.
func TestMeasureReleaseStartupExercisesControlFlow(parseT *testing.T) {
	parseOriginalResolveWasmExec := releaseResolveWasmExec
	parseOriginalRunStartupProbe := releaseRunStartupProbeWithPlaywright
	parseT.Cleanup(func() {
		releaseResolveWasmExec = parseOriginalResolveWasmExec
		releaseRunStartupProbeWithPlaywright = parseOriginalRunStartupProbe
	})

	parseT.Run("skip none mode", func(parseT2 *testing.T) {
		parseRecord, parseErr := measureReleaseStartup(releaseConfig{startupMeasure: "none"}, nil)
		if parseErr != nil {
			parseT2.Fatalf("measureReleaseStartup(none): %v", parseErr)
		}
		if parseRecord != nil {
			parseT2.Fatalf("expected no startup record for none mode, got %#v", parseRecord)
		}
	})

	parseT.Run("resolve wasm exec failure", func(parseT2 *testing.T) {
		releaseResolveWasmExec = func() (string, error) {
			return "", errors.New("missing wasm_exec.js")
		}
		releaseRunStartupProbeWithPlaywright = parseOriginalRunStartupProbe

		parseRecord, parseErr := measureReleaseStartup(releaseConfig{
			outDir:         parseT2.TempDir(),
			binaryName:     "app.wasm",
			startupMeasure: "browser",
		}, nil)
		if parseErr == nil || !strings.Contains(parseErr.Error(), "resolve wasm_exec.js for startup measurement") {
			parseT2.Fatalf("expected resolve wasm_exec error, got record=%#v err=%v", parseRecord, parseErr)
		}
	})

	parseT.Run("probe runner failure", func(parseT2 *testing.T) {
		releaseResolveWasmExec = func() (string, error) {
			return filepath.Join(parseT2.TempDir(), "wasm_exec.js"), nil
		}
		releaseRunStartupProbeWithPlaywright = func(parseProbeURL string, parseReportPath string, parseTimeoutMs int) error {
			return errors.New("probe failed")
		}

		parseRecord, parseErr := measureReleaseStartup(releaseConfig{
			outDir:           parseT2.TempDir(),
			binaryName:       "app.wasm",
			startupMeasure:   "browser",
			startupTimeoutMs: 9000,
		}, nil)
		if parseErr == nil || !strings.Contains(parseErr.Error(), "measure release startup: probe failed") {
			parseT2.Fatalf("expected probe failure, got record=%#v err=%v", parseRecord, parseErr)
		}
	})

	parseT.Run("report file required", func(parseT2 *testing.T) {
		parseOutDir := parseT2.TempDir()
		releaseResolveWasmExec = func() (string, error) {
			return filepath.Join(parseOutDir, "wasm_exec.js"), nil
		}
		releaseRunStartupProbeWithPlaywright = func(parseProbeURL string, parseReportPath string, parseTimeoutMs int) error {
			return nil
		}

		parseRecord, parseErr := measureReleaseStartup(releaseConfig{
			outDir:           parseOutDir,
			binaryName:       "app.wasm",
			startupMeasure:   "browser",
			startupTimeoutMs: 7500,
		}, map[string]releaseArtifactRecord{"gzip": {Path: "app.wasm.gz"}})
		if parseErr == nil || !strings.Contains(parseErr.Error(), "startup measurement did not produce a report") {
			parseT2.Fatalf("expected missing report error, got record=%#v err=%v", parseRecord, parseErr)
		}
	})

	parseT.Run("writes startup report", func(parseT2 *testing.T) {
		parseOutDir := parseT2.TempDir()
		releaseResolveWasmExec = func() (string, error) {
			return filepath.Join(parseOutDir, "wasm_exec.js"), nil
		}
		releaseRunStartupProbeWithPlaywright = func(parseProbeURL string, parseReportPath string, parseTimeoutMs int) error {
			if !strings.Contains(parseProbeURL, "/__gwc/startup-probe.html") {
				parseT2.Fatalf("expected probe URL to target startup probe page, got %q", parseProbeURL)
			}
			if parseTimeoutMs != 1234 {
				parseT2.Fatalf("expected startup timeout to propagate, got %d", parseTimeoutMs)
			}
			return os.WriteFile(parseReportPath, []byte(`{"startup":{"readyMs":12}}`), 0644)
		}

		parseRecord, parseErr := measureReleaseStartup(releaseConfig{
			outDir:           parseOutDir,
			binaryName:       "app.wasm",
			startupMeasure:   "browser",
			startupTimeoutMs: 1234,
		}, map[string]releaseArtifactRecord{"gzip": {Path: "app.wasm.gz"}})
		if parseErr != nil {
			parseT2.Fatalf("measureReleaseStartup(browser): %v", parseErr)
		}
		if parseRecord == nil || parseRecord.Mode != "browser" || parseRecord.Path != "wasm-startup-report.json" || parseRecord.TransportEncoding != "gzip" {
			parseT2.Fatalf("unexpected startup record %#v", parseRecord)
		}
		if _, parseErr2 := os.Stat(filepath.Join(parseOutDir, parseRecord.Path)); parseErr2 != nil {
			parseT2.Fatalf("expected startup report artifact to exist: %v", parseErr2)
		}
	})
}

// TestValidateReleaseSmokeExercisesValidationPaths verifies release smoke validation success and error branches.
func TestValidateReleaseSmokeExercisesValidationPaths(parseT *testing.T) {
	parseOriginalMarshalIndent := releaseMarshalIndent
	parseT.Cleanup(func() {
		releaseMarshalIndent = parseOriginalMarshalIndent
	})

	parseT.Run("skip disabled smoke validation", func(parseT2 *testing.T) {
		parseRecord, parseErr := validateReleaseSmoke(releaseConfig{}, "", nil, nil)
		if parseErr != nil {
			parseT2.Fatalf("validateReleaseSmoke(disabled): %v", parseErr)
		}
		if parseRecord != nil {
			parseT2.Fatalf("expected nil validation record when smoke validation is disabled, got %#v", parseRecord)
		}
	})

	parseT.Run("successful validation writes report", func(parseT2 *testing.T) {
		parseOutDir := parseT2.TempDir()
		parseArtifacts := buildReleaseArtifacts(parseT2, parseOutDir)
		parseManifestPath := buildReleaseManifestPath(parseT2, parseOutDir, parseArtifacts)

		parseRecord, parseErr := validateReleaseSmoke(releaseConfig{
			outDir:        parseOutDir,
			binaryName:    "app.wasm",
			validateSmoke: true,
		}, parseManifestPath, parseArtifacts, &releaseStartupRecord{Path: "wasm-startup-report.json"})
		if parseErr != nil {
			parseT2.Fatalf("validateReleaseSmoke(success): %v", parseErr)
		}
		if parseRecord == nil || parseRecord.Path != "wasm-release-validation.json" || parseRecord.StartupReportPath != "wasm-startup-report.json" {
			parseT2.Fatalf("unexpected validation record %#v", parseRecord)
		}
		if len(parseRecord.Checks) < 3 {
			parseT2.Fatalf("expected validation checks, got %#v", parseRecord)
		}
		if !strings.Contains(strings.ToLower(parseRecord.WasmContentType), "application/wasm") || parseRecord.WasmContentEncoding != "gzip" {
			parseT2.Fatalf("unexpected validation wasm headers %#v", parseRecord)
		}
		if _, parseErr2 := os.Stat(filepath.Join(parseOutDir, parseRecord.Path)); parseErr2 != nil {
			parseT2.Fatalf("expected validation report artifact: %v", parseErr2)
		}
	})

	parseT.Run("read and parse manifest failures", func(parseT2 *testing.T) {
		parseOutDir := parseT2.TempDir()
		parseArtifacts := buildReleaseArtifacts(parseT2, parseOutDir)

		parseRecord, parseErr := validateReleaseSmoke(releaseConfig{
			outDir:        parseOutDir,
			binaryName:    "app.wasm",
			validateSmoke: true,
		}, filepath.Join(parseOutDir, "missing-manifest.json"), parseArtifacts, &releaseStartupRecord{Path: "wasm-startup-report.json"})
		if parseErr == nil || !strings.Contains(parseErr.Error(), "read release manifest for smoke validation") {
			parseT2.Fatalf("expected manifest read failure, got record=%#v err=%v", parseRecord, parseErr)
		}

		parseInvalidManifestPath := filepath.Join(parseOutDir, "invalid-manifest.json")
		if parseErr2 := os.WriteFile(parseInvalidManifestPath, []byte(`{}`), 0644); parseErr2 != nil {
			parseT2.Fatalf("write invalid manifest: %v", parseErr2)
		}
		parseRecord, parseErr = validateReleaseSmoke(releaseConfig{
			outDir:        parseOutDir,
			binaryName:    "app.wasm",
			validateSmoke: true,
		}, parseInvalidManifestPath, parseArtifacts, &releaseStartupRecord{Path: "wasm-startup-report.json"})
		if parseErr == nil || !strings.Contains(parseErr.Error(), "validate release manifest for smoke validation") {
			parseT2.Fatalf("expected manifest parse failure, got record=%#v err=%v", parseRecord, parseErr)
		}
	})

	parseT.Run("startup report required", func(parseT2 *testing.T) {
		parseOutDir := parseT2.TempDir()
		parseArtifacts := buildReleaseArtifacts(parseT2, parseOutDir)
		parseManifestPath := buildReleaseManifestPath(parseT2, parseOutDir, parseArtifacts)

		parseRecord, parseErr := validateReleaseSmoke(releaseConfig{
			outDir:        parseOutDir,
			binaryName:    "app.wasm",
			validateSmoke: true,
		}, parseManifestPath, parseArtifacts, nil)
		if parseErr == nil || !strings.Contains(parseErr.Error(), "release smoke validation requires a startup probe result") {
			parseT2.Fatalf("expected startup report requirement error, got record=%#v err=%v", parseRecord, parseErr)
		}
	})

	parseT.Run("artifact mismatch", func(parseT2 *testing.T) {
		parseOutDir := parseT2.TempDir()
		parseArtifacts := buildReleaseArtifacts(parseT2, parseOutDir)
		parseManifestPath := buildReleaseManifestPath(parseT2, parseOutDir, parseArtifacts)

		parseManifestBytes, parseErr := os.ReadFile(parseManifestPath)
		if parseErr != nil {
			parseT2.Fatalf("read manifest: %v", parseErr)
		}
		var parseManifest pwa.WasmReleaseManifest
		if parseErr2 := json.Unmarshal(parseManifestBytes, &parseManifest); parseErr2 != nil {
			parseT2.Fatalf("decode manifest: %v", parseErr2)
		}
		parseWasm := parseManifest.Artifacts["wasm"]
		parseWasm.SHA256 = strings.Repeat("0", 64)
		parseManifest.Artifacts["wasm"] = parseWasm
		parseEncoded, parseErr3 := json.Marshal(parseManifest)
		if parseErr3 != nil {
			parseT2.Fatalf("re-encode manifest: %v", parseErr3)
		}
		if parseErr4 := os.WriteFile(parseManifestPath, parseEncoded, 0644); parseErr4 != nil {
			parseT2.Fatalf("write mismatched manifest: %v", parseErr4)
		}

		parseRecord, parseErr := validateReleaseSmoke(releaseConfig{
			outDir:        parseOutDir,
			binaryName:    "app.wasm",
			validateSmoke: true,
		}, parseManifestPath, parseArtifacts, &releaseStartupRecord{Path: "wasm-startup-report.json"})
		if parseErr == nil || !strings.Contains(parseErr.Error(), "manifest record does not match on-disk artifact") {
			parseT2.Fatalf("expected artifact mismatch error, got record=%#v err=%v", parseRecord, parseErr)
		}
	})

	parseT.Run("marshal and write failures", func(parseT2 *testing.T) {
		parseOutDir := parseT2.TempDir()
		parseArtifacts := buildReleaseArtifacts(parseT2, parseOutDir)
		parseManifestPath := buildReleaseManifestPath(parseT2, parseOutDir, parseArtifacts)

		releaseMarshalIndent = func(parseV any, parsePrefix string, parseIndent string) ([]byte, error) {
			return nil, errors.New("marshal failed")
		}
		parseRecord, parseErr := validateReleaseSmoke(releaseConfig{
			outDir:        parseOutDir,
			binaryName:    "app.wasm",
			validateSmoke: true,
		}, parseManifestPath, parseArtifacts, &releaseStartupRecord{Path: "wasm-startup-report.json"})
		if parseErr == nil || !strings.Contains(parseErr.Error(), "encode release validation report") {
			parseT2.Fatalf("expected marshal failure, got record=%#v err=%v", parseRecord, parseErr)
		}

		releaseMarshalIndent = parseOriginalMarshalIndent
		if parseErr2 := os.MkdirAll(filepath.Join(parseOutDir, "wasm-release-validation.json"), 0755); parseErr2 != nil {
			parseT2.Fatalf("mkdir validation report collision path: %v", parseErr2)
		}
		parseRecord, parseErr = validateReleaseSmoke(releaseConfig{
			outDir:        parseOutDir,
			binaryName:    "app.wasm",
			validateSmoke: true,
		}, parseManifestPath, parseArtifacts, &releaseStartupRecord{Path: "wasm-startup-report.json"})
		if parseErr == nil || !strings.Contains(parseErr.Error(), "write release validation report") {
			parseT2.Fatalf("expected write failure, got record=%#v err=%v", parseRecord, parseErr)
		}
	})
}

// TestRunReleaseStartupProbeWithPlaywrightReportsEarlyFailures verifies the install and runtime startup error branches.
func TestRunReleaseStartupProbeWithPlaywrightReportsEarlyFailures(parseT *testing.T) {
	parseOriginalInstall := releasePlaywrightInstall
	parseOriginalRun := releasePlaywrightRun
	parseT.Cleanup(func() {
		releasePlaywrightInstall = parseOriginalInstall
		releasePlaywrightRun = parseOriginalRun
	})

	parseT.Run("install failure", func(parseT2 *testing.T) {
		releasePlaywrightInstall = func(parseOptions *playwright.RunOptions) error {
			return errors.New("install failed")
		}
		releasePlaywrightRun = parseOriginalRun

		parseErr := runReleaseStartupProbeWithPlaywright("http://127.0.0.1:1", filepath.Join(parseT2.TempDir(), "report.json"), 1000)
		if parseErr == nil || !strings.Contains(parseErr.Error(), "install playwright-go runtime") {
			parseT2.Fatalf("expected install failure, got %v", parseErr)
		}
	})

	parseT.Run("runtime startup failure", func(parseT2 *testing.T) {
		releasePlaywrightInstall = func(parseOptions *playwright.RunOptions) error {
			return nil
		}
		releasePlaywrightRun = func(parseOptions *playwright.RunOptions) (*playwright.Playwright, error) {
			return nil, errors.New("runtime failed")
		}

		parseErr := runReleaseStartupProbeWithPlaywright("http://127.0.0.1:1", filepath.Join(parseT2.TempDir(), "report.json"), 1000)
		if parseErr == nil || !strings.Contains(parseErr.Error(), "run playwright-go runtime") {
			parseT2.Fatalf("expected runtime startup failure, got %v", parseErr)
		}
	})
}

// buildReleaseArtifacts writes a minimal wasm release artifact set for smoke validation tests.
func buildReleaseArtifacts(parseT *testing.T, parseDir string) map[string]releaseArtifactRecord {
	parseT.Helper()

	parseWasmPath := filepath.Join(parseDir, "app.wasm")
	parseGzipPath := filepath.Join(parseDir, "app.wasm.gz")
	if parseErr := os.WriteFile(parseWasmPath, []byte("wasm-binary"), 0644); parseErr != nil {
		parseT.Fatalf("write wasm artifact: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(parseGzipPath, []byte("gzip-binary"), 0644); parseErr2 != nil {
		parseT.Fatalf("write gzip artifact: %v", parseErr2)
	}

	parseWasmRecord, parseErr := releaseArtifactRecordForPath(parseDir, parseWasmPath)
	if parseErr != nil {
		parseT.Fatalf("record wasm artifact: %v", parseErr)
	}
	parseGzipRecord, parseErr2 := releaseArtifactRecordForPath(parseDir, parseGzipPath)
	if parseErr2 != nil {
		parseT.Fatalf("record gzip artifact: %v", parseErr2)
	}
	return map[string]releaseArtifactRecord{
		"wasm": parseWasmRecord,
		"gzip": parseGzipRecord,
	}
}

// buildReleaseManifestPath writes a release manifest that matches the provided artifacts.
func buildReleaseManifestPath(parseT *testing.T, parseDir string, parseArtifacts map[string]releaseArtifactRecord) string {
	parseT.Helper()

	parseManifest := pwa.WasmReleaseManifest{
		Package: "example.com/app",
		Profile: "release",
		GOOS:    "js",
		GOARCH:  "wasm",
		Flags:   pwa.WasmReleaseFlags{Compression: true},
		Artifacts: map[string]pwa.WasmReleaseArtifact{
			"wasm": {
				Path:   parseArtifacts["wasm"].Path,
				Bytes:  parseArtifacts["wasm"].Bytes,
				SHA256: parseArtifacts["wasm"].SHA256,
			},
			"gzip": {
				Path:   parseArtifacts["gzip"].Path,
				Bytes:  parseArtifacts["gzip"].Bytes,
				SHA256: parseArtifacts["gzip"].SHA256,
			},
		},
	}
	parseEncoded, parseErr := json.Marshal(parseManifest)
	if parseErr != nil {
		parseT.Fatalf("marshal release manifest: %v", parseErr)
	}

	parseManifestPath := filepath.Join(parseDir, "manifest.json")
	if parseErr2 := os.WriteFile(parseManifestPath, parseEncoded, 0644); parseErr2 != nil {
		parseT.Fatalf("write release manifest: %v", parseErr2)
	}
	return parseManifestPath
}
