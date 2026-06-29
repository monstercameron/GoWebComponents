package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type launcherHookInvocationRequest struct {
	SchemaVersion string                          `json:"schemaVersion"`
	HookPoint     string                          `json:"hookPoint"`
	Command       string                          `json:"command"`
	CommandArgs   []string                        `json:"commandArgs,omitempty"`
	RepoRoot      string                          `json:"repoRoot,omitempty"`
	Sources       launcherEnterpriseConfigSources `json:"sources"`
}

type launcherHookInvocationResponse struct {
	OK          *bool    `json:"ok,omitempty"`
	Summary     string   `json:"summary,omitempty"`
	Diagnostics []string `json:"diagnostics,omitempty"`
}

type launcherPluginExecutionResult struct {
	Plugin   launcherExecutablePlugin
	Response launcherPluginInvocationResponse
}

var launcherActiveEnterpriseConfig = defaultLauncherEnterpriseConfig()

var launcherActiveEnterpriseSources = launcherEnterpriseConfigSources{FrameworkDefaults: true}

var launcherRunHookProcess = runLauncherHookProcess

var launcherRunPluginProcess = runLauncherHookProcess

var launcherExtensionReportWriter io.Writer = os.Stderr

var launcherPrintExtensionReport = printLauncherExtensionReport

var launcherEnterpriseLookupEnv = os.LookupEnv

func parseLauncherGlobalCLIOptions(parseArgs []string) (launcherGlobalCLIOptions, []string, error) {
	parseOptions := launcherGlobalCLIOptions{}
	parseIndex := 0
	for parseIndex < len(parseArgs) {
		parseArg := strings.TrimSpace(parseArgs[parseIndex])
		if parseArg == "--" {
			parseIndex++
			break
		}
		if parseArg == "-h" || parseArg == "--help" {
			break
		}
		if !strings.HasPrefix(parseArg, "-") {
			break
		}
		switch parseArg {
		case "-policy-pack", "--policy-pack":
			if parseIndex+1 >= len(parseArgs) {
				return launcherGlobalCLIOptions{}, nil, fmt.Errorf("%s requires a path value", parseArg)
			}
			parseOptions.PolicyPackPath = strings.TrimSpace(parseArgs[parseIndex+1])
			parseIndex += 2
		case "-no-hooks", "--no-hooks":
			parseOptions.DisableHooks = true
			parseIndex++
		case "-no-plugins", "--no-plugins":
			parseOptions.DisablePlugins = true
			parseIndex++
		default:
			return launcherGlobalCLIOptions{}, nil, fmt.Errorf("unknown global flag %q", parseArg)
		}
	}
	return parseOptions, parseArgs[parseIndex:], nil
}

func hookPointName(parsePhase string, parseCommand string) string {
	parsePhase = strings.TrimSpace(strings.ToLower(parsePhase))
	parseCommand = strings.TrimSpace(strings.ToLower(parseCommand))
	if parsePhase == "" || parseCommand == "" {
		return ""
	}
	return parsePhase + "-" + parseCommand
}

func runLauncherCommandHooks(parseHooks map[string][]launcherExecutableHook, parsePhase string, parseCommand string, parseCommandArgs []string, parseRepoRoot string, parseSources launcherEnterpriseConfigSources) error {
	if len(parseHooks) == 0 {
		return nil
	}
	parsePoints := []string{
		hookPointName(parsePhase, "all"),
		hookPointName(parsePhase, parseCommand),
	}
	for _, parsePoint := range parsePoints {
		parsePointHooks := parseHooks[parsePoint]
		for _, parseHook := range parsePointHooks {
			if parseErr := runSingleLauncherHook(parsePoint, parseHook, parseCommand, parseCommandArgs, parseRepoRoot, parseSources); parseErr != nil {
				return parseErr
			}
		}
	}
	return nil
}

func runSingleLauncherHook(parsePoint string, parseHook launcherExecutableHook, parseCommand string, parseCommandArgs []string, parseRepoRoot string, parseSources launcherEnterpriseConfigSources) error {
	parsePath := strings.TrimSpace(parseHook.Path)
	if parsePath == "" {
		return fmt.Errorf("hook %q has an empty executable path", parseHook.Name)
	}
	if parseErr := validateLauncherExecutableSecurity("hook", parseHook.Name, parsePath, parseHook.Trusted, launcherActiveEnterpriseConfig.Security); parseErr != nil {
		if parseHook.AllowFailure {
			fmt.Printf("GWC hook warn: %s blocked by security boundary (%v); continuing because allowFailure=true\n", parseHook.Name, parseErr)
			return nil
		}
		return parseErr
	}
	if !launcherEnterprisePathExists(parsePath) {
		if parseHook.AllowFailure {
			fmt.Printf("GWC hook warn: %s at %s does not exist; continuing because allowFailure=true\n", parseHook.Name, parsePath)
			return nil
		}
		return fmt.Errorf("hook %q executable does not exist at %s", parseHook.Name, parsePath)
	}

	parseRequest := launcherHookInvocationRequest{
		SchemaVersion: "gwc-hook-v1",
		HookPoint:     parsePoint,
		Command:       parseCommand,
		CommandArgs:   append([]string(nil), parseCommandArgs...),
		RepoRoot:      parseRepoRoot,
		Sources:       parseSources,
	}
	parsePayload, parseErr2 := json.Marshal(parseRequest)
	if parseErr2 != nil {
		return fmt.Errorf("encode hook request for %q: %w", parseHook.Name, parseErr2)
	}

	parseTimeout := 30 * time.Second
	if parseHook.TimeoutMS > 0 {
		parseTimeout = time.Duration(parseHook.TimeoutMS) * time.Millisecond
	}
	parseEnv, parseErr2 := buildLauncherExtensionEnv(parseHook.InheritEnv, parseHook.Env, launcherActiveEnterpriseConfig.Security)
	if parseErr2 != nil {
		if parseHook.AllowFailure {
			fmt.Printf("GWC hook warn: %s has invalid environment (%v); continuing because allowFailure=true\n", parseHook.Name, parseErr2)
			return nil
		}
		return fmt.Errorf("hook %q environment: %w", parseHook.Name, parseErr2)
	}

	parseStdout, parseStderr, parseErr2 := launcherRunHookProcess(parsePath, parseHook.Args, parseEnv, parsePayload, parseTimeout)
	if parseErr2 != nil {
		if parseHook.AllowFailure {
			fmt.Printf("GWC hook warn: %s failed (%v); continuing because allowFailure=true\n", parseHook.Name, parseErr2)
			return nil
		}
		parseDetails := strings.TrimSpace(parseStderr)
		if parseDetails == "" {
			parseDetails = strings.TrimSpace(parseStdout)
		}
		if parseDetails == "" {
			return fmt.Errorf("hook %q failed: %w", parseHook.Name, parseErr2)
		}
		return fmt.Errorf("hook %q failed: %w (%s)", parseHook.Name, parseErr2, parseDetails)
	}

	parseResponseText := strings.TrimSpace(parseStdout)
	if parseResponseText == "" {
		return nil
	}
	var parseResponse launcherHookInvocationResponse
	if parseErr3 := json.Unmarshal([]byte(parseResponseText), &parseResponse); parseErr3 != nil {
		if parseHook.AllowFailure {
			fmt.Printf("GWC hook warn: %s returned non-JSON output; continuing because allowFailure=true\n", parseHook.Name)
			return nil
		}
		return fmt.Errorf("hook %q returned invalid JSON: %w", parseHook.Name, parseErr3)
	}
	if parseResponse.OK != nil && !*parseResponse.OK {
		parseMessage := strings.TrimSpace(parseResponse.Summary)
		if parseMessage == "" {
			parseMessage = "hook reported failure"
		}
		if len(parseResponse.Diagnostics) > 0 {
			parseMessage = parseMessage + " | " + strings.Join(parseResponse.Diagnostics, " | ")
		}
		if parseHook.AllowFailure {
			fmt.Printf("GWC hook warn: %s reported failure (%s); continuing because allowFailure=true\n", parseHook.Name, parseMessage)
			return nil
		}
		return fmt.Errorf("hook %q reported failure: %s", parseHook.Name, parseMessage)
	}
	return nil
}

func runLauncherHookProcess(parsePath string, parseArgs []string, parseEnv []string, parseStdin []byte, parseTimeout time.Duration) (string, string, error) {
	parseCtx, parseCancel := context.WithTimeout(context.Background(), parseTimeout)
	defer parseCancel()

	parseCmd := exec.CommandContext(parseCtx, parsePath, parseArgs...)
	if len(parseEnv) > 0 {
		parseCmd.Env = parseEnv
	}
	parseCmd.Stdin = strings.NewReader(string(parseStdin))
	parseOutput, parseErr := parseCmd.CombinedOutput()
	parseStdout := string(parseOutput)
	parseStderr := ""
	if parseCtx.Err() == context.DeadlineExceeded {
		return parseStdout, parseStderr, fmt.Errorf("timed out after %s", parseTimeout)
	}
	return parseStdout, parseStderr, parseErr
}

func runLauncherPluginsForCapability(parseCapability string, parseCommand string, parseCommandArgs []string, parseRepoRoot string, parseSources launcherEnterpriseConfigSources) ([]launcherPluginExecutionResult, error) {
	parseCapability = strings.TrimSpace(strings.ToLower(parseCapability))
	if parseCapability == "" {
		return nil, nil
	}
	parseResults := []launcherPluginExecutionResult{}
	for _, parsePlugin := range launcherActiveEnterpriseConfig.Plugins {
		if !pluginHasCapability(parsePlugin, parseCapability) {
			continue
		}
		if parseErr := validateLauncherExecutableSecurity("plugin", parsePlugin.Name, parsePlugin.Path, parsePlugin.Trusted, launcherActiveEnterpriseConfig.Security); parseErr != nil {
			return nil, parseErr
		}
		if !launcherEnterprisePathExists(parsePlugin.Path) {
			return nil, fmt.Errorf("plugin %q executable does not exist at %s", parsePlugin.Name, parsePlugin.Path)
		}
		parseRequest := launcherPluginInvocationRequest{
			SchemaVersion: "gwc-plugin-v1",
			PluginName:    parsePlugin.Name,
			Capability:    parseCapability,
			Command:       parseCommand,
			CommandArgs:   append([]string(nil), parseCommandArgs...),
			RepoRoot:      parseRepoRoot,
			Sources:       parseSources,
		}
		parsePayload, parseErr2 := json.Marshal(parseRequest)
		if parseErr2 != nil {
			return nil, fmt.Errorf("encode plugin request for %q: %w", parsePlugin.Name, parseErr2)
		}

		parseTimeout := 30 * time.Second
		parseEnv, parseErr2 := buildLauncherExtensionEnv(parsePlugin.InheritEnv, parsePlugin.Env, launcherActiveEnterpriseConfig.Security)
		if parseErr2 != nil {
			return nil, fmt.Errorf("plugin %q environment: %w", parsePlugin.Name, parseErr2)
		}
		parseStdout, parseStderr, parseErr2 := launcherRunPluginProcess(parsePlugin.Path, parsePlugin.Args, parseEnv, parsePayload, parseTimeout)
		if parseErr2 != nil {
			parseDetails := strings.TrimSpace(parseStderr)
			if parseDetails == "" {
				parseDetails = strings.TrimSpace(parseStdout)
			}
			if parseDetails == "" {
				return nil, fmt.Errorf("plugin %q failed: %w", parsePlugin.Name, parseErr2)
			}
			return nil, fmt.Errorf("plugin %q failed: %w (%s)", parsePlugin.Name, parseErr2, parseDetails)
		}

		parseResponse := launcherPluginInvocationResponse{}
		parseResponseText := strings.TrimSpace(parseStdout)
		if parseResponseText != "" {
			if parseErr3 := json.Unmarshal([]byte(parseResponseText), &parseResponse); parseErr3 != nil {
				return nil, fmt.Errorf("plugin %q returned invalid JSON: %w", parsePlugin.Name, parseErr3)
			}
		}
		if parseResponse.OK != nil && !*parseResponse.OK {
			parseSummary := strings.TrimSpace(parseResponse.Summary)
			if parseSummary == "" {
				parseSummary = "plugin reported failure"
			}
			return nil, fmt.Errorf("plugin %q reported failure: %s", parsePlugin.Name, parseSummary)
		}
		parseResults = append(parseResults, launcherPluginExecutionResult{
			Plugin:   parsePlugin,
			Response: parseResponse,
		})
	}
	return parseResults, nil
}

func pluginHasCapability(parsePlugin launcherExecutablePlugin, parseCapability string) bool {
	parseCapability = strings.TrimSpace(strings.ToLower(parseCapability))
	for _, parseCandidate := range parsePlugin.Capabilities {
		if strings.TrimSpace(strings.ToLower(parseCandidate)) == parseCapability {
			return true
		}
	}
	return false
}

func enforcePluginChecks(parseResults []launcherPluginExecutionResult, parseCapability string) error {
	switch strings.TrimSpace(strings.ToLower(parseCapability)) {
	case "verify_check":
		for _, parseResult := range parseResults {
			for _, parseCheck := range parseResult.Response.VerifyChecks {
				if !parseCheck.Passed {
					parseSummary := strings.TrimSpace(parseCheck.Summary)
					if parseSummary == "" {
						parseSummary = "verify check failed"
					}
					parseDiagnostics := formatPluginDiagnostics(parseResult.Response.Diagnostics)
					if parseDiagnostics != "" {
						return fmt.Errorf("plugin %q verify check %q failed: %s (diagnostics: %s)", parseResult.Plugin.Name, parseCheck.Name, parseSummary, parseDiagnostics)
					}
					return fmt.Errorf("plugin %q verify check %q failed: %s", parseResult.Plugin.Name, parseCheck.Name, parseSummary)
				}
			}
		}
	case "release_validator":
		for _, parseResult2 := range parseResults {
			for _, parseCheck2 := range parseResult2.Response.ReleaseValidators {
				if !parseCheck2.Passed {
					parseSummary2 := strings.TrimSpace(parseCheck2.Summary)
					if parseSummary2 == "" {
						parseSummary2 = "release validator failed"
					}
					parseDiagnostics2 := formatPluginDiagnostics(parseResult2.Response.Diagnostics)
					if parseDiagnostics2 != "" {
						return fmt.Errorf("plugin %q release validator %q failed: %s (diagnostics: %s)", parseResult2.Plugin.Name, parseCheck2.Name, parseSummary2, parseDiagnostics2)
					}
					return fmt.Errorf("plugin %q release validator %q failed: %s", parseResult2.Plugin.Name, parseCheck2.Name, parseSummary2)
				}
			}
		}
	}
	return nil
}

func formatPluginDiagnostics(parseDiagnostics []launcherPluginDiagnostic) string {
	if len(parseDiagnostics) == 0 {
		return ""
	}
	parseFormatted := make([]string, 0, len(parseDiagnostics))
	for _, parseDiagnostic := range parseDiagnostics {
		parseCode := strings.TrimSpace(parseDiagnostic.Code)
		parseSummary := strings.TrimSpace(parseDiagnostic.Summary)
		if parseCode == "" && parseSummary == "" {
			continue
		}
		if parseCode == "" {
			parseFormatted = append(parseFormatted, parseSummary)
			continue
		}
		if parseSummary == "" {
			parseFormatted = append(parseFormatted, parseCode)
			continue
		}
		parseFormatted = append(parseFormatted, parseCode+": "+parseSummary)
	}
	return strings.Join(parseFormatted, " | ")
}

func printLauncherExtensionReport(parseCommand string, parseLayered launcherEnterpriseLayeredConfig) {
	parseWriter := launcherExtensionReportWriter
	if parseWriter == nil {
		return
	}
	fmt.Fprintln(parseWriter, "GWC extension report")
	fmt.Fprintf(parseWriter, "  command: %s\n", parseCommand)
	if strings.TrimSpace(parseLayered.Sources.OrganizationPolicyPath) != "" {
		fmt.Fprintf(parseWriter, "  org policy: %s\n", parseLayered.Sources.OrganizationPolicyPath)
	} else {
		fmt.Fprintln(parseWriter, "  org policy: (none)")
	}
	if strings.TrimSpace(parseLayered.Sources.ProjectConfigPath) != "" {
		fmt.Fprintf(parseWriter, "  project config: %s\n", parseLayered.Sources.ProjectConfigPath)
	} else {
		fmt.Fprintln(parseWriter, "  project config: (none)")
	}
	fmt.Fprintf(
		parseWriter,
		"  security: allowUntrusted=%t executableRoots=%d inheritedEnvAllow=%d inheritedEnvDeny=%d\n",
		launcherAllowUntrustedExtensions(parseLayered.Effective.Security),
		len(parseLayered.Effective.Security.AllowedExecutableRoots),
		len(parseLayered.Effective.Security.InheritedEnvAllowlist),
		len(parseLayered.Effective.Security.InheritedEnvDenylist),
	)

	parseHookKeys := make([]string, 0, len(parseLayered.Effective.Hooks))
	for parseKey := range parseLayered.Effective.Hooks {
		parseHookKeys = append(parseHookKeys, parseKey)
	}
	sort.Strings(parseHookKeys)
	if len(parseHookKeys) == 0 {
		fmt.Fprintln(parseWriter, "  hooks: (none)")
	} else {
		fmt.Fprintln(parseWriter, "  hooks:")
		for _, parseKey2 := range parseHookKeys {
			parseHooks := parseLayered.Effective.Hooks[parseKey2]
			parseTrustedCount := 0
			for _, parseHook := range parseHooks {
				if parseHook.Trusted {
					parseTrustedCount++
				}
			}
			fmt.Fprintf(parseWriter, "    - %s: %d loaded (%d trusted, %d untrusted)\n", parseKey2, len(parseHooks), parseTrustedCount, len(parseHooks)-parseTrustedCount)
		}
	}

	if len(parseLayered.Effective.Plugins) == 0 {
		fmt.Fprintln(parseWriter, "  plugins: (none)")
	} else {
		fmt.Fprintln(parseWriter, "  plugins:")
		for _, parsePlugin := range parseLayered.Effective.Plugins {
			parseTrust := "untrusted"
			if parsePlugin.Trusted {
				parseTrust = "trusted"
			}
			parseCapabilities := strings.Join(parsePlugin.Capabilities, ",")
			if strings.TrimSpace(parseCapabilities) == "" {
				parseCapabilities = "(none)"
			}
			fmt.Fprintf(parseWriter, "    - %s [%s] caps=%s path=%s\n", parsePlugin.Name, parseTrust, parseCapabilities, parsePlugin.Path)
		}
	}
}

func validateLauncherExtensionSecurity(parseConfig launcherEnterpriseConfig) error {
	for parseHookPoint, parseHooks := range parseConfig.Hooks {
		for _, parseHook := range parseHooks {
			if parseErr := validateLauncherExecutableSecurity("hook "+parseHookPoint, parseHook.Name, parseHook.Path, parseHook.Trusted, parseConfig.Security); parseErr != nil {
				return parseErr
			}
		}
	}
	for _, parsePlugin := range parseConfig.Plugins {
		if parseErr2 := validateLauncherExecutableSecurity("plugin", parsePlugin.Name, parsePlugin.Path, parsePlugin.Trusted, parseConfig.Security); parseErr2 != nil {
			return parseErr2
		}
	}
	return nil
}

func validateLauncherExecutableSecurity(parseKind string, parseName string, parsePath string, isTrusted bool, parseSecurity launcherEnterpriseSecurityPolicy) error {
	parseDisplayName := strings.TrimSpace(parseName)
	if parseDisplayName == "" {
		parseDisplayName = strings.TrimSpace(parsePath)
	}
	if !isTrusted && !launcherAllowUntrustedExtensions(parseSecurity) {
		return fmt.Errorf("%s %q is untrusted and blocked by enterprise security policy", parseKind, parseDisplayName)
	}
	if len(parseSecurity.AllowedExecutableRoots) == 0 {
		return nil
	}
	for _, parseRoot := range parseSecurity.AllowedExecutableRoots {
		if launcherPathWithinRoot(parsePath, parseRoot) {
			return nil
		}
	}
	return fmt.Errorf("%s %q at %s is outside allowed executable roots", parseKind, parseDisplayName, parsePath)
}

func launcherAllowUntrustedExtensions(parseSecurity launcherEnterpriseSecurityPolicy) bool {
	if parseSecurity.AllowUntrustedExtensions == nil {
		return true
	}
	return *parseSecurity.AllowUntrustedExtensions
}

func launcherPathWithinRoot(parsePath string, parseRoot string) bool {
	parseNormalizedPath, parseErr := filepath.Abs(strings.TrimSpace(parsePath))
	if parseErr != nil {
		return false
	}
	parseNormalizedRoot, parseErr := filepath.Abs(strings.TrimSpace(parseRoot))
	if parseErr != nil {
		return false
	}
	parseRel, parseErr := filepath.Rel(parseNormalizedRoot, parseNormalizedPath)
	if parseErr != nil {
		return false
	}
	if parseRel == "." {
		return true
	}
	return !strings.HasPrefix(parseRel, "..")
}

func buildLauncherExtensionEnv(isInherit bool, parseExplicit map[string]string, parseSecurity launcherEnterpriseSecurityPolicy) ([]string, error) {
	parseMerged := map[string]string{}
	if isInherit {
		for _, parsePair := range os.Environ() {
			parseName, parseValue, parseOk := strings.Cut(parsePair, "=")
			if !parseOk {
				continue
			}
			if !launcherInheritedEnvAllowed(parseName, parseSecurity) {
				continue
			}
			parseMerged[parseName] = parseValue
		}
	}
	for parseKey, parseRawValue := range parseExplicit {
		parseName2 := strings.TrimSpace(parseKey)
		if parseName2 == "" {
			continue
		}
		if launcherEnvDenied(parseName2, parseSecurity) {
			return nil, fmt.Errorf("environment variable %q is blocked by inheritedEnvDenylist", parseName2)
		}
		parseResolved, parseErr := resolveLauncherExtensionEnvValue(parseRawValue)
		if parseErr != nil {
			return nil, fmt.Errorf("resolve environment variable %q: %w", parseName2, parseErr)
		}
		parseMerged[parseName2] = parseResolved
	}
	if len(parseMerged) == 0 {
		return []string{}, nil
	}
	parseKeys := make([]string, 0, len(parseMerged))
	for parseKey2 := range parseMerged {
		parseKeys = append(parseKeys, parseKey2)
	}
	sort.Strings(parseKeys)
	parseEnv := make([]string, 0, len(parseKeys))
	for _, parseKey3 := range parseKeys {
		parseEnv = append(parseEnv, fmt.Sprintf("%s=%s", parseKey3, parseMerged[parseKey3]))
	}
	return parseEnv, nil
}

func launcherInheritedEnvAllowed(parseName string, parseSecurity launcherEnterpriseSecurityPolicy) bool {
	if launcherEnvDenied(parseName, parseSecurity) {
		return false
	}
	if len(parseSecurity.InheritedEnvAllowlist) == 0 {
		return true
	}
	parseCandidate := strings.ToUpper(strings.TrimSpace(parseName))
	for _, parseAllowed := range parseSecurity.InheritedEnvAllowlist {
		if parseCandidate == strings.ToUpper(strings.TrimSpace(parseAllowed)) {
			return true
		}
	}
	return false
}

func launcherEnvDenied(parseName string, parseSecurity launcherEnterpriseSecurityPolicy) bool {
	parseCandidate := strings.ToUpper(strings.TrimSpace(parseName))
	for _, parseDenied := range parseSecurity.InheritedEnvDenylist {
		if parseCandidate == strings.ToUpper(strings.TrimSpace(parseDenied)) {
			return true
		}
	}
	return false
}

func resolveLauncherExtensionEnvValue(parseRawValue string) (string, error) {
	parseValue := strings.TrimSpace(parseRawValue)
	if strings.HasPrefix(parseValue, "${ENV:") && strings.HasSuffix(parseValue, "}") {
		parseEnvKey := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(parseValue, "${ENV:"), "}"))
		if parseEnvKey == "" {
			return "", fmt.Errorf("empty env key in %q", parseRawValue)
		}
		parseSecretValue, parseOk := launcherEnterpriseLookupEnv(parseEnvKey)
		if !parseOk {
			return "", fmt.Errorf("environment variable %q is not set", parseEnvKey)
		}
		return parseSecretValue, nil
	}
	return parseRawValue, nil
}
