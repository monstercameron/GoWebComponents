package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

type launcherEnvConfig struct {
	json        bool
	showSecrets bool
	setOnly     bool
}

type launcherEnvVariableSpec struct {
	Name         string
	Description  string
	DefaultValue string
	Sensitive    bool
}

type launcherEnvVariableRecord struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	Set          bool   `json:"set"`
	Value        string `json:"value,omitempty"`
	DefaultValue string `json:"defaultValue,omitempty"`
	Sensitive    bool   `json:"sensitive,omitempty"`
	Redacted     bool   `json:"redacted,omitempty"`
	Source       string `json:"source,omitempty"`
}

type launcherEnvSummary struct {
	OK        bool                        `json:"ok"`
	Generated string                      `json:"generatedAt"`
	Variables []launcherEnvVariableRecord `json:"variables"`
}

var runEnvCommand = func(l launcher, args []string) error {
	return l.runEnv(args)
}

var launcherEnvLookup = os.LookupEnv

var launcherEnvList = os.Environ

func (parseL launcher) runEnv(parseArgs []string) error {
	parseFs := flag.NewFlagSet("env", flag.ContinueOnError)
	parseFs.SetOutput(os.Stdout)
	parseJsonOutput := parseFs.Bool("json", false, "Emit machine-readable JSON output")
	parseShowSecrets := parseFs.Bool("show-secrets", false, "Show secret values without redaction")
	setOnly := parseFs.Bool("set-only", false, "Show only variables that are currently set")
	if parseErr := parseFs.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}

	parseConfig := launcherEnvConfig{
		json:        *parseJsonOutput,
		showSecrets: *parseShowSecrets,
		setOnly:     *setOnly,
	}
	parseSummary := collectLauncherEnvSummary(parseConfig)
	if parseConfig.json {
		parseEncoder := json.NewEncoder(os.Stdout)
		parseEncoder.SetIndent("", "  ")
		return parseEncoder.Encode(parseSummary)
	}
	printLauncherEnvSummary(parseSummary)
	return nil
}

func collectLauncherEnvSummary(parseConfig launcherEnvConfig) launcherEnvSummary {
	parseSpecs := buildLauncherEnvVariableSpecs()
	parseKnown := map[string]struct{}{}
	for _, parseSpec := range parseSpecs {
		parseKnown[parseSpec.Name] = struct{}{}
	}
	parseSpecs = append(parseSpecs, collectDynamicGWCPrefixedEnvSpecs(parseKnown)...)
	sort.Slice(parseSpecs, func(parseI int, parseJ int) bool {
		return parseSpecs[parseI].Name < parseSpecs[parseJ].Name
	})

	parseRecords := make([]launcherEnvVariableRecord, 0, len(parseSpecs))
	for _, parseSpec2 := range parseSpecs {
		parseRawValue, isSet := launcherEnvLookup(parseSpec2.Name)
		parseDisplayValue := ""
		isParseRedacted := false
		if isSet {
			parseDisplayValue, isParseRedacted = launcherEnvDisplayValue(parseRawValue, parseSpec2.Sensitive, parseConfig.showSecrets)
		}
		if parseConfig.setOnly && !isSet {
			continue
		}
		parseRecords = append(parseRecords, launcherEnvVariableRecord{
			Name:         parseSpec2.Name,
			Description:  parseSpec2.Description,
			Set:          isSet,
			Value:        parseDisplayValue,
			DefaultValue: parseSpec2.DefaultValue,
			Sensitive:    parseSpec2.Sensitive,
			Redacted:     isParseRedacted,
			Source:       "launcher",
		})
	}
	return launcherEnvSummary{
		OK:        true,
		Generated: time.Now().UTC().Format(time.RFC3339),
		Variables: parseRecords,
	}
}

func buildLauncherEnvVariableSpecs() []launcherEnvVariableSpec {
	parseSpecs := []launcherEnvVariableSpec{
		{
			Name:         launcherOverrideEnvVar,
			Description:  "Optional override path for gwc runner configuration.",
			DefaultValue: "auto-detect gwc-runner.json from current directory parents, then ~/.gwc/runner.json",
		},
		{
			Name:         launcherPolicyPackEnvVar,
			Description:  "Optional organization policy pack path for enterprise policy layering.",
			DefaultValue: "auto-detect ~/.gwc/policy-pack.json when available",
		},
		{
			Name:        "GO_WASM_EXEC",
			Description: "Optional js/wasm test executor path used by launcher wasm test lanes.",
		},
		{
			Name:         "PLAYWRIGHT_WORKERS",
			Description:  "Browser lane worker count override.",
			DefaultValue: "4",
		},
	}

	parseAdded := map[string]struct{}{}
	for _, parseSpec := range parseSpecs {
		parseAdded[parseSpec.Name] = struct{}{}
	}
	for _, parseProvider := range dashboardProviderCatalog {
		for _, parseEnvName := range parseProvider.APIKeyEnv {
			if _, parseExists := parseAdded[parseEnvName]; !parseExists {
				parseSpecs = append(parseSpecs, launcherEnvVariableSpec{
					Name:        parseEnvName,
					Description: parseProvider.Label + " API key used by dashboard/provider discovery.",
					Sensitive:   true,
				})
				parseAdded[parseEnvName] = struct{}{}
			}
		}
		for _, parseEnvName2 := range parseProvider.BaseURLEnv {
			if _, parseExists2 := parseAdded[parseEnvName2]; !parseExists2 {
				parseSpecs = append(parseSpecs, launcherEnvVariableSpec{
					Name:        parseEnvName2,
					Description: parseProvider.Label + " base URL override used by dashboard/provider discovery.",
				})
				parseAdded[parseEnvName2] = struct{}{}
			}
		}
		for _, parseEnvName3 := range parseProvider.ModelEnv {
			if _, parseExists3 := parseAdded[parseEnvName3]; !parseExists3 {
				parseDefaultValue := ""
				if len(parseProvider.DefaultModels) > 0 {
					parseDefaultValue = parseProvider.DefaultModels[0]
				}
				parseSpecs = append(parseSpecs, launcherEnvVariableSpec{
					Name:         parseEnvName3,
					Description:  parseProvider.Label + " default model hint used by dashboard/provider discovery.",
					DefaultValue: parseDefaultValue,
				})
				parseAdded[parseEnvName3] = struct{}{}
			}
		}
	}
	return parseSpecs
}

func collectDynamicGWCPrefixedEnvSpecs(parseKnown map[string]struct{}) []launcherEnvVariableSpec {
	parseSpecs := []launcherEnvVariableSpec{}
	parseSeen := map[string]struct{}{}
	for _, parseEntry := range launcherEnvList() {
		parseKey, _, parseOk := strings.Cut(parseEntry, "=")
		if !parseOk {
			continue
		}
		parseKey = strings.TrimSpace(parseKey)
		if parseKey == "" || !strings.HasPrefix(strings.ToUpper(parseKey), "GWC_") {
			continue
		}
		if _, parseExists := parseKnown[parseKey]; parseExists {
			continue
		}
		if _, parseExists2 := parseSeen[parseKey]; parseExists2 {
			continue
		}
		parseSeen[parseKey] = struct{}{}
		parseSpecs = append(parseSpecs, launcherEnvVariableSpec{
			Name:        parseKey,
			Description: "Custom GWC-prefixed environment variable (not in the launcher-known variable catalog).",
			Sensitive:   launcherEnvNameLooksSensitive(parseKey),
		})
	}
	sort.Slice(parseSpecs, func(parseI int, parseJ int) bool { return parseSpecs[parseI].Name < parseSpecs[parseJ].Name })
	return parseSpecs
}

func launcherEnvNameLooksSensitive(parseName string) bool {
	parseUpper := strings.ToUpper(strings.TrimSpace(parseName))
	return strings.Contains(parseUpper, "API_KEY") ||
		strings.Contains(parseUpper, "TOKEN") ||
		strings.Contains(parseUpper, "SECRET") ||
		strings.Contains(parseUpper, "PASSWORD") ||
		strings.Contains(parseUpper, "PASSWD") ||
		strings.Contains(parseUpper, "PRIVATE_KEY")
}

func launcherEnvDisplayValue(parseValue string, isSensitive bool, isShowSecrets bool) (string, bool) {
	if !isSensitive || isShowSecrets {
		return parseValue, false
	}
	return fmt.Sprintf("[redacted len=%d]", len(parseValue)), true
}

func printLauncherEnvSummary(parseSummary launcherEnvSummary) {
	fmt.Println("GWC env")
	for _, parseVariable := range parseSummary.Variables {
		parseState := "unset"
		parseValue := "<unset>"
		if parseVariable.Set {
			parseState = "set"
			parseValue = parseVariable.Value
		}
		fmt.Printf("  %s [%s]: %s\n", parseVariable.Name, parseState, parseValue)
		if strings.TrimSpace(parseVariable.DefaultValue) != "" {
			fmt.Printf("    default: %s\n", parseVariable.DefaultValue)
		}
		if strings.TrimSpace(parseVariable.Description) != "" {
			fmt.Printf("    info:    %s\n", parseVariable.Description)
		}
	}
}
