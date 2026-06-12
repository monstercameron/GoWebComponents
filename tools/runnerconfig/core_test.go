package runnerconfig

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOverridesJSONWireShapePreservesPathsObject(parseT *testing.T) {
	parseEncoded, parseErr := json.Marshal(Overrides{})
	if parseErr != nil {
		parseT.Fatalf("marshal runner overrides: %v", parseErr)
	}
	if parseText := string(parseEncoded); !strings.Contains(parseText, `"paths":{}`) {
		parseT.Fatalf("expected zero-value overrides to preserve paths object, got %s", parseText)
	}
}

func TestResolveConfigPathFindsParentConfig(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseConfigPath := filepath.Join(parseRoot, "gwc-runner.json")
	if parseErr := os.WriteFile(parseConfigPath, []byte(`{"paths":{"generatedProjectRoot":"generated"}}`), 0644); parseErr != nil {
		parseT.Fatalf("write config: %v", parseErr)
	}
	parseNested := filepath.Join(parseRoot, "apps", "demo")
	if parseErr2 := os.MkdirAll(parseNested, 0755); parseErr2 != nil {
		parseT.Fatalf("mkdir nested: %v", parseErr2)
	}

	parseResolved, parseErr3 := ResolveConfigPath(parseNested, FS{})
	if parseErr3 != nil {
		parseT.Fatalf("ResolveConfigPath: %v", parseErr3)
	}
	if parseResolved != parseConfigPath {
		parseT.Fatalf("expected parent config %q, got %q", parseConfigPath, parseResolved)
	}

	parseOverrides, parseFoundPath, parseOk, parseErr3 := Load(parseNested, FS{})
	if parseErr3 != nil {
		parseT.Fatalf("Load: %v", parseErr3)
	}
	if !parseOk || parseFoundPath != parseConfigPath {
		parseT.Fatalf("expected overrides from %q, got ok=%t path=%q", parseConfigPath, parseOk, parseFoundPath)
	}
	if parseOverrides.Paths.GeneratedProjectRoot != "generated" {
		parseT.Fatalf("expected decoded generatedProjectRoot, got %#v", parseOverrides.Paths)
	}
}

func TestResolveConfigPathSupportsExplicitRelativeOverride(parseT *testing.T) {
	parseCwd := parseT.TempDir()
	parseConfigDir := filepath.Join(parseCwd, "configs")
	if parseErr := os.MkdirAll(parseConfigDir, 0755); parseErr != nil {
		parseT.Fatalf("mkdir configs: %v", parseErr)
	}
	parseConfigPath := filepath.Join(parseConfigDir, "runner.json")
	if parseErr2 := os.WriteFile(parseConfigPath, []byte(`{"paths":{"artifactRoot":"artifacts"}}`), 0644); parseErr2 != nil {
		parseT.Fatalf("write config: %v", parseErr2)
	}
	parseT.Setenv(OverrideEnvVar, filepath.Join("configs", "runner.json"))

	parseResolved, parseErr3 := ResolveConfigPath(parseCwd, FS{})
	if parseErr3 != nil {
		parseT.Fatalf("ResolveConfigPath: %v", parseErr3)
	}
	if parseResolved != parseConfigPath {
		parseT.Fatalf("expected resolved env config %q, got %q", parseConfigPath, parseResolved)
	}
}

func TestResolveConfigPathFallsBackToHomeConfig(parseT *testing.T) {
	parseHome := parseT.TempDir()
	parseHomeConfig := filepath.Join(parseHome, ".gwc", "runner.json")
	if parseErr := os.MkdirAll(filepath.Dir(parseHomeConfig), 0755); parseErr != nil {
		parseT.Fatalf("mkdir home config dir: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(parseHomeConfig, []byte(`{"paths":{"browserWorkspace":"browser"}}`), 0644); parseErr2 != nil {
		parseT.Fatalf("write home config: %v", parseErr2)
	}

	parseFs := FS{
		UserHomeDir: func() (string, error) { return parseHome, nil },
	}
	parseResolved, parseErr3 := ResolveConfigPath("", parseFs)
	if parseErr3 != nil {
		parseT.Fatalf("ResolveConfigPath: %v", parseErr3)
	}
	if parseResolved != parseHomeConfig {
		parseT.Fatalf("expected home config %q, got %q", parseHomeConfig, parseResolved)
	}
}

func TestLoadReturnsReadAndParseErrors(parseT *testing.T) {
	parseCwd := parseT.TempDir()
	parseMissing := filepath.Join(parseCwd, "missing.json")
	parseT.Setenv(OverrideEnvVar, parseMissing)
	if _, _, _, parseErr := Load(parseCwd, FS{}); parseErr == nil || !strings.Contains(parseErr.Error(), "read launcher override file") {
		parseT.Fatalf("expected read error for missing config, got %v", parseErr)
	}

	parseInvalid := filepath.Join(parseCwd, "invalid.json")
	if parseErr2 := os.WriteFile(parseInvalid, []byte("{"), 0644); parseErr2 != nil {
		parseT.Fatalf("write invalid config: %v", parseErr2)
	}
	parseT.Setenv(OverrideEnvVar, parseInvalid)
	if _, _, _, parseErr3 := Load(parseCwd, FS{}); parseErr3 == nil || !strings.Contains(parseErr3.Error(), "parse launcher override file") {
		parseT.Fatalf("expected parse error for invalid config, got %v", parseErr3)
	}
}

func TestResolveValueAndConfiguredPathHelpers(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseConfigPath := filepath.Join(parseRoot, "gwc-runner.json")
	if parseErr := os.WriteFile(parseConfigPath, []byte(`{"paths":{"workspaceBuildRoot":"build","artifactRoot":"enterprise-artifacts"}}`), 0644); parseErr != nil {
		parseT.Fatalf("write config: %v", parseErr)
	}

	if parseGot, parseErr2 := ResolveValue(parseConfigPath, ""); parseErr2 != nil || parseGot != "" {
		parseT.Fatalf("expected blank ResolveValue result, got %q err=%v", parseGot, parseErr2)
	}
	parseAbsolute := filepath.Join(parseRoot, "abs", "path")
	if parseGot2, parseErr3 := ResolveValue(parseConfigPath, parseAbsolute); parseErr3 != nil || parseGot2 != filepath.Clean(parseAbsolute) {
		parseT.Fatalf("expected absolute ResolveValue path %q, got %q err=%v", parseAbsolute, parseGot2, parseErr3)
	}
	if parseGot3, parseErr4 := ResolveValue(parseConfigPath, filepath.Join("relative", "path")); parseErr4 != nil {
		parseT.Fatalf("ResolveValue relative path: %v", parseErr4)
	} else if parseWant := filepath.Join(parseRoot, "relative", "path"); parseGot3 != parseWant {
		parseT.Fatalf("expected resolved relative path %q, got %q", parseWant, parseGot3)
	}

	parseConfigured, parseOk, parseErr5 := ResolveConfiguredPath(parseRoot, func(parsePaths Paths) string { return parsePaths.WorkspaceBuildRoot }, "workspaceBuildRoot", FS{})
	if parseErr5 != nil {
		parseT.Fatalf("ResolveConfiguredPath: %v", parseErr5)
	}
	if !parseOk || parseConfigured != filepath.Join(parseRoot, "build") {
		parseT.Fatalf("expected configured workspace build path %q, got ok=%t path=%q", filepath.Join(parseRoot, "build"), parseOk, parseConfigured)
	}
}

func TestWorkspaceAndArtifactResolvers(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseConfigPath := filepath.Join(parseRoot, "gwc-runner.json")
	if parseErr := os.WriteFile(parseConfigPath, []byte(`{"paths":{"artifactRoot":"enterprise-artifacts","workspaceBuildRoot":"workspace-bin"}}`), 0644); parseErr != nil {
		parseT.Fatalf("write config: %v", parseErr)
	}

	buildRoot, parseErr2 := ResolveWorkspaceBuildRoot(parseRoot, FS{})
	if parseErr2 != nil {
		parseT.Fatalf("ResolveWorkspaceBuildRoot: %v", parseErr2)
	}
	if parseWant := filepath.Join(parseRoot, "workspace-bin"); buildRoot != parseWant {
		parseT.Fatalf("expected workspace build root %q, got %q", parseWant, buildRoot)
	}

	buildPath, parseErr2 := ResolveWorkspaceBuildPath(parseRoot, FS{}, "examples", "site")
	if parseErr2 != nil {
		parseT.Fatalf("ResolveWorkspaceBuildPath: %v", parseErr2)
	}
	if parseWant2 := filepath.Join(parseRoot, "workspace-bin", "examples", "site"); buildPath != parseWant2 {
		parseT.Fatalf("expected workspace build path %q, got %q", parseWant2, buildPath)
	}

	parseArtifactPath, parseOk, parseErr2 := ResolveArtifactPath(parseRoot, FS{}, "bin", "app.wasm")
	if parseErr2 != nil {
		parseT.Fatalf("ResolveArtifactPath: %v", parseErr2)
	}
	if !parseOk {
		parseT.Fatal("expected artifact path override to resolve")
	}
	if parseWant3 := filepath.Join(parseRoot, "enterprise-artifacts", filepath.Base(parseRoot), "bin", "app.wasm"); parseArtifactPath != parseWant3 {
		parseT.Fatalf("expected artifact path %q, got %q", parseWant3, parseArtifactPath)
	}

	if parseGot := GetArtifactNamespace(""); parseGot != "workspace" {
		parseT.Fatalf("expected empty root namespace to fall back to workspace, got %q", parseGot)
	}
	if parseGot2 := GetArtifactNamespace(parseRoot); parseGot2 != filepath.Base(parseRoot) {
		parseT.Fatalf("expected namespace %q, got %q", filepath.Base(parseRoot), parseGot2)
	}
}
