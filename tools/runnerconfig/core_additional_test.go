package runnerconfig

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveConfiguredPathReturnsNotFoundWhenNoConfigOrBlankOverride(parseT *testing.T) {
	parseRoot := parseT.TempDir()

	parseConfigured, parseOk, parseErr := ResolveConfiguredPath(parseRoot, func(parsePaths Paths) string {
		return parsePaths.WorkspaceBuildRoot
	}, "workspaceBuildRoot", FS{})
	if parseErr != nil {
		parseT.Fatalf("unexpected resolve configured path error: %v", parseErr)
	}
	if parseOk || parseConfigured != "" {
		parseT.Fatalf("expected missing config to return empty path and ok=false, got path=%q ok=%t", parseConfigured, parseOk)
	}

	parseConfigPath := filepath.Join(parseRoot, "gwc-runner.json")
	if parseErr2 := os.WriteFile(parseConfigPath, []byte(`{"paths":{"workspaceBuildRoot":"   "}}`), 0644); parseErr2 != nil {
		parseT.Fatalf("write config: %v", parseErr2)
	}
	parseConfigured, parseOk, parseErr = ResolveConfiguredPath(parseRoot, func(parsePaths2 Paths) string {
		return parsePaths2.WorkspaceBuildRoot
	}, "workspaceBuildRoot", FS{})
	if parseErr != nil {
		parseT.Fatalf("unexpected blank override resolve error: %v", parseErr)
	}
	if parseOk || parseConfigured != "" {
		parseT.Fatalf("expected blank override to return empty path and ok=false, got path=%q ok=%t", parseConfigured, parseOk)
	}
}

func TestResolveConfiguredPathIncludesLabelOnResolveError(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseConfigPath := filepath.Join(parseRoot, "gwc-runner.json")
	if parseErr := os.WriteFile(parseConfigPath, []byte("{"), 0644); parseErr != nil {
		parseT.Fatalf("write invalid config: %v", parseErr)
	}
	_, _, parseErr2 := ResolveConfiguredPath(parseRoot, func(parsePaths Paths) string {
		return parsePaths.WorkspaceBuildRoot
	}, "workspaceBuildRoot", FS{})
	if parseErr2 == nil || !strings.Contains(parseErr2.Error(), "parse launcher override file") {
		parseT.Fatalf("expected parse error passthrough, got %v", parseErr2)
	}
}

func TestResolveWorkspaceBuildRootFallbackBranches(parseT *testing.T) {
	buildRoot, parseErr := ResolveWorkspaceBuildRoot(" ", FS{})
	if parseErr != nil {
		parseT.Fatalf("resolve workspace build root fallback: %v", parseErr)
	}
	if !filepath.IsAbs(buildRoot) {
		parseT.Fatalf("expected absolute fallback workspace build root, got %q", buildRoot)
	}
	if filepath.Base(buildRoot) != "bin" {
		parseT.Fatalf("expected fallback workspace build root to end in bin, got %q", buildRoot)
	}

	parseRoot := parseT.TempDir()
	buildRoot, parseErr = ResolveWorkspaceBuildRoot(parseRoot, FS{})
	if parseErr != nil {
		parseT.Fatalf("resolve workspace build root from explicit root: %v", parseErr)
	}
	if parseWant := filepath.Join(parseRoot, "bin"); buildRoot != parseWant {
		parseT.Fatalf("expected default workspace build root %q, got %q", parseWant, buildRoot)
	}
}

func TestResolveWorkspaceBuildPathPropagatesErrors(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseT.Setenv(OverrideEnvVar, filepath.Join(parseRoot, "missing-runner.json"))
	if _, parseErr := ResolveWorkspaceBuildPath(parseRoot, FS{}, "artifacts"); parseErr == nil || !strings.Contains(parseErr.Error(), "read launcher override file") {
		parseT.Fatalf("expected read error propagation from build path resolver, got %v", parseErr)
	}
}

func TestResolveArtifactPathReturnsNotConfiguredWithoutOverride(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseArtifactPath, parseOk, parseErr := ResolveArtifactPath(parseRoot, FS{}, "dist", "app.wasm")
	if parseErr != nil {
		parseT.Fatalf("resolve artifact path without override: %v", parseErr)
	}
	if parseOk || parseArtifactPath != "" {
		parseT.Fatalf("expected unresolved artifact path when no override exists, got path=%q ok=%t", parseArtifactPath, parseOk)
	}
}

func TestFindConfigInParentsHandlesEmptyAndInvalidPaths(parseT *testing.T) {
	if parseGot := LocateConfigInParents("", FS{}); parseGot != "" {
		parseT.Fatalf("expected empty cwd to return no config path, got %q", parseGot)
	}
	if parseGot2 := LocateConfigInParents(string([]byte{0}), FS{}); parseGot2 != "" {
		parseT.Fatalf("expected invalid cwd to return no config path, got %q", parseGot2)
	}
}

func TestResolveConfigPathHomeFallbackErrorsAreIgnored(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseResolved, parseErr := ResolveConfigPath(parseRoot, FS{
		UserHomeDir: func() (string, error) {
			return "", errors.New("no home")
		},
	})
	if parseErr != nil {
		parseT.Fatalf("expected home-dir error to be ignored, got %v", parseErr)
	}
	if parseResolved != "" {
		parseT.Fatalf("expected no config path when nothing exists, got %q", parseResolved)
	}
}

func TestPathExistsBranches(parseT *testing.T) {
	parseFs := FS{
		Stat: func(parsePath string) (os.FileInfo, error) {
			return nil, errors.New("missing")
		},
	}
	if parseFs.pathExists("") {
		parseT.Fatal("expected empty path to report missing")
	}
	if parseFs.pathExists("something") {
		parseT.Fatal("expected stat error path to report missing")
	}
}

func TestResolveConfigPathSupportsExplicitRelativeOverrideWithoutCWD(parseT *testing.T) {
	parseConfigDir := parseT.TempDir()
	parseConfigPath := filepath.Join(parseConfigDir, "runner.json")
	if parseErr := os.WriteFile(parseConfigPath, []byte(`{"paths":{"artifactRoot":"artifacts"}}`), 0644); parseErr != nil {
		parseT.Fatalf("write config: %v", parseErr)
	}

	parsePreviousWD, parseErr2 := os.Getwd()
	if parseErr2 != nil {
		parseT.Fatalf("getwd: %v", parseErr2)
	}
	if parseErr3 := os.Chdir(parseConfigDir); parseErr3 != nil {
		parseT.Fatalf("chdir to config dir: %v", parseErr3)
	}
	defer func() {
		_ = os.Chdir(parsePreviousWD)
	}()

	parseT.Setenv(OverrideEnvVar, "runner.json")
	parseResolved, parseErr2 := ResolveConfigPath("", FS{})
	if parseErr2 != nil {
		parseT.Fatalf("ResolveConfigPath with blank cwd: %v", parseErr2)
	}
	if parseResolved != parseConfigPath {
		parseT.Fatalf("expected config path %q, got %q", parseConfigPath, parseResolved)
	}
}

func TestResolveValueWithoutConfigPathAndLoadWithoutConfig(parseT *testing.T) {
	parseRelative := filepath.Join("relative", "path")
	parseResolved, parseErr := ResolveValue("", parseRelative)
	if parseErr != nil {
		parseT.Fatalf("ResolveValue without config path: %v", parseErr)
	}
	if !filepath.IsAbs(parseResolved) || filepath.Base(parseResolved) != "path" {
		parseT.Fatalf("expected absolute resolved path ending in path, got %q", parseResolved)
	}

	parseRoot := parseT.TempDir()
	parseOverrides, parseFoundPath, parseOk, parseErr := Load(parseRoot, FS{})
	if parseErr != nil {
		parseT.Fatalf("Load without config: %v", parseErr)
	}
	if parseOk || parseFoundPath != "" || parseOverrides != (Overrides{}) {
		parseT.Fatalf("expected zero-value load result with no config, got overrides=%#v path=%q ok=%t", parseOverrides, parseFoundPath, parseOk)
	}
}

func TestArtifactNamespaceFallsBackForFilesystemRoot(parseT *testing.T) {
	if parseGot := GetArtifactNamespace(string(filepath.Separator)); parseGot != "workspace" {
		parseT.Fatalf("expected filesystem root namespace to fall back to workspace, got %q", parseGot)
	}
}

func TestResolveConfigPathAndLoadPropagateExplicitPathResolutionErrors(parseT *testing.T) {
	parseT.Setenv(OverrideEnvVar, "runner.json")
	parseInvalidCWD := string([]byte{0})

	if _, parseErr := ResolveConfigPath(parseInvalidCWD, FS{}); parseErr == nil || !strings.Contains(parseErr.Error(), "resolve "+OverrideEnvVar+" path") {
		parseT.Fatalf("expected explicit override resolution error, got %v", parseErr)
	}

	if _, _, _, parseErr2 := Load(parseInvalidCWD, FS{}); parseErr2 == nil || !strings.Contains(parseErr2.Error(), "resolve "+OverrideEnvVar+" path") {
		parseT.Fatalf("expected load to propagate explicit override resolution error, got %v", parseErr2)
	}
}

func TestResolveValueAndConfiguredPathPropagateResolutionErrors(parseT *testing.T) {
	parseInvalidPath := string([]byte{0})
	if _, parseErr := ResolveValue("", parseInvalidPath); parseErr == nil {
		parseT.Fatal("expected ResolveValue to fail for invalid relative path input")
	}

	parseRoot := parseT.TempDir()
	parseConfigPath := filepath.Join(parseRoot, "gwc-runner.json")
	if parseErr2 := os.WriteFile(parseConfigPath, []byte("{\"paths\":{\"workspaceBuildRoot\":\"\\u0000\"}}"), 0644); parseErr2 != nil {
		parseT.Fatalf("write invalid path config: %v", parseErr2)
	}

	_, _, parseErr3 := ResolveConfiguredPath(parseRoot, func(parsePaths Paths) string {
		return parsePaths.WorkspaceBuildRoot
	}, "workspaceBuildRoot", FS{})
	if parseErr3 == nil || !strings.Contains(parseErr3.Error(), "resolve workspaceBuildRoot override") {
		parseT.Fatalf("expected wrapped resolve configured path error, got %v", parseErr3)
	}
}
