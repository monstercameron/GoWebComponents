package runnerconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveConfigPathFindsParentConfig(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "gwc-runner.json")
	if err := os.WriteFile(configPath, []byte(`{"paths":{"generatedProjectRoot":"generated"}}`), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	nested := filepath.Join(root, "apps", "demo")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}

	resolved, err := ResolveConfigPath(nested, FS{})
	if err != nil {
		t.Fatalf("ResolveConfigPath: %v", err)
	}
	if resolved != configPath {
		t.Fatalf("expected parent config %q, got %q", configPath, resolved)
	}

	overrides, foundPath, ok, err := Load(nested, FS{})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !ok || foundPath != configPath {
		t.Fatalf("expected overrides from %q, got ok=%t path=%q", configPath, ok, foundPath)
	}
	if overrides.Paths.GeneratedProjectRoot != "generated" {
		t.Fatalf("expected decoded generatedProjectRoot, got %#v", overrides.Paths)
	}
}

func TestResolveConfigPathSupportsExplicitRelativeOverride(t *testing.T) {
	cwd := t.TempDir()
	configDir := filepath.Join(cwd, "configs")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("mkdir configs: %v", err)
	}
	configPath := filepath.Join(configDir, "runner.json")
	if err := os.WriteFile(configPath, []byte(`{"paths":{"artifactRoot":"artifacts"}}`), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	t.Setenv(OverrideEnvVar, filepath.Join("configs", "runner.json"))

	resolved, err := ResolveConfigPath(cwd, FS{})
	if err != nil {
		t.Fatalf("ResolveConfigPath: %v", err)
	}
	if resolved != configPath {
		t.Fatalf("expected resolved env config %q, got %q", configPath, resolved)
	}
}

func TestResolveConfigPathFallsBackToHomeConfig(t *testing.T) {
	home := t.TempDir()
	homeConfig := filepath.Join(home, ".gwc", "runner.json")
	if err := os.MkdirAll(filepath.Dir(homeConfig), 0755); err != nil {
		t.Fatalf("mkdir home config dir: %v", err)
	}
	if err := os.WriteFile(homeConfig, []byte(`{"paths":{"browserWorkspace":"browser"}}`), 0644); err != nil {
		t.Fatalf("write home config: %v", err)
	}

	fs := FS{
		UserHomeDir: func() (string, error) { return home, nil },
	}
	resolved, err := ResolveConfigPath("", fs)
	if err != nil {
		t.Fatalf("ResolveConfigPath: %v", err)
	}
	if resolved != homeConfig {
		t.Fatalf("expected home config %q, got %q", homeConfig, resolved)
	}
}

func TestLoadReturnsReadAndParseErrors(t *testing.T) {
	cwd := t.TempDir()
	missing := filepath.Join(cwd, "missing.json")
	t.Setenv(OverrideEnvVar, missing)
	if _, _, _, err := Load(cwd, FS{}); err == nil || !strings.Contains(err.Error(), "read launcher override file") {
		t.Fatalf("expected read error for missing config, got %v", err)
	}

	invalid := filepath.Join(cwd, "invalid.json")
	if err := os.WriteFile(invalid, []byte("{"), 0644); err != nil {
		t.Fatalf("write invalid config: %v", err)
	}
	t.Setenv(OverrideEnvVar, invalid)
	if _, _, _, err := Load(cwd, FS{}); err == nil || !strings.Contains(err.Error(), "parse launcher override file") {
		t.Fatalf("expected parse error for invalid config, got %v", err)
	}
}

func TestResolveValueAndConfiguredPathHelpers(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "gwc-runner.json")
	if err := os.WriteFile(configPath, []byte(`{"paths":{"workspaceBuildRoot":"build","artifactRoot":"enterprise-artifacts"}}`), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if got, err := ResolveValue(configPath, ""); err != nil || got != "" {
		t.Fatalf("expected blank ResolveValue result, got %q err=%v", got, err)
	}
	absolute := filepath.Join(root, "abs", "path")
	if got, err := ResolveValue(configPath, absolute); err != nil || got != filepath.Clean(absolute) {
		t.Fatalf("expected absolute ResolveValue path %q, got %q err=%v", absolute, got, err)
	}
	if got, err := ResolveValue(configPath, filepath.Join("relative", "path")); err != nil {
		t.Fatalf("ResolveValue relative path: %v", err)
	} else if want := filepath.Join(root, "relative", "path"); got != want {
		t.Fatalf("expected resolved relative path %q, got %q", want, got)
	}

	configured, ok, err := ResolveConfiguredPath(root, func(paths Paths) string { return paths.WorkspaceBuildRoot }, "workspaceBuildRoot", FS{})
	if err != nil {
		t.Fatalf("ResolveConfiguredPath: %v", err)
	}
	if !ok || configured != filepath.Join(root, "build") {
		t.Fatalf("expected configured workspace build path %q, got ok=%t path=%q", filepath.Join(root, "build"), ok, configured)
	}
}

func TestWorkspaceAndArtifactResolvers(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "gwc-runner.json")
	if err := os.WriteFile(configPath, []byte(`{"paths":{"artifactRoot":"enterprise-artifacts","workspaceBuildRoot":"workspace-bin"}}`), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	buildRoot, err := ResolveWorkspaceBuildRoot(root, FS{})
	if err != nil {
		t.Fatalf("ResolveWorkspaceBuildRoot: %v", err)
	}
	if want := filepath.Join(root, "workspace-bin"); buildRoot != want {
		t.Fatalf("expected workspace build root %q, got %q", want, buildRoot)
	}

	buildPath, err := ResolveWorkspaceBuildPath(root, FS{}, "examples", "site")
	if err != nil {
		t.Fatalf("ResolveWorkspaceBuildPath: %v", err)
	}
	if want := filepath.Join(root, "workspace-bin", "examples", "site"); buildPath != want {
		t.Fatalf("expected workspace build path %q, got %q", want, buildPath)
	}

	artifactPath, ok, err := ResolveArtifactPath(root, FS{}, "bin", "app.wasm")
	if err != nil {
		t.Fatalf("ResolveArtifactPath: %v", err)
	}
	if !ok {
		t.Fatal("expected artifact path override to resolve")
	}
	if want := filepath.Join(root, "enterprise-artifacts", filepath.Base(root), "bin", "app.wasm"); artifactPath != want {
		t.Fatalf("expected artifact path %q, got %q", want, artifactPath)
	}

	if got := GetArtifactNamespace(""); got != "workspace" {
		t.Fatalf("expected empty root namespace to fall back to workspace, got %q", got)
	}
	if got := GetArtifactNamespace(root); got != filepath.Base(root) {
		t.Fatalf("expected namespace %q, got %q", filepath.Base(root), got)
	}
}

