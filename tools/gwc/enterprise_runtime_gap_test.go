package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestEnterpriseRuntimeHookHelpersCoverBranches verifies direct hook helper failure and formatting branches.
func TestEnterpriseRuntimeHookHelpersCoverBranches(parseT *testing.T) {
	if parseGot := hookPointName("", "build"); parseGot != "" {
		parseT.Fatalf("hookPointName empty phase = %q, want empty", parseGot)
	}
	if parseGot := hookPointName(" PRE ", " Build "); parseGot != "pre-build" {
		parseT.Fatalf("hookPointName normalized = %q, want pre-build", parseGot)
	}

	parseOriginalConfig := launcherActiveEnterpriseConfig
	parseOriginalHookProcess := launcherRunHookProcess
	parseT.Cleanup(func() {
		launcherActiveEnterpriseConfig = parseOriginalConfig
		launcherRunHookProcess = parseOriginalHookProcess
	})
	launcherActiveEnterpriseConfig = defaultLauncherEnterpriseConfig()

	if parseErr := runSingleLauncherHook("pre-build", launcherExecutableHook{Name: "blank-hook"}, "build", nil, parseT.TempDir(), launcherEnterpriseConfigSources{}); parseErr == nil || !strings.Contains(parseErr.Error(), "empty executable path") {
		parseT.Fatalf("expected empty hook path error, got %v", parseErr)
	}

	parseHookPath := filepath.Join(parseT.TempDir(), "pre-build.exe")
	if parseErr2 := os.WriteFile(parseHookPath, []byte(""), 0o644); parseErr2 != nil {
		parseT.Fatalf("WriteFile hook placeholder: %v", parseErr2)
	}

	if parseErr3 := runSingleLauncherHook("pre-build", launcherExecutableHook{
		Name:         "env-hook",
		Path:         parseHookPath,
		AllowFailure: true,
		Env:          map[string]string{"SECRET": "${ENV:GWC_HOOK_MISSING_ENV}"},
	}, "build", nil, parseT.TempDir(), launcherEnterpriseConfigSources{}); parseErr3 != nil {
		parseT.Fatalf("expected allowFailure env-hook to continue, got %v", parseErr3)
	}

	if parseErr4 := runSingleLauncherHook("pre-build", launcherExecutableHook{
		Name:         "missing-hook",
		Path:         filepath.Join(parseT.TempDir(), "missing.exe"),
		AllowFailure: true,
	}, "build", nil, parseT.TempDir(), launcherEnterpriseConfigSources{}); parseErr4 != nil {
		parseT.Fatalf("expected allowFailure missing-hook to continue, got %v", parseErr4)
	}

	launcherRunHookProcess = func(parsePath string, parseArgs []string, parseEnv []string, parseStdin []byte, parseTimeout time.Duration) (string, string, error) {
		return `not-json`, "", nil
	}
	if parseErr5 := runSingleLauncherHook("pre-build", launcherExecutableHook{
		Name:         "json-hook",
		Path:         parseHookPath,
		AllowFailure: true,
	}, "build", nil, parseT.TempDir(), launcherEnterpriseConfigSources{}); parseErr5 != nil {
		parseT.Fatalf("expected allowFailure invalid-json hook to continue, got %v", parseErr5)
	}

	launcherRunHookProcess = func(parsePath string, parseArgs []string, parseEnv []string, parseStdin []byte, parseTimeout time.Duration) (string, string, error) {
		return `{"ok":false,"summary":"blocked","diagnostics":["policy","ownership"]}`, "", nil
	}
	if parseErr6 := runSingleLauncherHook("pre-build", launcherExecutableHook{
		Name: "failing-hook",
		Path: parseHookPath,
	}, "build", nil, parseT.TempDir(), launcherEnterpriseConfigSources{}); parseErr6 == nil || !strings.Contains(parseErr6.Error(), "blocked | policy | ownership") {
		parseT.Fatalf("expected structured hook failure, got %v", parseErr6)
	}
}

// TestEnterpriseRuntimeHookProcessCoversSuccessAndTimeout verifies the real hook-process wrapper.
func TestEnterpriseRuntimeHookProcessCoversSuccessAndTimeout(parseT *testing.T) {
	parseGoPath, parseErr := exec.LookPath("go")
	if parseErr != nil {
		parseT.Fatalf("LookPath go: %v", parseErr)
	}
	parseStdout, parseStderr, parseErr := runLauncherHookProcess(parseGoPath, []string{"version"}, nil, nil, time.Second)
	if parseErr != nil {
		parseT.Fatalf("runLauncherHookProcess success: %v", parseErr)
	}
	if strings.TrimSpace(parseStderr) != "" || !strings.Contains(parseStdout, "go version") {
		parseT.Fatalf("unexpected hook process output: stdout=%q stderr=%q", parseStdout, parseStderr)
	}

	parseSleepPath := "sh"
	parseSleepArgs := []string{"-c", "sleep 1"}
	if runtime.GOOS == "windows" {
		parseSleepPath = "powershell"
		parseSleepArgs = []string{"-NoProfile", "-Command", "Start-Sleep -Milliseconds 250"}
	}
	parseSleepPath, parseErr = exec.LookPath(parseSleepPath)
	if parseErr != nil {
		parseT.Skipf("sleep command unavailable for timeout branch: %v", parseErr)
	}
	if _, _, parseErr2 := runLauncherHookProcess(parseSleepPath, parseSleepArgs, nil, nil, 20*time.Millisecond); parseErr2 == nil || !strings.Contains(parseErr2.Error(), "timed out") {
		parseT.Fatalf("expected hook process timeout, got %v", parseErr2)
	}
}

// TestEnterpriseRuntimePluginFailuresCoverBranches verifies plugin execution failure paths.
func TestEnterpriseRuntimePluginFailuresCoverBranches(parseT *testing.T) {
	parseOriginalConfig := launcherActiveEnterpriseConfig
	parseOriginalPluginProcess := launcherRunPluginProcess
	parseT.Cleanup(func() {
		launcherActiveEnterpriseConfig = parseOriginalConfig
		launcherRunPluginProcess = parseOriginalPluginProcess
	})

	parsePluginPath := filepath.Join(parseT.TempDir(), "verify-plugin.exe")
	if parseErr := os.WriteFile(parsePluginPath, []byte(""), 0o644); parseErr != nil {
		parseT.Fatalf("WriteFile plugin placeholder: %v", parseErr)
	}

	parseResults, parseErr := runLauncherPluginsForCapability("", "verify", nil, parseT.TempDir(), launcherEnterpriseConfigSources{})
	if parseErr != nil || parseResults != nil {
		parseT.Fatalf("expected empty capability to no-op, results=%#v err=%v", parseResults, parseErr)
	}

	launcherActiveEnterpriseConfig = launcherEnterpriseConfig{
		Plugins: []launcherExecutablePlugin{{
			Name:         "missing-plugin",
			Path:         filepath.Join(parseT.TempDir(), "missing.exe"),
			Capabilities: []string{"verify_check"},
		}},
	}
	if _, parseErr2 := runLauncherPluginsForCapability("verify_check", "verify", nil, parseT.TempDir(), launcherEnterpriseConfigSources{}); parseErr2 == nil || !strings.Contains(parseErr2.Error(), "does not exist") {
		parseT.Fatalf("expected missing plugin path error, got %v", parseErr2)
	}

	launcherActiveEnterpriseConfig = launcherEnterpriseConfig{
		Plugins: []launcherExecutablePlugin{{
			Name:         "env-plugin",
			Path:         parsePluginPath,
			Capabilities: []string{"verify_check"},
			Env:          map[string]string{"SECRET": "${ENV:GWC_PLUGIN_MISSING_ENV}"},
		}},
	}
	if _, parseErr3 := runLauncherPluginsForCapability("verify_check", "verify", nil, parseT.TempDir(), launcherEnterpriseConfigSources{}); parseErr3 == nil || !strings.Contains(parseErr3.Error(), "environment") {
		parseT.Fatalf("expected plugin environment error, got %v", parseErr3)
	}

	launcherActiveEnterpriseConfig = launcherEnterpriseConfig{
		Plugins: []launcherExecutablePlugin{{
			Name:         "stderr-plugin",
			Path:         parsePluginPath,
			Capabilities: []string{"verify_check"},
		}},
	}
	launcherRunPluginProcess = func(parsePath string, parseArgs []string, parseEnv []string, parseStdin []byte, parseTimeout time.Duration) (string, string, error) {
		return "", "stderr details", errors.New("boom")
	}
	if _, parseErr4 := runLauncherPluginsForCapability("verify_check", "verify", nil, parseT.TempDir(), launcherEnterpriseConfigSources{}); parseErr4 == nil || !strings.Contains(parseErr4.Error(), "stderr details") {
		parseT.Fatalf("expected plugin stderr failure details, got %v", parseErr4)
	}

	launcherRunPluginProcess = func(parsePath string, parseArgs []string, parseEnv []string, parseStdin []byte, parseTimeout time.Duration) (string, string, error) {
		return "not-json", "", nil
	}
	if _, parseErr5 := runLauncherPluginsForCapability("verify_check", "verify", nil, parseT.TempDir(), launcherEnterpriseConfigSources{}); parseErr5 == nil || !strings.Contains(parseErr5.Error(), "invalid JSON") {
		parseT.Fatalf("expected plugin invalid JSON error, got %v", parseErr5)
	}

	launcherRunPluginProcess = func(parsePath string, parseArgs []string, parseEnv []string, parseStdin []byte, parseTimeout time.Duration) (string, string, error) {
		return `{"ok":false}`, "", nil
	}
	if _, parseErr6 := runLauncherPluginsForCapability("verify_check", "verify", nil, parseT.TempDir(), launcherEnterpriseConfigSources{}); parseErr6 == nil || !strings.Contains(parseErr6.Error(), "plugin reported failure") {
		parseT.Fatalf("expected default plugin failure summary, got %v", parseErr6)
	}
}

// TestEnterpriseRuntimePluginHelpersCoverFormattingAndPaths verifies plugin formatting and path/env helpers.
func TestEnterpriseRuntimePluginHelpersCoverFormattingAndPaths(parseT *testing.T) {
	if !pluginHasCapability(launcherExecutablePlugin{Capabilities: []string{" VERIFY_CHECK "}}, "verify_check") {
		parseT.Fatal("expected pluginHasCapability to be case-insensitive")
	}
	if pluginHasCapability(launcherExecutablePlugin{Capabilities: []string{"release_validator"}}, "verify_check") {
		parseT.Fatal("expected pluginHasCapability to reject missing capabilities")
	}

	parseFormatted := formatPluginDiagnostics([]launcherPluginDiagnostic{
		{},
		{Summary: "summary only"},
		{Code: "CODE"},
		{Code: "RULE", Summary: "full detail"},
	})
	if parseFormatted != "summary only | CODE | RULE: full detail" {
		parseT.Fatalf("unexpected formatted diagnostics: %q", parseFormatted)
	}

	parseRoot := parseT.TempDir()
	parseInsidePath := filepath.Join(parseRoot, "hooks", "tool.exe")
	if !launcherPathWithinRoot(parseRoot, parseRoot) {
		parseT.Fatal("expected root path to count as within root")
	}
	if !launcherPathWithinRoot(parseInsidePath, parseRoot) {
		parseT.Fatal("expected nested path to count as within root")
	}
	if launcherPathWithinRoot(parseT.TempDir(), parseRoot) {
		parseT.Fatal("expected outside path to be rejected")
	}

	parseOriginalLookupEnv := launcherEnterpriseLookupEnv
	parseT.Cleanup(func() {
		launcherEnterpriseLookupEnv = parseOriginalLookupEnv
	})
	launcherEnterpriseLookupEnv = func(parseKey string) (string, bool) {
		if parseKey == "GWC_PLUGIN_SECRET" {
			return "resolved-secret", true
		}
		return "", false
	}

	if parseValue, parseErr := resolveLauncherExtensionEnvValue("plain-text"); parseErr != nil || parseValue != "plain-text" {
		parseT.Fatalf("resolveLauncherExtensionEnvValue plain = %q err=%v", parseValue, parseErr)
	}
	if parseValue, parseErr := resolveLauncherExtensionEnvValue("${ENV:GWC_PLUGIN_SECRET}"); parseErr != nil || parseValue != "resolved-secret" {
		parseT.Fatalf("resolveLauncherExtensionEnvValue env = %q err=%v", parseValue, parseErr)
	}
	if _, parseErr := resolveLauncherExtensionEnvValue("${ENV:}"); parseErr == nil || !strings.Contains(parseErr.Error(), "empty env key") {
		parseT.Fatalf("expected empty env key error, got %v", parseErr)
	}
	if _, parseErr := resolveLauncherExtensionEnvValue("${ENV:GWC_PLUGIN_MISSING}"); parseErr == nil || !strings.Contains(parseErr.Error(), "is not set") {
		parseT.Fatalf("expected missing env error, got %v", parseErr)
	}
}
