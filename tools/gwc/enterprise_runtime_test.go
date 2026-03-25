package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseLauncherGlobalCLIOptions(t *testing.T) {
	options, remaining, err := parseLauncherGlobalCLIOptions([]string{"-policy-pack", "org/policy.json", "-no-hooks", "-no-plugins", "verify", "-json"})
	if err != nil {
		t.Fatalf("parse global options: %v", err)
	}
	if options.PolicyPackPath != "org/policy.json" || !options.DisableHooks || !options.DisablePlugins {
		t.Fatalf("unexpected parsed options: %#v", options)
	}
	if got := strings.Join(remaining, " "); got != "verify -json" {
		t.Fatalf("unexpected remaining args: %q", got)
	}

	if _, _, err := parseLauncherGlobalCLIOptions([]string{"-unknown", "verify"}); err == nil || !strings.Contains(err.Error(), "unknown global flag") {
		t.Fatalf("expected unknown global flag error, got %v", err)
	}
}

func TestLauncherRunExecutesPreAndPostHooks(t *testing.T) {
	projectRoot := t.TempDir()
	hookDir := filepath.Join(projectRoot, "hooks")
	if err := os.MkdirAll(hookDir, 0755); err != nil {
		t.Fatalf("mkdir hook dir: %v", err)
	}
	preHookPath := filepath.Join(hookDir, "pre-test.exe")
	postHookPath := filepath.Join(hookDir, "post-test.exe")
	if err := os.WriteFile(preHookPath, []byte(""), 0644); err != nil {
		t.Fatalf("write pre hook placeholder: %v", err)
	}
	if err := os.WriteFile(postHookPath, []byte(""), 0644); err != nil {
		t.Fatalf("write post hook placeholder: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "gwc-runner.json"), []byte(`{
  "enterprise": {
    "hooks": {
      "pre-test": [{ "name": "pre-test-hook", "path": "hooks/pre-test.exe" }],
      "post-test": [{ "name": "post-test-hook", "path": "hooks/post-test.exe" }]
    }
  }
}`), 0644); err != nil {
		t.Fatalf("write project config: %v", err)
	}

	originalGetwd := launcherConfigGetwd
	originalHomeDir := launcherConfigUserHomeDir
	originalRunTestCommand := runTestCommand
	originalRunHookProcess := launcherRunHookProcess
	t.Cleanup(func() {
		launcherConfigGetwd = originalGetwd
		launcherConfigUserHomeDir = originalHomeDir
		runTestCommand = originalRunTestCommand
		launcherRunHookProcess = originalRunHookProcess
	})
	launcherConfigGetwd = func() (string, error) { return projectRoot, nil }
	launcherConfigUserHomeDir = func() (string, error) { return filepath.Join(projectRoot, "home"), nil }

	invokedHooks := []string{}
	launcherRunHookProcess = func(path string, args []string, env []string, stdin []byte, timeout time.Duration) (string, string, error) {
		invokedHooks = append(invokedHooks, filepath.Base(path))
		var request launcherHookInvocationRequest
		if err := json.Unmarshal(stdin, &request); err != nil {
			t.Fatalf("parse hook request payload: %v", err)
		}
		if request.Command != "test" {
			t.Fatalf("expected hook request command test, got %#v", request)
		}
		return `{"ok":true}`, "", nil
	}

	testCommandCalled := false
	runTestCommand = func(l launcher, args []string) error {
		testCommandCalled = true
		return nil
	}

	if err := (launcher{}).run([]string{"test", "-json"}); err != nil {
		t.Fatalf("run test with hooks: %v", err)
	}
	if !testCommandCalled {
		t.Fatal("expected test command to be dispatched")
	}
	if got := strings.Join(invokedHooks, ","); got != "pre-test.exe,post-test.exe" {
		t.Fatalf("expected pre/post hook order, got %q", got)
	}
}

func TestLauncherRunStopsOnPreHookFailure(t *testing.T) {
	projectRoot := t.TempDir()
	hookDir := filepath.Join(projectRoot, "hooks")
	if err := os.MkdirAll(hookDir, 0755); err != nil {
		t.Fatalf("mkdir hook dir: %v", err)
	}
	preHookPath := filepath.Join(hookDir, "pre-build.exe")
	if err := os.WriteFile(preHookPath, []byte(""), 0644); err != nil {
		t.Fatalf("write pre hook placeholder: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "gwc-runner.json"), []byte(`{
  "enterprise": {
    "hooks": {
      "pre-build": [{ "name": "pre-build-hook", "path": "hooks/pre-build.exe" }]
    }
  }
}`), 0644); err != nil {
		t.Fatalf("write project config: %v", err)
	}

	originalGetwd := launcherConfigGetwd
	originalHomeDir := launcherConfigUserHomeDir
	originalRunBuildCommand := runBuildCommand
	originalRunHookProcess := launcherRunHookProcess
	t.Cleanup(func() {
		launcherConfigGetwd = originalGetwd
		launcherConfigUserHomeDir = originalHomeDir
		runBuildCommand = originalRunBuildCommand
		launcherRunHookProcess = originalRunHookProcess
	})
	launcherConfigGetwd = func() (string, error) { return projectRoot, nil }
	launcherConfigUserHomeDir = func() (string, error) { return filepath.Join(projectRoot, "home"), nil }
	launcherRunHookProcess = func(path string, args []string, env []string, stdin []byte, timeout time.Duration) (string, string, error) {
		return "", "", errors.New("blocked by policy")
	}

	buildCalled := false
	runBuildCommand = func(l launcher, args []string) error {
		buildCalled = true
		return nil
	}

	err := (launcher{}).run([]string{"build", "-json"})
	if err == nil || !strings.Contains(err.Error(), "pre-build-hook") {
		t.Fatalf("expected pre-hook failure attribution, got %v", err)
	}
	if buildCalled {
		t.Fatal("expected build command to be skipped after pre-hook failure")
	}
}

func TestLauncherRunContinuesWhenHookAllowsFailure(t *testing.T) {
	projectRoot := t.TempDir()
	hookDir := filepath.Join(projectRoot, "hooks")
	if err := os.MkdirAll(hookDir, 0755); err != nil {
		t.Fatalf("mkdir hook dir: %v", err)
	}
	preHookPath := filepath.Join(hookDir, "pre-verify.exe")
	if err := os.WriteFile(preHookPath, []byte(""), 0644); err != nil {
		t.Fatalf("write pre hook placeholder: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "gwc-runner.json"), []byte(`{
  "enterprise": {
    "hooks": {
      "pre-verify": [{ "name": "pre-verify-hook", "path": "hooks/pre-verify.exe", "allowFailure": true }]
    }
  }
}`), 0644); err != nil {
		t.Fatalf("write project config: %v", err)
	}

	originalGetwd := launcherConfigGetwd
	originalHomeDir := launcherConfigUserHomeDir
	originalRunVerifyCommand := runVerifyCommand
	originalRunHookProcess := launcherRunHookProcess
	t.Cleanup(func() {
		launcherConfigGetwd = originalGetwd
		launcherConfigUserHomeDir = originalHomeDir
		runVerifyCommand = originalRunVerifyCommand
		launcherRunHookProcess = originalRunHookProcess
	})
	launcherConfigGetwd = func() (string, error) { return projectRoot, nil }
	launcherConfigUserHomeDir = func() (string, error) { return filepath.Join(projectRoot, "home"), nil }
	launcherRunHookProcess = func(path string, args []string, env []string, stdin []byte, timeout time.Duration) (string, string, error) {
		return "", "", errors.New("temporary hook failure")
	}

	verifyCalled := false
	runVerifyCommand = func(l launcher, args []string) error {
		verifyCalled = true
		return nil
	}

	if err := (launcher{}).run([]string{"verify", "-json"}); err != nil {
		t.Fatalf("expected verify command to continue with allowFailure hook, got %v", err)
	}
	if !verifyCalled {
		t.Fatal("expected verify command to run when allowFailure hook fails")
	}
}

func TestRunLauncherPluginsForCapabilityUsesStructuredJSONIO(t *testing.T) {
	pluginPath := filepath.Join(t.TempDir(), "verify-plugin.exe")
	if err := os.WriteFile(pluginPath, []byte(""), 0644); err != nil {
		t.Fatalf("write plugin placeholder: %v", err)
	}

	originalPlugins := launcherActiveEnterpriseConfig
	originalPluginProcess := launcherRunPluginProcess
	t.Cleanup(func() {
		launcherActiveEnterpriseConfig = originalPlugins
		launcherRunPluginProcess = originalPluginProcess
	})
	launcherActiveEnterpriseConfig = launcherEnterpriseConfig{
		Plugins: []launcherExecutablePlugin{
			{
				Name:         "verify-plugin",
				Path:         pluginPath,
				Capabilities: []string{"verify_check"},
			},
		},
	}

	launcherRunPluginProcess = func(path string, args []string, env []string, stdin []byte, timeout time.Duration) (string, string, error) {
		var request launcherPluginInvocationRequest
		if err := json.Unmarshal(stdin, &request); err != nil {
			t.Fatalf("parse plugin request payload: %v", err)
		}
		if request.SchemaVersion != "gwc-plugin-v1" || request.Capability != "verify_check" || request.Command != "verify" {
			t.Fatalf("unexpected plugin request payload: %#v", request)
		}
		return `{"ok":true,"verifyChecks":[{"name":"policy","passed":true}]}`, "", nil
	}

	results, err := runLauncherPluginsForCapability("verify_check", "verify", []string{"-json"}, `C:\repo`, launcherEnterpriseConfigSources{FrameworkDefaults: true})
	if err != nil {
		t.Fatalf("run plugins for verify_check: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected one plugin execution result, got %#v", results)
	}
	if err := enforcePluginChecks(results, "verify_check"); err != nil {
		t.Fatalf("expected verify checks to pass, got %v", err)
	}
}

func TestRunVerifyFailsWhenPluginVerifyCheckFails(t *testing.T) {
	pluginPath := filepath.Join(t.TempDir(), "verify-plugin.exe")
	if err := os.WriteFile(pluginPath, []byte(""), 0644); err != nil {
		t.Fatalf("write plugin placeholder: %v", err)
	}

	originalPlugins := launcherActiveEnterpriseConfig
	originalSources := launcherActiveEnterpriseSources
	originalPluginProcess := launcherRunPluginProcess
	t.Cleanup(func() {
		launcherActiveEnterpriseConfig = originalPlugins
		launcherActiveEnterpriseSources = originalSources
		launcherRunPluginProcess = originalPluginProcess
	})
	launcherActiveEnterpriseConfig = launcherEnterpriseConfig{
		Plugins: []launcherExecutablePlugin{
			{
				Name:         "verify-plugin",
				Path:         pluginPath,
				Capabilities: []string{"verify_check"},
			},
		},
	}
	launcherActiveEnterpriseSources = launcherEnterpriseConfigSources{FrameworkDefaults: true}
	launcherRunPluginProcess = func(path string, args []string, env []string, stdin []byte, timeout time.Duration) (string, string, error) {
		return `{"verifyChecks":[{"name":"org-verify","passed":false,"summary":"verify lane policy failed"}]}`, "", nil
	}

	err := (launcher{}).runVerify(nil)
	if err == nil || !strings.Contains(err.Error(), "org-verify") {
		t.Fatalf("expected plugin verify check failure attribution, got %v", err)
	}
}

func TestRunReleaseFailsWhenPluginReleaseValidatorFails(t *testing.T) {
	pluginPath := filepath.Join(t.TempDir(), "release-plugin.exe")
	if err := os.WriteFile(pluginPath, []byte(""), 0644); err != nil {
		t.Fatalf("write plugin placeholder: %v", err)
	}

	originalPlugins := launcherActiveEnterpriseConfig
	originalSources := launcherActiveEnterpriseSources
	originalPluginProcess := launcherRunPluginProcess
	t.Cleanup(func() {
		launcherActiveEnterpriseConfig = originalPlugins
		launcherActiveEnterpriseSources = originalSources
		launcherRunPluginProcess = originalPluginProcess
	})
	launcherActiveEnterpriseConfig = launcherEnterpriseConfig{
		Plugins: []launcherExecutablePlugin{
			{
				Name:         "release-plugin",
				Path:         pluginPath,
				Capabilities: []string{"release_validator"},
			},
		},
	}
	launcherActiveEnterpriseSources = launcherEnterpriseConfigSources{FrameworkDefaults: true}
	launcherRunPluginProcess = func(path string, args []string, env []string, stdin []byte, timeout time.Duration) (string, string, error) {
		return `{"releaseValidators":[{"name":"org-release-validator","passed":false,"summary":"artifact naming policy failed"}]}`, "", nil
	}

	err := (launcher{}).runRelease(nil)
	if err == nil || !strings.Contains(err.Error(), "org-release-validator") {
		t.Fatalf("expected plugin release validator failure attribution, got %v", err)
	}
}

func TestPrintLauncherExtensionReportIncludesTrustAndDiscovery(t *testing.T) {
	var buffer bytes.Buffer
	originalWriter := launcherExtensionReportWriter
	t.Cleanup(func() { launcherExtensionReportWriter = originalWriter })
	launcherExtensionReportWriter = &buffer

	layered := launcherEnterpriseLayeredConfig{
		Effective: launcherEnterpriseConfig{
			Hooks: map[string][]launcherExecutableHook{
				"pre-verify": {
					{Name: "pre-verify-1", Trusted: true},
					{Name: "pre-verify-2", Trusted: false},
				},
			},
			Plugins: []launcherExecutablePlugin{
				{Name: "org-plugin", Trusted: true, Capabilities: []string{"verify_check"}, Path: `C:\plugins\org.exe`},
			},
		},
		Sources: launcherEnterpriseConfigSources{
			FrameworkDefaults:      true,
			OrganizationPolicyPath: `C:\policy\org-policy.json`,
			ProjectConfigPath:      `C:\repo\gwc-runner.json`,
		},
	}

	printLauncherExtensionReport("verify", layered)
	output := buffer.String()
	for _, expected := range []string{
		"GWC extension report",
		"command: verify",
		"org policy: C:\\policy\\org-policy.json",
		"project config: C:\\repo\\gwc-runner.json",
		"pre-verify: 2 loaded (1 trusted, 1 untrusted)",
		"org-plugin [trusted] caps=verify_check",
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("expected extension report output to contain %q, got:\n%s", expected, output)
		}
	}
}

func TestLauncherRunInvokesExtensionReportForPlainTextCommands(t *testing.T) {
	originalGetwd := launcherConfigGetwd
	originalHome := launcherConfigUserHomeDir
	originalReportPrinter := launcherPrintExtensionReport
	originalRunDoctorCommand := runDoctorCommand
	t.Cleanup(func() {
		launcherConfigGetwd = originalGetwd
		launcherConfigUserHomeDir = originalHome
		launcherPrintExtensionReport = originalReportPrinter
		runDoctorCommand = originalRunDoctorCommand
	})
	workspace := t.TempDir()
	launcherConfigGetwd = func() (string, error) { return workspace, nil }
	launcherConfigUserHomeDir = func() (string, error) { return filepath.Join(workspace, "home"), nil }

	reportCalled := false
	launcherPrintExtensionReport = func(command string, layered launcherEnterpriseLayeredConfig) {
		reportCalled = true
		if command != "doctor" {
			t.Fatalf("expected extension report command doctor, got %q", command)
		}
	}
	runDoctorCommand = func(l launcher, args []string) error { return nil }

	if err := (launcher{}).run([]string{"doctor"}); err != nil {
		t.Fatalf("run doctor: %v", err)
	}
	if !reportCalled {
		t.Fatal("expected launcher run to invoke extension report printer")
	}
}

func TestLauncherRunInvokesExtensionReport(t *testing.T) {
	TestLauncherRunInvokesExtensionReportForPlainTextCommands(t)
}

func TestLauncherRunSkipsExtensionReportForJSONCommands(t *testing.T) {
	originalGetwd := launcherConfigGetwd
	originalHome := launcherConfigUserHomeDir
	originalReportPrinter := launcherPrintExtensionReport
	originalRunDoctorCommand := runDoctorCommand
	t.Cleanup(func() {
		launcherConfigGetwd = originalGetwd
		launcherConfigUserHomeDir = originalHome
		launcherPrintExtensionReport = originalReportPrinter
		runDoctorCommand = originalRunDoctorCommand
	})
	workspace := t.TempDir()
	launcherConfigGetwd = func() (string, error) { return workspace, nil }
	launcherConfigUserHomeDir = func() (string, error) { return filepath.Join(workspace, "home"), nil }

	reportCalled := false
	launcherPrintExtensionReport = func(command string, layered launcherEnterpriseLayeredConfig) {
		reportCalled = true
	}
	runDoctorCommand = func(l launcher, args []string) error { return nil }

	if err := (launcher{}).run([]string{"doctor", "-json"}); err != nil {
		t.Fatalf("run doctor: %v", err)
	}
	if reportCalled {
		t.Fatal("expected launcher run to skip extension report printer for json commands")
	}
}

func TestLauncherRunBlocksUntrustedExtensionsWhenPolicyDisallows(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectRoot, "hooks"), 0755); err != nil {
		t.Fatalf("mkdir hooks: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "hooks", "pre-build.exe"), []byte(""), 0644); err != nil {
		t.Fatalf("write hook placeholder: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "gwc-runner.json"), []byte(`{
  "enterprise": {
    "security": { "allowUntrustedExtensions": false },
    "hooks": { "pre-build": [{ "name": "pre-build-hook", "path": "hooks/pre-build.exe" }] }
  }
}`), 0644); err != nil {
		t.Fatalf("write project config: %v", err)
	}

	originalGetwd := launcherConfigGetwd
	originalHome := launcherConfigUserHomeDir
	originalRunBuild := runBuildCommand
	t.Cleanup(func() {
		launcherConfigGetwd = originalGetwd
		launcherConfigUserHomeDir = originalHome
		runBuildCommand = originalRunBuild
	})
	launcherConfigGetwd = func() (string, error) { return projectRoot, nil }
	launcherConfigUserHomeDir = func() (string, error) { return filepath.Join(projectRoot, "home"), nil }

	buildCalled := false
	runBuildCommand = func(l launcher, args []string) error {
		buildCalled = true
		return nil
	}

	err := (launcher{}).run([]string{"build"})
	if err == nil || !strings.Contains(err.Error(), "untrusted and blocked") {
		t.Fatalf("expected untrusted extension policy failure, got %v", err)
	}
	if buildCalled {
		t.Fatal("expected command dispatch to be blocked by security policy")
	}
}

func TestLauncherRunBlocksExtensionsOutsideAllowedRoots(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectRoot, "hooks"), 0755); err != nil {
		t.Fatalf("mkdir hooks: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "hooks", "pre-build.exe"), []byte(""), 0644); err != nil {
		t.Fatalf("write hook placeholder: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectRoot, "trusted-hooks"), 0755); err != nil {
		t.Fatalf("mkdir trusted hooks: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "gwc-runner.json"), []byte(`{
  "enterprise": {
    "security": { "allowedExecutableRoots": ["trusted-hooks"] },
    "hooks": { "pre-build": [{ "name": "pre-build-hook", "path": "hooks/pre-build.exe", "trusted": true }] }
  }
}`), 0644); err != nil {
		t.Fatalf("write project config: %v", err)
	}

	originalGetwd := launcherConfigGetwd
	originalHome := launcherConfigUserHomeDir
	t.Cleanup(func() {
		launcherConfigGetwd = originalGetwd
		launcherConfigUserHomeDir = originalHome
	})
	launcherConfigGetwd = func() (string, error) { return projectRoot, nil }
	launcherConfigUserHomeDir = func() (string, error) { return filepath.Join(projectRoot, "home"), nil }

	err := (launcher{}).run([]string{"build"})
	if err == nil || !strings.Contains(err.Error(), "outside allowed executable roots") {
		t.Fatalf("expected allowed-root security failure, got %v", err)
	}
}

func TestBuildLauncherExtensionEnvSupportsSecretRefsAndInheritedFiltering(t *testing.T) {
	t.Setenv("GWC_ALLOWED_INHERITED", "allowed")
	t.Setenv("GWC_SECRET_TOKEN", "secret-value")
	t.Setenv("GWC_BLOCKED", "blocked")

	env, err := buildLauncherExtensionEnv(true, map[string]string{
		"API_TOKEN": "${ENV:GWC_SECRET_TOKEN}",
	}, launcherEnterpriseSecurityPolicy{
		InheritedEnvAllowlist: []string{"GWC_ALLOWED_INHERITED", "GWC_BLOCKED"},
		InheritedEnvDenylist:  []string{"GWC_BLOCKED"},
	})
	if err != nil {
		t.Fatalf("build launcher extension env: %v", err)
	}
	envMap := map[string]string{}
	for _, pair := range env {
		name, value, ok := strings.Cut(pair, "=")
		if !ok {
			continue
		}
		envMap[name] = value
	}
	if envMap["GWC_ALLOWED_INHERITED"] != "allowed" {
		t.Fatalf("expected inherited allowlisted env variable, got %#v", envMap)
	}
	if _, exists := envMap["GWC_BLOCKED"]; exists {
		t.Fatalf("expected denied env variable to be filtered, got %#v", envMap)
	}
	if envMap["API_TOKEN"] != "secret-value" {
		t.Fatalf("expected secret env reference to resolve, got %#v", envMap)
	}
}

func TestBuildLauncherExtensionEnvFailsForMissingSecretRef(t *testing.T) {
	_, err := buildLauncherExtensionEnv(false, map[string]string{
		"API_TOKEN": "${ENV:GWC_NOT_SET}",
	}, launcherEnterpriseSecurityPolicy{})
	if err == nil || !strings.Contains(err.Error(), "is not set") {
		t.Fatalf("expected missing secret env reference error, got %v", err)
	}
}

func TestLauncherEnterpriseVerifyIntegrationRespectsPolicyPrecedenceAndDiagnostics(t *testing.T) {
	root := t.TempDir()
	projectRoot := filepath.Join(root, "project")
	if err := os.MkdirAll(filepath.Join(projectRoot, "plugins"), 0755); err != nil {
		t.Fatalf("mkdir project plugins: %v", err)
	}
	projectPluginPath := filepath.Join(projectRoot, "plugins", "project-verify.exe")
	if err := os.WriteFile(projectPluginPath, []byte(""), 0644); err != nil {
		t.Fatalf("write project plugin placeholder: %v", err)
	}
	orgPolicyPath := filepath.Join(root, "org-policy-pack.json")
	if err := os.MkdirAll(filepath.Join(root, "plugins"), 0755); err != nil {
		t.Fatalf("mkdir org plugins: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "plugins", "org-verify.exe"), []byte(""), 0644); err != nil {
		t.Fatalf("write org plugin placeholder: %v", err)
	}
	if err := os.WriteFile(orgPolicyPath, []byte(`{
  "plugins": [
    { "name": "org-verify", "path": "plugins/org-verify.exe", "capabilities": ["verify_check"] }
  ]
}`), 0644); err != nil {
		t.Fatalf("write org policy pack: %v", err)
	}
	projectConfigPath := filepath.Join(projectRoot, "gwc-runner.json")
	if err := os.WriteFile(projectConfigPath, []byte(`{
  "enterprise": {
    "plugins": [
      { "name": "org-verify", "path": "plugins/project-verify.exe", "capabilities": ["verify_check"] }
    ]
  }
}`), 0644); err != nil {
		t.Fatalf("write project config: %v", err)
	}

	originalGetwd := launcherConfigGetwd
	originalHome := launcherConfigUserHomeDir
	originalRunPluginProcess := launcherRunPluginProcess
	t.Cleanup(func() {
		launcherConfigGetwd = originalGetwd
		launcherConfigUserHomeDir = originalHome
		launcherRunPluginProcess = originalRunPluginProcess
	})
	launcherConfigGetwd = func() (string, error) { return projectRoot, nil }
	launcherConfigUserHomeDir = func() (string, error) { return filepath.Join(projectRoot, "home"), nil }

	invokedPath := ""
	var capturedRequest launcherPluginInvocationRequest
	launcherRunPluginProcess = func(path string, args []string, env []string, stdin []byte, timeout time.Duration) (string, string, error) {
		invokedPath = path
		if err := json.Unmarshal(stdin, &capturedRequest); err != nil {
			t.Fatalf("parse plugin request: %v", err)
		}
		return `{
  "diagnostics": [{"code":"GWC-POLICY-LANE","severity":"error","summary":"browser lane required"}],
  "verifyChecks": [{"name":"org-lane-policy","passed":false,"summary":"missing required browser lane"}]
}`, "", nil
	}

	err := (launcher{repoRoot: projectRoot}).run([]string{"-policy-pack", orgPolicyPath, "verify", "-skip-tests"})
	if err == nil {
		t.Fatal("expected verify policy/plugin integration failure")
	}
	if !strings.Contains(err.Error(), "org-lane-policy") || !strings.Contains(err.Error(), "GWC-POLICY-LANE") {
		t.Fatalf("expected failure attribution with machine-readable diagnostic code, got %v", err)
	}
	if invokedPath != projectPluginPath {
		t.Fatalf("expected project plugin path to win by precedence, got %q", invokedPath)
	}
	if capturedRequest.Sources.OrganizationPolicyPath != orgPolicyPath || capturedRequest.Sources.ProjectConfigPath != projectConfigPath {
		t.Fatalf("expected plugin request sources to include policy and project discovery paths, got %#v", capturedRequest.Sources)
	}
}

func TestLauncherEnterpriseHookIntegrationOrdersOrgAndProjectHooks(t *testing.T) {
	root := t.TempDir()
	projectRoot := filepath.Join(root, "project")
	if err := os.MkdirAll(filepath.Join(projectRoot, "hooks"), 0755); err != nil {
		t.Fatalf("mkdir project hooks: %v", err)
	}
	orgPolicyPath := filepath.Join(root, "org-policy-pack.json")
	if err := os.MkdirAll(filepath.Join(root, "hooks"), 0755); err != nil {
		t.Fatalf("mkdir org hooks: %v", err)
	}
	orgPreAll := filepath.Join(root, "hooks", "org-pre-all.exe")
	orgPreVerify := filepath.Join(root, "hooks", "org-pre-verify.exe")
	projectPreVerify := filepath.Join(projectRoot, "hooks", "project-pre-verify.exe")
	for _, path := range []string{orgPreAll, orgPreVerify, projectPreVerify} {
		if err := os.WriteFile(path, []byte(""), 0644); err != nil {
			t.Fatalf("write hook placeholder %s: %v", path, err)
		}
	}
	if err := os.WriteFile(orgPolicyPath, []byte(`{
  "hooks": {
    "pre-all": [{ "name": "org-pre-all", "path": "hooks/org-pre-all.exe" }],
    "pre-verify": [{ "name": "org-pre-verify", "path": "hooks/org-pre-verify.exe" }]
  }
}`), 0644); err != nil {
		t.Fatalf("write org policy pack: %v", err)
	}
	projectConfigPath := filepath.Join(projectRoot, "gwc-runner.json")
	if err := os.WriteFile(projectConfigPath, []byte(`{
  "enterprise": {
    "hooks": {
      "pre-verify": [{ "name": "project-pre-verify", "path": "hooks/project-pre-verify.exe" }]
    }
  }
}`), 0644); err != nil {
		t.Fatalf("write project config: %v", err)
	}

	originalGetwd := launcherConfigGetwd
	originalHome := launcherConfigUserHomeDir
	originalRunVerifyCommand := runVerifyCommand
	originalRunHookProcess := launcherRunHookProcess
	t.Cleanup(func() {
		launcherConfigGetwd = originalGetwd
		launcherConfigUserHomeDir = originalHome
		runVerifyCommand = originalRunVerifyCommand
		launcherRunHookProcess = originalRunHookProcess
	})
	launcherConfigGetwd = func() (string, error) { return projectRoot, nil }
	launcherConfigUserHomeDir = func() (string, error) { return filepath.Join(projectRoot, "home"), nil }
	runVerifyCommand = func(l launcher, args []string) error { return nil }

	hookOrder := []string{}
	var capturedSources launcherEnterpriseConfigSources
	launcherRunHookProcess = func(path string, args []string, env []string, stdin []byte, timeout time.Duration) (string, string, error) {
		hookOrder = append(hookOrder, filepath.Base(path))
		var request launcherHookInvocationRequest
		if err := json.Unmarshal(stdin, &request); err != nil {
			t.Fatalf("parse hook request: %v", err)
		}
		capturedSources = request.Sources
		return `{"ok":true}`, "", nil
	}

	if err := (launcher{repoRoot: projectRoot}).run([]string{"-policy-pack", orgPolicyPath, "verify", "-skip-tests"}); err != nil {
		t.Fatalf("run verify with org+project hook integration: %v", err)
	}
	if got := strings.Join(hookOrder, ","); got != "org-pre-all.exe,org-pre-verify.exe,project-pre-verify.exe" {
		t.Fatalf("expected ordered pre-hook chain from policy then project overrides, got %q", got)
	}
	if capturedSources.OrganizationPolicyPath != orgPolicyPath || capturedSources.ProjectConfigPath != projectConfigPath {
		t.Fatalf("expected hook request sources to include policy and project paths, got %#v", capturedSources)
	}
}
