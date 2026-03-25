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

func parseLauncherGlobalCLIOptions(args []string) (launcherGlobalCLIOptions, []string, error) {
	options := launcherGlobalCLIOptions{}
	index := 0
	for index < len(args) {
		arg := strings.TrimSpace(args[index])
		if arg == "--" {
			index++
			break
		}
		if arg == "-h" || arg == "--help" {
			break
		}
		if !strings.HasPrefix(arg, "-") {
			break
		}
		switch arg {
		case "-policy-pack", "--policy-pack":
			if index+1 >= len(args) {
				return launcherGlobalCLIOptions{}, nil, fmt.Errorf("%s requires a path value", arg)
			}
			options.PolicyPackPath = strings.TrimSpace(args[index+1])
			index += 2
		case "-no-hooks", "--no-hooks":
			options.DisableHooks = true
			index++
		case "-no-plugins", "--no-plugins":
			options.DisablePlugins = true
			index++
		default:
			return launcherGlobalCLIOptions{}, nil, fmt.Errorf("unknown global flag %q", arg)
		}
	}
	return options, args[index:], nil
}

func hookPointName(phase string, command string) string {
	phase = strings.TrimSpace(strings.ToLower(phase))
	command = strings.TrimSpace(strings.ToLower(command))
	if phase == "" || command == "" {
		return ""
	}
	return phase + "-" + command
}

func runLauncherCommandHooks(hooks map[string][]launcherExecutableHook, phase string, command string, commandArgs []string, repoRoot string, sources launcherEnterpriseConfigSources) error {
	if len(hooks) == 0 {
		return nil
	}
	points := []string{
		hookPointName(phase, "all"),
		hookPointName(phase, command),
	}
	for _, point := range points {
		pointHooks := hooks[point]
		for _, hook := range pointHooks {
			if err := runSingleLauncherHook(point, hook, command, commandArgs, repoRoot, sources); err != nil {
				return err
			}
		}
	}
	return nil
}

func runSingleLauncherHook(point string, hook launcherExecutableHook, command string, commandArgs []string, repoRoot string, sources launcherEnterpriseConfigSources) error {
	path := strings.TrimSpace(hook.Path)
	if path == "" {
		return fmt.Errorf("hook %q has an empty executable path", hook.Name)
	}
	if err := validateLauncherExecutableSecurity("hook", hook.Name, path, hook.Trusted, launcherActiveEnterpriseConfig.Security); err != nil {
		if hook.AllowFailure {
			fmt.Printf("GWC hook warn: %s blocked by security boundary (%v); continuing because allowFailure=true\n", hook.Name, err)
			return nil
		}
		return err
	}
	if !launcherEnterprisePathExists(path) {
		if hook.AllowFailure {
			fmt.Printf("GWC hook warn: %s at %s does not exist; continuing because allowFailure=true\n", hook.Name, path)
			return nil
		}
		return fmt.Errorf("hook %q executable does not exist at %s", hook.Name, path)
	}

	request := launcherHookInvocationRequest{
		SchemaVersion: "gwc-hook-v1",
		HookPoint:     point,
		Command:       command,
		CommandArgs:   append([]string(nil), commandArgs...),
		RepoRoot:      repoRoot,
		Sources:       sources,
	}
	payload, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("encode hook request for %q: %w", hook.Name, err)
	}

	timeout := 30 * time.Second
	if hook.TimeoutMS > 0 {
		timeout = time.Duration(hook.TimeoutMS) * time.Millisecond
	}
	env, err := buildLauncherExtensionEnv(hook.InheritEnv, hook.Env, launcherActiveEnterpriseConfig.Security)
	if err != nil {
		if hook.AllowFailure {
			fmt.Printf("GWC hook warn: %s has invalid environment (%v); continuing because allowFailure=true\n", hook.Name, err)
			return nil
		}
		return fmt.Errorf("hook %q environment: %w", hook.Name, err)
	}

	stdout, stderr, err := launcherRunHookProcess(path, hook.Args, env, payload, timeout)
	if err != nil {
		if hook.AllowFailure {
			fmt.Printf("GWC hook warn: %s failed (%v); continuing because allowFailure=true\n", hook.Name, err)
			return nil
		}
		details := strings.TrimSpace(stderr)
		if details == "" {
			details = strings.TrimSpace(stdout)
		}
		if details == "" {
			return fmt.Errorf("hook %q failed: %w", hook.Name, err)
		}
		return fmt.Errorf("hook %q failed: %w (%s)", hook.Name, err, details)
	}

	responseText := strings.TrimSpace(stdout)
	if responseText == "" {
		return nil
	}
	var response launcherHookInvocationResponse
	if err := json.Unmarshal([]byte(responseText), &response); err != nil {
		if hook.AllowFailure {
			fmt.Printf("GWC hook warn: %s returned non-JSON output; continuing because allowFailure=true\n", hook.Name)
			return nil
		}
		return fmt.Errorf("hook %q returned invalid JSON: %w", hook.Name, err)
	}
	if response.OK != nil && !*response.OK {
		message := strings.TrimSpace(response.Summary)
		if message == "" {
			message = "hook reported failure"
		}
		if len(response.Diagnostics) > 0 {
			message = message + " | " + strings.Join(response.Diagnostics, " | ")
		}
		if hook.AllowFailure {
			fmt.Printf("GWC hook warn: %s reported failure (%s); continuing because allowFailure=true\n", hook.Name, message)
			return nil
		}
		return fmt.Errorf("hook %q reported failure: %s", hook.Name, message)
	}
	return nil
}

func runLauncherHookProcess(path string, args []string, env []string, stdin []byte, timeout time.Duration) (string, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, path, args...)
	if len(env) > 0 {
		cmd.Env = env
	}
	cmd.Stdin = strings.NewReader(string(stdin))
	output, err := cmd.CombinedOutput()
	stdout := string(output)
	stderr := ""
	if ctx.Err() == context.DeadlineExceeded {
		return stdout, stderr, fmt.Errorf("timed out after %s", timeout)
	}
	return stdout, stderr, err
}

func runLauncherPluginsForCapability(capability string, command string, commandArgs []string, repoRoot string, sources launcherEnterpriseConfigSources) ([]launcherPluginExecutionResult, error) {
	capability = strings.TrimSpace(strings.ToLower(capability))
	if capability == "" {
		return nil, nil
	}
	results := []launcherPluginExecutionResult{}
	for _, plugin := range launcherActiveEnterpriseConfig.Plugins {
		if !pluginHasCapability(plugin, capability) {
			continue
		}
		if err := validateLauncherExecutableSecurity("plugin", plugin.Name, plugin.Path, plugin.Trusted, launcherActiveEnterpriseConfig.Security); err != nil {
			return nil, err
		}
		if !launcherEnterprisePathExists(plugin.Path) {
			return nil, fmt.Errorf("plugin %q executable does not exist at %s", plugin.Name, plugin.Path)
		}
		request := launcherPluginInvocationRequest{
			SchemaVersion: "gwc-plugin-v1",
			PluginName:    plugin.Name,
			Capability:    capability,
			Command:       command,
			CommandArgs:   append([]string(nil), commandArgs...),
			RepoRoot:      repoRoot,
			Sources:       sources,
		}
		payload, err := json.Marshal(request)
		if err != nil {
			return nil, fmt.Errorf("encode plugin request for %q: %w", plugin.Name, err)
		}

		timeout := 30 * time.Second
		env, err := buildLauncherExtensionEnv(plugin.InheritEnv, plugin.Env, launcherActiveEnterpriseConfig.Security)
		if err != nil {
			return nil, fmt.Errorf("plugin %q environment: %w", plugin.Name, err)
		}
		stdout, stderr, err := launcherRunPluginProcess(plugin.Path, plugin.Args, env, payload, timeout)
		if err != nil {
			details := strings.TrimSpace(stderr)
			if details == "" {
				details = strings.TrimSpace(stdout)
			}
			if details == "" {
				return nil, fmt.Errorf("plugin %q failed: %w", plugin.Name, err)
			}
			return nil, fmt.Errorf("plugin %q failed: %w (%s)", plugin.Name, err, details)
		}

		response := launcherPluginInvocationResponse{}
		responseText := strings.TrimSpace(stdout)
		if responseText != "" {
			if err := json.Unmarshal([]byte(responseText), &response); err != nil {
				return nil, fmt.Errorf("plugin %q returned invalid JSON: %w", plugin.Name, err)
			}
		}
		if response.OK != nil && !*response.OK {
			summary := strings.TrimSpace(response.Summary)
			if summary == "" {
				summary = "plugin reported failure"
			}
			return nil, fmt.Errorf("plugin %q reported failure: %s", plugin.Name, summary)
		}
		results = append(results, launcherPluginExecutionResult{
			Plugin:   plugin,
			Response: response,
		})
	}
	return results, nil
}

func pluginHasCapability(plugin launcherExecutablePlugin, capability string) bool {
	capability = strings.TrimSpace(strings.ToLower(capability))
	for _, candidate := range plugin.Capabilities {
		if strings.TrimSpace(strings.ToLower(candidate)) == capability {
			return true
		}
	}
	return false
}

func enforcePluginChecks(results []launcherPluginExecutionResult, capability string) error {
	switch strings.TrimSpace(strings.ToLower(capability)) {
	case "verify_check":
		for _, result := range results {
			for _, check := range result.Response.VerifyChecks {
				if !check.Passed {
					summary := strings.TrimSpace(check.Summary)
					if summary == "" {
						summary = "verify check failed"
					}
					diagnostics := formatPluginDiagnostics(result.Response.Diagnostics)
					if diagnostics != "" {
						return fmt.Errorf("plugin %q verify check %q failed: %s (diagnostics: %s)", result.Plugin.Name, check.Name, summary, diagnostics)
					}
					return fmt.Errorf("plugin %q verify check %q failed: %s", result.Plugin.Name, check.Name, summary)
				}
			}
		}
	case "release_validator":
		for _, result := range results {
			for _, check := range result.Response.ReleaseValidators {
				if !check.Passed {
					summary := strings.TrimSpace(check.Summary)
					if summary == "" {
						summary = "release validator failed"
					}
					diagnostics := formatPluginDiagnostics(result.Response.Diagnostics)
					if diagnostics != "" {
						return fmt.Errorf("plugin %q release validator %q failed: %s (diagnostics: %s)", result.Plugin.Name, check.Name, summary, diagnostics)
					}
					return fmt.Errorf("plugin %q release validator %q failed: %s", result.Plugin.Name, check.Name, summary)
				}
			}
		}
	}
	return nil
}

func formatPluginDiagnostics(diagnostics []launcherPluginDiagnostic) string {
	if len(diagnostics) == 0 {
		return ""
	}
	formatted := make([]string, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		code := strings.TrimSpace(diagnostic.Code)
		summary := strings.TrimSpace(diagnostic.Summary)
		if code == "" && summary == "" {
			continue
		}
		if code == "" {
			formatted = append(formatted, summary)
			continue
		}
		if summary == "" {
			formatted = append(formatted, code)
			continue
		}
		formatted = append(formatted, code+": "+summary)
	}
	return strings.Join(formatted, " | ")
}

func printLauncherExtensionReport(command string, layered launcherEnterpriseLayeredConfig) {
	writer := launcherExtensionReportWriter
	if writer == nil {
		return
	}
	fmt.Fprintln(writer, "GWC extension report")
	fmt.Fprintf(writer, "  command: %s\n", command)
	if strings.TrimSpace(layered.Sources.OrganizationPolicyPath) != "" {
		fmt.Fprintf(writer, "  org policy: %s\n", layered.Sources.OrganizationPolicyPath)
	} else {
		fmt.Fprintln(writer, "  org policy: (none)")
	}
	if strings.TrimSpace(layered.Sources.ProjectConfigPath) != "" {
		fmt.Fprintf(writer, "  project config: %s\n", layered.Sources.ProjectConfigPath)
	} else {
		fmt.Fprintln(writer, "  project config: (none)")
	}
	fmt.Fprintf(
		writer,
		"  security: allowUntrusted=%t executableRoots=%d inheritedEnvAllow=%d inheritedEnvDeny=%d\n",
		launcherAllowUntrustedExtensions(layered.Effective.Security),
		len(layered.Effective.Security.AllowedExecutableRoots),
		len(layered.Effective.Security.InheritedEnvAllowlist),
		len(layered.Effective.Security.InheritedEnvDenylist),
	)

	hookKeys := make([]string, 0, len(layered.Effective.Hooks))
	for key := range layered.Effective.Hooks {
		hookKeys = append(hookKeys, key)
	}
	sort.Strings(hookKeys)
	if len(hookKeys) == 0 {
		fmt.Fprintln(writer, "  hooks: (none)")
	} else {
		fmt.Fprintln(writer, "  hooks:")
		for _, key := range hookKeys {
			hooks := layered.Effective.Hooks[key]
			trustedCount := 0
			for _, hook := range hooks {
				if hook.Trusted {
					trustedCount++
				}
			}
			fmt.Fprintf(writer, "    - %s: %d loaded (%d trusted, %d untrusted)\n", key, len(hooks), trustedCount, len(hooks)-trustedCount)
		}
	}

	if len(layered.Effective.Plugins) == 0 {
		fmt.Fprintln(writer, "  plugins: (none)")
	} else {
		fmt.Fprintln(writer, "  plugins:")
		for _, plugin := range layered.Effective.Plugins {
			trust := "untrusted"
			if plugin.Trusted {
				trust = "trusted"
			}
			capabilities := strings.Join(plugin.Capabilities, ",")
			if strings.TrimSpace(capabilities) == "" {
				capabilities = "(none)"
			}
			fmt.Fprintf(writer, "    - %s [%s] caps=%s path=%s\n", plugin.Name, trust, capabilities, plugin.Path)
		}
	}
}

func validateLauncherExtensionSecurity(config launcherEnterpriseConfig) error {
	for hookPoint, hooks := range config.Hooks {
		for _, hook := range hooks {
			if err := validateLauncherExecutableSecurity("hook "+hookPoint, hook.Name, hook.Path, hook.Trusted, config.Security); err != nil {
				return err
			}
		}
	}
	for _, plugin := range config.Plugins {
		if err := validateLauncherExecutableSecurity("plugin", plugin.Name, plugin.Path, plugin.Trusted, config.Security); err != nil {
			return err
		}
	}
	return nil
}

func validateLauncherExecutableSecurity(kind string, name string, path string, trusted bool, security launcherEnterpriseSecurityPolicy) error {
	displayName := strings.TrimSpace(name)
	if displayName == "" {
		displayName = strings.TrimSpace(path)
	}
	if !trusted && !launcherAllowUntrustedExtensions(security) {
		return fmt.Errorf("%s %q is untrusted and blocked by enterprise security policy", kind, displayName)
	}
	if len(security.AllowedExecutableRoots) == 0 {
		return nil
	}
	for _, root := range security.AllowedExecutableRoots {
		if launcherPathWithinRoot(path, root) {
			return nil
		}
	}
	return fmt.Errorf("%s %q at %s is outside allowed executable roots", kind, displayName, path)
}

func launcherAllowUntrustedExtensions(security launcherEnterpriseSecurityPolicy) bool {
	if security.AllowUntrustedExtensions == nil {
		return true
	}
	return *security.AllowUntrustedExtensions
}

func launcherPathWithinRoot(path string, root string) bool {
	normalizedPath, err := filepath.Abs(strings.TrimSpace(path))
	if err != nil {
		return false
	}
	normalizedRoot, err := filepath.Abs(strings.TrimSpace(root))
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(normalizedRoot, normalizedPath)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	return !strings.HasPrefix(rel, "..")
}

func buildLauncherExtensionEnv(inherit bool, explicit map[string]string, security launcherEnterpriseSecurityPolicy) ([]string, error) {
	merged := map[string]string{}
	if inherit {
		for _, pair := range os.Environ() {
			name, value, ok := strings.Cut(pair, "=")
			if !ok {
				continue
			}
			if !launcherInheritedEnvAllowed(name, security) {
				continue
			}
			merged[name] = value
		}
	}
	for key, rawValue := range explicit {
		name := strings.TrimSpace(key)
		if name == "" {
			continue
		}
		if launcherEnvDenied(name, security) {
			return nil, fmt.Errorf("environment variable %q is blocked by inheritedEnvDenylist", name)
		}
		resolved, err := resolveLauncherExtensionEnvValue(rawValue)
		if err != nil {
			return nil, fmt.Errorf("resolve environment variable %q: %w", name, err)
		}
		merged[name] = resolved
	}
	if len(merged) == 0 {
		return []string{}, nil
	}
	keys := make([]string, 0, len(merged))
	for key := range merged {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	env := make([]string, 0, len(keys))
	for _, key := range keys {
		env = append(env, fmt.Sprintf("%s=%s", key, merged[key]))
	}
	return env, nil
}

func launcherInheritedEnvAllowed(name string, security launcherEnterpriseSecurityPolicy) bool {
	if launcherEnvDenied(name, security) {
		return false
	}
	if len(security.InheritedEnvAllowlist) == 0 {
		return true
	}
	candidate := strings.ToUpper(strings.TrimSpace(name))
	for _, allowed := range security.InheritedEnvAllowlist {
		if candidate == strings.ToUpper(strings.TrimSpace(allowed)) {
			return true
		}
	}
	return false
}

func launcherEnvDenied(name string, security launcherEnterpriseSecurityPolicy) bool {
	candidate := strings.ToUpper(strings.TrimSpace(name))
	for _, denied := range security.InheritedEnvDenylist {
		if candidate == strings.ToUpper(strings.TrimSpace(denied)) {
			return true
		}
	}
	return false
}

func resolveLauncherExtensionEnvValue(rawValue string) (string, error) {
	value := strings.TrimSpace(rawValue)
	if strings.HasPrefix(value, "${ENV:") && strings.HasSuffix(value, "}") {
		envKey := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(value, "${ENV:"), "}"))
		if envKey == "" {
			return "", fmt.Errorf("empty env key in %q", rawValue)
		}
		secretValue, ok := launcherEnterpriseLookupEnv(envKey)
		if !ok {
			return "", fmt.Errorf("environment variable %q is not set", envKey)
		}
		return secretValue, nil
	}
	return rawValue, nil
}
