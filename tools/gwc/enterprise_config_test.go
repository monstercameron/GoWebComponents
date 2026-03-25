package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveLauncherEnterpriseConfigAppliesLayeringOrder(t *testing.T) {
	root := t.TempDir()
	projectRoot := filepath.Join(root, "project")
	if err := os.MkdirAll(projectRoot, 0755); err != nil {
		t.Fatalf("mkdir project root: %v", err)
	}

	homeRoot := filepath.Join(root, "home")
	if err := os.MkdirAll(filepath.Join(homeRoot, ".gwc"), 0755); err != nil {
		t.Fatalf("mkdir home policy root: %v", err)
	}
	orgPolicyPath := filepath.Join(homeRoot, ".gwc", "policy-pack.json")
	if err := os.WriteFile(orgPolicyPath, []byte(`{
  "policy": {
    "requiredTestLanes": ["unit"],
    "requireReleaseBudgets": true
  },
  "plugins": [
    { "name": "org-plugin", "path": "plugins/org.exe", "capabilities": ["verify_check"] }
  ]
}`), 0644); err != nil {
		t.Fatalf("write org policy pack: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(homeRoot, ".gwc", "plugins"), 0755); err != nil {
		t.Fatalf("mkdir org plugin dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(homeRoot, ".gwc", "plugins", "org.exe"), []byte(""), 0644); err != nil {
		t.Fatalf("write org plugin placeholder: %v", err)
	}

	projectConfigPath := filepath.Join(projectRoot, "gwc-runner.json")
	if err := os.WriteFile(projectConfigPath, []byte(`{
  "paths": { "artifactRoot": "artifacts" },
  "enterprise": {
    "policy": {
      "requiredTestLanes": ["browser"],
      "releaseBinaryPattern": "^project-[a-z0-9._-]+\\.wasm$"
    },
    "hooks": {
      "pre-release": [{ "name": "project-pre-release", "path": "hooks/pre-release.exe" }]
    }
  }
}`), 0644); err != nil {
		t.Fatalf("write project config: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectRoot, "hooks"), 0755); err != nil {
		t.Fatalf("mkdir project hooks dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "hooks", "pre-release.exe"), []byte(""), 0644); err != nil {
		t.Fatalf("write project hook placeholder: %v", err)
	}

	cliPolicyPath := filepath.Join(root, "cli-policy.json")
	if err := os.WriteFile(cliPolicyPath, []byte(`{
  "policy": {
    "requiredTestLanes": ["release"],
    "requireReleaseBudgets": true,
    "requiredReleaseCompression": "gzip+brotli",
    "releaseBinaryPattern": "^cli-[a-z0-9._-]+\\.wasm$"
  },
  "plugins": [
    { "name": "org-plugin", "path": "cli/org.exe", "capabilities": ["release_validator"] }
  ]
}`), 0644); err != nil {
		t.Fatalf("write cli policy pack: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "cli"), 0755); err != nil {
		t.Fatalf("mkdir cli policy plugin dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "cli", "org.exe"), []byte(""), 0644); err != nil {
		t.Fatalf("write cli plugin placeholder: %v", err)
	}

	originalHomeDir := launcherConfigUserHomeDir
	t.Cleanup(func() { launcherConfigUserHomeDir = originalHomeDir })
	launcherConfigUserHomeDir = func() (string, error) { return homeRoot, nil }
	t.Setenv(launcherPolicyPackEnvVar, "")

	layered, err := resolveLauncherEnterpriseConfig(projectRoot, launcherGlobalCLIOptions{
		PolicyPackPath: cliPolicyPath,
	})
	if err != nil {
		t.Fatalf("resolve enterprise config layering: %v", err)
	}

	if !layered.Sources.FrameworkDefaults {
		t.Fatal("expected framework defaults source flag")
	}
	if layered.Sources.OrganizationPolicyPath != cliPolicyPath {
		t.Fatalf("expected org policy path to use CLI override %q, got %q", cliPolicyPath, layered.Sources.OrganizationPolicyPath)
	}
	if layered.Sources.CLIOverridePolicyPack != cliPolicyPath {
		t.Fatalf("expected CLI policy source to record override path %q, got %q", cliPolicyPath, layered.Sources.CLIOverridePolicyPack)
	}
	if layered.Sources.ProjectConfigPath != projectConfigPath {
		t.Fatalf("expected project config source %q, got %q", projectConfigPath, layered.Sources.ProjectConfigPath)
	}

	lanes := strings.Join(layered.Effective.Policy.RequiredTestLanes, ",")
	for _, required := range []string{"browser", "release"} {
		if !strings.Contains(lanes, required) {
			t.Fatalf("expected required test lanes to include %q, got %q", required, lanes)
		}
	}
	if layered.Effective.Policy.RequireReleaseBudgets == nil || !*layered.Effective.Policy.RequireReleaseBudgets {
		t.Fatalf("expected requireReleaseBudgets to remain true from lower layer, got %#v", layered.Effective.Policy.RequireReleaseBudgets)
	}
	if layered.Effective.Policy.RequiredReleaseCompression != "gzip+brotli" {
		t.Fatalf("expected required release compression from CLI policy pack, got %q", layered.Effective.Policy.RequiredReleaseCompression)
	}
	if layered.Effective.Policy.ReleaseBinaryPattern != "^project-[a-z0-9._-]+\\.wasm$" {
		t.Fatalf("expected release binary pattern from project layer precedence, got %q", layered.Effective.Policy.ReleaseBinaryPattern)
	}
	if len(layered.Effective.Hooks["pre-release"]) != 1 {
		t.Fatalf("expected project hook to be merged, got %#v", layered.Effective.Hooks)
	}
	if len(layered.Effective.Plugins) != 1 {
		t.Fatalf("expected plugin replacement by name across layers, got %#v", layered.Effective.Plugins)
	}
	if !strings.Contains(layered.Effective.Plugins[0].Path, filepath.Join("cli", "org.exe")) {
		t.Fatalf("expected plugin path to resolve from CLI policy pack, got %#v", layered.Effective.Plugins[0])
	}
}

func TestResolveLauncherEnterpriseConfigLoadsOrganizationPolicyFromEnv(t *testing.T) {
	root := t.TempDir()
	projectRoot := filepath.Join(root, "project")
	if err := os.MkdirAll(projectRoot, 0755); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	envPolicy := filepath.Join(root, "env-policy.json")
	if err := os.WriteFile(envPolicy, []byte(`{"policy":{"requiredTestLanes":["hydration"]}}`), 0644); err != nil {
		t.Fatalf("write env policy: %v", err)
	}
	t.Setenv(launcherPolicyPackEnvVar, envPolicy)

	layered, err := resolveLauncherEnterpriseConfig(projectRoot, launcherGlobalCLIOptions{})
	if err != nil {
		t.Fatalf("resolve enterprise config: %v", err)
	}
	if layered.Sources.OrganizationPolicyPath != envPolicy {
		t.Fatalf("expected env policy path %q, got %q", envPolicy, layered.Sources.OrganizationPolicyPath)
	}
	if len(layered.Effective.Policy.RequiredTestLanes) != 1 || layered.Effective.Policy.RequiredTestLanes[0] != "hydration" {
		t.Fatalf("expected env policy lane merge, got %#v", layered.Effective.Policy.RequiredTestLanes)
	}
}

func TestResolveLauncherEnterpriseConfigErrorsForMissingExplicitPolicyPack(t *testing.T) {
	projectRoot := t.TempDir()
	_, err := resolveLauncherEnterpriseConfig(projectRoot, launcherGlobalCLIOptions{
		PolicyPackPath: filepath.Join(projectRoot, "missing-policy.json"),
	})
	if err == nil || !strings.Contains(err.Error(), "CLI policy pack path does not exist") {
		t.Fatalf("expected missing explicit policy-pack error, got %v", err)
	}
}

func TestResolveLauncherEnterpriseConfigCanDisableHooksAndPluginsFromCLI(t *testing.T) {
	root := t.TempDir()
	projectRoot := filepath.Join(root, "project")
	if err := os.MkdirAll(projectRoot, 0755); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "gwc-runner.json"), []byte(`{
  "enterprise": {
    "hooks": { "pre-build": [{ "path": "hooks/pre-build.exe" }] },
    "plugins": [{ "name": "project-plugin", "path": "plugins/project.exe", "capabilities": ["verify_check"] }]
  }
}`), 0644); err != nil {
		t.Fatalf("write project config: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectRoot, "hooks"), 0755); err != nil {
		t.Fatalf("mkdir hooks dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "hooks", "pre-build.exe"), []byte(""), 0644); err != nil {
		t.Fatalf("write hook placeholder: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectRoot, "plugins"), 0755); err != nil {
		t.Fatalf("mkdir plugin dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "plugins", "project.exe"), []byte(""), 0644); err != nil {
		t.Fatalf("write plugin placeholder: %v", err)
	}

	layered, err := resolveLauncherEnterpriseConfig(projectRoot, launcherGlobalCLIOptions{
		DisableHooks:   true,
		DisablePlugins: true,
	})
	if err != nil {
		t.Fatalf("resolve enterprise config: %v", err)
	}
	if len(layered.Effective.Hooks) != 0 {
		t.Fatalf("expected CLI hook disable to clear hooks, got %#v", layered.Effective.Hooks)
	}
	if len(layered.Effective.Plugins) != 0 {
		t.Fatalf("expected CLI plugin disable to clear plugins, got %#v", layered.Effective.Plugins)
	}
}

func TestLoadEnterpriseConfigRejectsUnknownPluginCapability(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "gwc-runner.json")
	if err := os.WriteFile(configPath, []byte(`{
  "enterprise": {
    "plugins": [
      { "name": "bad-plugin", "path": "plugins/bad.exe", "capabilities": ["rewrite_core_runtime"] }
    ]
  }
}`), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(filepath.Dir(configPath), "plugins"), 0755); err != nil {
		t.Fatalf("mkdir plugins dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(filepath.Dir(configPath), "plugins", "bad.exe"), []byte(""), 0644); err != nil {
		t.Fatalf("write plugin placeholder: %v", err)
	}

	_, err := loadEnterpriseConfigFromPath(configPath)
	if err == nil || !strings.Contains(err.Error(), "unsupported capability") {
		t.Fatalf("expected unsupported plugin capability error, got %v", err)
	}
}

func TestLoadEnterpriseConfigNormalizesPolicyFields(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "policy-pack.json")
	if err := os.WriteFile(configPath, []byte(`{
  "policy": {
    "requiredTestLanes": ["go-native", "hydrate"],
    "requiredReleaseCompression": "both"
  }
}`), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	config, err := loadEnterpriseConfigFromPath(configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if got := strings.Join(config.Policy.RequiredTestLanes, ","); got != "unit,hydration" {
		t.Fatalf("expected normalized policy lanes unit,hydration, got %q", got)
	}
	if config.Policy.RequiredReleaseCompression != "gzip+brotli" {
		t.Fatalf("expected normalized release compression gzip+brotli, got %q", config.Policy.RequiredReleaseCompression)
	}
}

func TestLoadEnterpriseConfigRejectsInvalidPolicyRegexAndCompression(t *testing.T) {
	t.Run("invalid release binary pattern", func(t *testing.T) {
		configPath := filepath.Join(t.TempDir(), "policy-pack.json")
		if err := os.WriteFile(configPath, []byte(`{"policy":{"releaseBinaryPattern":"["}}`), 0644); err != nil {
			t.Fatalf("write config: %v", err)
		}
		_, err := loadEnterpriseConfigFromPath(configPath)
		if err == nil || !strings.Contains(err.Error(), "compile release binary pattern") {
			t.Fatalf("expected release binary pattern error, got %v", err)
		}
	})

	t.Run("invalid required release compression", func(t *testing.T) {
		configPath := filepath.Join(t.TempDir(), "policy-pack.json")
		if err := os.WriteFile(configPath, []byte(`{"policy":{"requiredReleaseCompression":"mystery"}}`), 0644); err != nil {
			t.Fatalf("write config: %v", err)
		}
		_, err := loadEnterpriseConfigFromPath(configPath)
		if err == nil || !strings.Contains(err.Error(), "normalize required release compression") {
			t.Fatalf("expected required release compression error, got %v", err)
		}
	})
}

func TestResolveLauncherEnterpriseConfigMergesSecuritySettings(t *testing.T) {
	root := t.TempDir()
	projectRoot := filepath.Join(root, "project")
	if err := os.MkdirAll(projectRoot, 0755); err != nil {
		t.Fatalf("mkdir project root: %v", err)
	}

	orgPolicyPath := filepath.Join(root, "org-policy.json")
	if err := os.WriteFile(orgPolicyPath, []byte(`{
  "security": {
    "allowUntrustedExtensions": false,
    "allowedExecutableRoots": ["org-hooks"],
    "inheritedEnvAllowlist": ["CI"],
    "inheritedEnvDenylist": ["TOKEN"]
  }
}`), 0644); err != nil {
		t.Fatalf("write org policy: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "gwc-runner.json"), []byte(`{
  "enterprise": {
    "security": {
      "allowedExecutableRoots": ["project-hooks"],
      "inheritedEnvAllowlist": ["PATH"],
      "inheritedEnvDenylist": ["PASSWORD"]
    }
  }
}`), 0644); err != nil {
		t.Fatalf("write project config: %v", err)
	}

	layered, err := resolveLauncherEnterpriseConfig(projectRoot, launcherGlobalCLIOptions{
		PolicyPackPath: orgPolicyPath,
	})
	if err != nil {
		t.Fatalf("resolve enterprise config: %v", err)
	}
	if layered.Effective.Security.AllowUntrustedExtensions == nil || *layered.Effective.Security.AllowUntrustedExtensions {
		t.Fatalf("expected merged security allowUntrustedExtensions=false, got %#v", layered.Effective.Security.AllowUntrustedExtensions)
	}
	if len(layered.Effective.Security.AllowedExecutableRoots) != 2 {
		t.Fatalf("expected merged allowed executable roots, got %#v", layered.Effective.Security.AllowedExecutableRoots)
	}
	if got := strings.Join(layered.Effective.Security.InheritedEnvAllowlist, ","); got != "CI,PATH" {
		t.Fatalf("expected merged inherited env allowlist CI,PATH, got %q", got)
	}
	if got := strings.Join(layered.Effective.Security.InheritedEnvDenylist, ","); got != "TOKEN,PASSWORD" {
		t.Fatalf("expected merged inherited env denylist TOKEN,PASSWORD, got %q", got)
	}
}
