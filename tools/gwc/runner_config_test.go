package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLoadLauncherOverridesFindsParentConfig(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseNested := filepath.Join(parseRoot, "apps", "sample")
	if parseErr := os.MkdirAll(parseNested, 0755); parseErr != nil {
		parseT.Fatalf("mkdir nested dir: %v", parseErr)
	}
	parseConfigPath := filepath.Join(parseRoot, "gwc-runner.json")
	if parseErr2 := os.WriteFile(parseConfigPath, []byte(`{"paths":{"generatedProjectRoot":"company-projects"}}`), 0644); parseErr2 != nil {
		parseT.Fatalf("write config: %v", parseErr2)
	}

	parseOverrides, parseFoundPath, parseOk, parseErr3 := loadLauncherOverrides(parseNested)
	if parseErr3 != nil {
		parseT.Fatalf("load launcher overrides: %v", parseErr3)
	}
	if !parseOk {
		parseT.Fatal("expected launcher overrides to be discovered from a parent directory")
	}
	if parseFoundPath != parseConfigPath {
		parseT.Fatalf("expected config path %q, got %q", parseConfigPath, parseFoundPath)
	}
	if parseOverrides.Paths.GeneratedProjectRoot != "company-projects" {
		parseT.Fatalf("expected generated project root override, got %#v", parseOverrides)
	}
}

func TestCanonicalRunnerConfigExampleMatchesLauncherSchema(parseT *testing.T) {
	parseExamplePath := filepath.Join("..", "..", "docs", "examples", "gwc-runner.example.json")
	parseContent, parseErr := os.ReadFile(parseExamplePath)
	if parseErr != nil {
		parseT.Fatalf("read canonical runner config example: %v", parseErr)
	}

	var parseOverrides launcherOverrides
	if parseErr2 := json.Unmarshal(parseContent, &parseOverrides); parseErr2 != nil {
		parseT.Fatalf("parse canonical runner config example: %v", parseErr2)
	}

	if parseOverrides.Paths.GeneratedProjectRoot != "generated-projects" {
		parseT.Fatalf("expected generatedProjectRoot in canonical example, got %#v", parseOverrides.Paths)
	}
	if parseOverrides.Paths.ArtifactRoot != "enterprise-artifacts" {
		parseT.Fatalf("expected artifactRoot in canonical example, got %#v", parseOverrides.Paths)
	}
	if parseOverrides.Paths.WASMExecJS != "vendor/wasm_exec.js" {
		parseT.Fatalf("expected wasmExecJS in canonical example, got %#v", parseOverrides.Paths)
	}
	if parseOverrides.Paths.GoWASMExec != "tools/go_js_wasm_exec.bat" {
		parseT.Fatalf("expected goWasmExec in canonical example, got %#v", parseOverrides.Paths)
	}
	if parseOverrides.Paths.BrowserWorkspace != "test" {
		parseT.Fatalf("expected browserWorkspace in canonical example, got %#v", parseOverrides.Paths)
	}
	if parseOverrides.Paths.LivereloadWorkspace != "tools/livereload" {
		parseT.Fatalf("expected livereloadWorkspace in canonical example, got %#v", parseOverrides.Paths)
	}
	if parseOverrides.Paths.LivereloadClientScript != "" {
		parseT.Fatalf("expected canonical example to omit livereloadClientScript, got %#v", parseOverrides.Paths)
	}
	if len(parseOverrides.Paths.GeneratedProjectRoot) == 0 {
		parseT.Fatalf("expected canonical example to stay non-empty, got %#v", parseOverrides.Paths)
	}
	if len(parseOverrides.Paths.ArtifactRoot) == 0 {
		parseT.Fatalf("expected canonical example to stay non-empty, got %#v", parseOverrides.Paths)
	}
	if len(parseOverrides.Paths.WASMExecJS) == 0 || len(parseOverrides.Paths.GoWASMExec) == 0 {
		parseT.Fatalf("expected canonical example exec fields to stay non-empty, got %#v", parseOverrides.Paths)
	}
}

func TestLoadLauncherOverridesPrefersExplicitEnvPath(parseT *testing.T) {
	parseConfigPath := filepath.Join(parseT.TempDir(), "enterprise-runner.json")
	if parseErr := os.WriteFile(parseConfigPath, []byte(`{"paths":{"browserWorkspace":"browser-suite"}}`), 0644); parseErr != nil {
		parseT.Fatalf("write explicit env config: %v", parseErr)
	}
	parseT.Setenv(launcherOverrideEnvVar, parseConfigPath)

	parseOverrides, parseFoundPath, parseOk, parseErr2 := loadLauncherOverrides("")
	if parseErr2 != nil {
		parseT.Fatalf("load launcher overrides: %v", parseErr2)
	}
	if !parseOk {
		parseT.Fatal("expected launcher overrides from explicit env path")
	}
	if parseFoundPath != parseConfigPath {
		parseT.Fatalf("expected explicit env config path %q, got %q", parseConfigPath, parseFoundPath)
	}
	if parseOverrides.Paths.BrowserWorkspace != "browser-suite" {
		parseT.Fatalf("expected browser workspace override, got %#v", parseOverrides)
	}
}

func TestDefaultGeneratedScaffoldRootUsesOverride(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseConfigPath := filepath.Join(parseRoot, "gwc-runner.json")
	if parseErr := os.WriteFile(parseConfigPath, []byte(`{"paths":{"generatedProjectRoot":"custom-projects"}}`), 0644); parseErr != nil {
		parseT.Fatalf("write config: %v", parseErr)
	}

	parseOriginalGetwd := launcherConfigGetwd
	parseOriginalHomeDir := startUserHomeDir
	parseT.Cleanup(func() {
		launcherConfigGetwd = parseOriginalGetwd
		startUserHomeDir = parseOriginalHomeDir
	})
	launcherConfigGetwd = func() (string, error) { return parseRoot, nil }
	startUserHomeDir = func() (string, error) { return filepath.Join(parseRoot, "home"), nil }

	parseGot := defaultGeneratedScaffoldRoot()
	parseWant := filepath.Join(parseRoot, "custom-projects")
	if parseGot != parseWant {
		parseT.Fatalf("expected override scaffold root %q, got %q", parseWant, parseGot)
	}
}

func TestResolveLauncherArtifactPathUsesOverride(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	if parseErr := os.WriteFile(filepath.Join(parseRoot, "gwc-runner.json"), []byte(`{"paths":{"artifactRoot":"enterprise-artifacts"}}`), 0644); parseErr != nil {
		parseT.Fatalf("write config: %v", parseErr)
	}

	parseGot, parseOk, parseErr2 := resolveLauncherArtifactPath(parseRoot, "wasm-release")
	if parseErr2 != nil {
		parseT.Fatalf("resolve launcher artifact path: %v", parseErr2)
	}
	if !parseOk {
		parseT.Fatal("expected artifact root override to be discovered")
	}
	parseWant := filepath.Join(parseRoot, "enterprise-artifacts", filepath.Base(parseRoot), "wasm-release")
	if parseGot != parseWant {
		parseT.Fatalf("expected artifact path %q, got %q", parseWant, parseGot)
	}
}

func TestResolveLauncherDefaultBuildOutputUsesSharedResolver(parseT *testing.T) {
	parseT.Run("artifact root override", func(parseT2 *testing.T) {
		parseRoot := parseT2.TempDir()
		if parseErr := os.WriteFile(filepath.Join(parseRoot, "gwc-runner.json"), []byte(`{"paths":{"artifactRoot":"enterprise-artifacts"}}`), 0644); parseErr != nil {
			parseT2.Fatalf("write config: %v", parseErr)
		}

		parseGot, parseSource, parseErr2 := resolveLauncherDefaultBuildOutput(parseRoot)
		if parseErr2 != nil {
			parseT2.Fatalf("resolve default build output: %v", parseErr2)
		}
		parseWant := filepath.Join(parseRoot, "enterprise-artifacts", filepath.Base(parseRoot), "bin", "main.wasm")
		if parseGot != parseWant || parseSource != "gwc-runner.json paths.artifactRoot" {
			parseT2.Fatalf("expected shared build output resolver to return %q via gwc-runner.json paths.artifactRoot, got path=%q source=%q", parseWant, parseGot, parseSource)
		}
	})

	parseT.Run("convention fallback", func(parseT3 *testing.T) {
		parseRoot2 := parseT3.TempDir()
		parseGot2, parseSource2, parseErr3 := resolveLauncherDefaultBuildOutput(parseRoot2)
		if parseErr3 != nil {
			parseT3.Fatalf("resolve default build output: %v", parseErr3)
		}
		parseWant2 := filepath.Join(parseRoot2, "bin", "main.wasm")
		if parseGot2 != parseWant2 || parseSource2 != "convention fallback" {
			parseT3.Fatalf("expected convention fallback build output %q, got path=%q source=%q", parseWant2, parseGot2, parseSource2)
		}
	})
}

func TestResolveLauncherDefaultReleaseOutDirUsesSharedResolver(parseT *testing.T) {
	parseT.Run("artifact root override", func(parseT2 *testing.T) {
		parseRoot := parseT2.TempDir()
		if parseErr := os.WriteFile(filepath.Join(parseRoot, "gwc-runner.json"), []byte(`{"paths":{"artifactRoot":"enterprise-artifacts"}}`), 0644); parseErr != nil {
			parseT2.Fatalf("write config: %v", parseErr)
		}

		parseGot, parseSource, parseErr2 := resolveLauncherDefaultReleaseOutDir(parseRoot)
		if parseErr2 != nil {
			parseT2.Fatalf("resolve default release out dir: %v", parseErr2)
		}
		parseWant := filepath.Join(parseRoot, "enterprise-artifacts", filepath.Base(parseRoot), "wasm-release")
		if parseGot != parseWant || parseSource != "gwc-runner.json paths.artifactRoot" {
			parseT2.Fatalf("expected shared release output resolver to return %q via gwc-runner.json paths.artifactRoot, got path=%q source=%q", parseWant, parseGot, parseSource)
		}
	})

	parseT.Run("convention fallback", func(parseT3 *testing.T) {
		parseRoot2 := parseT3.TempDir()
		parseGot2, parseSource2, parseErr3 := resolveLauncherDefaultReleaseOutDir(parseRoot2)
		if parseErr3 != nil {
			parseT3.Fatalf("resolve default release out dir: %v", parseErr3)
		}
		parseWant2 := filepath.Join(parseRoot2, "bin", "wasm-release")
		if parseGot2 != parseWant2 || parseSource2 != "convention fallback" {
			parseT3.Fatalf("expected convention fallback release out dir %q, got path=%q source=%q", parseWant2, parseGot2, parseSource2)
		}
	})
}

func TestResolveLauncherExamplesWasmDirUsesSharedResolver(parseT *testing.T) {
	parseRepoRoot := parseT.TempDir()
	parseExamplesBuildDir := filepath.Join(parseRepoRoot, "bin", "examples")
	if parseErr := os.MkdirAll(parseExamplesBuildDir, 0755); parseErr != nil {
		parseT.Fatalf("mkdir examples build dir: %v", parseErr)
	}

	parseGot := resolveLauncherExamplesWasmDir(parseRepoRoot, filepath.Join(parseRepoRoot, "examples", "static"))
	if parseGot != parseExamplesBuildDir {
		parseT.Fatalf("expected shared examples wasm dir resolver to return %q, got %q", parseExamplesBuildDir, parseGot)
	}

	parseStaticDir := filepath.Join(parseT.TempDir(), "static")
	if parseErr2 := os.MkdirAll(parseStaticDir, 0755); parseErr2 != nil {
		parseT.Fatalf("mkdir static dir: %v", parseErr2)
	}
	parseGot = resolveLauncherExamplesWasmDir(parseT.TempDir(), parseStaticDir)
	if parseGot != filepath.Join(parseStaticDir, "bin") {
		parseT.Fatalf("expected static-dir fallback examples wasm dir, got %q", parseGot)
	}
}

func TestResolveLauncherTempRootUsesSharedResolver(parseT *testing.T) {
	parseT.Run("artifact root override", func(parseT2 *testing.T) {
		parseRoot := parseT2.TempDir()
		if parseErr := os.WriteFile(filepath.Join(parseRoot, "gwc-runner.json"), []byte(`{"paths":{"artifactRoot":"enterprise-artifacts"}}`), 0644); parseErr != nil {
			parseT2.Fatalf("write config: %v", parseErr)
		}

		parseGot, parseSource, parseErr2 := resolveLauncherTempRoot(parseRoot)
		if parseErr2 != nil {
			parseT2.Fatalf("resolve launcher temp root: %v", parseErr2)
		}
		parseWant := filepath.Join(parseRoot, "enterprise-artifacts", filepath.Base(parseRoot), "tmp")
		if parseGot != parseWant || parseSource != "gwc-runner.json paths.artifactRoot" {
			parseT2.Fatalf("expected shared temp root %q via gwc-runner.json paths.artifactRoot, got path=%q source=%q", parseWant, parseGot, parseSource)
		}
	})

	parseT.Run("convention fallback", func(parseT3 *testing.T) {
		parseRoot2 := parseT3.TempDir()
		parseGot2, parseSource2, parseErr3 := resolveLauncherTempRoot(parseRoot2)
		if parseErr3 != nil {
			parseT3.Fatalf("resolve launcher temp root: %v", parseErr3)
		}
		parseWant2 := filepath.Join(parseRoot2, "bin", "tmp")
		if parseGot2 != parseWant2 || parseSource2 != "convention fallback" {
			parseT3.Fatalf("expected convention temp root %q, got path=%q source=%q", parseWant2, parseGot2, parseSource2)
		}
	})
}

func TestResolveLauncherLivereloadWorkspaceUsesOverride(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseOverrideDir := filepath.Join(parseRoot, "enterprise-livereload")
	if parseErr := os.MkdirAll(parseOverrideDir, 0755); parseErr != nil {
		parseT.Fatalf("mkdir override dir: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(filepath.Join(parseRoot, "gwc-runner.json"), []byte(`{"paths":{"livereloadWorkspace":"enterprise-livereload"}}`), 0644); parseErr2 != nil {
		parseT.Fatalf("write config: %v", parseErr2)
	}

	parseGot, parseErr3 := resolveLauncherLivereloadWorkspace(filepath.Join(parseRoot, "repo"), parseRoot)
	if parseErr3 != nil {
		parseT.Fatalf("resolve livereload workspace: %v", parseErr3)
	}
	if parseGot != parseOverrideDir {
		parseT.Fatalf("expected livereload workspace %q, got %q", parseOverrideDir, parseGot)
	}
}

func TestResolveLauncherLivereloadClientScriptUsesOverride(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseOverrideFile := filepath.Join(parseRoot, "vendor", "livereload-client.js")
	if parseErr := os.MkdirAll(filepath.Dir(parseOverrideFile), 0755); parseErr != nil {
		parseT.Fatalf("mkdir override dir: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(parseOverrideFile, []byte("console.log('ok');\n"), 0644); parseErr2 != nil {
		parseT.Fatalf("write override file: %v", parseErr2)
	}
	if parseErr3 := os.WriteFile(filepath.Join(parseRoot, "gwc-runner.json"), []byte(`{"paths":{"livereloadClientScript":"vendor/livereload-client.js"}}`), 0644); parseErr3 != nil {
		parseT.Fatalf("write config: %v", parseErr3)
	}

	parseGot, parseOk, parseErr4 := resolveLauncherLivereloadClientScript(filepath.Join(parseRoot, "repo"), parseRoot)
	if parseErr4 != nil {
		parseT.Fatalf("resolve livereload client script: %v", parseErr4)
	}
	if !parseOk {
		parseT.Fatal("expected livereload client script override to be discovered")
	}
	if parseGot != parseOverrideFile {
		parseT.Fatalf("expected livereload client script %q, got %q", parseOverrideFile, parseGot)
	}
}

func TestResolveWasmExecPathUsesOverride(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseOverrideFile := filepath.Join(parseRoot, "vendor", "wasm_exec.js")
	if parseErr := os.MkdirAll(filepath.Dir(parseOverrideFile), 0755); parseErr != nil {
		parseT.Fatalf("mkdir override dir: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(parseOverrideFile, []byte("// wasm runtime"), 0644); parseErr2 != nil {
		parseT.Fatalf("write wasm_exec override: %v", parseErr2)
	}
	if parseErr3 := os.WriteFile(filepath.Join(parseRoot, "gwc-runner.json"), []byte(`{"paths":{"wasmExecJS":"vendor/wasm_exec.js"}}`), 0644); parseErr3 != nil {
		parseT.Fatalf("write config: %v", parseErr3)
	}

	parseOriginalGetwd := launcherConfigGetwd
	parseT.Cleanup(func() { launcherConfigGetwd = parseOriginalGetwd })
	launcherConfigGetwd = func() (string, error) { return parseRoot, nil }

	parseGot, parseErr4 := resolveWasmExecPath()
	if parseErr4 != nil {
		parseT.Fatalf("resolve wasm exec path: %v", parseErr4)
	}
	if parseGot != parseOverrideFile {
		parseT.Fatalf("expected wasm exec override %q, got %q", parseOverrideFile, parseGot)
	}
}

func TestResolveWasmExecPathRejectsMissingOverrideFile(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	if parseErr := os.WriteFile(filepath.Join(parseRoot, "gwc-runner.json"), []byte(`{"paths":{"wasmExecJS":"vendor/missing-wasm_exec.js"}}`), 0644); parseErr != nil {
		parseT.Fatalf("write config: %v", parseErr)
	}

	parseOriginalGetwd := launcherConfigGetwd
	parseT.Cleanup(func() { launcherConfigGetwd = parseOriginalGetwd })
	launcherConfigGetwd = func() (string, error) { return parseRoot, nil }

	parseGot, parseErr2 := resolveWasmExecPath()
	if parseErr2 == nil || !strings.Contains(parseErr2.Error(), "configured wasmExecJS path does not exist") {
		parseT.Fatalf("expected missing override file error, got path=%q err=%v", parseGot, parseErr2)
	}
}

func TestResolveWasmExecPathFallsBackToGOROOTRuntime(parseT *testing.T) {
	parseT.Setenv(launcherOverrideEnvVar, "")
	parseOriginalGetwd := launcherConfigGetwd
	parseT.Cleanup(func() { launcherConfigGetwd = parseOriginalGetwd })
	launcherConfigGetwd = func() (string, error) { return parseT.TempDir(), nil }

	parseGot, parseErr := resolveWasmExecPath()
	if parseErr != nil {
		parseT.Fatalf("resolve wasm exec path from GOROOT: %v", parseErr)
	}
	parseGoRoot := resolveWasmExecGoRoot()
	parseWantCandidates := []string{
		filepath.Join(parseGoRoot, "lib", "wasm", "wasm_exec.js"),
		filepath.Join(parseGoRoot, "misc", "wasm", "wasm_exec.js"),
	}
	if parseGot != parseWantCandidates[0] && parseGot != parseWantCandidates[1] {
		parseT.Fatalf("expected GOROOT wasm_exec.js path, got %q", parseGot)
	}
}

func TestResolveWasmExecPathRejectsMissingGOROOTVariants(parseT *testing.T) {
	parseT.Setenv(launcherOverrideEnvVar, "")
	parseOriginalGetwd := launcherConfigGetwd
	parseOriginalGoRoot := resolveWasmExecGoRoot
	parseT.Cleanup(func() {
		launcherConfigGetwd = parseOriginalGetwd
		resolveWasmExecGoRoot = parseOriginalGoRoot
	})
	launcherConfigGetwd = func() (string, error) { return parseT.TempDir(), nil }

	resolveWasmExecGoRoot = func() string { return "" }
	if parseGot, parseErr := resolveWasmExecPath(); parseErr == nil || !strings.Contains(parseErr.Error(), "GOROOT is not available") {
		parseT.Fatalf("expected empty GOROOT error, got path=%q err=%v", parseGot, parseErr)
	}

	resolveWasmExecGoRoot = func() string { return parseT.TempDir() }
	if parseGot2, parseErr2 := resolveWasmExecPath(); parseErr2 == nil || !strings.Contains(parseErr2.Error(), "wasm_exec.js not found under GOROOT") {
		parseT.Fatalf("expected missing GOROOT wasm_exec.js error, got path=%q err=%v", parseGot2, parseErr2)
	}
}

func TestRunnerConfigPathOverridesApplyAcrossLauncherCommands(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseRepoRoot, parseErr := resolveRepoRoot()
	if parseErr != nil {
		parseT.Fatalf("resolve repo root: %v", parseErr)
	}
	parseMainPath := filepath.Join(parseRoot, "main.go")
	if parseErr2 := os.WriteFile(filepath.Join(parseRoot, "go.mod"), []byte("module example.com/runnerconfigparity\n\ngo 1.25.0\n"), 0644); parseErr2 != nil {
		parseT.Fatalf("write go.mod: %v", parseErr2)
	}
	if parseErr3 := os.WriteFile(parseMainPath, []byte("package main\nfunc main() {}\n"), 0644); parseErr3 != nil {
		parseT.Fatalf("write main.go: %v", parseErr3)
	}
	parseBrowserWorkspace := filepath.Join(parseRoot, "enterprise-browser")
	if parseErr4 := os.MkdirAll(parseBrowserWorkspace, 0755); parseErr4 != nil {
		parseT.Fatalf("mkdir browser workspace: %v", parseErr4)
	}
	if parseErr5 := os.MkdirAll(filepath.Join(parseBrowserWorkspace, "playwrightgo"), 0755); parseErr5 != nil {
		parseT.Fatalf("mkdir browser playwrightgo package: %v", parseErr5)
	}
	if parseErr6 := os.WriteFile(filepath.Join(parseRoot, "gwc-runner.json"), []byte(`{"paths":{"artifactRoot":"enterprise-artifacts","browserWorkspace":"enterprise-browser"}}`), 0644); parseErr6 != nil {
		parseT.Fatalf("write gwc-runner.json: %v", parseErr6)
	}

	parseStdout, parseRestoreStdout, parseErr := captureExamplesStdout()
	if parseErr != nil {
		parseT.Fatalf("capture stdout: %v", parseErr)
	}
	defer parseRestoreStdout()

	parseLauncher := launcher{repoRoot: parseRepoRoot}
	if parseErr7 := parseLauncher.run([]string{"build", "-app", parseMainPath, "-root", parseRoot, "-json"}); parseErr7 != nil {
		parseT.Fatalf("run build with runner config: %v", parseErr7)
	}
	buildOutput, parseErr := parseStdout()
	if parseErr != nil {
		parseT.Fatalf("read build stdout: %v", parseErr)
	}
	var buildSummary buildSummary
	if parseErr8 := json.Unmarshal([]byte(buildOutput), &buildSummary); parseErr8 != nil {
		parseT.Fatalf("unmarshal build summary: %v\n%s", parseErr8, buildOutput)
	}
	if buildSummary.OutputPath != filepath.Join(parseRoot, "enterprise-artifacts", filepath.Base(parseRoot), "bin", "main.wasm") {
		parseT.Fatalf("expected build output to honor artifactRoot, got %#v", buildSummary)
	}

	parseStdout, parseRestoreStdout, parseErr = captureExamplesStdout()
	if parseErr != nil {
		parseT.Fatalf("capture release stdout: %v", parseErr)
	}
	defer parseRestoreStdout()

	if parseErr9 := parseLauncher.run([]string{"release", "-app", parseMainPath, "-root", parseRoot, "-compression", "none", "-json"}); parseErr9 != nil {
		parseT.Fatalf("run release with runner config: %v", parseErr9)
	}
	parseReleaseOutput, parseErr := parseStdout()
	if parseErr != nil {
		parseT.Fatalf("read release stdout: %v", parseErr)
	}
	var parseReleaseSummary releaseSummary
	if parseErr10 := json.Unmarshal([]byte(parseReleaseOutput), &parseReleaseSummary); parseErr10 != nil {
		parseT.Fatalf("unmarshal release summary: %v\n%s", parseErr10, parseReleaseOutput)
	}
	if parseReleaseSummary.OutDir != filepath.Join(parseRoot, "enterprise-artifacts", filepath.Base(parseRoot), "wasm-release") {
		parseT.Fatalf("expected release output to honor artifactRoot, got %#v", parseReleaseSummary)
	}

	parseStdout, parseRestoreStdout, parseErr = captureExamplesStdout()
	if parseErr != nil {
		parseT.Fatalf("capture test stdout: %v", parseErr)
	}
	defer parseRestoreStdout()

	parseOriginalRunCommand := launcherRunCommand
	parseT.Cleanup(func() { launcherRunCommand = parseOriginalRunCommand })
	launcherRunCommand = func(parseCommand string, parseArgs []string, parseCwd string, parseEnv []string) (string, error) {
		if parseCommand != "go" {
			parseT.Fatalf("expected browser lane to invoke go, got %q", parseCommand)
		}
		parseExpectedArgs := []string{"test", "-tags", "playwrightgo", "./playwrightgo", "-run", "TestMainSuite", "-v"}
		if !reflect.DeepEqual(parseArgs, parseExpectedArgs) {
			parseT.Fatalf("expected browser lane args %#v, got %#v", parseExpectedArgs, parseArgs)
		}
		if parseCwd != parseBrowserWorkspace {
			parseT.Fatalf("expected browser workspace override %q, got %q", parseBrowserWorkspace, parseCwd)
		}
		return "playwright ok", nil
	}
	if parseErr11 := parseLauncher.run([]string{"test", "-root", parseRoot, "-lane", "browser", "-json"}); parseErr11 != nil {
		parseT.Fatalf("run browser test with runner config: %v", parseErr11)
	}
	parseTestOutput, parseErr := parseStdout()
	if parseErr != nil {
		parseT.Fatalf("read test stdout: %v", parseErr)
	}
	var parseTestSummary testSummary
	if parseErr12 := json.Unmarshal([]byte(parseTestOutput), &parseTestSummary); parseErr12 != nil {
		parseT.Fatalf("unmarshal test summary: %v\n%s", parseErr12, parseTestOutput)
	}
	if len(parseTestSummary.Lanes) != 1 || parseTestSummary.Lanes[0].Workspace != parseBrowserWorkspace {
		parseT.Fatalf("expected browser test lane to honor browserWorkspace override, got %#v", parseTestSummary)
	}
}

func TestResolveWasmTestExecUsesOverride(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseOverrideFile := filepath.Join(parseRoot, "tools", "go_js_wasm_exec.bat")
	if parseErr := os.MkdirAll(filepath.Dir(parseOverrideFile), 0755); parseErr != nil {
		parseT.Fatalf("mkdir override dir: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(parseOverrideFile, []byte("@echo off\n"), 0644); parseErr2 != nil {
		parseT.Fatalf("write go wasm exec override: %v", parseErr2)
	}
	if parseErr3 := os.WriteFile(filepath.Join(parseRoot, "gwc-runner.json"), []byte(`{"paths":{"goWasmExec":"tools/go_js_wasm_exec.bat"}}`), 0644); parseErr3 != nil {
		parseT.Fatalf("write config: %v", parseErr3)
	}

	parseOriginalGetwd := launcherConfigGetwd
	parseT.Cleanup(func() { launcherConfigGetwd = parseOriginalGetwd })
	launcherConfigGetwd = func() (string, error) { return parseRoot, nil }

	parseGot, parseErr4 := resolveWasmTestExec(parseRoot)
	if parseErr4 != nil {
		parseT.Fatalf("resolve wasm test exec: %v", parseErr4)
	}
	if parseGot != parseOverrideFile {
		parseT.Fatalf("expected goWasmExec override %q, got %q", parseOverrideFile, parseGot)
	}
}

func TestResolveWasmTestExecUsesRepoRootConfigInsteadOfCurrentContext(parseT *testing.T) {
	parseRepoRoot := parseT.TempDir()
	parseOtherRoot := parseT.TempDir()
	parseOverrideFile := filepath.Join(parseRepoRoot, "tools", "go_js_wasm_exec.bat")
	if parseErr := os.MkdirAll(filepath.Dir(parseOverrideFile), 0755); parseErr != nil {
		parseT.Fatalf("mkdir override dir: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(parseOverrideFile, []byte("@echo off\n"), 0644); parseErr2 != nil {
		parseT.Fatalf("write override file: %v", parseErr2)
	}
	if parseErr3 := os.WriteFile(filepath.Join(parseRepoRoot, "gwc-runner.json"), []byte(`{"paths":{"goWasmExec":"tools/go_js_wasm_exec.bat"}}`), 0644); parseErr3 != nil {
		parseT.Fatalf("write repo config: %v", parseErr3)
	}

	parseOriginalGetwd := launcherConfigGetwd
	parseT.Cleanup(func() { launcherConfigGetwd = parseOriginalGetwd })
	launcherConfigGetwd = func() (string, error) { return parseOtherRoot, nil }

	parseGot, parseErr4 := resolveWasmTestExec(parseRepoRoot)
	if parseErr4 != nil {
		parseT.Fatalf("resolve wasm test exec: %v", parseErr4)
	}
	if parseGot != parseOverrideFile {
		parseT.Fatalf("expected repo-root config override %q, got %q", parseOverrideFile, parseGot)
	}
}

func TestResolveWasmTestExecUsesEnvironmentFallback(parseT *testing.T) {
	parseT.Setenv("GO_WASM_EXEC", filepath.Join(`C:\tools`, "go_js_wasm_exec.bat"))

	parseGot, parseErr := resolveWasmTestExec(parseT.TempDir())
	if parseErr != nil {
		parseT.Fatalf("resolve wasm test exec from env: %v", parseErr)
	}
	if parseGot != filepath.Join(`C:\tools`, "go_js_wasm_exec.bat") {
		parseT.Fatalf("expected env-based wasm exec path, got %q", parseGot)
	}
}

func TestResolveWasmTestExecErrorsWhenNoHelperExists(parseT *testing.T) {
	parseT.Setenv("GO_WASM_EXEC", "")

	parseRoot := parseT.TempDir()
	parseGot, parseErr := resolveWasmTestExec(parseRoot)
	if parseErr == nil {
		parseT.Fatalf("expected missing wasm exec helper to fail, got path %q", parseGot)
	}
	if !strings.Contains(parseErr.Error(), "GO_WASM_EXEC is not set") {
		parseT.Fatalf("expected actionable missing wasm exec error, got %v", parseErr)
	}
}

func TestResolveWasmTestExecUsesRepoHelperOnWindows(parseT *testing.T) {
	parseT.Setenv("GO_WASM_EXEC", "")
	parseRoot := parseT.TempDir()
	parseHelperPath := filepath.Join(parseRoot, "tools", "go_js_wasm_exec.bat")
	if parseErr := os.MkdirAll(filepath.Dir(parseHelperPath), 0755); parseErr != nil {
		parseT.Fatalf("mkdir helper dir: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(parseHelperPath, []byte("@echo off\n"), 0644); parseErr2 != nil {
		parseT.Fatalf("write helper: %v", parseErr2)
	}

	parseGot, parseErr3 := resolveWasmTestExec(parseRoot)
	if parseErr3 != nil {
		parseT.Fatalf("resolve wasm test exec from repo helper: %v", parseErr3)
	}
	if parseGot != parseHelperPath {
		parseT.Fatalf("expected repo helper %q, got %q", parseHelperPath, parseGot)
	}
}

func TestDetectBrowserWorkspaceUsesOverride(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseBrowserWorkspace := filepath.Join(parseRoot, "enterprise-browser")
	if parseErr := os.MkdirAll(parseBrowserWorkspace, 0755); parseErr != nil {
		parseT.Fatalf("mkdir browser workspace: %v", parseErr)
	}
	if parseErr2 := os.MkdirAll(filepath.Join(parseBrowserWorkspace, "playwrightgo"), 0755); parseErr2 != nil {
		parseT.Fatalf("mkdir browser workspace playwrightgo package: %v", parseErr2)
	}
	if parseErr3 := os.WriteFile(filepath.Join(parseRoot, "gwc-runner.json"), []byte(`{"paths":{"browserWorkspace":"enterprise-browser"}}`), 0644); parseErr3 != nil {
		parseT.Fatalf("write config: %v", parseErr3)
	}

	parseGot := detectBrowserWorkspace(filepath.Join(parseRoot, "repo"), parseRoot)
	if parseGot != parseBrowserWorkspace {
		parseT.Fatalf("expected browser workspace override %q, got %q", parseBrowserWorkspace, parseGot)
	}
}

func TestResolveLauncherOverridePathResolvesRelativeEnvPath(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseConfigPath := filepath.Join(parseRoot, "configs", "runner.json")
	if parseErr := os.MkdirAll(filepath.Dir(parseConfigPath), 0755); parseErr != nil {
		parseT.Fatalf("mkdir config dir: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(parseConfigPath, []byte(`{"paths":{"generatedProjectRoot":"custom"}}`), 0644); parseErr2 != nil {
		parseT.Fatalf("write config: %v", parseErr2)
	}
	parseT.Setenv(launcherOverrideEnvVar, filepath.Join("configs", "runner.json"))

	parseResolved, parseErr3 := resolveLauncherOverridePath(parseRoot)
	if parseErr3 != nil {
		parseT.Fatalf("resolve launcher override path: %v", parseErr3)
	}
	if parseResolved != parseConfigPath {
		parseT.Fatalf("expected resolved config path %q, got %q", parseConfigPath, parseResolved)
	}
}

func TestLoadLauncherOverridesForCurrentContextFallsBackToHomeConfig(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseHome := filepath.Join(parseRoot, "home")
	parseConfigPath := filepath.Join(parseHome, ".gwc", "runner.json")
	if parseErr := os.MkdirAll(filepath.Dir(parseConfigPath), 0755); parseErr != nil {
		parseT.Fatalf("mkdir home config dir: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(parseConfigPath, []byte(`{"paths":{"browserWorkspace":"browser-suite"}}`), 0644); parseErr2 != nil {
		parseT.Fatalf("write home config: %v", parseErr2)
	}

	parseOriginalGetwd := launcherConfigGetwd
	parseOriginalHomeDir := launcherConfigUserHomeDir
	parseT.Cleanup(func() {
		launcherConfigGetwd = parseOriginalGetwd
		launcherConfigUserHomeDir = parseOriginalHomeDir
	})
	launcherConfigGetwd = func() (string, error) { return "", os.ErrPermission }
	launcherConfigUserHomeDir = func() (string, error) { return parseHome, nil }

	parseOverrides, parseFoundPath, parseOk, parseErr3 := loadLauncherOverridesForCurrentContext()
	if parseErr3 != nil {
		parseT.Fatalf("load launcher overrides for current context: %v", parseErr3)
	}
	if !parseOk {
		parseT.Fatal("expected home launcher override config to be discovered")
	}
	if parseFoundPath != parseConfigPath {
		parseT.Fatalf("expected home config path %q, got %q", parseConfigPath, parseFoundPath)
	}
	if parseOverrides.Paths.BrowserWorkspace != "browser-suite" {
		parseT.Fatalf("expected browser workspace override, got %#v", parseOverrides)
	}
}

func TestLoadLauncherOverridesReportsParseError(parseT *testing.T) {
	parseConfigPath := filepath.Join(parseT.TempDir(), "gwc-runner.json")
	if parseErr := os.WriteFile(parseConfigPath, []byte(`{"paths":`), 0644); parseErr != nil {
		parseT.Fatalf("write invalid config: %v", parseErr)
	}
	parseT.Setenv(launcherOverrideEnvVar, parseConfigPath)

	_, _, _, parseErr2 := loadLauncherOverrides("")
	if parseErr2 == nil {
		parseT.Fatal("expected invalid launcher override json to fail")
	}
	if !strings.Contains(parseErr2.Error(), "parse launcher override file") {
		parseT.Fatalf("expected parse error wrapper, got %v", parseErr2)
	}
}

func TestResolveLauncherOverrideValueHandlesBlankAbsoluteAndRelativePaths(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseConfigPath := filepath.Join(parseRoot, "gwc-runner.json")
	parseAbsolutePath := filepath.Join(parseRoot, "absolute", "runner.json")

	parseBlank, parseErr := resolveLauncherOverrideValue(parseConfigPath, "   ")
	if parseErr != nil {
		parseT.Fatalf("resolve blank override value: %v", parseErr)
	}
	if parseBlank != "" {
		parseT.Fatalf("expected blank override value to stay blank, got %q", parseBlank)
	}

	parseAbs, parseErr := resolveLauncherOverrideValue(parseConfigPath, parseAbsolutePath)
	if parseErr != nil {
		parseT.Fatalf("resolve absolute override value: %v", parseErr)
	}
	if parseAbs != filepath.Clean(parseAbsolutePath) {
		parseT.Fatalf("expected absolute override value %q, got %q", parseAbsolutePath, parseAbs)
	}

	parseRelative, parseErr := resolveLauncherOverrideValue(parseConfigPath, filepath.Join("vendor", "wasm_exec.js"))
	if parseErr != nil {
		parseT.Fatalf("resolve relative override value: %v", parseErr)
	}
	parseWantRelative := filepath.Join(parseRoot, "vendor", "wasm_exec.js")
	if parseRelative != parseWantRelative {
		parseT.Fatalf("expected relative override value %q, got %q", parseWantRelative, parseRelative)
	}
}
