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

func (l launcher) runEnv(args []string) error {
	fs := flag.NewFlagSet("env", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	jsonOutput := fs.Bool("json", false, "Emit machine-readable JSON output")
	showSecrets := fs.Bool("show-secrets", false, "Show secret values without redaction")
	setOnly := fs.Bool("set-only", false, "Show only variables that are currently set")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	config := launcherEnvConfig{
		json:        *jsonOutput,
		showSecrets: *showSecrets,
		setOnly:     *setOnly,
	}
	summary := collectLauncherEnvSummary(config)
	if config.json {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(summary)
	}
	printLauncherEnvSummary(summary)
	return nil
}

func collectLauncherEnvSummary(config launcherEnvConfig) launcherEnvSummary {
	specs := buildLauncherEnvVariableSpecs()
	known := map[string]struct{}{}
	for _, spec := range specs {
		known[spec.Name] = struct{}{}
	}
	for _, dynamic := range collectDynamicGWCPrefixedEnvSpecs(known) {
		specs = append(specs, dynamic)
	}
	sort.Slice(specs, func(i int, j int) bool {
		return specs[i].Name < specs[j].Name
	})

	records := make([]launcherEnvVariableRecord, 0, len(specs))
	for _, spec := range specs {
		rawValue, isSet := launcherEnvLookup(spec.Name)
		displayValue := ""
		redacted := false
		if isSet {
			displayValue, redacted = launcherEnvDisplayValue(rawValue, spec.Sensitive, config.showSecrets)
		}
		if config.setOnly && !isSet {
			continue
		}
		records = append(records, launcherEnvVariableRecord{
			Name:         spec.Name,
			Description:  spec.Description,
			Set:          isSet,
			Value:        displayValue,
			DefaultValue: spec.DefaultValue,
			Sensitive:    spec.Sensitive,
			Redacted:     redacted,
			Source:       "launcher",
		})
	}
	return launcherEnvSummary{
		OK:        true,
		Generated: time.Now().UTC().Format(time.RFC3339),
		Variables: records,
	}
}

func buildLauncherEnvVariableSpecs() []launcherEnvVariableSpec {
	specs := []launcherEnvVariableSpec{
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

	added := map[string]struct{}{}
	for _, spec := range specs {
		added[spec.Name] = struct{}{}
	}
	for _, provider := range dashboardProviderCatalog {
		for _, envName := range provider.APIKeyEnv {
			if _, exists := added[envName]; !exists {
				specs = append(specs, launcherEnvVariableSpec{
					Name:        envName,
					Description: provider.Label + " API key used by dashboard/provider discovery.",
					Sensitive:   true,
				})
				added[envName] = struct{}{}
			}
		}
		for _, envName := range provider.BaseURLEnv {
			if _, exists := added[envName]; !exists {
				specs = append(specs, launcherEnvVariableSpec{
					Name:        envName,
					Description: provider.Label + " base URL override used by dashboard/provider discovery.",
				})
				added[envName] = struct{}{}
			}
		}
		for _, envName := range provider.ModelEnv {
			if _, exists := added[envName]; !exists {
				defaultValue := ""
				if len(provider.DefaultModels) > 0 {
					defaultValue = provider.DefaultModels[0]
				}
				specs = append(specs, launcherEnvVariableSpec{
					Name:         envName,
					Description:  provider.Label + " default model hint used by dashboard/provider discovery.",
					DefaultValue: defaultValue,
				})
				added[envName] = struct{}{}
			}
		}
	}
	return specs
}

func collectDynamicGWCPrefixedEnvSpecs(known map[string]struct{}) []launcherEnvVariableSpec {
	specs := []launcherEnvVariableSpec{}
	seen := map[string]struct{}{}
	for _, entry := range launcherEnvList() {
		key, _, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if key == "" || !strings.HasPrefix(strings.ToUpper(key), "GWC_") {
			continue
		}
		if _, exists := known[key]; exists {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		specs = append(specs, launcherEnvVariableSpec{
			Name:        key,
			Description: "Custom GWC-prefixed environment variable (not in the launcher-known variable catalog).",
			Sensitive:   launcherEnvNameLooksSensitive(key),
		})
	}
	sort.Slice(specs, func(i int, j int) bool { return specs[i].Name < specs[j].Name })
	return specs
}

func launcherEnvNameLooksSensitive(name string) bool {
	upper := strings.ToUpper(strings.TrimSpace(name))
	return strings.Contains(upper, "API_KEY") ||
		strings.Contains(upper, "TOKEN") ||
		strings.Contains(upper, "SECRET") ||
		strings.Contains(upper, "PASSWORD") ||
		strings.Contains(upper, "PASSWD") ||
		strings.Contains(upper, "PRIVATE_KEY")
}

func launcherEnvDisplayValue(value string, sensitive bool, showSecrets bool) (string, bool) {
	if !sensitive || showSecrets {
		return value, false
	}
	return fmt.Sprintf("[redacted len=%d]", len(value)), true
}

func printLauncherEnvSummary(summary launcherEnvSummary) {
	fmt.Println("GWC env")
	for _, variable := range summary.Variables {
		state := "unset"
		value := "<unset>"
		if variable.Set {
			state = "set"
			value = variable.Value
		}
		fmt.Printf("  %s [%s]: %s\n", variable.Name, state, value)
		if strings.TrimSpace(variable.DefaultValue) != "" {
			fmt.Printf("    default: %s\n", variable.DefaultValue)
		}
		if strings.TrimSpace(variable.Description) != "" {
			fmt.Printf("    info:    %s\n", variable.Description)
		}
	}
}
