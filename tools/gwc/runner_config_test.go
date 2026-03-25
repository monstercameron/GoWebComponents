package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func TestLoadLauncherOverridesFindsParentConfig(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "apps", "sample")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatalf("mkdir nested dir: %v", err)
	}
	configPath := filepath.Join(root, "gwc-runner.json")
	if err := os.WriteFile(configPath, []byte(`{"paths":{"generatedProjectRoot":"company-projects"}}`), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	overrides, foundPath, ok, err := loadLauncherOverrides(nested)
	if err != nil {
		t.Fatalf("load launcher overrides: %v", err)
	}
	if !ok {
		t.Fatal("expected launcher overrides to be discovered from a parent directory")
	}
	if foundPath != configPath {
		t.Fatalf("expected config path %q, got %q", configPath, foundPath)
	}
	if overrides.Paths.GeneratedProjectRoot != "company-projects" {
		t.Fatalf("expected generated project root override, got %#v", overrides)
	}
}

func TestCanonicalRunnerConfigExampleMatchesLauncherSchema(t *testing.T) {
	examplePath := filepath.Join("..", "..", "docs", "examples", "gwc-runner.example.json")
	content, err := os.ReadFile(examplePath)
	if err != nil {
		t.Fatalf("read canonical runner config example: %v", err)
	}

	var overrides launcherOverrides
	if err := json.Unmarshal(content, &overrides); err != nil {
		t.Fatalf("parse canonical runner config example: %v", err)
	}

	if overrides.Paths.GeneratedProjectRoot != "generated-projects" {
		t.Fatalf("expected generatedProjectRoot in canonical example, got %#v", overrides.Paths)
	}
	if overrides.Paths.ArtifactRoot != "enterprise-artifacts" {
		t.Fatalf("expected artifactRoot in canonical example, got %#v", overrides.Paths)
	}
	if overrides.Paths.WASMExecJS != "vendor/wasm_exec.js" {
		t.Fatalf("expected wasmExecJS in canonical example, got %#v", overrides.Paths)
	}
	if overrides.Paths.GoWASMExec != "tools/go_js_wasm_exec.bat" {
		t.Fatalf("expected goWasmExec in canonical example, got %#v", overrides.Paths)
	}
	if overrides.Paths.BrowserWorkspace != "test" {
		t.Fatalf("expected browserWorkspace in canonical example, got %#v", overrides.Paths)
	}
	if overrides.Paths.LivereloadWorkspace != "tools/livereload" {
		t.Fatalf("expected livereloadWorkspace in canonical example, got %#v", overrides.Paths)
	}
	if overrides.Paths.LivereloadClientScript != "" {
		t.Fatalf("expected canonical example to omit livereloadClientScript, got %#v", overrides.Paths)
	}
	if len(overrides.Paths.GeneratedProjectRoot) == 0 {
		t.Fatalf("expected canonical example to stay non-empty, got %#v", overrides.Paths)
	}
	if len(overrides.Paths.ArtifactRoot) == 0 {
		t.Fatalf("expected canonical example to stay non-empty, got %#v", overrides.Paths)
	}
	if len(overrides.Paths.WASMExecJS) == 0 || len(overrides.Paths.GoWASMExec) == 0 {
		t.Fatalf("expected canonical example exec fields to stay non-empty, got %#v", overrides.Paths)
	}
}

func TestLoadLauncherOverridesPrefersExplicitEnvPath(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "enterprise-runner.json")
	if err := os.WriteFile(configPath, []byte(`{"paths":{"browserWorkspace":"browser-suite"}}`), 0644); err != nil {
		t.Fatalf("write explicit env config: %v", err)
	}
	t.Setenv(launcherOverrideEnvVar, configPath)

	overrides, foundPath, ok, err := loadLauncherOverrides("")
	if err != nil {
		t.Fatalf("load launcher overrides: %v", err)
	}
	if !ok {
		t.Fatal("expected launcher overrides from explicit env path")
	}
	if foundPath != configPath {
		t.Fatalf("expected explicit env config path %q, got %q", configPath, foundPath)
	}
	if overrides.Paths.BrowserWorkspace != "browser-suite" {
		t.Fatalf("expected browser workspace override, got %#v", overrides)
	}
}

func TestDefaultGeneratedScaffoldRootUsesOverride(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "gwc-runner.json")
	if err := os.WriteFile(configPath, []byte(`{"paths":{"generatedProjectRoot":"custom-projects"}}`), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	originalGetwd := launcherConfigGetwd
	originalHomeDir := startUserHomeDir
	t.Cleanup(func() {
		launcherConfigGetwd = originalGetwd
		startUserHomeDir = originalHomeDir
	})
	launcherConfigGetwd = func() (string, error) { return root, nil }
	startUserHomeDir = func() (string, error) { return filepath.Join(root, "home"), nil }

	got := defaultGeneratedScaffoldRoot()
	want := filepath.Join(root, "custom-projects")
	if got != want {
		t.Fatalf("expected override scaffold root %q, got %q", want, got)
	}
}

func TestResolveLauncherArtifactPathUsesOverride(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "gwc-runner.json"), []byte(`{"paths":{"artifactRoot":"enterprise-artifacts"}}`), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	got, ok, err := resolveLauncherArtifactPath(root, "wasm-release")
	if err != nil {
		t.Fatalf("resolve launcher artifact path: %v", err)
	}
	if !ok {
		t.Fatal("expected artifact root override to be discovered")
	}
	want := filepath.Join(root, "enterprise-artifacts", filepath.Base(root), "wasm-release")
	if got != want {
		t.Fatalf("expected artifact path %q, got %q", want, got)
	}
}

func TestResolveLauncherDefaultBuildOutputUsesSharedResolver(t *testing.T) {
	t.Run("artifact root override", func(t *testing.T) {
		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, "gwc-runner.json"), []byte(`{"paths":{"artifactRoot":"enterprise-artifacts"}}`), 0644); err != nil {
			t.Fatalf("write config: %v", err)
		}

		got, source, err := resolveLauncherDefaultBuildOutput(root)
		if err != nil {
			t.Fatalf("resolve default build output: %v", err)
		}
		want := filepath.Join(root, "enterprise-artifacts", filepath.Base(root), "main.wasm")
		if got != want || source != "gwc-runner.json paths.artifactRoot" {
			t.Fatalf("expected shared build output resolver to return %q via gwc-runner.json paths.artifactRoot, got path=%q source=%q", want, got, source)
		}
	})

	t.Run("convention fallback", func(t *testing.T) {
		root := t.TempDir()
		got, source, err := resolveLauncherDefaultBuildOutput(root)
		if err != nil {
			t.Fatalf("resolve default build output: %v", err)
		}
		want := filepath.Join(root, "main.wasm")
		if got != want || source != "convention fallback" {
			t.Fatalf("expected convention fallback build output %q, got path=%q source=%q", want, got, source)
		}
	})
}

func TestResolveLauncherDefaultReleaseOutDirUsesSharedResolver(t *testing.T) {
	t.Run("artifact root override", func(t *testing.T) {
		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, "gwc-runner.json"), []byte(`{"paths":{"artifactRoot":"enterprise-artifacts"}}`), 0644); err != nil {
			t.Fatalf("write config: %v", err)
		}

		got, source, err := resolveLauncherDefaultReleaseOutDir(root)
		if err != nil {
			t.Fatalf("resolve default release out dir: %v", err)
		}
		want := filepath.Join(root, "enterprise-artifacts", filepath.Base(root), "wasm-release")
		if got != want || source != "gwc-runner.json paths.artifactRoot" {
			t.Fatalf("expected shared release output resolver to return %q via gwc-runner.json paths.artifactRoot, got path=%q source=%q", want, got, source)
		}
	})

	t.Run("convention fallback", func(t *testing.T) {
		root := t.TempDir()
		got, source, err := resolveLauncherDefaultReleaseOutDir(root)
		if err != nil {
			t.Fatalf("resolve default release out dir: %v", err)
		}
		want := filepath.Join(root, "bin", "wasm-release")
		if got != want || source != "convention fallback" {
			t.Fatalf("expected convention fallback release out dir %q, got path=%q source=%q", want, got, source)
		}
	})
}

func TestResolveLauncherExamplesWasmDirUsesSharedResolver(t *testing.T) {
	repoRoot := t.TempDir()
	examplesBuildDir := filepath.Join(repoRoot, "bin", "examples")
	if err := os.MkdirAll(examplesBuildDir, 0755); err != nil {
		t.Fatalf("mkdir examples build dir: %v", err)
	}

	got := resolveLauncherExamplesWasmDir(repoRoot, filepath.Join(repoRoot, "examples", "static"))
	if got != examplesBuildDir {
		t.Fatalf("expected shared examples wasm dir resolver to return %q, got %q", examplesBuildDir, got)
	}

	staticDir := filepath.Join(t.TempDir(), "static")
	if err := os.MkdirAll(staticDir, 0755); err != nil {
		t.Fatalf("mkdir static dir: %v", err)
	}
	got = resolveLauncherExamplesWasmDir(t.TempDir(), staticDir)
	if got != filepath.Join(staticDir, "bin") {
		t.Fatalf("expected static-dir fallback examples wasm dir, got %q", got)
	}
}

func TestResolveLauncherTempRootUsesSharedResolver(t *testing.T) {
	t.Run("artifact root override", func(t *testing.T) {
		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, "gwc-runner.json"), []byte(`{"paths":{"artifactRoot":"enterprise-artifacts"}}`), 0644); err != nil {
			t.Fatalf("write config: %v", err)
		}

		got, source, err := resolveLauncherTempRoot(root)
		if err != nil {
			t.Fatalf("resolve launcher temp root: %v", err)
		}
		want := filepath.Join(root, "enterprise-artifacts", filepath.Base(root), "tmp")
		if got != want || source != "gwc-runner.json paths.artifactRoot" {
			t.Fatalf("expected shared temp root %q via gwc-runner.json paths.artifactRoot, got path=%q source=%q", want, got, source)
		}
	})

	t.Run("convention fallback", func(t *testing.T) {
		root := t.TempDir()
		got, source, err := resolveLauncherTempRoot(root)
		if err != nil {
			t.Fatalf("resolve launcher temp root: %v", err)
		}
		want := filepath.Join(root, "bin", "tmp")
		if got != want || source != "convention fallback" {
			t.Fatalf("expected convention temp root %q, got path=%q source=%q", want, got, source)
		}
	})
}

func TestResolveLauncherLivereloadWorkspaceUsesOverride(t *testing.T) {
	root := t.TempDir()
	overrideDir := filepath.Join(root, "enterprise-livereload")
	if err := os.MkdirAll(overrideDir, 0755); err != nil {
		t.Fatalf("mkdir override dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "gwc-runner.json"), []byte(`{"paths":{"livereloadWorkspace":"enterprise-livereload"}}`), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	got, err := resolveLauncherLivereloadWorkspace(filepath.Join(root, "repo"), root)
	if err != nil {
		t.Fatalf("resolve livereload workspace: %v", err)
	}
	if got != overrideDir {
		t.Fatalf("expected livereload workspace %q, got %q", overrideDir, got)
	}
}

func TestResolveLauncherLivereloadClientScriptUsesOverride(t *testing.T) {
	root := t.TempDir()
	overrideFile := filepath.Join(root, "vendor", "livereload-client.js")
	if err := os.MkdirAll(filepath.Dir(overrideFile), 0755); err != nil {
		t.Fatalf("mkdir override dir: %v", err)
	}
	if err := os.WriteFile(overrideFile, []byte("console.log('ok');\n"), 0644); err != nil {
		t.Fatalf("write override file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "gwc-runner.json"), []byte(`{"paths":{"livereloadClientScript":"vendor/livereload-client.js"}}`), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	got, ok, err := resolveLauncherLivereloadClientScript(filepath.Join(root, "repo"), root)
	if err != nil {
		t.Fatalf("resolve livereload client script: %v", err)
	}
	if !ok {
		t.Fatal("expected livereload client script override to be discovered")
	}
	if got != overrideFile {
		t.Fatalf("expected livereload client script %q, got %q", overrideFile, got)
	}
}

func TestResolveWasmExecPathUsesOverride(t *testing.T) {
	root := t.TempDir()
	overrideFile := filepath.Join(root, "vendor", "wasm_exec.js")
	if err := os.MkdirAll(filepath.Dir(overrideFile), 0755); err != nil {
		t.Fatalf("mkdir override dir: %v", err)
	}
	if err := os.WriteFile(overrideFile, []byte("// wasm runtime"), 0644); err != nil {
		t.Fatalf("write wasm_exec override: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "gwc-runner.json"), []byte(`{"paths":{"wasmExecJS":"vendor/wasm_exec.js"}}`), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	originalGetwd := launcherConfigGetwd
	t.Cleanup(func() { launcherConfigGetwd = originalGetwd })
	launcherConfigGetwd = func() (string, error) { return root, nil }

	got, err := resolveWasmExecPath()
	if err != nil {
		t.Fatalf("resolve wasm exec path: %v", err)
	}
	if got != overrideFile {
		t.Fatalf("expected wasm exec override %q, got %q", overrideFile, got)
	}
}

func TestResolveWasmExecPathRejectsMissingOverrideFile(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "gwc-runner.json"), []byte(`{"paths":{"wasmExecJS":"vendor/missing-wasm_exec.js"}}`), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	originalGetwd := launcherConfigGetwd
	t.Cleanup(func() { launcherConfigGetwd = originalGetwd })
	launcherConfigGetwd = func() (string, error) { return root, nil }

	got, err := resolveWasmExecPath()
	if err == nil || !strings.Contains(err.Error(), "configured wasmExecJS path does not exist") {
		t.Fatalf("expected missing override file error, got path=%q err=%v", got, err)
	}
}

func TestResolveWasmExecPathFallsBackToGOROOTRuntime(t *testing.T) {
	t.Setenv(launcherOverrideEnvVar, "")
	originalGetwd := launcherConfigGetwd
	t.Cleanup(func() { launcherConfigGetwd = originalGetwd })
	launcherConfigGetwd = func() (string, error) { return t.TempDir(), nil }

	got, err := resolveWasmExecPath()
	if err != nil {
		t.Fatalf("resolve wasm exec path from GOROOT: %v", err)
	}
	wantCandidates := []string{
		filepath.Join(runtime.GOROOT(), "lib", "wasm", "wasm_exec.js"),
		filepath.Join(runtime.GOROOT(), "misc", "wasm", "wasm_exec.js"),
	}
	if got != wantCandidates[0] && got != wantCandidates[1] {
		t.Fatalf("expected GOROOT wasm_exec.js path, got %q", got)
	}
}

func TestResolveWasmExecPathRejectsMissingGOROOTVariants(t *testing.T) {
	t.Setenv(launcherOverrideEnvVar, "")
	originalGetwd := launcherConfigGetwd
	originalGoRoot := resolveWasmExecGoRoot
	t.Cleanup(func() {
		launcherConfigGetwd = originalGetwd
		resolveWasmExecGoRoot = originalGoRoot
	})
	launcherConfigGetwd = func() (string, error) { return t.TempDir(), nil }

	resolveWasmExecGoRoot = func() string { return "" }
	if got, err := resolveWasmExecPath(); err == nil || !strings.Contains(err.Error(), "GOROOT is not available") {
		t.Fatalf("expected empty GOROOT error, got path=%q err=%v", got, err)
	}

	resolveWasmExecGoRoot = func() string { return t.TempDir() }
	if got, err := resolveWasmExecPath(); err == nil || !strings.Contains(err.Error(), "wasm_exec.js not found under GOROOT") {
		t.Fatalf("expected missing GOROOT wasm_exec.js error, got path=%q err=%v", got, err)
	}
}

func TestRunnerConfigPathOverridesApplyAcrossLauncherCommands(t *testing.T) {
	root := t.TempDir()
	repoRoot, err := resolveRepoRoot()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	mainPath := filepath.Join(root, "main.go")
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/runnerconfigparity\n\ngo 1.25.0\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(mainPath, []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}
	browserWorkspace := filepath.Join(root, "enterprise-browser")
	if err := os.MkdirAll(browserWorkspace, 0755); err != nil {
		t.Fatalf("mkdir browser workspace: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(browserWorkspace, "playwrightgo"), 0755); err != nil {
		t.Fatalf("mkdir browser playwrightgo package: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "gwc-runner.json"), []byte(`{"paths":{"artifactRoot":"enterprise-artifacts","browserWorkspace":"enterprise-browser"}}`), 0644); err != nil {
		t.Fatalf("write gwc-runner.json: %v", err)
	}

	stdout, restoreStdout, err := captureExamplesStdout()
	if err != nil {
		t.Fatalf("capture stdout: %v", err)
	}
	defer restoreStdout()

	launcher := launcher{repoRoot: repoRoot}
	if err := launcher.run([]string{"build", "-app", mainPath, "-root", root, "-json"}); err != nil {
		t.Fatalf("run build with runner config: %v", err)
	}
	buildOutput, err := stdout()
	if err != nil {
		t.Fatalf("read build stdout: %v", err)
	}
	var buildSummary buildSummary
	if err := json.Unmarshal([]byte(buildOutput), &buildSummary); err != nil {
		t.Fatalf("unmarshal build summary: %v\n%s", err, buildOutput)
	}
	if buildSummary.OutputPath != filepath.Join(root, "enterprise-artifacts", filepath.Base(root), "main.wasm") {
		t.Fatalf("expected build output to honor artifactRoot, got %#v", buildSummary)
	}

	stdout, restoreStdout, err = captureExamplesStdout()
	if err != nil {
		t.Fatalf("capture release stdout: %v", err)
	}
	defer restoreStdout()

	if err := launcher.run([]string{"release", "-app", mainPath, "-root", root, "-compression", "none", "-json"}); err != nil {
		t.Fatalf("run release with runner config: %v", err)
	}
	releaseOutput, err := stdout()
	if err != nil {
		t.Fatalf("read release stdout: %v", err)
	}
	var releaseSummary releaseSummary
	if err := json.Unmarshal([]byte(releaseOutput), &releaseSummary); err != nil {
		t.Fatalf("unmarshal release summary: %v\n%s", err, releaseOutput)
	}
	if releaseSummary.OutDir != filepath.Join(root, "enterprise-artifacts", filepath.Base(root), "wasm-release") {
		t.Fatalf("expected release output to honor artifactRoot, got %#v", releaseSummary)
	}

	stdout, restoreStdout, err = captureExamplesStdout()
	if err != nil {
		t.Fatalf("capture test stdout: %v", err)
	}
	defer restoreStdout()

	originalRunCommand := launcherRunCommand
	t.Cleanup(func() { launcherRunCommand = originalRunCommand })
	launcherRunCommand = func(command string, args []string, cwd string, env []string) (string, error) {
		if command != "go" {
			t.Fatalf("expected browser lane to invoke go, got %q", command)
		}
		expectedArgs := []string{"test", "-tags", "playwrightgo", "./playwrightgo", "-run", "TestMainSuite", "-v"}
		if !reflect.DeepEqual(args, expectedArgs) {
			t.Fatalf("expected browser lane args %#v, got %#v", expectedArgs, args)
		}
		if cwd != browserWorkspace {
			t.Fatalf("expected browser workspace override %q, got %q", browserWorkspace, cwd)
		}
		return "playwright ok", nil
	}
	if err := launcher.run([]string{"test", "-root", root, "-lane", "browser", "-json"}); err != nil {
		t.Fatalf("run browser test with runner config: %v", err)
	}
	testOutput, err := stdout()
	if err != nil {
		t.Fatalf("read test stdout: %v", err)
	}
	var testSummary testSummary
	if err := json.Unmarshal([]byte(testOutput), &testSummary); err != nil {
		t.Fatalf("unmarshal test summary: %v\n%s", err, testOutput)
	}
	if len(testSummary.Lanes) != 1 || testSummary.Lanes[0].Workspace != browserWorkspace {
		t.Fatalf("expected browser test lane to honor browserWorkspace override, got %#v", testSummary)
	}
}

func TestResolveWasmTestExecUsesOverride(t *testing.T) {
	root := t.TempDir()
	overrideFile := filepath.Join(root, "tools", "go_js_wasm_exec.bat")
	if err := os.MkdirAll(filepath.Dir(overrideFile), 0755); err != nil {
		t.Fatalf("mkdir override dir: %v", err)
	}
	if err := os.WriteFile(overrideFile, []byte("@echo off\n"), 0644); err != nil {
		t.Fatalf("write go wasm exec override: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "gwc-runner.json"), []byte(`{"paths":{"goWasmExec":"tools/go_js_wasm_exec.bat"}}`), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	originalGetwd := launcherConfigGetwd
	t.Cleanup(func() { launcherConfigGetwd = originalGetwd })
	launcherConfigGetwd = func() (string, error) { return root, nil }

	got, err := resolveWasmTestExec(root)
	if err != nil {
		t.Fatalf("resolve wasm test exec: %v", err)
	}
	if got != overrideFile {
		t.Fatalf("expected goWasmExec override %q, got %q", overrideFile, got)
	}
}

func TestResolveWasmTestExecUsesRepoRootConfigInsteadOfCurrentContext(t *testing.T) {
	repoRoot := t.TempDir()
	otherRoot := t.TempDir()
	overrideFile := filepath.Join(repoRoot, "tools", "go_js_wasm_exec.bat")
	if err := os.MkdirAll(filepath.Dir(overrideFile), 0755); err != nil {
		t.Fatalf("mkdir override dir: %v", err)
	}
	if err := os.WriteFile(overrideFile, []byte("@echo off\n"), 0644); err != nil {
		t.Fatalf("write override file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "gwc-runner.json"), []byte(`{"paths":{"goWasmExec":"tools/go_js_wasm_exec.bat"}}`), 0644); err != nil {
		t.Fatalf("write repo config: %v", err)
	}

	originalGetwd := launcherConfigGetwd
	t.Cleanup(func() { launcherConfigGetwd = originalGetwd })
	launcherConfigGetwd = func() (string, error) { return otherRoot, nil }

	got, err := resolveWasmTestExec(repoRoot)
	if err != nil {
		t.Fatalf("resolve wasm test exec: %v", err)
	}
	if got != overrideFile {
		t.Fatalf("expected repo-root config override %q, got %q", overrideFile, got)
	}
}

func TestResolveWasmTestExecUsesEnvironmentFallback(t *testing.T) {
	t.Setenv("GO_WASM_EXEC", filepath.Join(`C:\tools`, "go_js_wasm_exec.bat"))

	got, err := resolveWasmTestExec(t.TempDir())
	if err != nil {
		t.Fatalf("resolve wasm test exec from env: %v", err)
	}
	if got != filepath.Join(`C:\tools`, "go_js_wasm_exec.bat") {
		t.Fatalf("expected env-based wasm exec path, got %q", got)
	}
}

func TestResolveWasmTestExecErrorsWhenNoHelperExists(t *testing.T) {
	t.Setenv("GO_WASM_EXEC", "")

	root := t.TempDir()
	got, err := resolveWasmTestExec(root)
	if err == nil {
		t.Fatalf("expected missing wasm exec helper to fail, got path %q", got)
	}
	if !strings.Contains(err.Error(), "GO_WASM_EXEC is not set") {
		t.Fatalf("expected actionable missing wasm exec error, got %v", err)
	}
}

func TestResolveWasmTestExecUsesRepoHelperOnWindows(t *testing.T) {
	t.Setenv("GO_WASM_EXEC", "")
	root := t.TempDir()
	helperPath := filepath.Join(root, "tools", "go_js_wasm_exec.bat")
	if err := os.MkdirAll(filepath.Dir(helperPath), 0755); err != nil {
		t.Fatalf("mkdir helper dir: %v", err)
	}
	if err := os.WriteFile(helperPath, []byte("@echo off\n"), 0644); err != nil {
		t.Fatalf("write helper: %v", err)
	}

	got, err := resolveWasmTestExec(root)
	if err != nil {
		t.Fatalf("resolve wasm test exec from repo helper: %v", err)
	}
	if got != helperPath {
		t.Fatalf("expected repo helper %q, got %q", helperPath, got)
	}
}

func TestDetectBrowserWorkspaceUsesOverride(t *testing.T) {
	root := t.TempDir()
	browserWorkspace := filepath.Join(root, "enterprise-browser")
	if err := os.MkdirAll(browserWorkspace, 0755); err != nil {
		t.Fatalf("mkdir browser workspace: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(browserWorkspace, "playwrightgo"), 0755); err != nil {
		t.Fatalf("mkdir browser workspace playwrightgo package: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "gwc-runner.json"), []byte(`{"paths":{"browserWorkspace":"enterprise-browser"}}`), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	got := detectBrowserWorkspace(filepath.Join(root, "repo"), root)
	if got != browserWorkspace {
		t.Fatalf("expected browser workspace override %q, got %q", browserWorkspace, got)
	}
}

func TestResolveLauncherOverridePathResolvesRelativeEnvPath(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "configs", "runner.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte(`{"paths":{"generatedProjectRoot":"custom"}}`), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	t.Setenv(launcherOverrideEnvVar, filepath.Join("configs", "runner.json"))

	resolved, err := resolveLauncherOverridePath(root)
	if err != nil {
		t.Fatalf("resolve launcher override path: %v", err)
	}
	if resolved != configPath {
		t.Fatalf("expected resolved config path %q, got %q", configPath, resolved)
	}
}

func TestLoadLauncherOverridesForCurrentContextFallsBackToHomeConfig(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	configPath := filepath.Join(home, ".gwc", "runner.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		t.Fatalf("mkdir home config dir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte(`{"paths":{"browserWorkspace":"browser-suite"}}`), 0644); err != nil {
		t.Fatalf("write home config: %v", err)
	}

	originalGetwd := launcherConfigGetwd
	originalHomeDir := launcherConfigUserHomeDir
	t.Cleanup(func() {
		launcherConfigGetwd = originalGetwd
		launcherConfigUserHomeDir = originalHomeDir
	})
	launcherConfigGetwd = func() (string, error) { return "", os.ErrPermission }
	launcherConfigUserHomeDir = func() (string, error) { return home, nil }

	overrides, foundPath, ok, err := loadLauncherOverridesForCurrentContext()
	if err != nil {
		t.Fatalf("load launcher overrides for current context: %v", err)
	}
	if !ok {
		t.Fatal("expected home launcher override config to be discovered")
	}
	if foundPath != configPath {
		t.Fatalf("expected home config path %q, got %q", configPath, foundPath)
	}
	if overrides.Paths.BrowserWorkspace != "browser-suite" {
		t.Fatalf("expected browser workspace override, got %#v", overrides)
	}
}

func TestLoadLauncherOverridesReportsParseError(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "gwc-runner.json")
	if err := os.WriteFile(configPath, []byte(`{"paths":`), 0644); err != nil {
		t.Fatalf("write invalid config: %v", err)
	}
	t.Setenv(launcherOverrideEnvVar, configPath)

	_, _, _, err := loadLauncherOverrides("")
	if err == nil {
		t.Fatal("expected invalid launcher override json to fail")
	}
	if !strings.Contains(err.Error(), "parse launcher override file") {
		t.Fatalf("expected parse error wrapper, got %v", err)
	}
}

func TestResolveLauncherOverrideValueHandlesBlankAbsoluteAndRelativePaths(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "gwc-runner.json")
	absolutePath := filepath.Join(root, "absolute", "runner.json")

	blank, err := resolveLauncherOverrideValue(configPath, "   ")
	if err != nil {
		t.Fatalf("resolve blank override value: %v", err)
	}
	if blank != "" {
		t.Fatalf("expected blank override value to stay blank, got %q", blank)
	}

	abs, err := resolveLauncherOverrideValue(configPath, absolutePath)
	if err != nil {
		t.Fatalf("resolve absolute override value: %v", err)
	}
	if abs != filepath.Clean(absolutePath) {
		t.Fatalf("expected absolute override value %q, got %q", absolutePath, abs)
	}

	relative, err := resolveLauncherOverrideValue(configPath, filepath.Join("vendor", "wasm_exec.js"))
	if err != nil {
		t.Fatalf("resolve relative override value: %v", err)
	}
	wantRelative := filepath.Join(root, "vendor", "wasm_exec.js")
	if relative != wantRelative {
		t.Fatalf("expected relative override value %q, got %q", wantRelative, relative)
	}
}
