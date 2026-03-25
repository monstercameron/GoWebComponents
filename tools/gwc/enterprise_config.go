package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

const launcherPolicyPackEnvVar = "GWC_POLICY_PACK"

type launcherGlobalCLIOptions struct {
	PolicyPackPath string
	DisableHooks   bool
	DisablePlugins bool
}

type launcherEnterprisePolicy struct {
	RequiredTestLanes          []string `json:"requiredTestLanes,omitempty"`
	RequireReleaseBudgets      *bool    `json:"requireReleaseBudgets,omitempty"`
	ReleaseBinaryPattern       string   `json:"releaseBinaryPattern,omitempty"`
	ReleaseManifestPattern     string   `json:"releaseManifestPattern,omitempty"`
	RequiredReleaseCompression string   `json:"requiredReleaseCompression,omitempty"`
	ApprovedGoToolchains       []string `json:"approvedGoToolchains,omitempty"`
}

type launcherEnterpriseSecurityPolicy struct {
	AllowUntrustedExtensions *bool    `json:"allowUntrustedExtensions,omitempty"`
	AllowedExecutableRoots   []string `json:"allowedExecutableRoots,omitempty"`
	InheritedEnvAllowlist    []string `json:"inheritedEnvAllowlist,omitempty"`
	InheritedEnvDenylist     []string `json:"inheritedEnvDenylist,omitempty"`
}

type launcherExecutableHook struct {
	Name         string            `json:"name,omitempty"`
	Path         string            `json:"path,omitempty"`
	Args         []string          `json:"args,omitempty"`
	Env          map[string]string `json:"env,omitempty"`
	TimeoutMS    int               `json:"timeoutMs,omitempty"`
	AllowFailure bool              `json:"allowFailure,omitempty"`
	Trusted      bool              `json:"trusted,omitempty"`
	InheritEnv   bool              `json:"inheritEnv,omitempty"`
}

type launcherExecutablePlugin struct {
	Name         string            `json:"name,omitempty"`
	Path         string            `json:"path,omitempty"`
	Args         []string          `json:"args,omitempty"`
	Env          map[string]string `json:"env,omitempty"`
	Capabilities []string          `json:"capabilities,omitempty"`
	Trusted      bool              `json:"trusted,omitempty"`
	InheritEnv   bool              `json:"inheritEnv,omitempty"`
}

type launcherEnterpriseConfig struct {
	Hooks    map[string][]launcherExecutableHook `json:"hooks,omitempty"`
	Plugins  []launcherExecutablePlugin          `json:"plugins,omitempty"`
	Policy   launcherEnterprisePolicy            `json:"policy,omitempty"`
	Security launcherEnterpriseSecurityPolicy    `json:"security,omitempty"`
}

type launcherEnterpriseConfigFile struct {
	Enterprise launcherEnterpriseConfig `json:"enterprise,omitempty"`
}

type launcherEnterpriseConfigSources struct {
	FrameworkDefaults      bool   `json:"frameworkDefaults"`
	OrganizationPolicyPath string `json:"organizationPolicyPath,omitempty"`
	ProjectConfigPath      string `json:"projectConfigPath,omitempty"`
	CLIOverridePolicyPack  string `json:"cliOverridePolicyPack,omitempty"`
}

type launcherEnterpriseLayeredConfig struct {
	Effective launcherEnterpriseConfig        `json:"effective"`
	Sources   launcherEnterpriseConfigSources `json:"sources"`
}

var launcherEnterpriseReadFile = os.ReadFile

var launcherEnterpriseStat = os.Stat

var launcherEnterpriseGetenv = os.Getenv

var launcherPluginCapabilityAllowlist = map[string]struct{}{
	"scaffold_feature":    {},
	"test_lane":           {},
	"verify_check":        {},
	"release_validator":   {},
	"deployment_packager": {},
}

func defaultLauncherEnterpriseConfig() launcherEnterpriseConfig {
	return launcherEnterpriseConfig{
		Hooks:    map[string][]launcherExecutableHook{},
		Plugins:  []launcherExecutablePlugin{},
		Policy:   launcherEnterprisePolicy{},
		Security: launcherEnterpriseSecurityPolicy{},
	}
}

func resolveLauncherEnterpriseConfig(cwd string, cli launcherGlobalCLIOptions) (launcherEnterpriseLayeredConfig, error) {
	layered := launcherEnterpriseLayeredConfig{
		Effective: defaultLauncherEnterpriseConfig(),
		Sources: launcherEnterpriseConfigSources{
			FrameworkDefaults: true,
		},
	}

	orgPath, hasOrgPath, fromCLI, err := resolveOrganizationPolicyPackPath(cwd, cli.PolicyPackPath)
	if err != nil {
		return launcherEnterpriseLayeredConfig{}, err
	}
	if hasOrgPath {
		orgConfig, loadErr := loadEnterpriseConfigFromPath(orgPath)
		if loadErr != nil {
			return launcherEnterpriseLayeredConfig{}, fmt.Errorf("load organization policy pack: %w", loadErr)
		}
		layered.Effective = mergeLauncherEnterpriseConfig(layered.Effective, orgConfig)
		layered.Sources.OrganizationPolicyPath = orgPath
		if fromCLI {
			layered.Sources.CLIOverridePolicyPack = orgPath
		}
	}

	projectConfigPath := findLauncherOverrideInParents(cwd)
	if strings.TrimSpace(projectConfigPath) != "" {
		projectConfig, loadErr := loadEnterpriseConfigFromPath(projectConfigPath)
		if loadErr != nil {
			return launcherEnterpriseLayeredConfig{}, fmt.Errorf("load project enterprise config: %w", loadErr)
		}
		layered.Effective = mergeLauncherEnterpriseConfig(layered.Effective, projectConfig)
		layered.Sources.ProjectConfigPath = projectConfigPath
	}

	if cli.DisableHooks {
		layered.Effective.Hooks = map[string][]launcherExecutableHook{}
	}
	if cli.DisablePlugins {
		layered.Effective.Plugins = []launcherExecutablePlugin{}
	}

	return layered, nil
}

func resolveOrganizationPolicyPackPath(cwd string, cliPolicyPath string) (string, bool, bool, error) {
	if strings.TrimSpace(cliPolicyPath) != "" {
		resolved, err := resolveEnterpriseConfigPath(cwd, cliPolicyPath)
		if err != nil {
			return "", false, false, fmt.Errorf("resolve CLI policy pack path: %w", err)
		}
		if !launcherEnterprisePathExists(resolved) {
			return "", false, false, fmt.Errorf("CLI policy pack path does not exist: %s", resolved)
		}
		return resolved, true, true, nil
	}

	if envPath := strings.TrimSpace(launcherEnterpriseGetenv(launcherPolicyPackEnvVar)); envPath != "" {
		resolved, err := resolveEnterpriseConfigPath(cwd, envPath)
		if err != nil {
			return "", false, false, fmt.Errorf("resolve %s path: %w", launcherPolicyPackEnvVar, err)
		}
		if !launcherEnterprisePathExists(resolved) {
			return "", false, false, fmt.Errorf("%s path does not exist: %s", launcherPolicyPackEnvVar, resolved)
		}
		return resolved, true, false, nil
	}

	homeDir, err := launcherConfigUserHomeDir()
	if err != nil {
		return "", false, false, nil
	}
	defaultPath := filepath.Join(homeDir, ".gwc", "policy-pack.json")
	if !launcherEnterprisePathExists(defaultPath) {
		return "", false, false, nil
	}
	return defaultPath, true, false, nil
}

func resolveEnterpriseConfigPath(baseDir string, rawPath string) (string, error) {
	rawPath = strings.TrimSpace(rawPath)
	if rawPath == "" {
		return "", nil
	}
	if filepath.IsAbs(rawPath) {
		return filepath.Clean(rawPath), nil
	}
	if strings.TrimSpace(baseDir) == "" {
		baseDir = "."
	}
	return filepath.Abs(filepath.Join(baseDir, rawPath))
}

func launcherEnterprisePathExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	_, err := launcherEnterpriseStat(path)
	return err == nil
}

func loadEnterpriseConfigFromPath(path string) (launcherEnterpriseConfig, error) {
	content, err := launcherEnterpriseReadFile(path)
	if err != nil {
		return launcherEnterpriseConfig{}, fmt.Errorf("read config %s: %w", path, err)
	}

	var wrapped launcherEnterpriseConfigFile
	if err := json.Unmarshal(content, &wrapped); err != nil {
		return launcherEnterpriseConfig{}, fmt.Errorf("parse config %s: %w", path, err)
	}
	if hasLauncherEnterprisePayload(wrapped.Enterprise) {
		normalized, normalizeErr := normalizeLauncherEnterpriseConfig(path, wrapped.Enterprise)
		if normalizeErr != nil {
			return launcherEnterpriseConfig{}, normalizeErr
		}
		return normalized, nil
	}

	var direct launcherEnterpriseConfig
	if err := json.Unmarshal(content, &direct); err != nil {
		return launcherEnterpriseConfig{}, fmt.Errorf("parse config %s: %w", path, err)
	}
	normalized, normalizeErr := normalizeLauncherEnterpriseConfig(path, direct)
	if normalizeErr != nil {
		return launcherEnterpriseConfig{}, normalizeErr
	}
	return normalized, nil
}

func hasLauncherEnterprisePayload(config launcherEnterpriseConfig) bool {
	return len(config.Hooks) > 0 ||
		len(config.Plugins) > 0 ||
		hasLauncherEnterprisePolicyPayload(config.Policy) ||
		hasLauncherEnterpriseSecurityPayload(config.Security)
}

func hasLauncherEnterprisePolicyPayload(policy launcherEnterprisePolicy) bool {
	return len(policy.RequiredTestLanes) > 0 ||
		policy.RequireReleaseBudgets != nil ||
		strings.TrimSpace(policy.ReleaseBinaryPattern) != "" ||
		strings.TrimSpace(policy.ReleaseManifestPattern) != "" ||
		strings.TrimSpace(policy.RequiredReleaseCompression) != "" ||
		len(policy.ApprovedGoToolchains) > 0
}

func hasLauncherEnterpriseSecurityPayload(security launcherEnterpriseSecurityPolicy) bool {
	return security.AllowUntrustedExtensions != nil ||
		len(security.AllowedExecutableRoots) > 0 ||
		len(security.InheritedEnvAllowlist) > 0 ||
		len(security.InheritedEnvDenylist) > 0
}

func normalizeLauncherEnterpriseConfig(configPath string, config launcherEnterpriseConfig) (launcherEnterpriseConfig, error) {
	normalized := defaultLauncherEnterpriseConfig()
	normalized.Policy = config.Policy

	for hookPoint, hooks := range config.Hooks {
		point := normalizeHookPoint(hookPoint)
		if point == "" {
			continue
		}
		for _, hook := range hooks {
			path, err := resolveLauncherOverrideValue(configPath, hook.Path)
			if err != nil {
				return launcherEnterpriseConfig{}, fmt.Errorf("resolve hook path %q at %s: %w", hook.Path, configPath, err)
			}
			hook.Path = path
			hook.Name = strings.TrimSpace(hook.Name)
			if hook.Name == "" {
				hook.Name = filepath.Base(path)
			}
			normalized.Hooks[point] = append(normalized.Hooks[point], hook)
		}
	}

	for _, plugin := range config.Plugins {
		path, err := resolveLauncherOverrideValue(configPath, plugin.Path)
		if err != nil {
			return launcherEnterpriseConfig{}, fmt.Errorf("resolve plugin path %q at %s: %w", plugin.Path, configPath, err)
		}
		plugin.Path = path
		plugin.Name = strings.TrimSpace(plugin.Name)
		if plugin.Name == "" {
			plugin.Name = filepath.Base(path)
		}
		normalizedCapabilities, capabilityErr := normalizePluginCapabilities(plugin.Capabilities)
		if capabilityErr != nil {
			return launcherEnterpriseConfig{}, fmt.Errorf("normalize plugin capabilities for %q: %w", plugin.Name, capabilityErr)
		}
		plugin.Capabilities = normalizedCapabilities
		normalized.Plugins = append(normalized.Plugins, plugin)
	}

	normalized.Policy.RequiredTestLanes = normalizeStringList(config.Policy.RequiredTestLanes)
	if len(normalized.Policy.RequiredTestLanes) > 0 {
		laneValues, err := normalizeTestLanes(normalized.Policy.RequiredTestLanes)
		if err != nil {
			return launcherEnterpriseConfig{}, fmt.Errorf("normalize required test lanes at %s: %w", configPath, err)
		}
		normalized.Policy.RequiredTestLanes = laneValues
	}
	normalized.Policy.ReleaseBinaryPattern = strings.TrimSpace(config.Policy.ReleaseBinaryPattern)
	if normalized.Policy.ReleaseBinaryPattern != "" {
		if _, err := regexp.Compile(normalized.Policy.ReleaseBinaryPattern); err != nil {
			return launcherEnterpriseConfig{}, fmt.Errorf("compile release binary pattern at %s: %w", configPath, err)
		}
	}
	normalized.Policy.ReleaseManifestPattern = strings.TrimSpace(config.Policy.ReleaseManifestPattern)
	if normalized.Policy.ReleaseManifestPattern != "" {
		if _, err := regexp.Compile(normalized.Policy.ReleaseManifestPattern); err != nil {
			return launcherEnterpriseConfig{}, fmt.Errorf("compile release manifest pattern at %s: %w", configPath, err)
		}
	}
	normalized.Policy.RequiredReleaseCompression = strings.TrimSpace(strings.ToLower(config.Policy.RequiredReleaseCompression))
	if normalized.Policy.RequiredReleaseCompression != "" {
		compression, err := normalizeReleaseCompressionPolicy(normalized.Policy.RequiredReleaseCompression)
		if err != nil {
			return launcherEnterpriseConfig{}, fmt.Errorf("normalize required release compression at %s: %w", configPath, err)
		}
		normalized.Policy.RequiredReleaseCompression = compression
	}
	normalized.Policy.ApprovedGoToolchains = normalizeStringList(config.Policy.ApprovedGoToolchains)
	for index := range normalized.Policy.ApprovedGoToolchains {
		normalized.Policy.ApprovedGoToolchains[index] = strings.ToLower(strings.TrimSpace(normalized.Policy.ApprovedGoToolchains[index]))
	}

	normalized.Security = launcherEnterpriseSecurityPolicy{
		AllowUntrustedExtensions: config.Security.AllowUntrustedExtensions,
		AllowedExecutableRoots:   []string{},
		InheritedEnvAllowlist:    normalizeEnvNameList(config.Security.InheritedEnvAllowlist),
		InheritedEnvDenylist:     normalizeEnvNameList(config.Security.InheritedEnvDenylist),
	}
	for _, root := range config.Security.AllowedExecutableRoots {
		resolvedRoot, err := resolveLauncherOverrideValue(configPath, root)
		if err != nil {
			return launcherEnterpriseConfig{}, fmt.Errorf("resolve security allowed executable root %q at %s: %w", root, configPath, err)
		}
		normalized.Security.AllowedExecutableRoots = append(normalized.Security.AllowedExecutableRoots, resolvedRoot)
	}
	normalized.Security.AllowedExecutableRoots = normalizeStringList(normalized.Security.AllowedExecutableRoots)
	return normalized, nil
}

func normalizeHookPoint(point string) string {
	point = strings.TrimSpace(strings.ToLower(point))
	if point == "" {
		return ""
	}
	return point
}

func normalizePluginCapabilities(capabilities []string) ([]string, error) {
	normalized := normalizeStringList(capabilities)
	for _, capability := range normalized {
		key := strings.TrimSpace(strings.ToLower(capability))
		if _, ok := launcherPluginCapabilityAllowlist[key]; !ok {
			return nil, fmt.Errorf("unsupported capability %q", capability)
		}
	}
	slices.Sort(normalized)
	return normalized, nil
}

func normalizeStringList(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	normalized := make([]string, 0, len(values))
	for _, raw := range values {
		value := strings.TrimSpace(raw)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, value)
	}
	return normalized
}

func normalizeEnvNameList(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	normalized := make([]string, 0, len(values))
	for _, raw := range values {
		value := strings.ToUpper(strings.TrimSpace(raw))
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}
	return normalized
}

func mergeLauncherEnterpriseConfig(base launcherEnterpriseConfig, overlay launcherEnterpriseConfig) launcherEnterpriseConfig {
	merged := defaultLauncherEnterpriseConfig()
	for key, hooks := range base.Hooks {
		merged.Hooks[key] = append(merged.Hooks[key], hooks...)
	}
	for key, hooks := range overlay.Hooks {
		merged.Hooks[key] = append(merged.Hooks[key], hooks...)
	}

	merged.Plugins = append(merged.Plugins, base.Plugins...)
	pluginIndex := make(map[string]int, len(merged.Plugins))
	for index, plugin := range merged.Plugins {
		pluginIndex[strings.ToLower(strings.TrimSpace(plugin.Name))] = index
	}
	for _, plugin := range overlay.Plugins {
		key := strings.ToLower(strings.TrimSpace(plugin.Name))
		if existing, ok := pluginIndex[key]; ok {
			merged.Plugins[existing] = plugin
			continue
		}
		pluginIndex[key] = len(merged.Plugins)
		merged.Plugins = append(merged.Plugins, plugin)
	}

	merged.Policy = base.Policy
	merged.Policy.RequiredTestLanes = normalizeStringList(append(base.Policy.RequiredTestLanes, overlay.Policy.RequiredTestLanes...))
	if overlay.Policy.RequireReleaseBudgets != nil {
		merged.Policy.RequireReleaseBudgets = overlay.Policy.RequireReleaseBudgets
	}
	if strings.TrimSpace(overlay.Policy.ReleaseBinaryPattern) != "" {
		merged.Policy.ReleaseBinaryPattern = overlay.Policy.ReleaseBinaryPattern
	}
	if strings.TrimSpace(overlay.Policy.ReleaseManifestPattern) != "" {
		merged.Policy.ReleaseManifestPattern = overlay.Policy.ReleaseManifestPattern
	}
	if strings.TrimSpace(overlay.Policy.RequiredReleaseCompression) != "" {
		merged.Policy.RequiredReleaseCompression = overlay.Policy.RequiredReleaseCompression
	}
	merged.Policy.ApprovedGoToolchains = normalizeStringList(append(base.Policy.ApprovedGoToolchains, overlay.Policy.ApprovedGoToolchains...))

	merged.Security = base.Security
	if overlay.Security.AllowUntrustedExtensions != nil {
		merged.Security.AllowUntrustedExtensions = overlay.Security.AllowUntrustedExtensions
	}
	merged.Security.AllowedExecutableRoots = normalizeStringList(append(base.Security.AllowedExecutableRoots, overlay.Security.AllowedExecutableRoots...))
	merged.Security.InheritedEnvAllowlist = normalizeEnvNameList(append(base.Security.InheritedEnvAllowlist, overlay.Security.InheritedEnvAllowlist...))
	merged.Security.InheritedEnvDenylist = normalizeEnvNameList(append(base.Security.InheritedEnvDenylist, overlay.Security.InheritedEnvDenylist...))

	return merged
}
