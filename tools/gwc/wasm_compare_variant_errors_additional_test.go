package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestExecuteWasmCompareCacheCoversSummaryFailureBranches verifies cache summary error reporting and write failures.
func TestExecuteWasmCompareCacheCoversSummaryFailureBranches(parseT *testing.T) {
	parseT.Run("returns summary plus error when one cache variant fails", func(parseT2 *testing.T) {
		parsePackageDir := buildWasmPackageFixture(parseT2)
		parseOutDir := parseT2.TempDir()

		parseOriginalRunCommand := wasmRunCommand
		parseT2.Cleanup(func() {
			wasmRunCommand = parseOriginalRunCommand
		})
		wasmRunCommand = func(parseCommand string, parseArgs []string, parseCwd string, parseEnv []string) (string, error) {
			if parseCommand != "go" || len(parseArgs) == 0 {
				parseT2.Fatalf("unexpected command %q args=%#v", parseCommand, parseArgs)
			}
			switch parseArgs[0] {
			case "env":
				if len(parseArgs) < 2 {
					parseT2.Fatalf("expected env key, got %#v", parseArgs)
				}
				switch parseArgs[1] {
				case "GOCACHE":
					return filepath.Join(parseOutDir, "default-gocache"), nil
				case "GOMODCACHE":
					return filepath.Join(parseOutDir, "default-gomodcache"), nil
				default:
					parseT2.Fatalf("unexpected env args %#v", parseArgs)
					return "", nil
				}
			case "mod":
				return "", errors.New("download failed")
			case "build":
				parseOutputIndex := -1
				for parseIndex := 0; parseIndex < len(parseArgs)-1; parseIndex++ {
					if parseArgs[parseIndex] == "-o" {
						parseOutputIndex = parseIndex + 1
						break
					}
				}
				if parseOutputIndex < 0 {
					parseT2.Fatalf("expected -o in build args: %#v", parseArgs)
				}
				parseWasmPath := parseArgs[parseOutputIndex]
				if parseErr := os.MkdirAll(filepath.Dir(parseWasmPath), 0755); parseErr != nil {
					parseT2.Fatalf("mkdir wasm output dir: %v", parseErr)
				}
				if parseErr2 := os.WriteFile(parseWasmPath, []byte("cache-variant"), 0644); parseErr2 != nil {
					parseT2.Fatalf("write wasm output: %v", parseErr2)
				}
				return "", nil
			case "version":
				return "go version go1.26.0 windows/amd64", nil
			default:
				parseT2.Fatalf("unexpected go args %#v", parseArgs)
				return "", nil
			}
		}

		parseSummary, parseErr := executeWasmCompareCache(wasmCacheConfig{
			packagePath: parsePackageDir,
			outDir:      parseOutDir,
			binaryName:  "app.wasm",
			summaryName: "summary.json",
		})
		if parseErr == nil || !strings.Contains(parseErr.Error(), "one or more cache variants failed to measure") {
			parseT2.Fatalf("expected cache variant summary failure, got summary=%#v err=%v", parseSummary, parseErr)
		}
		if parseSummary.OK {
			parseT2.Fatalf("expected non-ok cache summary, got %#v", parseSummary)
		}
		parseColdVariant := parseSummary.Variants["ci-style-cold"]
		if parseColdVariant.Status != "failed" || parseColdVariant.Error != "go mod download failed" {
			parseT2.Fatalf("expected ci-style-cold failure details, got %#v", parseColdVariant)
		}
		if _, parseErr2 := os.Stat(filepath.Join(parseOutDir, "summary.json")); parseErr2 != nil {
			parseT2.Fatalf("expected cache summary artifact: %v", parseErr2)
		}
	})

	parseT.Run("reports cache summary write failures", func(parseT2 *testing.T) {
		parsePackageDir := buildWasmPackageFixture(parseT2)
		parseOutDir := parseT2.TempDir()
		parseSummaryPath := filepath.Join(parseOutDir, "summary.json")
		if parseErr := os.MkdirAll(parseSummaryPath, 0755); parseErr != nil {
			parseT2.Fatalf("mkdir summary collision path: %v", parseErr)
		}

		parseOriginalRunCommand := wasmRunCommand
		parseT2.Cleanup(func() {
			wasmRunCommand = parseOriginalRunCommand
		})
		wasmRunCommand = func(parseCommand string, parseArgs []string, parseCwd string, parseEnv []string) (string, error) {
			if parseCommand != "go" || len(parseArgs) == 0 {
				parseT2.Fatalf("unexpected command %q args=%#v", parseCommand, parseArgs)
			}
			switch parseArgs[0] {
			case "env":
				if parseArgs[1] == "GOCACHE" {
					return filepath.Join(parseOutDir, "default-gocache"), nil
				}
				if parseArgs[1] == "GOMODCACHE" {
					return filepath.Join(parseOutDir, "default-gomodcache"), nil
				}
				parseT2.Fatalf("unexpected env args %#v", parseArgs)
				return "", nil
			case "mod":
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
					parseT2.Fatalf("expected -o in build args: %#v", parseArgs)
				}
				parseWasmPath := parseArgs[parseOutputIndex]
				if parseErr2 := os.MkdirAll(filepath.Dir(parseWasmPath), 0755); parseErr2 != nil {
					parseT2.Fatalf("mkdir wasm output dir: %v", parseErr2)
				}
				if parseErr3 := os.WriteFile(parseWasmPath, []byte("cache-variant"), 0644); parseErr3 != nil {
					parseT2.Fatalf("write wasm output: %v", parseErr3)
				}
				return "", nil
			case "version":
				return "go version go1.26.0 windows/amd64", nil
			default:
				parseT2.Fatalf("unexpected go args %#v", parseArgs)
				return "", nil
			}
		}

		_, parseErr := executeWasmCompareCache(wasmCacheConfig{
			packagePath: parsePackageDir,
			outDir:      parseOutDir,
			binaryName:  "app.wasm",
			summaryName: "summary.json",
		})
		if parseErr == nil || !strings.Contains(parseErr.Error(), "write wasm cache summary") {
			parseT2.Fatalf("expected cache summary write failure, got %v", parseErr)
		}
	})
}

// TestExecuteWasmCompareToolchainCoversErrorBranches verifies toolchain version and summary write failures.
func TestExecuteWasmCompareToolchainCoversErrorBranches(parseT *testing.T) {
	parseT.Run("returns candidate version failure after compare artifacts are produced", func(parseT2 *testing.T) {
		parseOutDir := parseT2.TempDir()

		parseOriginalRunCommand := wasmRunCommand
		parseT2.Cleanup(func() {
			wasmRunCommand = parseOriginalRunCommand
		})
		wasmRunCommand = func(parseCommand string, parseArgs []string, parseCwd string, parseEnv []string) (string, error) {
			if len(parseArgs) == 0 {
				parseT2.Fatalf("expected args for %q", parseCommand)
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
						parseT2.Fatalf("expected -o in build args: %#v", parseArgs)
					}
					parseWasmPath := parseArgs[parseOutputIndex]
					if parseErr := os.MkdirAll(filepath.Dir(parseWasmPath), 0755); parseErr != nil {
						parseT2.Fatalf("mkdir wasm output dir: %v", parseErr)
					}
					if parseErr2 := os.WriteFile(parseWasmPath, []byte("same-wasm"), 0644); parseErr2 != nil {
						parseT2.Fatalf("write wasm output: %v", parseErr2)
					}
					return "", nil
				case "version":
					if parseCommand == "go-candidate" {
						return "", errors.New("candidate version failed")
					}
					return "go version go-baseline windows/amd64", nil
				default:
					parseT2.Fatalf("unexpected args %#v", parseArgs)
					return "", nil
				}
			default:
				parseT2.Fatalf("unexpected command %q", parseCommand)
				return "", nil
			}
		}

		_, parseErr := executeWasmCompareToolchain(wasmToolchainConfig{
			packagePath:           "./examples/public/ui-render",
			baselineGoExecutable:  "go-baseline",
			candidateGoExecutable: "go-candidate",
			binaryName:            "app.wasm",
			outDir:                parseOutDir,
			skipCompression:       true,
		})
		if parseErr == nil || !strings.Contains(parseErr.Error(), "candidate version failed") {
			parseT2.Fatalf("expected candidate version failure, got %v", parseErr)
		}
	})

	parseT.Run("reports summary write failure", func(parseT2 *testing.T) {
		parseOutDir := parseT2.TempDir()
		parseSummaryPath := filepath.Join(parseOutDir, "wasm-toolchain-comparison.json")
		if parseErr := os.MkdirAll(parseSummaryPath, 0755); parseErr != nil {
			parseT2.Fatalf("mkdir summary collision path: %v", parseErr)
		}

		parseOriginalRunCommand := wasmRunCommand
		parseT2.Cleanup(func() {
			wasmRunCommand = parseOriginalRunCommand
		})
		wasmRunCommand = func(parseCommand string, parseArgs []string, parseCwd string, parseEnv []string) (string, error) {
			if len(parseArgs) == 0 {
				parseT2.Fatalf("expected args for %q", parseCommand)
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
						parseT2.Fatalf("expected -o in build args: %#v", parseArgs)
					}
					parseWasmPath := parseArgs[parseOutputIndex]
					if parseErr2 := os.MkdirAll(filepath.Dir(parseWasmPath), 0755); parseErr2 != nil {
						parseT2.Fatalf("mkdir wasm output dir: %v", parseErr2)
					}
					if parseErr3 := os.WriteFile(parseWasmPath, []byte("same-wasm"), 0644); parseErr3 != nil {
						parseT2.Fatalf("write wasm output: %v", parseErr3)
					}
					return "", nil
				case "version":
					return "go version stable windows/amd64", nil
				default:
					parseT2.Fatalf("unexpected args %#v", parseArgs)
					return "", nil
				}
			default:
				parseT2.Fatalf("unexpected command %q", parseCommand)
				return "", nil
			}
		}

		_, parseErr := executeWasmCompareToolchain(wasmToolchainConfig{
			packagePath:             "./examples/public/ui-render",
			baselineGoExecutable:    "go-baseline",
			candidateGoExecutable:   "go-candidate",
			binaryName:              "app.wasm",
			outDir:                  parseOutDir,
			timingRegressionPercent: 100000,
			skipCompression:         true,
		})
		if parseErr == nil || !strings.Contains(parseErr.Error(), "write wasm toolchain summary") {
			parseT2.Fatalf("expected toolchain summary write failure, got %v", parseErr)
		}
	})
}

// buildWasmPackageFixture creates a minimal Go package directory for wasm cache tests that apply temporary edits.
func buildWasmPackageFixture(parseT *testing.T) string {
	parseT.Helper()

	parsePackageDir := parseT.TempDir()
	parseSourcePath := filepath.Join(parsePackageDir, "fixture.go")
	if parseErr := os.WriteFile(parseSourcePath, []byte("package fixture\n\nfunc Value() int {\n\treturn 1\n}\n"), 0644); parseErr != nil {
		parseT.Fatalf("write package fixture: %v", parseErr)
	}
	return parsePackageDir
}
