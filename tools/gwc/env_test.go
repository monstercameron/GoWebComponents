package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRunEnvJSONRedactsSecretsAndIncludesDynamicGWCPrefixVars(t *testing.T) {
	originalLookup := launcherEnvLookup
	originalList := launcherEnvList
	t.Cleanup(func() {
		launcherEnvLookup = originalLookup
		launcherEnvList = originalList
	})

	envMap := map[string]string{
		"GWC_RUNNER_CONFIG": `C:\configs\gwc-runner.json`,
		"OPENAI_API_KEY":    "sk-live-secret-value",
		"GWC_CUSTOM_FLAG":   "enabled",
	}
	launcherEnvLookup = func(key string) (string, bool) {
		value, ok := envMap[key]
		return value, ok
	}
	launcherEnvList = func() []string {
		return []string{
			"GWC_RUNNER_CONFIG=" + envMap["GWC_RUNNER_CONFIG"],
			"OPENAI_API_KEY=" + envMap["OPENAI_API_KEY"],
			"GWC_CUSTOM_FLAG=" + envMap["GWC_CUSTOM_FLAG"],
		}
	}

	stdout, restoreStdout, err := captureExamplesStdout()
	if err != nil {
		t.Fatalf("capture stdout: %v", err)
	}
	defer restoreStdout()

	if err := (launcher{}).runEnv([]string{"-json"}); err != nil {
		t.Fatalf("run env command: %v", err)
	}

	output, err := stdout()
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	var summary launcherEnvSummary
	if err := json.Unmarshal([]byte(output), &summary); err != nil {
		t.Fatalf("decode env summary: %v\n%s", err, output)
	}

	openAI, ok := findLauncherEnvRecord(summary.Variables, "OPENAI_API_KEY")
	if !ok {
		t.Fatalf("expected OPENAI_API_KEY in env summary, got %#v", summary.Variables)
	}
	if !openAI.Set || !openAI.Redacted || !openAI.Sensitive {
		t.Fatalf("expected OPENAI_API_KEY to be set, sensitive, and redacted: %#v", openAI)
	}
	if !strings.Contains(openAI.Value, "[redacted len=") {
		t.Fatalf("expected redacted OPENAI_API_KEY value, got %#v", openAI)
	}

	custom, ok := findLauncherEnvRecord(summary.Variables, "GWC_CUSTOM_FLAG")
	if !ok {
		t.Fatalf("expected dynamic GWC_CUSTOM_FLAG in env summary, got %#v", summary.Variables)
	}
	if !custom.Set || custom.Value != "enabled" {
		t.Fatalf("expected dynamic custom var value to be visible, got %#v", custom)
	}
}

func TestRunEnvJSONShowSecretsAndSetOnly(t *testing.T) {
	originalLookup := launcherEnvLookup
	originalList := launcherEnvList
	t.Cleanup(func() {
		launcherEnvLookup = originalLookup
		launcherEnvList = originalList
	})

	envMap := map[string]string{
		"OPENAI_API_KEY": "sk-live-secret-value",
	}
	launcherEnvLookup = func(key string) (string, bool) {
		value, ok := envMap[key]
		return value, ok
	}
	launcherEnvList = func() []string {
		return []string{"OPENAI_API_KEY=" + envMap["OPENAI_API_KEY"]}
	}

	stdout, restoreStdout, err := captureExamplesStdout()
	if err != nil {
		t.Fatalf("capture stdout: %v", err)
	}
	defer restoreStdout()

	if err := (launcher{}).runEnv([]string{"-json", "-show-secrets", "-set-only"}); err != nil {
		t.Fatalf("run env command: %v", err)
	}

	output, err := stdout()
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	var summary launcherEnvSummary
	if err := json.Unmarshal([]byte(output), &summary); err != nil {
		t.Fatalf("decode env summary: %v\n%s", err, output)
	}

	openAI, ok := findLauncherEnvRecord(summary.Variables, "OPENAI_API_KEY")
	if !ok {
		t.Fatalf("expected OPENAI_API_KEY in env summary, got %#v", summary.Variables)
	}
	if openAI.Redacted {
		t.Fatalf("expected OPENAI_API_KEY not to be redacted when -show-secrets is set, got %#v", openAI)
	}
	if openAI.Value != envMap["OPENAI_API_KEY"] {
		t.Fatalf("expected OPENAI_API_KEY raw value, got %#v", openAI)
	}
	if _, exists := findLauncherEnvRecord(summary.Variables, "PLAYWRIGHT_WORKERS"); exists {
		t.Fatalf("expected unset PLAYWRIGHT_WORKERS to be omitted by -set-only, got %#v", summary.Variables)
	}
}

func findLauncherEnvRecord(records []launcherEnvVariableRecord, name string) (launcherEnvVariableRecord, bool) {
	for _, record := range records {
		if record.Name == name {
			return record, true
		}
	}
	return launcherEnvVariableRecord{}, false
}
