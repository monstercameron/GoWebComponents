package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunWasmMeasureWritesManifestAndArtifacts(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseOutDir := filepath.Join(parseRoot, "out")

	parseOriginalRunCommand := wasmRunCommand
	parseT.Cleanup(func() {
		wasmRunCommand = parseOriginalRunCommand
	})
	wasmRunCommand = func(parseCommand string, parseArgs []string, parseCwd string, parseEnv []string) (string, error) {
		if parseCommand != "go" {
			parseT.Fatalf("expected go command, got %q", parseCommand)
		}
		if len(parseArgs) == 0 {
			parseT.Fatal("expected args for go command")
		}
		switch parseArgs[0] {
		case "build":
			parseOutputIndex := -1
			for parseIndex := 0; parseIndex < len(parseArgs)-1; parseIndex++ {
				if parseArgs[parseIndex] == "-o" {
					parseOutputIndex = parseIndex + 1
					break
				}
			}
			if parseOutputIndex < 0 {
				parseT.Fatalf("expected -o in build args: %#v", parseArgs)
			}
			parseWasmPath := parseArgs[parseOutputIndex]
			if parseErr := os.MkdirAll(filepath.Dir(parseWasmPath), 0755); parseErr != nil {
				parseT.Fatalf("mkdir wasm output dir: %v", parseErr)
			}
			if parseErr2 := os.WriteFile(parseWasmPath, []byte("wasm-binary"), 0644); parseErr2 != nil {
				parseT.Fatalf("write wasm output: %v", parseErr2)
			}
			return "", nil
		case "version":
			return "go version go1.26.0 windows/amd64", nil
		default:
			parseT.Fatalf("unexpected go args: %#v", parseArgs)
			return "", nil
		}
	}

	parseStdout, parseRestoreStdout, parseErr3 := captureExamplesStdout()
	if parseErr3 != nil {
		parseT.Fatalf("capture stdout: %v", parseErr3)
	}
	defer parseRestoreStdout()

	parseErr3 = (launcher{}).runWasm([]string{
		"measure",
		"-package", "./examples/public/ui-render",
		"-out-dir", parseOutDir,
		"-binary-name", "test.wasm",
		"-manifest-name", "report.json",
		"-release-profile",
		"-json",
	})
	if parseErr3 != nil {
		parseT.Fatalf("run wasm measure: %v", parseErr3)
	}

	parseOutput, parseErr3 := parseStdout()
	if parseErr3 != nil {
		parseT.Fatalf("read stdout: %v", parseErr3)
	}
	var parseSummary wasmMeasureSummary
	if parseErr4 := json.Unmarshal([]byte(parseOutput), &parseSummary); parseErr4 != nil {
		parseT.Fatalf("decode wasm summary: %v\n%s", parseErr4, parseOutput)
	}
	if !parseSummary.OK {
		parseT.Fatalf("expected successful summary, got %#v", parseSummary)
	}
	if parseSummary.Manifest.Profile != "release" {
		parseT.Fatalf("expected release profile, got %#v", parseSummary)
	}
	for _, parseKey := range []string{"wasm", "gzip", "brotli"} {
		if _, parseOk := parseSummary.Manifest.Artifacts[parseKey]; !parseOk {
			parseT.Fatalf("expected artifact %q in summary: %#v", parseKey, parseSummary)
		}
	}
	if _, parseErr5 := os.Stat(filepath.FromSlash(parseSummary.ManifestPath)); parseErr5 != nil {
		parseT.Fatalf("expected manifest path to exist: %v", parseErr5)
	}
}

func TestRunWasmRejectsUnknownSubcommand(parseT *testing.T) {
	parseErr := (launcher{}).runWasm([]string{"mystery"})
	if parseErr == nil || !strings.Contains(parseErr.Error(), `unknown wasm subcommand "mystery"`) {
		parseT.Fatalf("expected unknown subcommand error, got %v", parseErr)
	}
}

func TestRunWasmCompareDetectsRegressionBeyondThreshold(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseBaselinePath := filepath.Join(parseRoot, "baseline.json")
	parseCandidatePath := filepath.Join(parseRoot, "candidate.json")
	if parseErr := os.WriteFile(parseBaselinePath, []byte(`{"phases":{"go_build_ms":100},"artifacts":{"wasm":{"bytes":100}}}`), 0644); parseErr != nil {
		parseT.Fatalf("write baseline: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(parseCandidatePath, []byte(`{"phases":{"go_build_ms":130},"artifacts":{"wasm":{"bytes":100}}}`), 0644); parseErr2 != nil {
		parseT.Fatalf("write candidate: %v", parseErr2)
	}

	parseStdout, parseRestoreStdout, parseErr3 := captureExamplesStdout()
	if parseErr3 != nil {
		parseT.Fatalf("capture stdout: %v", parseErr3)
	}
	defer parseRestoreStdout()

	parseErr3 = (launcher{}).runWasm([]string{"compare", "-baseline", parseBaselinePath, "-candidate", parseCandidatePath, "-json"})
	if parseErr3 == nil || !strings.Contains(parseErr3.Error(), "regressions") {
		parseT.Fatalf("expected regression error, got %v", parseErr3)
	}
	parseOutput, parseErr3 := parseStdout()
	if parseErr3 != nil {
		parseT.Fatalf("read stdout: %v", parseErr3)
	}
	var parseSummary wasmCompareSummary
	if parseErr4 := json.Unmarshal([]byte(parseOutput), &parseSummary); parseErr4 != nil {
		parseT.Fatalf("decode compare summary: %v\n%s", parseErr4, parseOutput)
	}
	if parseSummary.OK {
		parseT.Fatalf("expected failed compare summary, got %#v", parseSummary)
	}
	if parseSummary.Counts.Regressed == 0 {
		parseT.Fatalf("expected regressed metrics, got %#v", parseSummary)
	}
}

func TestRunWasmComparePassesWithinThreshold(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseBaselinePath := filepath.Join(parseRoot, "baseline.json")
	parseCandidatePath := filepath.Join(parseRoot, "candidate.json")
	if parseErr := os.WriteFile(parseBaselinePath, []byte(`{"phases":{"go_build_ms":100}}`), 0644); parseErr != nil {
		parseT.Fatalf("write baseline: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(parseCandidatePath, []byte(`{"phases":{"go_build_ms":105}}`), 0644); parseErr2 != nil {
		parseT.Fatalf("write candidate: %v", parseErr2)
	}

	parseStdout, parseRestoreStdout, parseErr3 := captureExamplesStdout()
	if parseErr3 != nil {
		parseT.Fatalf("capture stdout: %v", parseErr3)
	}
	defer parseRestoreStdout()

	if parseErr4 := (launcher{}).runWasm([]string{"compare", "-baseline", parseBaselinePath, "-candidate", parseCandidatePath, "-timing-regression-percent", "10", "-json"}); parseErr4 != nil {
		parseT.Fatalf("compare within threshold should pass: %v", parseErr4)
	}
	parseOutput, parseErr3 := parseStdout()
	if parseErr3 != nil {
		parseT.Fatalf("read stdout: %v", parseErr3)
	}
	var parseSummary wasmCompareSummary
	if parseErr5 := json.Unmarshal([]byte(parseOutput), &parseSummary); parseErr5 != nil {
		parseT.Fatalf("decode compare summary: %v\n%s", parseErr5, parseOutput)
	}
	if !parseSummary.OK || parseSummary.Counts.Regressed != 0 {
		parseT.Fatalf("expected passing summary, got %#v", parseSummary)
	}
}

func TestRunWasmCompareCompressionWritesSummaryWithoutOptimizer(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseOutDir := filepath.Join(parseRoot, "comparison")

	parseOriginalRunCommand := wasmRunCommand
	parseOriginalLookPath := wasmLookPath
	parseT.Cleanup(func() {
		wasmRunCommand = parseOriginalRunCommand
		wasmLookPath = parseOriginalLookPath
	})
	wasmLookPath = func(parseFile string) (string, error) {
		return "", errors.New("not found")
	}
	wasmRunCommand = func(parseCommand string, parseArgs []string, parseCwd string, parseEnv []string) (string, error) {
		if parseCommand != "go" {
			parseT.Fatalf("expected go command, got %q", parseCommand)
		}
		if len(parseArgs) == 0 {
			parseT.Fatal("expected args for go command")
		}
		switch parseArgs[0] {
		case "build":
			parseOutputIndex := -1
			for parseIndex := 0; parseIndex < len(parseArgs)-1; parseIndex++ {
				if parseArgs[parseIndex] == "-o" {
					parseOutputIndex = parseIndex + 1
					break
				}
			}
			if parseOutputIndex < 0 {
				parseT.Fatalf("expected -o in build args: %#v", parseArgs)
			}
			parseWasmPath := parseArgs[parseOutputIndex]
			if parseErr := os.MkdirAll(filepath.Dir(parseWasmPath), 0755); parseErr != nil {
				parseT.Fatalf("mkdir wasm output dir: %v", parseErr)
			}
			if parseErr2 := os.WriteFile(parseWasmPath, []byte("wasm-binary"), 0644); parseErr2 != nil {
				parseT.Fatalf("write wasm output: %v", parseErr2)
			}
			return "", nil
		case "version":
			return "go version go1.26.0 windows/amd64", nil
		default:
			parseT.Fatalf("unexpected command args: %#v", parseArgs)
			return "", nil
		}
	}

	parseStdout, parseRestoreStdout, parseErr3 := captureExamplesStdout()
	if parseErr3 != nil {
		parseT.Fatalf("capture stdout: %v", parseErr3)
	}
	defer parseRestoreStdout()

	if parseErr4 := (launcher{}).runWasm([]string{
		"compare-compression",
		"-package", "./examples/public/ui-render",
		"-out-dir", parseOutDir,
		"-summary-name", "summary.json",
		"-json",
	}); parseErr4 != nil {
		parseT.Fatalf("run compare-compression: %v", parseErr4)
	}

	parseOutput, parseErr3 := parseStdout()
	if parseErr3 != nil {
		parseT.Fatalf("read stdout: %v", parseErr3)
	}
	var parseSummary wasmCompressionSummary
	if parseErr5 := json.Unmarshal([]byte(parseOutput), &parseSummary); parseErr5 != nil {
		parseT.Fatalf("decode compression summary: %v\n%s", parseErr5, parseOutput)
	}
	if !parseSummary.OK {
		parseT.Fatalf("expected ok summary, got %#v", parseSummary)
	}
	if parseSummary.Environment.WasmOptAvailable {
		parseT.Fatalf("expected wasm-opt unavailable summary, got %#v", parseSummary.Environment)
	}
	for _, parseKey := range []string{"plain_raw", "stripped_raw", "stripped_compressed"} {
		if _, parseOk := parseSummary.Variants[parseKey]; !parseOk {
			parseT.Fatalf("expected variant %q in summary", parseKey)
		}
	}
	if _, parseOk2 := parseSummary.Variants["optimized_raw"]; parseOk2 {
		parseT.Fatalf("did not expect optimized variant when optimizer is unavailable")
	}
	if _, parseErr6 := os.Stat(filepath.FromSlash(parseSummary.SummaryPath)); parseErr6 != nil {
		parseT.Fatalf("expected summary file to exist: %v", parseErr6)
	}
}

func TestRunWasmCompareCompressionIncludesOptimizedVariants(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseOutDir := filepath.Join(parseRoot, "comparison")

	parseOriginalRunCommand := wasmRunCommand
	parseOriginalLookPath := wasmLookPath
	parseOriginalArtifactRecord := wasmReleaseArtifactRecordForPath
	parseT.Cleanup(func() {
		wasmRunCommand = parseOriginalRunCommand
		wasmLookPath = parseOriginalLookPath
		wasmReleaseArtifactRecordForPath = parseOriginalArtifactRecord
	})
	wasmReleaseArtifactRecordForPath = releaseArtifactRecordForPath
	wasmLookPath = func(parseFile string) (string, error) {
		if parseFile == "wasm-opt" {
			return filepath.Join(parseRoot, "fake-wasm-opt"), nil
		}
		return "", errors.New("not found")
	}
	wasmRunCommand = func(parseCommand string, parseArgs []string, parseCwd string, parseEnv []string) (string, error) {
		if len(parseArgs) == 0 {
			parseT.Fatalf("expected args for command %q", parseCommand)
		}
		switch parseCommand {
		case "go":
			switch parseArgs[0] {
			case "build":
				parseOutputIndex := -1
				for parseIndex := 0; parseIndex < len(parseArgs)-1; parseIndex++ {
					if parseArgs[parseIndex] == "-o" {
						parseOutputIndex = parseIndex + 1
						break
					}
				}
				if parseOutputIndex < 0 {
					parseT.Fatalf("expected -o in build args: %#v", parseArgs)
				}
				parseWasmPath := parseArgs[parseOutputIndex]
				if parseErr := os.MkdirAll(filepath.Dir(parseWasmPath), 0755); parseErr != nil {
					parseT.Fatalf("mkdir wasm output dir: %v", parseErr)
				}
				if parseErr2 := os.WriteFile(parseWasmPath, []byte("wasm-binary"), 0644); parseErr2 != nil {
					parseT.Fatalf("write wasm output: %v", parseErr2)
				}
				return "", nil
			case "version":
				return "go version go1.26.0 windows/amd64", nil
			default:
				parseT.Fatalf("unexpected go command args: %#v", parseArgs)
				return "", nil
			}
		case "wasm-opt":
			if len(parseArgs) < 4 {
				parseT.Fatalf("expected wasm-opt args, got %#v", parseArgs)
			}
			parseSourcePath := parseArgs[0]
			parseTargetPath := parseArgs[len(parseArgs)-1]
			parseOptimizedBytes, parseErr3 := os.ReadFile(parseSourcePath)
			if parseErr3 != nil {
				parseT.Fatalf("read source wasm: %v", parseErr3)
			}
			parseOptimizedBytes = append(parseOptimizedBytes, []byte("-optimized")...)
			if parseErr4 := os.WriteFile(parseTargetPath, parseOptimizedBytes, 0644); parseErr4 != nil {
				parseT.Fatalf("write optimized wasm: %v", parseErr4)
			}
			return "", nil
		default:
			parseT.Fatalf("unexpected command %q", parseCommand)
			return "", nil
		}
	}

	parseStdout, parseRestoreStdout, parseErr5 := captureExamplesStdout()
	if parseErr5 != nil {
		parseT.Fatalf("capture stdout: %v", parseErr5)
	}
	defer parseRestoreStdout()

	if parseErr6 := (launcher{}).runWasm([]string{
		"compare-compression",
		"-package", "./examples/public/ui-render",
		"-out-dir", parseOutDir,
		"-summary-name", "summary.json",
		"-json",
	}); parseErr6 != nil {
		parseT.Fatalf("run compare-compression: %v", parseErr6)
	}

	parseOutput, parseErr5 := parseStdout()
	if parseErr5 != nil {
		parseT.Fatalf("read stdout: %v", parseErr5)
	}
	var parseSummary wasmCompressionSummary
	if parseErr7 := json.Unmarshal([]byte(parseOutput), &parseSummary); parseErr7 != nil {
		parseT.Fatalf("decode compression summary: %v\n%s", parseErr7, parseOutput)
	}
	if !parseSummary.Environment.WasmOptAvailable {
		parseT.Fatalf("expected wasm-opt available summary, got %#v", parseSummary.Environment)
	}
	for _, parseKey := range []string{"optimized_raw", "optimized_compressed"} {
		if _, parseOk := parseSummary.Variants[parseKey]; !parseOk {
			parseT.Fatalf("expected optimized variant %q in summary", parseKey)
		}
	}
	parseOptimizedCompressed, parseOk2 := parseSummary.Variants["optimized_compressed"].(map[string]interface{})
	if !parseOk2 {
		parseT.Fatalf("expected optimized compressed variant payload, got %#v", parseSummary.Variants["optimized_compressed"])
	}
	parseParityChecks, parseOk2 := parseOptimizedCompressed["parity_checks"].(map[string]interface{})
	if !parseOk2 {
		parseT.Fatalf("expected parity checks in optimized compressed payload, got %#v", parseOptimizedCompressed)
	}
	parseRawToDelivery, parseOk2 := parseParityChecks["raw_to_delivery_copy"].(map[string]interface{})
	if !parseOk2 {
		parseT.Fatalf("expected raw_to_delivery_copy parity entry, got %#v", parseParityChecks)
	}
	parseOkValue, parseOk2 := parseRawToDelivery["ok"].(bool)
	if !parseOk2 || !parseOkValue {
		parseT.Fatalf("expected successful parity check status, got %#v", parseRawToDelivery)
	}
}

func TestRunWasmCompareCompressionFailsOnParityMismatch(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseOutDir := filepath.Join(parseRoot, "comparison")

	parseOriginalRunCommand := wasmRunCommand
	parseOriginalLookPath := wasmLookPath
	parseOriginalArtifactRecord := wasmReleaseArtifactRecordForPath
	parseT.Cleanup(func() {
		wasmRunCommand = parseOriginalRunCommand
		wasmLookPath = parseOriginalLookPath
		wasmReleaseArtifactRecordForPath = parseOriginalArtifactRecord
	})
	wasmReleaseArtifactRecordForPath = func(parseBaseDir string, parseArtifactPath string) (releaseArtifactRecord, error) {
		parseRecord, parseErr := releaseArtifactRecordForPath(parseBaseDir, parseArtifactPath)
		if parseErr != nil {
			return releaseArtifactRecord{}, parseErr
		}
		if strings.Contains(filepath.ToSlash(parseBaseDir), "optimized-compressed") && strings.HasSuffix(filepath.ToSlash(parseArtifactPath), ".wasm") {
			parseRecord.SHA256 = "parity-mismatch"
		}
		return parseRecord, nil
	}
	wasmLookPath = func(parseFile string) (string, error) {
		if parseFile == "wasm-opt" {
			return filepath.Join(parseRoot, "fake-wasm-opt"), nil
		}
		return "", errors.New("not found")
	}
	wasmRunCommand = func(parseCommand string, parseArgs []string, parseCwd string, parseEnv []string) (string, error) {
		if len(parseArgs) == 0 {
			parseT.Fatalf("expected args for command %q", parseCommand)
		}
		switch parseCommand {
		case "go":
			switch parseArgs[0] {
			case "build":
				parseOutputIndex := -1
				for parseIndex := 0; parseIndex < len(parseArgs)-1; parseIndex++ {
					if parseArgs[parseIndex] == "-o" {
						parseOutputIndex = parseIndex + 1
						break
					}
				}
				if parseOutputIndex < 0 {
					parseT.Fatalf("expected -o in build args: %#v", parseArgs)
				}
				parseWasmPath := parseArgs[parseOutputIndex]
				if parseErr2 := os.MkdirAll(filepath.Dir(parseWasmPath), 0755); parseErr2 != nil {
					parseT.Fatalf("mkdir wasm output dir: %v", parseErr2)
				}
				if parseErr3 := os.WriteFile(parseWasmPath, []byte("wasm-binary"), 0644); parseErr3 != nil {
					parseT.Fatalf("write wasm output: %v", parseErr3)
				}
				return "", nil
			case "version":
				return "go version go1.26.0 windows/amd64", nil
			default:
				parseT.Fatalf("unexpected go command args: %#v", parseArgs)
				return "", nil
			}
		case "wasm-opt":
			parseSourcePath := parseArgs[0]
			parseTargetPath := parseArgs[len(parseArgs)-1]
			parseOptimizedBytes, parseErr4 := os.ReadFile(parseSourcePath)
			if parseErr4 != nil {
				parseT.Fatalf("read source wasm: %v", parseErr4)
			}
			if parseErr5 := os.WriteFile(parseTargetPath, append(parseOptimizedBytes, []byte("-optimized")...), 0644); parseErr5 != nil {
				parseT.Fatalf("write optimized wasm: %v", parseErr5)
			}
			return "", nil
		default:
			parseT.Fatalf("unexpected command %q", parseCommand)
			return "", nil
		}
	}

	parseErr6 := (launcher{}).runWasm([]string{
		"compare-compression",
		"-package", "./examples/public/ui-render",
		"-out-dir", parseOutDir,
		"-summary-name", "summary.json",
		"-json",
	})
	if parseErr6 == nil || !strings.Contains(parseErr6.Error(), "parity mismatch") {
		parseT.Fatalf("expected parity mismatch error, got %v", parseErr6)
	}
}

func TestRunWasmCompareCacheWritesSummary(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseOutDir := filepath.Join(parseRoot, "cache")
	parsePackageDir := filepath.Join(parseRoot, "pkg")
	if parseErr := os.MkdirAll(parsePackageDir, 0755); parseErr != nil {
		parseT.Fatalf("mkdir package dir: %v", parseErr)
	}
	parseSourcePath := filepath.Join(parsePackageDir, "alpha.go")
	parseOriginalSource := "package pkg\n\nfunc Value() int { return 1 }\n"
	if parseErr2 := os.WriteFile(parseSourcePath, []byte(parseOriginalSource), 0644); parseErr2 != nil {
		parseT.Fatalf("write source file: %v", parseErr2)
	}

	parseOriginalRunCommand := wasmRunCommand
	parseT.Cleanup(func() {
		wasmRunCommand = parseOriginalRunCommand
	})
	parseDefaultGoCache := filepath.Join(parseRoot, "default-gocache")
	parseDefaultGoModCache := filepath.Join(parseRoot, "default-gomodcache")
	parseModDownloadCalls := 0
	wasmRunCommand = func(parseCommand string, parseArgs []string, parseCwd string, parseEnv []string) (string, error) {
		if parseCommand != "go" {
			parseT.Fatalf("expected go command, got %q", parseCommand)
		}
		if len(parseArgs) == 0 {
			parseT.Fatal("expected args for go command")
		}
		switch parseArgs[0] {
		case "env":
			if len(parseArgs) < 2 {
				parseT.Fatalf("expected go env key in args: %#v", parseArgs)
			}
			switch parseArgs[1] {
			case "GOCACHE":
				return parseDefaultGoCache, nil
			case "GOMODCACHE":
				return parseDefaultGoModCache, nil
			default:
				parseT.Fatalf("unexpected go env key: %q", parseArgs[1])
				return "", nil
			}
		case "mod":
			if len(parseArgs) == 2 && parseArgs[1] == "download" {
				parseModDownloadCalls++
				return "", nil
			}
			parseT.Fatalf("unexpected go mod args: %#v", parseArgs)
			return "", nil
		case "build":
			parseOutputIndex := -1
			for parseIndex := 0; parseIndex < len(parseArgs)-1; parseIndex++ {
				if parseArgs[parseIndex] == "-o" {
					parseOutputIndex = parseIndex + 1
					break
				}
			}
			if parseOutputIndex < 0 {
				parseT.Fatalf("expected -o in build args: %#v", parseArgs)
			}
			parseWasmPath := parseArgs[parseOutputIndex]
			if parseErr3 := os.MkdirAll(filepath.Dir(parseWasmPath), 0755); parseErr3 != nil {
				parseT.Fatalf("mkdir wasm output dir: %v", parseErr3)
			}
			if parseErr4 := os.WriteFile(parseWasmPath, []byte("cache-wasm"), 0644); parseErr4 != nil {
				parseT.Fatalf("write wasm output: %v", parseErr4)
			}
			return "", nil
		case "version":
			return "go version go1.26.0 windows/amd64", nil
		default:
			parseT.Fatalf("unexpected command args: %#v", parseArgs)
			return "", nil
		}
	}

	parseStdout, parseRestoreStdout, parseErr5 := captureExamplesStdout()
	if parseErr5 != nil {
		parseT.Fatalf("capture stdout: %v", parseErr5)
	}
	defer parseRestoreStdout()

	if parseErr6 := (launcher{}).runWasm([]string{
		"compare-cache",
		"-package", parsePackageDir,
		"-out-dir", parseOutDir,
		"-summary-name", "summary.json",
		"-json",
	}); parseErr6 != nil {
		parseT.Fatalf("run compare-cache: %v", parseErr6)
	}

	parseOutput, parseErr5 := parseStdout()
	if parseErr5 != nil {
		parseT.Fatalf("read stdout: %v", parseErr5)
	}
	var parseSummary wasmCacheSummary
	if parseErr7 := json.Unmarshal([]byte(parseOutput), &parseSummary); parseErr7 != nil {
		parseT.Fatalf("decode cache summary: %v\n%s", parseErr7, parseOutput)
	}
	if !parseSummary.OK {
		parseT.Fatalf("expected ok summary, got %#v", parseSummary)
	}
	parseExpectedVariants := []string{
		"shared-cache-cold",
		"shared-cache-warm",
		"shared-cache-small-edit",
		"isolated-build-cache",
		"ci-style-cold",
		"ci-style-warm",
		"ci-style-small-edit",
	}
	for _, parseKey := range parseExpectedVariants {
		parseVariant, parseOk := parseSummary.Variants[parseKey]
		if !parseOk {
			parseT.Fatalf("expected variant %q in summary", parseKey)
		}
		if parseVariant.Status != "ok" {
			parseT.Fatalf("expected variant %q status ok, got %#v", parseKey, parseVariant)
		}
	}
	if parseSummary.Variants["ci-style-cold"].ModuleDownloadMS == nil {
		parseT.Fatalf("expected ci-style-cold to include module download timing")
	}
	if strings.TrimSpace(parseSummary.Variants["shared-cache-small-edit"].EditedFile) == "" {
		parseT.Fatalf("expected shared-cache-small-edit edited file to be set")
	}
	if strings.TrimSpace(parseSummary.Variants["ci-style-small-edit"].EditedFile) == "" {
		parseT.Fatalf("expected ci-style-small-edit edited file to be set")
	}
	if parseModDownloadCalls != 1 {
		parseT.Fatalf("expected one go mod download call, got %d", parseModDownloadCalls)
	}
	if _, parseErr8 := os.Stat(filepath.FromSlash(parseSummary.SummaryPath)); parseErr8 != nil {
		parseT.Fatalf("expected summary file to exist: %v", parseErr8)
	}
	parseFinalSource, parseErr5 := os.ReadFile(parseSourcePath)
	if parseErr5 != nil {
		parseT.Fatalf("read source after run: %v", parseErr5)
	}
	if string(parseFinalSource) != parseOriginalSource {
		parseT.Fatalf("expected source to be restored after small-edit variants")
	}
}

func TestRunWasmCompareToolchainDetectsRegression(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseOutDir := filepath.Join(parseRoot, "toolchain")

	parseOriginalRunCommand := wasmRunCommand
	parseT.Cleanup(func() {
		wasmRunCommand = parseOriginalRunCommand
	})
	wasmRunCommand = func(parseCommand string, parseArgs []string, parseCwd string, parseEnv []string) (string, error) {
		if len(parseArgs) == 0 {
			parseT.Fatalf("expected args for command %q", parseCommand)
		}
		switch parseCommand {
		case "go-baseline", "go-candidate":
			switch parseArgs[0] {
			case "build":
				parseOutputIndex := -1
				for parseIndex := 0; parseIndex < len(parseArgs)-1; parseIndex++ {
					if parseArgs[parseIndex] == "-o" {
						parseOutputIndex = parseIndex + 1
						break
					}
				}
				if parseOutputIndex < 0 {
					parseT.Fatalf("expected -o in build args: %#v", parseArgs)
				}
				parseWasmPath := parseArgs[parseOutputIndex]
				if parseErr := os.MkdirAll(filepath.Dir(parseWasmPath), 0755); parseErr != nil {
					parseT.Fatalf("mkdir wasm output dir: %v", parseErr)
				}
				parsePayload := []byte("baseline-wasm")
				if parseCommand == "go-candidate" {
					parsePayload = append(parsePayload, []byte("-regression")...)
				}
				if parseErr2 := os.WriteFile(parseWasmPath, parsePayload, 0644); parseErr2 != nil {
					parseT.Fatalf("write wasm output: %v", parseErr2)
				}
				return "", nil
			case "version":
				return fmt.Sprintf("go version %s windows/amd64", parseCommand), nil
			default:
				parseT.Fatalf("unexpected toolchain command args: %#v", parseArgs)
				return "", nil
			}
		default:
			parseT.Fatalf("unexpected command %q", parseCommand)
			return "", nil
		}
	}

	parseStdout, parseRestoreStdout, parseErr3 := captureExamplesStdout()
	if parseErr3 != nil {
		parseT.Fatalf("capture stdout: %v", parseErr3)
	}
	defer parseRestoreStdout()

	parseErr3 = (launcher{}).runWasm([]string{
		"compare-toolchain",
		"-package", "./examples/public/ui-render",
		"-baseline-go", "go-baseline",
		"-candidate-go", "go-candidate",
		"-out-dir", parseOutDir,
		"-skip-compression",
		"-json",
	})
	if parseErr3 == nil || !strings.Contains(parseErr3.Error(), "regressions") {
		parseT.Fatalf("expected regression error, got %v", parseErr3)
	}

	parseOutput, parseErr3 := parseStdout()
	if parseErr3 != nil {
		parseT.Fatalf("read stdout: %v", parseErr3)
	}
	var parseSummary wasmToolchainSummary
	if parseErr4 := json.Unmarshal([]byte(parseOutput), &parseSummary); parseErr4 != nil {
		parseT.Fatalf("decode toolchain summary: %v\n%s", parseErr4, parseOutput)
	}
	if parseSummary.RegressionExitCode != 1 {
		parseT.Fatalf("expected regression exit code 1, got %#v", parseSummary)
	}
	if parseSummary.Baseline.GoExecutable != "go-baseline" {
		parseT.Fatalf("unexpected baseline executable: %#v", parseSummary.Baseline)
	}
	if parseSummary.Candidate.GoExecutable != "go-candidate" {
		parseT.Fatalf("unexpected candidate executable: %#v", parseSummary.Candidate)
	}
	if _, parseErr5 := os.Stat(filepath.FromSlash(parseSummary.SummaryPath)); parseErr5 != nil {
		parseT.Fatalf("expected summary file to exist: %v", parseErr5)
	}
	parseComparisonPath := filepath.Join(filepath.FromSlash(parseOutDir), "toolchain-comparison.json")
	if _, parseErr6 := os.Stat(parseComparisonPath); parseErr6 != nil {
		parseT.Fatalf("expected comparison file to exist: %v", parseErr6)
	}
}

func TestRunWasmCompareToolchainUsesCustomBinaryName(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseOutDir := filepath.Join(parseRoot, "toolchain")
	parseBaselineBinary := filepath.Join(parseOutDir, "baseline", "custom.wasm")
	parseCandidateBinary := filepath.Join(parseOutDir, "candidate", "custom.wasm")

	parseOriginalRunCommand := wasmRunCommand
	parseT.Cleanup(func() {
		wasmRunCommand = parseOriginalRunCommand
	})
	wasmRunCommand = func(parseCommand string, parseArgs []string, parseCwd string, parseEnv []string) (string, error) {
		if len(parseArgs) == 0 {
			parseT.Fatalf("expected args for command %q", parseCommand)
		}
		switch parseCommand {
		case "go-baseline", "go-candidate":
			switch parseArgs[0] {
			case "build":
				parseOutputIndex := -1
				for parseIndex := 0; parseIndex < len(parseArgs)-1; parseIndex++ {
					if parseArgs[parseIndex] == "-o" {
						parseOutputIndex = parseIndex + 1
						break
					}
				}
				if parseOutputIndex < 0 {
					parseT.Fatalf("expected -o in build args: %#v", parseArgs)
				}
				parseWasmPath := parseArgs[parseOutputIndex]
				if filepath.Base(parseWasmPath) != "custom.wasm" {
					parseT.Fatalf("expected custom binary name, got path %q", parseWasmPath)
				}
				if parseErr := os.MkdirAll(filepath.Dir(parseWasmPath), 0755); parseErr != nil {
					parseT.Fatalf("mkdir wasm output dir: %v", parseErr)
				}
				if parseErr2 := os.WriteFile(parseWasmPath, []byte("same-wasm"), 0644); parseErr2 != nil {
					parseT.Fatalf("write wasm output: %v", parseErr2)
				}
				return "", nil
			case "version":
				return fmt.Sprintf("go version %s windows/amd64", parseCommand), nil
			default:
				parseT.Fatalf("unexpected toolchain command args: %#v", parseArgs)
				return "", nil
			}
		default:
			parseT.Fatalf("unexpected command %q", parseCommand)
			return "", nil
		}
	}

	parseStdout, parseRestoreStdout, parseErr3 := captureExamplesStdout()
	if parseErr3 != nil {
		parseT.Fatalf("capture stdout: %v", parseErr3)
	}
	defer parseRestoreStdout()

	if parseErr4 := (launcher{}).runWasm([]string{
		"compare-toolchain",
		"-package", "./examples/public/ui-render",
		"-baseline-go", "go-baseline",
		"-candidate-go", "go-candidate",
		"-binary-name", "custom.wasm",
		"-out-dir", parseOutDir,
		"-skip-compression",
		"-timing-regression-percent", "100000",
		"-json",
	}); parseErr4 != nil {
		parseT.Fatalf("run compare-toolchain with custom binary name: %v", parseErr4)
	}

	parseOutput, parseErr3 := parseStdout()
	if parseErr3 != nil {
		parseT.Fatalf("read stdout: %v", parseErr3)
	}
	var parseSummary wasmToolchainSummary
	if parseErr5 := json.Unmarshal([]byte(parseOutput), &parseSummary); parseErr5 != nil {
		parseT.Fatalf("decode toolchain summary: %v\n%s", parseErr5, parseOutput)
	}
	if parseSummary.RegressionExitCode != 0 || !parseSummary.OK {
		parseT.Fatalf("expected successful comparison summary, got %#v", parseSummary)
	}
	if _, parseErr6 := os.Stat(parseBaselineBinary); parseErr6 != nil {
		parseT.Fatalf("expected baseline custom artifact to exist: %v", parseErr6)
	}
	if _, parseErr7 := os.Stat(parseCandidateBinary); parseErr7 != nil {
		parseT.Fatalf("expected candidate custom artifact to exist: %v", parseErr7)
	}
}
