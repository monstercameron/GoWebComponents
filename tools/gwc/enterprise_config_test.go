package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveLauncherEnterpriseConfigAppliesLayeringOrder(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseProjectRoot := filepath.Join(parseRoot, "project")
	if parseErr := os.MkdirAll(parseProjectRoot, 0755); parseErr != nil {
		parseT.Fatalf("mkdir project root: %v", parseErr)
	}

	parseHomeRoot := filepath.Join(parseRoot, "home")
	if parseErr2 := os.MkdirAll(filepath.Join(parseHomeRoot, ".gwc"), 0755); parseErr2 != nil {
		parseT.Fatalf("mkdir home policy root: %v", parseErr2)
	}
	parseOrgPolicyPath := filepath.Join(parseHomeRoot, ".gwc", "policy-pack.json")
	if parseErr3 := os.WriteFile(parseOrgPolicyPath, []byte(`{
  "policy": {
    "requiredTestLanes": ["unit"],
    "requireReleaseBudgets": true
  },
  "plugins": [
    { "name": "org-plugin", "path": "plugins/org.exe", "capabilities": ["verify_check"] }
  ]
}`), 0644); parseErr3 != nil {
		parseT.Fatalf("write org policy pack: %v", parseErr3)
	}
	if parseErr4 := os.MkdirAll(filepath.Join(parseHomeRoot, ".gwc", "plugins"), 0755); parseErr4 != nil {
		parseT.Fatalf("mkdir org plugin dir: %v", parseErr4)
	}
	if parseErr5 := os.WriteFile(filepath.Join(parseHomeRoot, ".gwc", "plugins", "org.exe"), []byte(""), 0644); parseErr5 != nil {
		parseT.Fatalf("write org plugin placeholder: %v", parseErr5)
	}

	parseProjectConfigPath := filepath.Join(parseProjectRoot, "gwc-runner.json")
	if parseErr6 := os.WriteFile(parseProjectConfigPath, []byte(`{
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
}`), 0644); parseErr6 != nil {
		parseT.Fatalf("write project config: %v", parseErr6)
	}
	if parseErr7 := os.MkdirAll(filepath.Join(parseProjectRoot, "hooks"), 0755); parseErr7 != nil {
		parseT.Fatalf("mkdir project hooks dir: %v", parseErr7)
	}
	if parseErr8 := os.WriteFile(filepath.Join(parseProjectRoot, "hooks", "pre-release.exe"), []byte(""), 0644); parseErr8 != nil {
		parseT.Fatalf("write project hook placeholder: %v", parseErr8)
	}

	parseCliPolicyPath := filepath.Join(parseRoot, "cli-policy.json")
	if parseErr9 := os.WriteFile(parseCliPolicyPath, []byte(`{
  "policy": {
    "requiredTestLanes": ["release"],
    "requireReleaseBudgets": true,
    "requiredReleaseCompression": "gzip+brotli",
    "releaseBinaryPattern": "^cli-[a-z0-9._-]+\\.wasm$"
  },
  "plugins": [
    { "name": "org-plugin", "path": "cli/org.exe", "capabilities": ["release_validator"] }
  ]
}`), 0644); parseErr9 != nil {
		parseT.Fatalf("write cli policy pack: %v", parseErr9)
	}
	if parseErr10 := os.MkdirAll(filepath.Join(parseRoot, "cli"), 0755); parseErr10 != nil {
		parseT.Fatalf("mkdir cli policy plugin dir: %v", parseErr10)
	}
	if parseErr11 := os.WriteFile(filepath.Join(parseRoot, "cli", "org.exe"), []byte(""), 0644); parseErr11 != nil {
		parseT.Fatalf("write cli plugin placeholder: %v", parseErr11)
	}

	parseOriginalHomeDir := launcherConfigUserHomeDir
	parseT.Cleanup(func() { launcherConfigUserHomeDir = parseOriginalHomeDir })
	launcherConfigUserHomeDir = func() (string, error) { return parseHomeRoot, nil }
	parseT.Setenv(launcherPolicyPackEnvVar, "")

	parseLayered, parseErr12 := resolveLauncherEnterpriseConfig(parseProjectRoot, launcherGlobalCLIOptions{
		PolicyPackPath: parseCliPolicyPath,
	})
	if parseErr12 != nil {
		parseT.Fatalf("resolve enterprise config layering: %v", parseErr12)
	}

	if !parseLayered.Sources.FrameworkDefaults {
		parseT.Fatal("expected framework defaults source flag")
	}
	if parseLayered.Sources.OrganizationPolicyPath != parseCliPolicyPath {
		parseT.Fatalf("expected org policy path to use CLI override %q, got %q", parseCliPolicyPath, parseLayered.Sources.OrganizationPolicyPath)
	}
	if parseLayered.Sources.CLIOverridePolicyPack != parseCliPolicyPath {
		parseT.Fatalf("expected CLI policy source to record override path %q, got %q", parseCliPolicyPath, parseLayered.Sources.CLIOverridePolicyPack)
	}
	if parseLayered.Sources.ProjectConfigPath != parseProjectConfigPath {
		parseT.Fatalf("expected project config source %q, got %q", parseProjectConfigPath, parseLayered.Sources.ProjectConfigPath)
	}

	parseLanes := strings.Join(parseLayered.Effective.Policy.RequiredTestLanes, ",")
	for _, parseRequired := range []string{"browser", "release"} {
		if !strings.Contains(parseLanes, parseRequired) {
			parseT.Fatalf("expected required test lanes to include %q, got %q", parseRequired, parseLanes)
		}
	}
	if parseLayered.Effective.Policy.RequireReleaseBudgets == nil || !*parseLayered.Effective.Policy.RequireReleaseBudgets {
		parseT.Fatalf("expected requireReleaseBudgets to remain true from lower layer, got %#v", parseLayered.Effective.Policy.RequireReleaseBudgets)
	}
	if parseLayered.Effective.Policy.RequiredReleaseCompression != "gzip+brotli" {
		parseT.Fatalf("expected required release compression from CLI policy pack, got %q", parseLayered.Effective.Policy.RequiredReleaseCompression)
	}
	if parseLayered.Effective.Policy.ReleaseBinaryPattern != "^project-[a-z0-9._-]+\\.wasm$" {
		parseT.Fatalf("expected release binary pattern from project layer precedence, got %q", parseLayered.Effective.Policy.ReleaseBinaryPattern)
	}
	if len(parseLayered.Effective.Hooks["pre-release"]) != 1 {
		parseT.Fatalf("expected project hook to be merged, got %#v", parseLayered.Effective.Hooks)
	}
	if len(parseLayered.Effective.Plugins) != 1 {
		parseT.Fatalf("expected plugin replacement by name across layers, got %#v", parseLayered.Effective.Plugins)
	}
	if !strings.Contains(parseLayered.Effective.Plugins[0].Path, filepath.Join("cli", "org.exe")) {
		parseT.Fatalf("expected plugin path to resolve from CLI policy pack, got %#v", parseLayered.Effective.Plugins[0])
	}
}

func TestResolveLauncherEnterpriseConfigLoadsOrganizationPolicyFromEnv(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseProjectRoot := filepath.Join(parseRoot, "project")
	if parseErr := os.MkdirAll(parseProjectRoot, 0755); parseErr != nil {
		parseT.Fatalf("mkdir project: %v", parseErr)
	}
	parseEnvPolicy := filepath.Join(parseRoot, "env-policy.json")
	if parseErr2 := os.WriteFile(parseEnvPolicy, []byte(`{"policy":{"requiredTestLanes":["hydration"]}}`), 0644); parseErr2 != nil {
		parseT.Fatalf("write env policy: %v", parseErr2)
	}
	parseT.Setenv(launcherPolicyPackEnvVar, parseEnvPolicy)

	parseLayered, parseErr3 := resolveLauncherEnterpriseConfig(parseProjectRoot, launcherGlobalCLIOptions{})
	if parseErr3 != nil {
		parseT.Fatalf("resolve enterprise config: %v", parseErr3)
	}
	if parseLayered.Sources.OrganizationPolicyPath != parseEnvPolicy {
		parseT.Fatalf("expected env policy path %q, got %q", parseEnvPolicy, parseLayered.Sources.OrganizationPolicyPath)
	}
	if len(parseLayered.Effective.Policy.RequiredTestLanes) != 1 || parseLayered.Effective.Policy.RequiredTestLanes[0] != "hydration" {
		parseT.Fatalf("expected env policy lane merge, got %#v", parseLayered.Effective.Policy.RequiredTestLanes)
	}
}

func TestResolveLauncherEnterpriseConfigErrorsForMissingExplicitPolicyPack(parseT *testing.T) {
	parseProjectRoot := parseT.TempDir()
	_, parseErr := resolveLauncherEnterpriseConfig(parseProjectRoot, launcherGlobalCLIOptions{
		PolicyPackPath: filepath.Join(parseProjectRoot, "missing-policy.json"),
	})
	if parseErr == nil || !strings.Contains(parseErr.Error(), "CLI policy pack path does not exist") {
		parseT.Fatalf("expected missing explicit policy-pack error, got %v", parseErr)
	}
}

func TestResolveLauncherEnterpriseConfigCanDisableHooksAndPluginsFromCLI(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseProjectRoot := filepath.Join(parseRoot, "project")
	if parseErr := os.MkdirAll(parseProjectRoot, 0755); parseErr != nil {
		parseT.Fatalf("mkdir project: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(filepath.Join(parseProjectRoot, "gwc-runner.json"), []byte(`{
  "enterprise": {
    "hooks": { "pre-build": [{ "path": "hooks/pre-build.exe" }] },
    "plugins": [{ "name": "project-plugin", "path": "plugins/project.exe", "capabilities": ["verify_check"] }]
  }
}`), 0644); parseErr2 != nil {
		parseT.Fatalf("write project config: %v", parseErr2)
	}
	if parseErr3 := os.MkdirAll(filepath.Join(parseProjectRoot, "hooks"), 0755); parseErr3 != nil {
		parseT.Fatalf("mkdir hooks dir: %v", parseErr3)
	}
	if parseErr4 := os.WriteFile(filepath.Join(parseProjectRoot, "hooks", "pre-build.exe"), []byte(""), 0644); parseErr4 != nil {
		parseT.Fatalf("write hook placeholder: %v", parseErr4)
	}
	if parseErr5 := os.MkdirAll(filepath.Join(parseProjectRoot, "plugins"), 0755); parseErr5 != nil {
		parseT.Fatalf("mkdir plugin dir: %v", parseErr5)
	}
	if parseErr6 := os.WriteFile(filepath.Join(parseProjectRoot, "plugins", "project.exe"), []byte(""), 0644); parseErr6 != nil {
		parseT.Fatalf("write plugin placeholder: %v", parseErr6)
	}

	parseLayered, parseErr7 := resolveLauncherEnterpriseConfig(parseProjectRoot, launcherGlobalCLIOptions{
		DisableHooks:   true,
		DisablePlugins: true,
	})
	if parseErr7 != nil {
		parseT.Fatalf("resolve enterprise config: %v", parseErr7)
	}
	if len(parseLayered.Effective.Hooks) != 0 {
		parseT.Fatalf("expected CLI hook disable to clear hooks, got %#v", parseLayered.Effective.Hooks)
	}
	if len(parseLayered.Effective.Plugins) != 0 {
		parseT.Fatalf("expected CLI plugin disable to clear plugins, got %#v", parseLayered.Effective.Plugins)
	}
}

func TestLoadEnterpriseConfigRejectsUnknownPluginCapability(parseT *testing.T) {
	parseConfigPath := filepath.Join(parseT.TempDir(), "gwc-runner.json")
	if parseErr := os.WriteFile(parseConfigPath, []byte(`{
  "enterprise": {
    "plugins": [
      { "name": "bad-plugin", "path": "plugins/bad.exe", "capabilities": ["rewrite_core_runtime"] }
    ]
  }
}`), 0644); parseErr != nil {
		parseT.Fatalf("write config: %v", parseErr)
	}
	if parseErr2 := os.MkdirAll(filepath.Join(filepath.Dir(parseConfigPath), "plugins"), 0755); parseErr2 != nil {
		parseT.Fatalf("mkdir plugins dir: %v", parseErr2)
	}
	if parseErr3 := os.WriteFile(filepath.Join(filepath.Dir(parseConfigPath), "plugins", "bad.exe"), []byte(""), 0644); parseErr3 != nil {
		parseT.Fatalf("write plugin placeholder: %v", parseErr3)
	}

	_, parseErr4 := loadEnterpriseConfigFromPath(parseConfigPath)
	if parseErr4 == nil || !strings.Contains(parseErr4.Error(), "unsupported capability") {
		parseT.Fatalf("expected unsupported plugin capability error, got %v", parseErr4)
	}
}

func TestLoadEnterpriseConfigNormalizesPolicyFields(parseT *testing.T) {
	parseConfigPath := filepath.Join(parseT.TempDir(), "policy-pack.json")
	if parseErr := os.WriteFile(parseConfigPath, []byte(`{
  "policy": {
    "requiredTestLanes": ["go-native", "hydrate"],
    "requiredReleaseCompression": "both"
  }
}`), 0644); parseErr != nil {
		parseT.Fatalf("write config: %v", parseErr)
	}

	parseConfig, parseErr2 := loadEnterpriseConfigFromPath(parseConfigPath)
	if parseErr2 != nil {
		parseT.Fatalf("load config: %v", parseErr2)
	}
	if parseGot := strings.Join(parseConfig.Policy.RequiredTestLanes, ","); parseGot != "unit,hydration" {
		parseT.Fatalf("expected normalized policy lanes unit,hydration, got %q", parseGot)
	}
	if parseConfig.Policy.RequiredReleaseCompression != "gzip+brotli" {
		parseT.Fatalf("expected normalized release compression gzip+brotli, got %q", parseConfig.Policy.RequiredReleaseCompression)
	}
}

func TestLoadEnterpriseConfigRejectsInvalidPolicyRegexAndCompression(parseT *testing.T) {
	parseT.Run("invalid release binary pattern", func(parseT2 *testing.T) {
		parseConfigPath := filepath.Join(parseT2.TempDir(), "policy-pack.json")
		if parseErr := os.WriteFile(parseConfigPath, []byte(`{"policy":{"releaseBinaryPattern":"["}}`), 0644); parseErr != nil {
			parseT2.Fatalf("write config: %v", parseErr)
		}
		_, parseErr2 := loadEnterpriseConfigFromPath(parseConfigPath)
		if parseErr2 == nil || !strings.Contains(parseErr2.Error(), "compile release binary pattern") {
			parseT2.Fatalf("expected release binary pattern error, got %v", parseErr2)
		}
	})

	parseT.Run("invalid required release compression", func(parseT3 *testing.T) {
		parseConfigPath2 := filepath.Join(parseT3.TempDir(), "policy-pack.json")
		if parseErr3 := os.WriteFile(parseConfigPath2, []byte(`{"policy":{"requiredReleaseCompression":"mystery"}}`), 0644); parseErr3 != nil {
			parseT3.Fatalf("write config: %v", parseErr3)
		}
		_, parseErr4 := loadEnterpriseConfigFromPath(parseConfigPath2)
		if parseErr4 == nil || !strings.Contains(parseErr4.Error(), "normalize required release compression") {
			parseT3.Fatalf("expected required release compression error, got %v", parseErr4)
		}
	})
}

func TestResolveLauncherEnterpriseConfigMergesSecuritySettings(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseProjectRoot := filepath.Join(parseRoot, "project")
	if parseErr := os.MkdirAll(parseProjectRoot, 0755); parseErr != nil {
		parseT.Fatalf("mkdir project root: %v", parseErr)
	}

	parseOrgPolicyPath := filepath.Join(parseRoot, "org-policy.json")
	if parseErr2 := os.WriteFile(parseOrgPolicyPath, []byte(`{
  "security": {
    "allowUntrustedExtensions": false,
    "allowedExecutableRoots": ["org-hooks"],
    "inheritedEnvAllowlist": ["CI"],
    "inheritedEnvDenylist": ["TOKEN"]
  }
}`), 0644); parseErr2 != nil {
		parseT.Fatalf("write org policy: %v", parseErr2)
	}
	if parseErr3 := os.WriteFile(filepath.Join(parseProjectRoot, "gwc-runner.json"), []byte(`{
  "enterprise": {
    "security": {
      "allowedExecutableRoots": ["project-hooks"],
      "inheritedEnvAllowlist": ["PATH"],
      "inheritedEnvDenylist": ["PASSWORD"]
    }
  }
}`), 0644); parseErr3 != nil {
		parseT.Fatalf("write project config: %v", parseErr3)
	}

	parseLayered, parseErr4 := resolveLauncherEnterpriseConfig(parseProjectRoot, launcherGlobalCLIOptions{
		PolicyPackPath: parseOrgPolicyPath,
	})
	if parseErr4 != nil {
		parseT.Fatalf("resolve enterprise config: %v", parseErr4)
	}
	if parseLayered.Effective.Security.AllowUntrustedExtensions == nil || *parseLayered.Effective.Security.AllowUntrustedExtensions {
		parseT.Fatalf("expected merged security allowUntrustedExtensions=false, got %#v", parseLayered.Effective.Security.AllowUntrustedExtensions)
	}
	if len(parseLayered.Effective.Security.AllowedExecutableRoots) != 2 {
		parseT.Fatalf("expected merged allowed executable roots, got %#v", parseLayered.Effective.Security.AllowedExecutableRoots)
	}
	if parseGot := strings.Join(parseLayered.Effective.Security.InheritedEnvAllowlist, ","); parseGot != "CI,PATH" {
		parseT.Fatalf("expected merged inherited env allowlist CI,PATH, got %q", parseGot)
	}
	if parseGot2 := strings.Join(parseLayered.Effective.Security.InheritedEnvDenylist, ","); parseGot2 != "TOKEN,PASSWORD" {
		parseT.Fatalf("expected merged inherited env denylist TOKEN,PASSWORD, got %q", parseGot2)
	}
}
