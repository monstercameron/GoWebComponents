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

func TestParseLauncherGlobalCLIOptions(parseT *testing.T) {
	parseOptions, parseRemaining, parseErr := parseLauncherGlobalCLIOptions([]string{"-policy-pack", "org/policy.json", "-no-hooks", "-no-plugins", "verify", "-json"})
	if parseErr != nil {
		parseT.Fatalf("parse global options: %v", parseErr)
	}
	if parseOptions.PolicyPackPath != "org/policy.json" || !parseOptions.DisableHooks || !parseOptions.DisablePlugins {
		parseT.Fatalf("unexpected parsed options: %#v", parseOptions)
	}
	if parseGot := strings.Join(parseRemaining, " "); parseGot != "verify -json" {
		parseT.Fatalf("unexpected remaining args: %q", parseGot)
	}

	if _, _, parseErr2 := parseLauncherGlobalCLIOptions([]string{"-unknown", "verify"}); parseErr2 == nil || !strings.Contains(parseErr2.Error(), "unknown global flag") {
		parseT.Fatalf("expected unknown global flag error, got %v", parseErr2)
	}

	parseT.Run("stops at explicit separator", func(parseT2 *testing.T) {
		parseOptions2, parseRemaining2, parseErr3 := parseLauncherGlobalCLIOptions([]string{"-no-hooks", "--", "-policy-pack", "ignored", "verify"})
		if parseErr3 != nil {
			parseT2.Fatalf("parse global options with separator: %v", parseErr3)
		}
		if !parseOptions2.DisableHooks {
			parseT2.Fatalf("expected -no-hooks before separator to apply, got %#v", parseOptions2)
		}
		if parseGot2 := strings.Join(parseRemaining2, " "); parseGot2 != "-policy-pack ignored verify" {
			parseT2.Fatalf("unexpected remaining args after separator: %q", parseGot2)
		}
	})

	parseT.Run("stops at help", func(parseT3 *testing.T) {
		parseOptions3, parseRemaining3, parseErr4 := parseLauncherGlobalCLIOptions([]string{"--help", "verify"})
		if parseErr4 != nil {
			parseT3.Fatalf("parse help global options: %v", parseErr4)
		}
		if parseOptions3 != (launcherGlobalCLIOptions{}) {
			parseT3.Fatalf("expected zero options when help stops parsing, got %#v", parseOptions3)
		}
		if parseGot3 := strings.Join(parseRemaining3, " "); parseGot3 != "--help verify" {
			parseT3.Fatalf("unexpected remaining args when help stops parsing: %q", parseGot3)
		}
	})

	parseT.Run("rejects missing policy pack value", func(parseT4 *testing.T) {
		if _, _, parseErr5 := parseLauncherGlobalCLIOptions([]string{"-policy-pack"}); parseErr5 == nil || !strings.Contains(parseErr5.Error(), "requires a path value") {
			parseT4.Fatalf("expected missing policy-pack value error, got %v", parseErr5)
		}
	})
}

func TestLauncherRunExecutesPreAndPostHooks(parseT *testing.T) {
	parseProjectRoot := parseT.TempDir()
	parseHookDir := filepath.Join(parseProjectRoot, "hooks")
	if parseErr := os.MkdirAll(parseHookDir, 0755); parseErr != nil {
		parseT.Fatalf("mkdir hook dir: %v", parseErr)
	}
	parsePreHookPath := filepath.Join(parseHookDir, "pre-test.exe")
	parsePostHookPath := filepath.Join(parseHookDir, "post-test.exe")
	if parseErr2 := os.WriteFile(parsePreHookPath, []byte(""), 0644); parseErr2 != nil {
		parseT.Fatalf("write pre hook placeholder: %v", parseErr2)
	}
	if parseErr3 := os.WriteFile(parsePostHookPath, []byte(""), 0644); parseErr3 != nil {
		parseT.Fatalf("write post hook placeholder: %v", parseErr3)
	}
	if parseErr4 := os.WriteFile(filepath.Join(parseProjectRoot, "gwc-runner.json"), []byte(`{
  "enterprise": {
    "hooks": {
      "pre-test": [{ "name": "pre-test-hook", "path": "hooks/pre-test.exe" }],
      "post-test": [{ "name": "post-test-hook", "path": "hooks/post-test.exe" }]
    }
  }
}`), 0644); parseErr4 != nil {
		parseT.Fatalf("write project config: %v", parseErr4)
	}

	parseOriginalGetwd := launcherConfigGetwd
	parseOriginalHomeDir := launcherConfigUserHomeDir
	parseOriginalRunTestCommand := runTestCommand
	parseOriginalRunHookProcess := launcherRunHookProcess
	parseT.Cleanup(func() {
		launcherConfigGetwd = parseOriginalGetwd
		launcherConfigUserHomeDir = parseOriginalHomeDir
		runTestCommand = parseOriginalRunTestCommand
		launcherRunHookProcess = parseOriginalRunHookProcess
	})
	launcherConfigGetwd = func() (string, error) { return parseProjectRoot, nil }
	launcherConfigUserHomeDir = func() (string, error) { return filepath.Join(parseProjectRoot, "home"), nil }

	parseInvokedHooks := []string{}
	launcherRunHookProcess = func(parsePath string, parseArgs []string, parseEnv []string, parseStdin []byte, parseTimeout time.Duration) (string, string, error) {
		parseInvokedHooks = append(parseInvokedHooks, filepath.Base(parsePath))
		var parseRequest launcherHookInvocationRequest
		if parseErr5 := json.Unmarshal(parseStdin, &parseRequest); parseErr5 != nil {
			parseT.Fatalf("parse hook request payload: %v", parseErr5)
		}
		if parseRequest.Command != "test" {
			parseT.Fatalf("expected hook request command test, got %#v", parseRequest)
		}
		return `{"ok":true}`, "", nil
	}

	isParseTestCommandCalled := false
	runTestCommand = func(parseL launcher, parseArgs2 []string) error {
		isParseTestCommandCalled = true
		return nil
	}

	if parseErr6 := (launcher{}).run([]string{"test", "-json"}); parseErr6 != nil {
		parseT.Fatalf("run test with hooks: %v", parseErr6)
	}
	if !isParseTestCommandCalled {
		parseT.Fatal("expected test command to be dispatched")
	}
	if parseGot := strings.Join(parseInvokedHooks, ","); parseGot != "pre-test.exe,post-test.exe" {
		parseT.Fatalf("expected pre/post hook order, got %q", parseGot)
	}
}

func TestLauncherRunStopsOnPreHookFailure(parseT *testing.T) {
	parseProjectRoot := parseT.TempDir()
	parseHookDir := filepath.Join(parseProjectRoot, "hooks")
	if parseErr := os.MkdirAll(parseHookDir, 0755); parseErr != nil {
		parseT.Fatalf("mkdir hook dir: %v", parseErr)
	}
	parsePreHookPath := filepath.Join(parseHookDir, "pre-build.exe")
	if parseErr2 := os.WriteFile(parsePreHookPath, []byte(""), 0644); parseErr2 != nil {
		parseT.Fatalf("write pre hook placeholder: %v", parseErr2)
	}
	if parseErr3 := os.WriteFile(filepath.Join(parseProjectRoot, "gwc-runner.json"), []byte(`{
  "enterprise": {
    "hooks": {
      "pre-build": [{ "name": "pre-build-hook", "path": "hooks/pre-build.exe" }]
    }
  }
}`), 0644); parseErr3 != nil {
		parseT.Fatalf("write project config: %v", parseErr3)
	}

	parseOriginalGetwd := launcherConfigGetwd
	parseOriginalHomeDir := launcherConfigUserHomeDir
	parseOriginalRunBuildCommand := runBuildCommand
	parseOriginalRunHookProcess := launcherRunHookProcess
	parseT.Cleanup(func() {
		launcherConfigGetwd = parseOriginalGetwd
		launcherConfigUserHomeDir = parseOriginalHomeDir
		runBuildCommand = parseOriginalRunBuildCommand
		launcherRunHookProcess = parseOriginalRunHookProcess
	})
	launcherConfigGetwd = func() (string, error) { return parseProjectRoot, nil }
	launcherConfigUserHomeDir = func() (string, error) { return filepath.Join(parseProjectRoot, "home"), nil }
	launcherRunHookProcess = func(parsePath string, parseArgs []string, parseEnv []string, parseStdin []byte, parseTimeout time.Duration) (string, string, error) {
		return "", "", errors.New("blocked by policy")
	}

	isBuildCalled := false
	runBuildCommand = func(parseL launcher, parseArgs2 []string) error {
		isBuildCalled = true
		return nil
	}

	parseErr4 := (launcher{}).run([]string{"build", "-json"})
	if parseErr4 == nil || !strings.Contains(parseErr4.Error(), "pre-build-hook") {
		parseT.Fatalf("expected pre-hook failure attribution, got %v", parseErr4)
	}
	if isBuildCalled {
		parseT.Fatal("expected build command to be skipped after pre-hook failure")
	}
}

func TestLauncherRunContinuesWhenHookAllowsFailure(parseT *testing.T) {
	parseProjectRoot := parseT.TempDir()
	parseHookDir := filepath.Join(parseProjectRoot, "hooks")
	if parseErr := os.MkdirAll(parseHookDir, 0755); parseErr != nil {
		parseT.Fatalf("mkdir hook dir: %v", parseErr)
	}
	parsePreHookPath := filepath.Join(parseHookDir, "pre-verify.exe")
	if parseErr2 := os.WriteFile(parsePreHookPath, []byte(""), 0644); parseErr2 != nil {
		parseT.Fatalf("write pre hook placeholder: %v", parseErr2)
	}
	if parseErr3 := os.WriteFile(filepath.Join(parseProjectRoot, "gwc-runner.json"), []byte(`{
  "enterprise": {
    "hooks": {
      "pre-verify": [{ "name": "pre-verify-hook", "path": "hooks/pre-verify.exe", "allowFailure": true }]
    }
  }
}`), 0644); parseErr3 != nil {
		parseT.Fatalf("write project config: %v", parseErr3)
	}

	parseOriginalGetwd := launcherConfigGetwd
	parseOriginalHomeDir := launcherConfigUserHomeDir
	parseOriginalRunVerifyCommand := runVerifyCommand
	parseOriginalRunHookProcess := launcherRunHookProcess
	parseT.Cleanup(func() {
		launcherConfigGetwd = parseOriginalGetwd
		launcherConfigUserHomeDir = parseOriginalHomeDir
		runVerifyCommand = parseOriginalRunVerifyCommand
		launcherRunHookProcess = parseOriginalRunHookProcess
	})
	launcherConfigGetwd = func() (string, error) { return parseProjectRoot, nil }
	launcherConfigUserHomeDir = func() (string, error) { return filepath.Join(parseProjectRoot, "home"), nil }
	launcherRunHookProcess = func(parsePath string, parseArgs []string, parseEnv []string, parseStdin []byte, parseTimeout time.Duration) (string, string, error) {
		return "", "", errors.New("temporary hook failure")
	}

	isParseVerifyCalled := false
	runVerifyCommand = func(parseL launcher, parseArgs2 []string) error {
		isParseVerifyCalled = true
		return nil
	}

	if parseErr4 := (launcher{}).run([]string{"verify", "-json"}); parseErr4 != nil {
		parseT.Fatalf("expected verify command to continue with allowFailure hook, got %v", parseErr4)
	}
	if !isParseVerifyCalled {
		parseT.Fatal("expected verify command to run when allowFailure hook fails")
	}
}

func TestRunLauncherPluginsForCapabilityUsesStructuredJSONIO(parseT *testing.T) {
	parsePluginPath := filepath.Join(parseT.TempDir(), "verify-plugin.exe")
	if parseErr := os.WriteFile(parsePluginPath, []byte(""), 0644); parseErr != nil {
		parseT.Fatalf("write plugin placeholder: %v", parseErr)
	}

	parseOriginalPlugins := launcherActiveEnterpriseConfig
	parseOriginalPluginProcess := launcherRunPluginProcess
	parseT.Cleanup(func() {
		launcherActiveEnterpriseConfig = parseOriginalPlugins
		launcherRunPluginProcess = parseOriginalPluginProcess
	})
	launcherActiveEnterpriseConfig = launcherEnterpriseConfig{
		Plugins: []launcherExecutablePlugin{
			{
				Name:         "verify-plugin",
				Path:         parsePluginPath,
				Capabilities: []string{"verify_check"},
			},
		},
	}

	launcherRunPluginProcess = func(parsePath string, parseArgs []string, parseEnv []string, parseStdin []byte, parseTimeout time.Duration) (string, string, error) {
		var parseRequest launcherPluginInvocationRequest
		if parseErr2 := json.Unmarshal(parseStdin, &parseRequest); parseErr2 != nil {
			parseT.Fatalf("parse plugin request payload: %v", parseErr2)
		}
		if parseRequest.SchemaVersion != "gwc-plugin-v1" || parseRequest.Capability != "verify_check" || parseRequest.Command != "verify" {
			parseT.Fatalf("unexpected plugin request payload: %#v", parseRequest)
		}
		return `{"ok":true,"verifyChecks":[{"name":"policy","passed":true}]}`, "", nil
	}

	parseResults, parseErr3 := runLauncherPluginsForCapability("verify_check", "verify", []string{"-json"}, `C:\repo`, launcherEnterpriseConfigSources{FrameworkDefaults: true})
	if parseErr3 != nil {
		parseT.Fatalf("run plugins for verify_check: %v", parseErr3)
	}
	if len(parseResults) != 1 {
		parseT.Fatalf("expected one plugin execution result, got %#v", parseResults)
	}
	if parseErr4 := enforcePluginChecks(parseResults, "verify_check"); parseErr4 != nil {
		parseT.Fatalf("expected verify checks to pass, got %v", parseErr4)
	}
}

func TestRunVerifyFailsWhenPluginVerifyCheckFails(parseT *testing.T) {
	parsePluginPath := filepath.Join(parseT.TempDir(), "verify-plugin.exe")
	if parseErr := os.WriteFile(parsePluginPath, []byte(""), 0644); parseErr != nil {
		parseT.Fatalf("write plugin placeholder: %v", parseErr)
	}

	parseOriginalPlugins := launcherActiveEnterpriseConfig
	parseOriginalSources := launcherActiveEnterpriseSources
	parseOriginalPluginProcess := launcherRunPluginProcess
	parseT.Cleanup(func() {
		launcherActiveEnterpriseConfig = parseOriginalPlugins
		launcherActiveEnterpriseSources = parseOriginalSources
		launcherRunPluginProcess = parseOriginalPluginProcess
	})
	launcherActiveEnterpriseConfig = launcherEnterpriseConfig{
		Plugins: []launcherExecutablePlugin{
			{
				Name:         "verify-plugin",
				Path:         parsePluginPath,
				Capabilities: []string{"verify_check"},
			},
		},
	}
	launcherActiveEnterpriseSources = launcherEnterpriseConfigSources{FrameworkDefaults: true}
	launcherRunPluginProcess = func(parsePath string, parseArgs []string, parseEnv []string, parseStdin []byte, parseTimeout time.Duration) (string, string, error) {
		return `{"verifyChecks":[{"name":"org-verify","passed":false,"summary":"verify lane policy failed"}]}`, "", nil
	}

	parseErr2 := (launcher{}).runVerify(nil)
	if parseErr2 == nil || !strings.Contains(parseErr2.Error(), "org-verify") {
		parseT.Fatalf("expected plugin verify check failure attribution, got %v", parseErr2)
	}
}

func TestRunReleaseFailsWhenPluginReleaseValidatorFails(parseT *testing.T) {
	parsePluginPath := filepath.Join(parseT.TempDir(), "release-plugin.exe")
	if parseErr := os.WriteFile(parsePluginPath, []byte(""), 0644); parseErr != nil {
		parseT.Fatalf("write plugin placeholder: %v", parseErr)
	}

	parseOriginalPlugins := launcherActiveEnterpriseConfig
	parseOriginalSources := launcherActiveEnterpriseSources
	parseOriginalPluginProcess := launcherRunPluginProcess
	parseT.Cleanup(func() {
		launcherActiveEnterpriseConfig = parseOriginalPlugins
		launcherActiveEnterpriseSources = parseOriginalSources
		launcherRunPluginProcess = parseOriginalPluginProcess
	})
	launcherActiveEnterpriseConfig = launcherEnterpriseConfig{
		Plugins: []launcherExecutablePlugin{
			{
				Name:         "release-plugin",
				Path:         parsePluginPath,
				Capabilities: []string{"release_validator"},
			},
		},
	}
	launcherActiveEnterpriseSources = launcherEnterpriseConfigSources{FrameworkDefaults: true}
	launcherRunPluginProcess = func(parsePath string, parseArgs []string, parseEnv []string, parseStdin []byte, parseTimeout time.Duration) (string, string, error) {
		return `{"releaseValidators":[{"name":"org-release-validator","passed":false,"summary":"artifact naming policy failed"}]}`, "", nil
	}

	parseErr2 := (launcher{}).runRelease(nil)
	if parseErr2 == nil || !strings.Contains(parseErr2.Error(), "org-release-validator") {
		parseT.Fatalf("expected plugin release validator failure attribution, got %v", parseErr2)
	}
}

func TestPrintLauncherExtensionReportIncludesTrustAndDiscovery(parseT *testing.T) {
	var parseBuffer bytes.Buffer
	parseOriginalWriter := launcherExtensionReportWriter
	parseT.Cleanup(func() { launcherExtensionReportWriter = parseOriginalWriter })
	launcherExtensionReportWriter = &parseBuffer

	parseLayered := launcherEnterpriseLayeredConfig{
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

	printLauncherExtensionReport("verify", parseLayered)
	parseOutput := parseBuffer.String()
	for _, parseExpected := range []string{
		"GWC extension report",
		"command: verify",
		"org policy: C:\\policy\\org-policy.json",
		"project config: C:\\repo\\gwc-runner.json",
		"pre-verify: 2 loaded (1 trusted, 1 untrusted)",
		"org-plugin [trusted] caps=verify_check",
	} {
		if !strings.Contains(parseOutput, parseExpected) {
			parseT.Fatalf("expected extension report output to contain %q, got:\n%s", parseExpected, parseOutput)
		}
	}
}

func TestLauncherRunInvokesExtensionReportForPlainTextCommands(parseT *testing.T) {
	parseOriginalGetwd := launcherConfigGetwd
	parseOriginalHome := launcherConfigUserHomeDir
	parseOriginalReportPrinter := launcherPrintExtensionReport
	parseOriginalRunDoctorCommand := runDoctorCommand
	parseT.Cleanup(func() {
		launcherConfigGetwd = parseOriginalGetwd
		launcherConfigUserHomeDir = parseOriginalHome
		launcherPrintExtensionReport = parseOriginalReportPrinter
		runDoctorCommand = parseOriginalRunDoctorCommand
	})
	parseWorkspace := parseT.TempDir()
	launcherConfigGetwd = func() (string, error) { return parseWorkspace, nil }
	launcherConfigUserHomeDir = func() (string, error) { return filepath.Join(parseWorkspace, "home"), nil }

	isParseReportCalled := false
	launcherPrintExtensionReport = func(parseCommand string, parseLayered launcherEnterpriseLayeredConfig) {
		isParseReportCalled = true
		if parseCommand != "doctor" {
			parseT.Fatalf("expected extension report command doctor, got %q", parseCommand)
		}
	}
	runDoctorCommand = func(parseL launcher, parseArgs []string) error { return nil }

	if parseErr := (launcher{}).run([]string{"doctor"}); parseErr != nil {
		parseT.Fatalf("run doctor: %v", parseErr)
	}
	if !isParseReportCalled {
		parseT.Fatal("expected launcher run to invoke extension report printer")
	}
}

func TestLauncherRunInvokesExtensionReport(parseT *testing.T) {
	TestLauncherRunInvokesExtensionReportForPlainTextCommands(parseT)
}

func TestLauncherRunSkipsExtensionReportForJSONCommands(parseT *testing.T) {
	parseOriginalGetwd := launcherConfigGetwd
	parseOriginalHome := launcherConfigUserHomeDir
	parseOriginalReportPrinter := launcherPrintExtensionReport
	parseOriginalRunDoctorCommand := runDoctorCommand
	parseT.Cleanup(func() {
		launcherConfigGetwd = parseOriginalGetwd
		launcherConfigUserHomeDir = parseOriginalHome
		launcherPrintExtensionReport = parseOriginalReportPrinter
		runDoctorCommand = parseOriginalRunDoctorCommand
	})
	parseWorkspace := parseT.TempDir()
	launcherConfigGetwd = func() (string, error) { return parseWorkspace, nil }
	launcherConfigUserHomeDir = func() (string, error) { return filepath.Join(parseWorkspace, "home"), nil }

	isParseReportCalled := false
	launcherPrintExtensionReport = func(parseCommand string, parseLayered launcherEnterpriseLayeredConfig) {
		isParseReportCalled = true
	}
	runDoctorCommand = func(parseL launcher, parseArgs []string) error { return nil }

	if parseErr := (launcher{}).run([]string{"doctor", "-json"}); parseErr != nil {
		parseT.Fatalf("run doctor: %v", parseErr)
	}
	if isParseReportCalled {
		parseT.Fatal("expected launcher run to skip extension report printer for json commands")
	}
}

func TestLauncherRunBlocksUntrustedExtensionsWhenPolicyDisallows(parseT *testing.T) {
	parseProjectRoot := parseT.TempDir()
	if parseErr := os.MkdirAll(filepath.Join(parseProjectRoot, "hooks"), 0755); parseErr != nil {
		parseT.Fatalf("mkdir hooks: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(filepath.Join(parseProjectRoot, "hooks", "pre-build.exe"), []byte(""), 0644); parseErr2 != nil {
		parseT.Fatalf("write hook placeholder: %v", parseErr2)
	}
	if parseErr3 := os.WriteFile(filepath.Join(parseProjectRoot, "gwc-runner.json"), []byte(`{
  "enterprise": {
    "security": { "allowUntrustedExtensions": false },
    "hooks": { "pre-build": [{ "name": "pre-build-hook", "path": "hooks/pre-build.exe" }] }
  }
}`), 0644); parseErr3 != nil {
		parseT.Fatalf("write project config: %v", parseErr3)
	}

	parseOriginalGetwd := launcherConfigGetwd
	parseOriginalHome := launcherConfigUserHomeDir
	parseOriginalRunBuild := runBuildCommand
	parseT.Cleanup(func() {
		launcherConfigGetwd = parseOriginalGetwd
		launcherConfigUserHomeDir = parseOriginalHome
		runBuildCommand = parseOriginalRunBuild
	})
	launcherConfigGetwd = func() (string, error) { return parseProjectRoot, nil }
	launcherConfigUserHomeDir = func() (string, error) { return filepath.Join(parseProjectRoot, "home"), nil }

	isBuildCalled := false
	runBuildCommand = func(parseL launcher, parseArgs []string) error {
		isBuildCalled = true
		return nil
	}

	parseErr4 := (launcher{}).run([]string{"build"})
	if parseErr4 == nil || !strings.Contains(parseErr4.Error(), "untrusted and blocked") {
		parseT.Fatalf("expected untrusted extension policy failure, got %v", parseErr4)
	}
	if isBuildCalled {
		parseT.Fatal("expected command dispatch to be blocked by security policy")
	}
}

func TestLauncherRunBlocksExtensionsOutsideAllowedRoots(parseT *testing.T) {
	parseProjectRoot := parseT.TempDir()
	if parseErr := os.MkdirAll(filepath.Join(parseProjectRoot, "hooks"), 0755); parseErr != nil {
		parseT.Fatalf("mkdir hooks: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(filepath.Join(parseProjectRoot, "hooks", "pre-build.exe"), []byte(""), 0644); parseErr2 != nil {
		parseT.Fatalf("write hook placeholder: %v", parseErr2)
	}
	if parseErr3 := os.MkdirAll(filepath.Join(parseProjectRoot, "trusted-hooks"), 0755); parseErr3 != nil {
		parseT.Fatalf("mkdir trusted hooks: %v", parseErr3)
	}
	if parseErr4 := os.WriteFile(filepath.Join(parseProjectRoot, "gwc-runner.json"), []byte(`{
  "enterprise": {
    "security": { "allowedExecutableRoots": ["trusted-hooks"] },
    "hooks": { "pre-build": [{ "name": "pre-build-hook", "path": "hooks/pre-build.exe", "trusted": true }] }
  }
}`), 0644); parseErr4 != nil {
		parseT.Fatalf("write project config: %v", parseErr4)
	}

	parseOriginalGetwd := launcherConfigGetwd
	parseOriginalHome := launcherConfigUserHomeDir
	parseT.Cleanup(func() {
		launcherConfigGetwd = parseOriginalGetwd
		launcherConfigUserHomeDir = parseOriginalHome
	})
	launcherConfigGetwd = func() (string, error) { return parseProjectRoot, nil }
	launcherConfigUserHomeDir = func() (string, error) { return filepath.Join(parseProjectRoot, "home"), nil }

	parseErr5 := (launcher{}).run([]string{"build"})
	if parseErr5 == nil || !strings.Contains(parseErr5.Error(), "outside allowed executable roots") {
		parseT.Fatalf("expected allowed-root security failure, got %v", parseErr5)
	}
}

func TestBuildLauncherExtensionEnvSupportsSecretRefsAndInheritedFiltering(parseT *testing.T) {
	parseT.Setenv("GWC_ALLOWED_INHERITED", "allowed")
	parseT.Setenv("GWC_SECRET_TOKEN", "secret-value")
	parseT.Setenv("GWC_BLOCKED", "blocked")

	parseEnv, parseErr := buildLauncherExtensionEnv(true, map[string]string{
		"API_TOKEN": "${ENV:GWC_SECRET_TOKEN}",
	}, launcherEnterpriseSecurityPolicy{
		InheritedEnvAllowlist: []string{"GWC_ALLOWED_INHERITED", "GWC_BLOCKED"},
		InheritedEnvDenylist:  []string{"GWC_BLOCKED"},
	})
	if parseErr != nil {
		parseT.Fatalf("build launcher extension env: %v", parseErr)
	}
	parseEnvMap := map[string]string{}
	for _, parsePair := range parseEnv {
		parseName, parseValue, parseOk := strings.Cut(parsePair, "=")
		if !parseOk {
			continue
		}
		parseEnvMap[parseName] = parseValue
	}
	if parseEnvMap["GWC_ALLOWED_INHERITED"] != "allowed" {
		parseT.Fatalf("expected inherited allowlisted env variable, got %#v", parseEnvMap)
	}
	if _, parseExists := parseEnvMap["GWC_BLOCKED"]; parseExists {
		parseT.Fatalf("expected denied env variable to be filtered, got %#v", parseEnvMap)
	}
	if parseEnvMap["API_TOKEN"] != "secret-value" {
		parseT.Fatalf("expected secret env reference to resolve, got %#v", parseEnvMap)
	}
}

func TestBuildLauncherExtensionEnvFailsForMissingSecretRef(parseT *testing.T) {
	_, parseErr := buildLauncherExtensionEnv(false, map[string]string{
		"API_TOKEN": "${ENV:GWC_NOT_SET}",
	}, launcherEnterpriseSecurityPolicy{})
	if parseErr == nil || !strings.Contains(parseErr.Error(), "is not set") {
		parseT.Fatalf("expected missing secret env reference error, got %v", parseErr)
	}
}

func TestLauncherEnterpriseVerifyIntegrationRespectsPolicyPrecedenceAndDiagnostics(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseProjectRoot := filepath.Join(parseRoot, "project")
	if parseErr := os.MkdirAll(filepath.Join(parseProjectRoot, "plugins"), 0755); parseErr != nil {
		parseT.Fatalf("mkdir project plugins: %v", parseErr)
	}
	parseProjectPluginPath := filepath.Join(parseProjectRoot, "plugins", "project-verify.exe")
	if parseErr2 := os.WriteFile(parseProjectPluginPath, []byte(""), 0644); parseErr2 != nil {
		parseT.Fatalf("write project plugin placeholder: %v", parseErr2)
	}
	parseOrgPolicyPath := filepath.Join(parseRoot, "org-policy-pack.json")
	if parseErr3 := os.MkdirAll(filepath.Join(parseRoot, "plugins"), 0755); parseErr3 != nil {
		parseT.Fatalf("mkdir org plugins: %v", parseErr3)
	}
	if parseErr4 := os.WriteFile(filepath.Join(parseRoot, "plugins", "org-verify.exe"), []byte(""), 0644); parseErr4 != nil {
		parseT.Fatalf("write org plugin placeholder: %v", parseErr4)
	}
	if parseErr5 := os.WriteFile(parseOrgPolicyPath, []byte(`{
  "plugins": [
    { "name": "org-verify", "path": "plugins/org-verify.exe", "capabilities": ["verify_check"] }
  ]
}`), 0644); parseErr5 != nil {
		parseT.Fatalf("write org policy pack: %v", parseErr5)
	}
	parseProjectConfigPath := filepath.Join(parseProjectRoot, "gwc-runner.json")
	if parseErr6 := os.WriteFile(parseProjectConfigPath, []byte(`{
  "enterprise": {
    "plugins": [
      { "name": "org-verify", "path": "plugins/project-verify.exe", "capabilities": ["verify_check"] }
    ]
  }
}`), 0644); parseErr6 != nil {
		parseT.Fatalf("write project config: %v", parseErr6)
	}

	parseOriginalGetwd := launcherConfigGetwd
	parseOriginalHome := launcherConfigUserHomeDir
	parseOriginalRunPluginProcess := launcherRunPluginProcess
	parseT.Cleanup(func() {
		launcherConfigGetwd = parseOriginalGetwd
		launcherConfigUserHomeDir = parseOriginalHome
		launcherRunPluginProcess = parseOriginalRunPluginProcess
	})
	launcherConfigGetwd = func() (string, error) { return parseProjectRoot, nil }
	launcherConfigUserHomeDir = func() (string, error) { return filepath.Join(parseProjectRoot, "home"), nil }

	parseInvokedPath := ""
	var parseCapturedRequest launcherPluginInvocationRequest
	launcherRunPluginProcess = func(parsePath string, parseArgs []string, parseEnv []string, parseStdin []byte, parseTimeout time.Duration) (string, string, error) {
		parseInvokedPath = parsePath
		if parseErr7 := json.Unmarshal(parseStdin, &parseCapturedRequest); parseErr7 != nil {
			parseT.Fatalf("parse plugin request: %v", parseErr7)
		}
		return `{
  "diagnostics": [{"code":"GWC-POLICY-LANE","severity":"error","summary":"browser lane required"}],
  "verifyChecks": [{"name":"org-lane-policy","passed":false,"summary":"missing required browser lane"}]
}`, "", nil
	}

	parseErr8 := (launcher{repoRoot: parseProjectRoot}).run([]string{"-policy-pack", parseOrgPolicyPath, "verify", "-skip-tests"})
	if parseErr8 == nil {
		parseT.Fatal("expected verify policy/plugin integration failure")
	}
	if !strings.Contains(parseErr8.Error(), "org-lane-policy") || !strings.Contains(parseErr8.Error(), "GWC-POLICY-LANE") {
		parseT.Fatalf("expected failure attribution with machine-readable diagnostic code, got %v", parseErr8)
	}
	if parseInvokedPath != parseProjectPluginPath {
		parseT.Fatalf("expected project plugin path to win by precedence, got %q", parseInvokedPath)
	}
	if parseCapturedRequest.Sources.OrganizationPolicyPath != parseOrgPolicyPath || parseCapturedRequest.Sources.ProjectConfigPath != parseProjectConfigPath {
		parseT.Fatalf("expected plugin request sources to include policy and project discovery paths, got %#v", parseCapturedRequest.Sources)
	}
}

func TestLauncherEnterpriseHookIntegrationOrdersOrgAndProjectHooks(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseProjectRoot := filepath.Join(parseRoot, "project")
	if parseErr := os.MkdirAll(filepath.Join(parseProjectRoot, "hooks"), 0755); parseErr != nil {
		parseT.Fatalf("mkdir project hooks: %v", parseErr)
	}
	parseOrgPolicyPath := filepath.Join(parseRoot, "org-policy-pack.json")
	if parseErr2 := os.MkdirAll(filepath.Join(parseRoot, "hooks"), 0755); parseErr2 != nil {
		parseT.Fatalf("mkdir org hooks: %v", parseErr2)
	}
	parseOrgPreAll := filepath.Join(parseRoot, "hooks", "org-pre-all.exe")
	parseOrgPreVerify := filepath.Join(parseRoot, "hooks", "org-pre-verify.exe")
	parseProjectPreVerify := filepath.Join(parseProjectRoot, "hooks", "project-pre-verify.exe")
	for _, parsePath := range []string{parseOrgPreAll, parseOrgPreVerify, parseProjectPreVerify} {
		if parseErr3 := os.WriteFile(parsePath, []byte(""), 0644); parseErr3 != nil {
			parseT.Fatalf("write hook placeholder %s: %v", parsePath, parseErr3)
		}
	}
	if parseErr4 := os.WriteFile(parseOrgPolicyPath, []byte(`{
  "hooks": {
    "pre-all": [{ "name": "org-pre-all", "path": "hooks/org-pre-all.exe" }],
    "pre-verify": [{ "name": "org-pre-verify", "path": "hooks/org-pre-verify.exe" }]
  }
}`), 0644); parseErr4 != nil {
		parseT.Fatalf("write org policy pack: %v", parseErr4)
	}
	parseProjectConfigPath := filepath.Join(parseProjectRoot, "gwc-runner.json")
	if parseErr5 := os.WriteFile(parseProjectConfigPath, []byte(`{
  "enterprise": {
    "hooks": {
      "pre-verify": [{ "name": "project-pre-verify", "path": "hooks/project-pre-verify.exe" }]
    }
  }
}`), 0644); parseErr5 != nil {
		parseT.Fatalf("write project config: %v", parseErr5)
	}

	parseOriginalGetwd := launcherConfigGetwd
	parseOriginalHome := launcherConfigUserHomeDir
	parseOriginalRunVerifyCommand := runVerifyCommand
	parseOriginalRunHookProcess := launcherRunHookProcess
	parseT.Cleanup(func() {
		launcherConfigGetwd = parseOriginalGetwd
		launcherConfigUserHomeDir = parseOriginalHome
		runVerifyCommand = parseOriginalRunVerifyCommand
		launcherRunHookProcess = parseOriginalRunHookProcess
	})
	launcherConfigGetwd = func() (string, error) { return parseProjectRoot, nil }
	launcherConfigUserHomeDir = func() (string, error) { return filepath.Join(parseProjectRoot, "home"), nil }
	runVerifyCommand = func(parseL launcher, parseArgs []string) error { return nil }

	parseHookOrder := []string{}
	var parseCapturedSources launcherEnterpriseConfigSources
	launcherRunHookProcess = func(parsePath2 string, parseArgs2 []string, parseEnv []string, parseStdin []byte, parseTimeout time.Duration) (string, string, error) {
		parseHookOrder = append(parseHookOrder, filepath.Base(parsePath2))
		var parseRequest launcherHookInvocationRequest
		if parseErr6 := json.Unmarshal(parseStdin, &parseRequest); parseErr6 != nil {
			parseT.Fatalf("parse hook request: %v", parseErr6)
		}
		parseCapturedSources = parseRequest.Sources
		return `{"ok":true}`, "", nil
	}

	if parseErr7 := (launcher{repoRoot: parseProjectRoot}).run([]string{"-policy-pack", parseOrgPolicyPath, "verify", "-skip-tests"}); parseErr7 != nil {
		parseT.Fatalf("run verify with org+project hook integration: %v", parseErr7)
	}
	if parseGot := strings.Join(parseHookOrder, ","); parseGot != "org-pre-all.exe,org-pre-verify.exe,project-pre-verify.exe" {
		parseT.Fatalf("expected ordered pre-hook chain from policy then project overrides, got %q", parseGot)
	}
	if parseCapturedSources.OrganizationPolicyPath != parseOrgPolicyPath || parseCapturedSources.ProjectConfigPath != parseProjectConfigPath {
		parseT.Fatalf("expected hook request sources to include policy and project paths, got %#v", parseCapturedSources)
	}
}
