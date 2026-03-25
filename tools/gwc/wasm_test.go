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

func TestRunWasmMeasureWritesManifestAndArtifacts(t *testing.T) {
	root := t.TempDir()
	outDir := filepath.Join(root, "out")

	originalRunCommand := wasmRunCommand
	t.Cleanup(func() {
		wasmRunCommand = originalRunCommand
	})
	wasmRunCommand = func(command string, args []string, cwd string, env []string) (string, error) {
		if command != "go" {
			t.Fatalf("expected go command, got %q", command)
		}
		if len(args) == 0 {
			t.Fatal("expected args for go command")
		}
		switch args[0] {
		case "build":
			outputIndex := -1
			for index := 0; index < len(args)-1; index++ {
				if args[index] == "-o" {
					outputIndex = index + 1
					break
				}
			}
			if outputIndex < 0 {
				t.Fatalf("expected -o in build args: %#v", args)
			}
			wasmPath := args[outputIndex]
			if err := os.MkdirAll(filepath.Dir(wasmPath), 0755); err != nil {
				t.Fatalf("mkdir wasm output dir: %v", err)
			}
			if err := os.WriteFile(wasmPath, []byte("wasm-binary"), 0644); err != nil {
				t.Fatalf("write wasm output: %v", err)
			}
			return "", nil
		case "version":
			return "go version go1.26.0 windows/amd64", nil
		default:
			t.Fatalf("unexpected go args: %#v", args)
			return "", nil
		}
	}

	stdout, restoreStdout, err := captureExamplesStdout()
	if err != nil {
		t.Fatalf("capture stdout: %v", err)
	}
	defer restoreStdout()

	err = (launcher{}).runWasm([]string{
		"measure",
		"-package", "./examples/21-ui-render",
		"-out-dir", outDir,
		"-binary-name", "test.wasm",
		"-manifest-name", "report.json",
		"-release-profile",
		"-json",
	})
	if err != nil {
		t.Fatalf("run wasm measure: %v", err)
	}

	output, err := stdout()
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	var summary wasmMeasureSummary
	if err := json.Unmarshal([]byte(output), &summary); err != nil {
		t.Fatalf("decode wasm summary: %v\n%s", err, output)
	}
	if !summary.OK {
		t.Fatalf("expected successful summary, got %#v", summary)
	}
	if summary.Manifest.Profile != "release" {
		t.Fatalf("expected release profile, got %#v", summary)
	}
	for _, key := range []string{"wasm", "gzip", "brotli"} {
		if _, ok := summary.Manifest.Artifacts[key]; !ok {
			t.Fatalf("expected artifact %q in summary: %#v", key, summary)
		}
	}
	if _, err := os.Stat(filepath.FromSlash(summary.ManifestPath)); err != nil {
		t.Fatalf("expected manifest path to exist: %v", err)
	}
}

func TestRunWasmRejectsUnknownSubcommand(t *testing.T) {
	err := (launcher{}).runWasm([]string{"mystery"})
	if err == nil || !strings.Contains(err.Error(), `unknown wasm subcommand "mystery"`) {
		t.Fatalf("expected unknown subcommand error, got %v", err)
	}
}

func TestRunWasmCompareDetectsRegressionBeyondThreshold(t *testing.T) {
	root := t.TempDir()
	baselinePath := filepath.Join(root, "baseline.json")
	candidatePath := filepath.Join(root, "candidate.json")
	if err := os.WriteFile(baselinePath, []byte(`{"phases":{"go_build_ms":100},"artifacts":{"wasm":{"bytes":100}}}`), 0644); err != nil {
		t.Fatalf("write baseline: %v", err)
	}
	if err := os.WriteFile(candidatePath, []byte(`{"phases":{"go_build_ms":130},"artifacts":{"wasm":{"bytes":100}}}`), 0644); err != nil {
		t.Fatalf("write candidate: %v", err)
	}

	stdout, restoreStdout, err := captureExamplesStdout()
	if err != nil {
		t.Fatalf("capture stdout: %v", err)
	}
	defer restoreStdout()

	err = (launcher{}).runWasm([]string{"compare", "-baseline", baselinePath, "-candidate", candidatePath, "-json"})
	if err == nil || !strings.Contains(err.Error(), "regressions") {
		t.Fatalf("expected regression error, got %v", err)
	}
	output, err := stdout()
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	var summary wasmCompareSummary
	if err := json.Unmarshal([]byte(output), &summary); err != nil {
		t.Fatalf("decode compare summary: %v\n%s", err, output)
	}
	if summary.OK {
		t.Fatalf("expected failed compare summary, got %#v", summary)
	}
	if summary.Counts.Regressed == 0 {
		t.Fatalf("expected regressed metrics, got %#v", summary)
	}
}

func TestRunWasmComparePassesWithinThreshold(t *testing.T) {
	root := t.TempDir()
	baselinePath := filepath.Join(root, "baseline.json")
	candidatePath := filepath.Join(root, "candidate.json")
	if err := os.WriteFile(baselinePath, []byte(`{"phases":{"go_build_ms":100}}`), 0644); err != nil {
		t.Fatalf("write baseline: %v", err)
	}
	if err := os.WriteFile(candidatePath, []byte(`{"phases":{"go_build_ms":105}}`), 0644); err != nil {
		t.Fatalf("write candidate: %v", err)
	}

	stdout, restoreStdout, err := captureExamplesStdout()
	if err != nil {
		t.Fatalf("capture stdout: %v", err)
	}
	defer restoreStdout()

	if err := (launcher{}).runWasm([]string{"compare", "-baseline", baselinePath, "-candidate", candidatePath, "-timing-regression-percent", "10", "-json"}); err != nil {
		t.Fatalf("compare within threshold should pass: %v", err)
	}
	output, err := stdout()
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	var summary wasmCompareSummary
	if err := json.Unmarshal([]byte(output), &summary); err != nil {
		t.Fatalf("decode compare summary: %v\n%s", err, output)
	}
	if !summary.OK || summary.Counts.Regressed != 0 {
		t.Fatalf("expected passing summary, got %#v", summary)
	}
}

func TestRunWasmCompareCompressionWritesSummaryWithoutOptimizer(t *testing.T) {
	root := t.TempDir()
	outDir := filepath.Join(root, "comparison")

	originalRunCommand := wasmRunCommand
	originalLookPath := wasmLookPath
	t.Cleanup(func() {
		wasmRunCommand = originalRunCommand
		wasmLookPath = originalLookPath
	})
	wasmLookPath = func(file string) (string, error) {
		return "", errors.New("not found")
	}
	wasmRunCommand = func(command string, args []string, cwd string, env []string) (string, error) {
		if command != "go" {
			t.Fatalf("expected go command, got %q", command)
		}
		if len(args) == 0 {
			t.Fatal("expected args for go command")
		}
		switch args[0] {
		case "build":
			outputIndex := -1
			for index := 0; index < len(args)-1; index++ {
				if args[index] == "-o" {
					outputIndex = index + 1
					break
				}
			}
			if outputIndex < 0 {
				t.Fatalf("expected -o in build args: %#v", args)
			}
			wasmPath := args[outputIndex]
			if err := os.MkdirAll(filepath.Dir(wasmPath), 0755); err != nil {
				t.Fatalf("mkdir wasm output dir: %v", err)
			}
			if err := os.WriteFile(wasmPath, []byte("wasm-binary"), 0644); err != nil {
				t.Fatalf("write wasm output: %v", err)
			}
			return "", nil
		case "version":
			return "go version go1.26.0 windows/amd64", nil
		default:
			t.Fatalf("unexpected command args: %#v", args)
			return "", nil
		}
	}

	stdout, restoreStdout, err := captureExamplesStdout()
	if err != nil {
		t.Fatalf("capture stdout: %v", err)
	}
	defer restoreStdout()

	if err := (launcher{}).runWasm([]string{
		"compare-compression",
		"-package", "./examples/21-ui-render",
		"-out-dir", outDir,
		"-summary-name", "summary.json",
		"-json",
	}); err != nil {
		t.Fatalf("run compare-compression: %v", err)
	}

	output, err := stdout()
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	var summary wasmCompressionSummary
	if err := json.Unmarshal([]byte(output), &summary); err != nil {
		t.Fatalf("decode compression summary: %v\n%s", err, output)
	}
	if !summary.OK {
		t.Fatalf("expected ok summary, got %#v", summary)
	}
	if summary.Environment.WasmOptAvailable {
		t.Fatalf("expected wasm-opt unavailable summary, got %#v", summary.Environment)
	}
	for _, key := range []string{"plain_raw", "stripped_raw", "stripped_compressed"} {
		if _, ok := summary.Variants[key]; !ok {
			t.Fatalf("expected variant %q in summary", key)
		}
	}
	if _, ok := summary.Variants["optimized_raw"]; ok {
		t.Fatalf("did not expect optimized variant when optimizer is unavailable")
	}
	if _, err := os.Stat(filepath.FromSlash(summary.SummaryPath)); err != nil {
		t.Fatalf("expected summary file to exist: %v", err)
	}
}

func TestRunWasmCompareCompressionIncludesOptimizedVariants(t *testing.T) {
	root := t.TempDir()
	outDir := filepath.Join(root, "comparison")

	originalRunCommand := wasmRunCommand
	originalLookPath := wasmLookPath
	originalArtifactRecord := wasmReleaseArtifactRecordForPath
	t.Cleanup(func() {
		wasmRunCommand = originalRunCommand
		wasmLookPath = originalLookPath
		wasmReleaseArtifactRecordForPath = originalArtifactRecord
	})
	wasmReleaseArtifactRecordForPath = releaseArtifactRecordForPath
	wasmLookPath = func(file string) (string, error) {
		if file == "wasm-opt" {
			return filepath.Join(root, "fake-wasm-opt"), nil
		}
		return "", errors.New("not found")
	}
	wasmRunCommand = func(command string, args []string, cwd string, env []string) (string, error) {
		if len(args) == 0 {
			t.Fatalf("expected args for command %q", command)
		}
		switch command {
		case "go":
			switch args[0] {
			case "build":
				outputIndex := -1
				for index := 0; index < len(args)-1; index++ {
					if args[index] == "-o" {
						outputIndex = index + 1
						break
					}
				}
				if outputIndex < 0 {
					t.Fatalf("expected -o in build args: %#v", args)
				}
				wasmPath := args[outputIndex]
				if err := os.MkdirAll(filepath.Dir(wasmPath), 0755); err != nil {
					t.Fatalf("mkdir wasm output dir: %v", err)
				}
				if err := os.WriteFile(wasmPath, []byte("wasm-binary"), 0644); err != nil {
					t.Fatalf("write wasm output: %v", err)
				}
				return "", nil
			case "version":
				return "go version go1.26.0 windows/amd64", nil
			default:
				t.Fatalf("unexpected go command args: %#v", args)
				return "", nil
			}
		case "wasm-opt":
			if len(args) < 4 {
				t.Fatalf("expected wasm-opt args, got %#v", args)
			}
			sourcePath := args[0]
			targetPath := args[len(args)-1]
			optimizedBytes, err := os.ReadFile(sourcePath)
			if err != nil {
				t.Fatalf("read source wasm: %v", err)
			}
			optimizedBytes = append(optimizedBytes, []byte("-optimized")...)
			if err := os.WriteFile(targetPath, optimizedBytes, 0644); err != nil {
				t.Fatalf("write optimized wasm: %v", err)
			}
			return "", nil
		default:
			t.Fatalf("unexpected command %q", command)
			return "", nil
		}
	}

	stdout, restoreStdout, err := captureExamplesStdout()
	if err != nil {
		t.Fatalf("capture stdout: %v", err)
	}
	defer restoreStdout()

	if err := (launcher{}).runWasm([]string{
		"compare-compression",
		"-package", "./examples/21-ui-render",
		"-out-dir", outDir,
		"-summary-name", "summary.json",
		"-json",
	}); err != nil {
		t.Fatalf("run compare-compression: %v", err)
	}

	output, err := stdout()
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	var summary wasmCompressionSummary
	if err := json.Unmarshal([]byte(output), &summary); err != nil {
		t.Fatalf("decode compression summary: %v\n%s", err, output)
	}
	if !summary.Environment.WasmOptAvailable {
		t.Fatalf("expected wasm-opt available summary, got %#v", summary.Environment)
	}
	for _, key := range []string{"optimized_raw", "optimized_compressed"} {
		if _, ok := summary.Variants[key]; !ok {
			t.Fatalf("expected optimized variant %q in summary", key)
		}
	}
	optimizedCompressed, ok := summary.Variants["optimized_compressed"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected optimized compressed variant payload, got %#v", summary.Variants["optimized_compressed"])
	}
	parityChecks, ok := optimizedCompressed["parity_checks"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected parity checks in optimized compressed payload, got %#v", optimizedCompressed)
	}
	rawToDelivery, ok := parityChecks["raw_to_delivery_copy"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected raw_to_delivery_copy parity entry, got %#v", parityChecks)
	}
	okValue, ok := rawToDelivery["ok"].(bool)
	if !ok || !okValue {
		t.Fatalf("expected successful parity check status, got %#v", rawToDelivery)
	}
}

func TestRunWasmCompareCompressionFailsOnParityMismatch(t *testing.T) {
	root := t.TempDir()
	outDir := filepath.Join(root, "comparison")

	originalRunCommand := wasmRunCommand
	originalLookPath := wasmLookPath
	originalArtifactRecord := wasmReleaseArtifactRecordForPath
	t.Cleanup(func() {
		wasmRunCommand = originalRunCommand
		wasmLookPath = originalLookPath
		wasmReleaseArtifactRecordForPath = originalArtifactRecord
	})
	wasmReleaseArtifactRecordForPath = func(baseDir string, artifactPath string) (releaseArtifactRecord, error) {
		record, err := releaseArtifactRecordForPath(baseDir, artifactPath)
		if err != nil {
			return releaseArtifactRecord{}, err
		}
		if strings.Contains(filepath.ToSlash(baseDir), "optimized-compressed") && strings.HasSuffix(filepath.ToSlash(artifactPath), ".wasm") {
			record.SHA256 = "parity-mismatch"
		}
		return record, nil
	}
	wasmLookPath = func(file string) (string, error) {
		if file == "wasm-opt" {
			return filepath.Join(root, "fake-wasm-opt"), nil
		}
		return "", errors.New("not found")
	}
	wasmRunCommand = func(command string, args []string, cwd string, env []string) (string, error) {
		if len(args) == 0 {
			t.Fatalf("expected args for command %q", command)
		}
		switch command {
		case "go":
			switch args[0] {
			case "build":
				outputIndex := -1
				for index := 0; index < len(args)-1; index++ {
					if args[index] == "-o" {
						outputIndex = index + 1
						break
					}
				}
				if outputIndex < 0 {
					t.Fatalf("expected -o in build args: %#v", args)
				}
				wasmPath := args[outputIndex]
				if err := os.MkdirAll(filepath.Dir(wasmPath), 0755); err != nil {
					t.Fatalf("mkdir wasm output dir: %v", err)
				}
				if err := os.WriteFile(wasmPath, []byte("wasm-binary"), 0644); err != nil {
					t.Fatalf("write wasm output: %v", err)
				}
				return "", nil
			case "version":
				return "go version go1.26.0 windows/amd64", nil
			default:
				t.Fatalf("unexpected go command args: %#v", args)
				return "", nil
			}
		case "wasm-opt":
			sourcePath := args[0]
			targetPath := args[len(args)-1]
			optimizedBytes, err := os.ReadFile(sourcePath)
			if err != nil {
				t.Fatalf("read source wasm: %v", err)
			}
			if err := os.WriteFile(targetPath, append(optimizedBytes, []byte("-optimized")...), 0644); err != nil {
				t.Fatalf("write optimized wasm: %v", err)
			}
			return "", nil
		default:
			t.Fatalf("unexpected command %q", command)
			return "", nil
		}
	}

	err := (launcher{}).runWasm([]string{
		"compare-compression",
		"-package", "./examples/21-ui-render",
		"-out-dir", outDir,
		"-summary-name", "summary.json",
		"-json",
	})
	if err == nil || !strings.Contains(err.Error(), "parity mismatch") {
		t.Fatalf("expected parity mismatch error, got %v", err)
	}
}

func TestRunWasmCompareCacheWritesSummary(t *testing.T) {
	root := t.TempDir()
	outDir := filepath.Join(root, "cache")
	packageDir := filepath.Join(root, "pkg")
	if err := os.MkdirAll(packageDir, 0755); err != nil {
		t.Fatalf("mkdir package dir: %v", err)
	}
	sourcePath := filepath.Join(packageDir, "alpha.go")
	originalSource := "package pkg\n\nfunc Value() int { return 1 }\n"
	if err := os.WriteFile(sourcePath, []byte(originalSource), 0644); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	originalRunCommand := wasmRunCommand
	t.Cleanup(func() {
		wasmRunCommand = originalRunCommand
	})
	defaultGoCache := filepath.Join(root, "default-gocache")
	defaultGoModCache := filepath.Join(root, "default-gomodcache")
	modDownloadCalls := 0
	wasmRunCommand = func(command string, args []string, cwd string, env []string) (string, error) {
		if command != "go" {
			t.Fatalf("expected go command, got %q", command)
		}
		if len(args) == 0 {
			t.Fatal("expected args for go command")
		}
		switch args[0] {
		case "env":
			if len(args) < 2 {
				t.Fatalf("expected go env key in args: %#v", args)
			}
			switch args[1] {
			case "GOCACHE":
				return defaultGoCache, nil
			case "GOMODCACHE":
				return defaultGoModCache, nil
			default:
				t.Fatalf("unexpected go env key: %q", args[1])
				return "", nil
			}
		case "mod":
			if len(args) == 2 && args[1] == "download" {
				modDownloadCalls++
				return "", nil
			}
			t.Fatalf("unexpected go mod args: %#v", args)
			return "", nil
		case "build":
			outputIndex := -1
			for index := 0; index < len(args)-1; index++ {
				if args[index] == "-o" {
					outputIndex = index + 1
					break
				}
			}
			if outputIndex < 0 {
				t.Fatalf("expected -o in build args: %#v", args)
			}
			wasmPath := args[outputIndex]
			if err := os.MkdirAll(filepath.Dir(wasmPath), 0755); err != nil {
				t.Fatalf("mkdir wasm output dir: %v", err)
			}
			if err := os.WriteFile(wasmPath, []byte("cache-wasm"), 0644); err != nil {
				t.Fatalf("write wasm output: %v", err)
			}
			return "", nil
		case "version":
			return "go version go1.26.0 windows/amd64", nil
		default:
			t.Fatalf("unexpected command args: %#v", args)
			return "", nil
		}
	}

	stdout, restoreStdout, err := captureExamplesStdout()
	if err != nil {
		t.Fatalf("capture stdout: %v", err)
	}
	defer restoreStdout()

	if err := (launcher{}).runWasm([]string{
		"compare-cache",
		"-package", packageDir,
		"-out-dir", outDir,
		"-summary-name", "summary.json",
		"-json",
	}); err != nil {
		t.Fatalf("run compare-cache: %v", err)
	}

	output, err := stdout()
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	var summary wasmCacheSummary
	if err := json.Unmarshal([]byte(output), &summary); err != nil {
		t.Fatalf("decode cache summary: %v\n%s", err, output)
	}
	if !summary.OK {
		t.Fatalf("expected ok summary, got %#v", summary)
	}
	expectedVariants := []string{
		"shared-cache-cold",
		"shared-cache-warm",
		"shared-cache-small-edit",
		"isolated-build-cache",
		"ci-style-cold",
		"ci-style-warm",
		"ci-style-small-edit",
	}
	for _, key := range expectedVariants {
		variant, ok := summary.Variants[key]
		if !ok {
			t.Fatalf("expected variant %q in summary", key)
		}
		if variant.Status != "ok" {
			t.Fatalf("expected variant %q status ok, got %#v", key, variant)
		}
	}
	if summary.Variants["ci-style-cold"].ModuleDownloadMS == nil {
		t.Fatalf("expected ci-style-cold to include module download timing")
	}
	if strings.TrimSpace(summary.Variants["shared-cache-small-edit"].EditedFile) == "" {
		t.Fatalf("expected shared-cache-small-edit edited file to be set")
	}
	if strings.TrimSpace(summary.Variants["ci-style-small-edit"].EditedFile) == "" {
		t.Fatalf("expected ci-style-small-edit edited file to be set")
	}
	if modDownloadCalls != 1 {
		t.Fatalf("expected one go mod download call, got %d", modDownloadCalls)
	}
	if _, err := os.Stat(filepath.FromSlash(summary.SummaryPath)); err != nil {
		t.Fatalf("expected summary file to exist: %v", err)
	}
	finalSource, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("read source after run: %v", err)
	}
	if string(finalSource) != originalSource {
		t.Fatalf("expected source to be restored after small-edit variants")
	}
}

func TestRunWasmCompareToolchainDetectsRegression(t *testing.T) {
	root := t.TempDir()
	outDir := filepath.Join(root, "toolchain")

	originalRunCommand := wasmRunCommand
	t.Cleanup(func() {
		wasmRunCommand = originalRunCommand
	})
	wasmRunCommand = func(command string, args []string, cwd string, env []string) (string, error) {
		if len(args) == 0 {
			t.Fatalf("expected args for command %q", command)
		}
		switch command {
		case "go-baseline", "go-candidate":
			switch args[0] {
			case "build":
				outputIndex := -1
				for index := 0; index < len(args)-1; index++ {
					if args[index] == "-o" {
						outputIndex = index + 1
						break
					}
				}
				if outputIndex < 0 {
					t.Fatalf("expected -o in build args: %#v", args)
				}
				wasmPath := args[outputIndex]
				if err := os.MkdirAll(filepath.Dir(wasmPath), 0755); err != nil {
					t.Fatalf("mkdir wasm output dir: %v", err)
				}
				payload := []byte("baseline-wasm")
				if command == "go-candidate" {
					payload = append(payload, []byte("-regression")...)
				}
				if err := os.WriteFile(wasmPath, payload, 0644); err != nil {
					t.Fatalf("write wasm output: %v", err)
				}
				return "", nil
			case "version":
				return fmt.Sprintf("go version %s windows/amd64", command), nil
			default:
				t.Fatalf("unexpected toolchain command args: %#v", args)
				return "", nil
			}
		default:
			t.Fatalf("unexpected command %q", command)
			return "", nil
		}
	}

	stdout, restoreStdout, err := captureExamplesStdout()
	if err != nil {
		t.Fatalf("capture stdout: %v", err)
	}
	defer restoreStdout()

	err = (launcher{}).runWasm([]string{
		"compare-toolchain",
		"-package", "./examples/21-ui-render",
		"-baseline-go", "go-baseline",
		"-candidate-go", "go-candidate",
		"-out-dir", outDir,
		"-skip-compression",
		"-json",
	})
	if err == nil || !strings.Contains(err.Error(), "regressions") {
		t.Fatalf("expected regression error, got %v", err)
	}

	output, err := stdout()
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	var summary wasmToolchainSummary
	if err := json.Unmarshal([]byte(output), &summary); err != nil {
		t.Fatalf("decode toolchain summary: %v\n%s", err, output)
	}
	if summary.RegressionExitCode != 1 {
		t.Fatalf("expected regression exit code 1, got %#v", summary)
	}
	if summary.Baseline.GoExecutable != "go-baseline" {
		t.Fatalf("unexpected baseline executable: %#v", summary.Baseline)
	}
	if summary.Candidate.GoExecutable != "go-candidate" {
		t.Fatalf("unexpected candidate executable: %#v", summary.Candidate)
	}
	if _, err := os.Stat(filepath.FromSlash(summary.SummaryPath)); err != nil {
		t.Fatalf("expected summary file to exist: %v", err)
	}
	comparisonPath := filepath.Join(filepath.FromSlash(outDir), "toolchain-comparison.json")
	if _, err := os.Stat(comparisonPath); err != nil {
		t.Fatalf("expected comparison file to exist: %v", err)
	}
}

func TestRunWasmCompareToolchainUsesCustomBinaryName(t *testing.T) {
	root := t.TempDir()
	outDir := filepath.Join(root, "toolchain")
	baselineBinary := filepath.Join(outDir, "baseline", "custom.wasm")
	candidateBinary := filepath.Join(outDir, "candidate", "custom.wasm")

	originalRunCommand := wasmRunCommand
	t.Cleanup(func() {
		wasmRunCommand = originalRunCommand
	})
	wasmRunCommand = func(command string, args []string, cwd string, env []string) (string, error) {
		if len(args) == 0 {
			t.Fatalf("expected args for command %q", command)
		}
		switch command {
		case "go-baseline", "go-candidate":
			switch args[0] {
			case "build":
				outputIndex := -1
				for index := 0; index < len(args)-1; index++ {
					if args[index] == "-o" {
						outputIndex = index + 1
						break
					}
				}
				if outputIndex < 0 {
					t.Fatalf("expected -o in build args: %#v", args)
				}
				wasmPath := args[outputIndex]
				if filepath.Base(wasmPath) != "custom.wasm" {
					t.Fatalf("expected custom binary name, got path %q", wasmPath)
				}
				if err := os.MkdirAll(filepath.Dir(wasmPath), 0755); err != nil {
					t.Fatalf("mkdir wasm output dir: %v", err)
				}
				if err := os.WriteFile(wasmPath, []byte("same-wasm"), 0644); err != nil {
					t.Fatalf("write wasm output: %v", err)
				}
				return "", nil
			case "version":
				return fmt.Sprintf("go version %s windows/amd64", command), nil
			default:
				t.Fatalf("unexpected toolchain command args: %#v", args)
				return "", nil
			}
		default:
			t.Fatalf("unexpected command %q", command)
			return "", nil
		}
	}

	stdout, restoreStdout, err := captureExamplesStdout()
	if err != nil {
		t.Fatalf("capture stdout: %v", err)
	}
	defer restoreStdout()

	if err := (launcher{}).runWasm([]string{
		"compare-toolchain",
		"-package", "./examples/21-ui-render",
		"-baseline-go", "go-baseline",
		"-candidate-go", "go-candidate",
		"-binary-name", "custom.wasm",
		"-out-dir", outDir,
		"-skip-compression",
		"-timing-regression-percent", "100000",
		"-json",
	}); err != nil {
		t.Fatalf("run compare-toolchain with custom binary name: %v", err)
	}

	output, err := stdout()
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	var summary wasmToolchainSummary
	if err := json.Unmarshal([]byte(output), &summary); err != nil {
		t.Fatalf("decode toolchain summary: %v\n%s", err, output)
	}
	if summary.RegressionExitCode != 0 || !summary.OK {
		t.Fatalf("expected successful comparison summary, got %#v", summary)
	}
	if _, err := os.Stat(baselineBinary); err != nil {
		t.Fatalf("expected baseline custom artifact to exist: %v", err)
	}
	if _, err := os.Stat(candidateBinary); err != nil {
		t.Fatalf("expected candidate custom artifact to exist: %v", err)
	}
}
