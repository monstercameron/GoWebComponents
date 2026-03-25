package runnerconfig

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveConfiguredPathReturnsNotFoundWhenNoConfigOrBlankOverride(t *testing.T) {
	root := t.TempDir()

	configured, ok, err := ResolveConfiguredPath(root, func(paths Paths) string {
		return paths.WorkspaceBuildRoot
	}, "workspaceBuildRoot", FS{})
	if err != nil {
		t.Fatalf("unexpected resolve configured path error: %v", err)
	}
	if ok || configured != "" {
		t.Fatalf("expected missing config to return empty path and ok=false, got path=%q ok=%t", configured, ok)
	}

	configPath := filepath.Join(root, "gwc-runner.json")
	if err := os.WriteFile(configPath, []byte(`{"paths":{"workspaceBuildRoot":"   "}}`), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	configured, ok, err = ResolveConfiguredPath(root, func(paths Paths) string {
		return paths.WorkspaceBuildRoot
	}, "workspaceBuildRoot", FS{})
	if err != nil {
		t.Fatalf("unexpected blank override resolve error: %v", err)
	}
	if ok || configured != "" {
		t.Fatalf("expected blank override to return empty path and ok=false, got path=%q ok=%t", configured, ok)
	}
}

func TestResolveConfiguredPathIncludesLabelOnResolveError(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "gwc-runner.json")
	if err := os.WriteFile(configPath, []byte("{"), 0644); err != nil {
		t.Fatalf("write invalid config: %v", err)
	}
	_, _, err := ResolveConfiguredPath(root, func(paths Paths) string {
		return paths.WorkspaceBuildRoot
	}, "workspaceBuildRoot", FS{})
	if err == nil || !strings.Contains(err.Error(), "parse launcher override file") {
		t.Fatalf("expected parse error passthrough, got %v", err)
	}
}

func TestResolveWorkspaceBuildRootFallbackBranches(t *testing.T) {
	buildRoot, err := ResolveWorkspaceBuildRoot(" ", FS{})
	if err != nil {
		t.Fatalf("resolve workspace build root fallback: %v", err)
	}
	if !filepath.IsAbs(buildRoot) {
		t.Fatalf("expected absolute fallback workspace build root, got %q", buildRoot)
	}
	if filepath.Base(buildRoot) != "bin" {
		t.Fatalf("expected fallback workspace build root to end in bin, got %q", buildRoot)
	}

	root := t.TempDir()
	buildRoot, err = ResolveWorkspaceBuildRoot(root, FS{})
	if err != nil {
		t.Fatalf("resolve workspace build root from explicit root: %v", err)
	}
	if want := filepath.Join(root, "bin"); buildRoot != want {
		t.Fatalf("expected default workspace build root %q, got %q", want, buildRoot)
	}
}

func TestResolveWorkspaceBuildPathPropagatesErrors(t *testing.T) {
	root := t.TempDir()
	t.Setenv(OverrideEnvVar, filepath.Join(root, "missing-runner.json"))
	if _, err := ResolveWorkspaceBuildPath(root, FS{}, "artifacts"); err == nil || !strings.Contains(err.Error(), "read launcher override file") {
		t.Fatalf("expected read error propagation from build path resolver, got %v", err)
	}
}

func TestResolveArtifactPathReturnsNotConfiguredWithoutOverride(t *testing.T) {
	root := t.TempDir()
	artifactPath, ok, err := ResolveArtifactPath(root, FS{}, "dist", "app.wasm")
	if err != nil {
		t.Fatalf("resolve artifact path without override: %v", err)
	}
	if ok || artifactPath != "" {
		t.Fatalf("expected unresolved artifact path when no override exists, got path=%q ok=%t", artifactPath, ok)
	}
}

func TestFindConfigInParentsHandlesEmptyAndInvalidPaths(t *testing.T) {
	if got := LocateConfigInParents("", FS{}); got != "" {
		t.Fatalf("expected empty cwd to return no config path, got %q", got)
	}
	if got := LocateConfigInParents(string([]byte{0}), FS{}); got != "" {
		t.Fatalf("expected invalid cwd to return no config path, got %q", got)
	}
}

func TestResolveConfigPathHomeFallbackErrorsAreIgnored(t *testing.T) {
	root := t.TempDir()
	resolved, err := ResolveConfigPath(root, FS{
		UserHomeDir: func() (string, error) {
			return "", errors.New("no home")
		},
	})
	if err != nil {
		t.Fatalf("expected home-dir error to be ignored, got %v", err)
	}
	if resolved != "" {
		t.Fatalf("expected no config path when nothing exists, got %q", resolved)
	}
}

func TestPathExistsBranches(t *testing.T) {
	fs := FS{
		Stat: func(path string) (os.FileInfo, error) {
			return nil, errors.New("missing")
		},
	}
	if fs.pathExists("") {
		t.Fatal("expected empty path to report missing")
	}
	if fs.pathExists("something") {
		t.Fatal("expected stat error path to report missing")
	}
}

func TestResolveConfigPathSupportsExplicitRelativeOverrideWithoutCWD(t *testing.T) {
	configDir := t.TempDir()
	configPath := filepath.Join(configDir, "runner.json")
	if err := os.WriteFile(configPath, []byte(`{"paths":{"artifactRoot":"artifacts"}}`), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	previousWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(configDir); err != nil {
		t.Fatalf("chdir to config dir: %v", err)
	}
	defer func() {
		_ = os.Chdir(previousWD)
	}()

	t.Setenv(OverrideEnvVar, "runner.json")
	resolved, err := ResolveConfigPath("", FS{})
	if err != nil {
		t.Fatalf("ResolveConfigPath with blank cwd: %v", err)
	}
	if resolved != configPath {
		t.Fatalf("expected config path %q, got %q", configPath, resolved)
	}
}

func TestResolveValueWithoutConfigPathAndLoadWithoutConfig(t *testing.T) {
	relative := filepath.Join("relative", "path")
	resolved, err := ResolveValue("", relative)
	if err != nil {
		t.Fatalf("ResolveValue without config path: %v", err)
	}
	if !filepath.IsAbs(resolved) || filepath.Base(resolved) != "path" {
		t.Fatalf("expected absolute resolved path ending in path, got %q", resolved)
	}

	root := t.TempDir()
	overrides, foundPath, ok, err := Load(root, FS{})
	if err != nil {
		t.Fatalf("Load without config: %v", err)
	}
	if ok || foundPath != "" || overrides != (Overrides{}) {
		t.Fatalf("expected zero-value load result with no config, got overrides=%#v path=%q ok=%t", overrides, foundPath, ok)
	}
}

func TestArtifactNamespaceFallsBackForFilesystemRoot(t *testing.T) {
	if got := GetArtifactNamespace(string(filepath.Separator)); got != "workspace" {
		t.Fatalf("expected filesystem root namespace to fall back to workspace, got %q", got)
	}
}

func TestResolveConfigPathAndLoadPropagateExplicitPathResolutionErrors(t *testing.T) {
	t.Setenv(OverrideEnvVar, "runner.json")
	invalidCWD := string([]byte{0})

	if _, err := ResolveConfigPath(invalidCWD, FS{}); err == nil || !strings.Contains(err.Error(), "resolve "+OverrideEnvVar+" path") {
		t.Fatalf("expected explicit override resolution error, got %v", err)
	}

	if _, _, _, err := Load(invalidCWD, FS{}); err == nil || !strings.Contains(err.Error(), "resolve "+OverrideEnvVar+" path") {
		t.Fatalf("expected load to propagate explicit override resolution error, got %v", err)
	}
}

func TestResolveValueAndConfiguredPathPropagateResolutionErrors(t *testing.T) {
	invalidPath := string([]byte{0})
	if _, err := ResolveValue("", invalidPath); err == nil {
		t.Fatal("expected ResolveValue to fail for invalid relative path input")
	}

	root := t.TempDir()
	configPath := filepath.Join(root, "gwc-runner.json")
	if err := os.WriteFile(configPath, []byte("{\"paths\":{\"workspaceBuildRoot\":\"\\u0000\"}}"), 0644); err != nil {
		t.Fatalf("write invalid path config: %v", err)
	}

	_, _, err := ResolveConfiguredPath(root, func(paths Paths) string {
		return paths.WorkspaceBuildRoot
	}, "workspaceBuildRoot", FS{})
	if err == nil || !strings.Contains(err.Error(), "resolve workspaceBuildRoot override") {
		t.Fatalf("expected wrapped resolve configured path error, got %v", err)
	}
}

