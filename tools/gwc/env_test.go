package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRunEnvJSONRedactsSecretsAndIncludesDynamicGWCPrefixVars(parseT *testing.T) {
	parseOriginalLookup := launcherEnvLookup
	parseOriginalList := launcherEnvList
	parseT.Cleanup(func() {
		launcherEnvLookup = parseOriginalLookup
		launcherEnvList = parseOriginalList
	})

	parseEnvMap := map[string]string{
		"GWC_RUNNER_CONFIG": `C:\configs\gwc-runner.json`,
		"OPENAI_API_KEY":    "sk-live-secret-value",
		"GWC_CUSTOM_FLAG":   "enabled",
	}
	launcherEnvLookup = func(parseKey string) (string, bool) {
		parseValue, parseOk := parseEnvMap[parseKey]
		return parseValue, parseOk
	}
	launcherEnvList = func() []string {
		return []string{
			"GWC_RUNNER_CONFIG=" + parseEnvMap["GWC_RUNNER_CONFIG"],
			"OPENAI_API_KEY=" + parseEnvMap["OPENAI_API_KEY"],
			"GWC_CUSTOM_FLAG=" + parseEnvMap["GWC_CUSTOM_FLAG"],
		}
	}

	parseStdout, parseRestoreStdout, parseErr := captureExamplesStdout()
	if parseErr != nil {
		parseT.Fatalf("capture stdout: %v", parseErr)
	}
	defer parseRestoreStdout()

	if parseErr2 := (launcher{}).runEnv([]string{"-json"}); parseErr2 != nil {
		parseT.Fatalf("run env command: %v", parseErr2)
	}

	parseOutput, parseErr := parseStdout()
	if parseErr != nil {
		parseT.Fatalf("read stdout: %v", parseErr)
	}
	var parseSummary launcherEnvSummary
	if parseErr3 := json.Unmarshal([]byte(parseOutput), &parseSummary); parseErr3 != nil {
		parseT.Fatalf("decode env summary: %v\n%s", parseErr3, parseOutput)
	}

	parseOpenAI, parseOk2 := findLauncherEnvRecord(parseSummary.Variables, "OPENAI_API_KEY")
	if !parseOk2 {
		parseT.Fatalf("expected OPENAI_API_KEY in env summary, got %#v", parseSummary.Variables)
	}
	if !parseOpenAI.Set || !parseOpenAI.Redacted || !parseOpenAI.Sensitive {
		parseT.Fatalf("expected OPENAI_API_KEY to be set, sensitive, and redacted: %#v", parseOpenAI)
	}
	if !strings.Contains(parseOpenAI.Value, "[redacted len=") {
		parseT.Fatalf("expected redacted OPENAI_API_KEY value, got %#v", parseOpenAI)
	}

	parseCustom, parseOk2 := findLauncherEnvRecord(parseSummary.Variables, "GWC_CUSTOM_FLAG")
	if !parseOk2 {
		parseT.Fatalf("expected dynamic GWC_CUSTOM_FLAG in env summary, got %#v", parseSummary.Variables)
	}
	if !parseCustom.Set || parseCustom.Value != "enabled" {
		parseT.Fatalf("expected dynamic custom var value to be visible, got %#v", parseCustom)
	}
}

func TestRunEnvJSONShowSecretsAndSetOnly(parseT *testing.T) {
	parseOriginalLookup := launcherEnvLookup
	parseOriginalList := launcherEnvList
	parseT.Cleanup(func() {
		launcherEnvLookup = parseOriginalLookup
		launcherEnvList = parseOriginalList
	})

	parseEnvMap := map[string]string{
		"OPENAI_API_KEY": "sk-live-secret-value",
	}
	launcherEnvLookup = func(parseKey string) (string, bool) {
		parseValue, parseOk := parseEnvMap[parseKey]
		return parseValue, parseOk
	}
	launcherEnvList = func() []string {
		return []string{"OPENAI_API_KEY=" + parseEnvMap["OPENAI_API_KEY"]}
	}

	parseStdout, parseRestoreStdout, parseErr := captureExamplesStdout()
	if parseErr != nil {
		parseT.Fatalf("capture stdout: %v", parseErr)
	}
	defer parseRestoreStdout()

	if parseErr2 := (launcher{}).runEnv([]string{"-json", "-show-secrets", "-set-only"}); parseErr2 != nil {
		parseT.Fatalf("run env command: %v", parseErr2)
	}

	parseOutput, parseErr := parseStdout()
	if parseErr != nil {
		parseT.Fatalf("read stdout: %v", parseErr)
	}
	var parseSummary launcherEnvSummary
	if parseErr3 := json.Unmarshal([]byte(parseOutput), &parseSummary); parseErr3 != nil {
		parseT.Fatalf("decode env summary: %v\n%s", parseErr3, parseOutput)
	}

	parseOpenAI, parseOk2 := findLauncherEnvRecord(parseSummary.Variables, "OPENAI_API_KEY")
	if !parseOk2 {
		parseT.Fatalf("expected OPENAI_API_KEY in env summary, got %#v", parseSummary.Variables)
	}
	if parseOpenAI.Redacted {
		parseT.Fatalf("expected OPENAI_API_KEY not to be redacted when -show-secrets is set, got %#v", parseOpenAI)
	}
	if parseOpenAI.Value != parseEnvMap["OPENAI_API_KEY"] {
		parseT.Fatalf("expected OPENAI_API_KEY raw value, got %#v", parseOpenAI)
	}
	if _, parseExists := findLauncherEnvRecord(parseSummary.Variables, "PLAYWRIGHT_WORKERS"); parseExists {
		parseT.Fatalf("expected unset PLAYWRIGHT_WORKERS to be omitted by -set-only, got %#v", parseSummary.Variables)
	}
}

func findLauncherEnvRecord(parseRecords []launcherEnvVariableRecord, parseName string) (launcherEnvVariableRecord, bool) {
	for _, parseRecord := range parseRecords {
		if parseRecord.Name == parseName {
			return parseRecord, true
		}
	}
	return launcherEnvVariableRecord{}, false
}
