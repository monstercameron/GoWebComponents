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

func resolveLauncherEnterpriseConfig(parseCwd string, parseCli launcherGlobalCLIOptions) (launcherEnterpriseLayeredConfig, error) {
	parseLayered := launcherEnterpriseLayeredConfig{
		Effective: defaultLauncherEnterpriseConfig(),
		Sources: launcherEnterpriseConfigSources{
			FrameworkDefaults: true,
		},
	}

	parseOrgPath, hasOrgPath, parseFromCLI, parseErr := resolveOrganizationPolicyPackPath(parseCwd, parseCli.PolicyPackPath)
	if parseErr != nil {
		return launcherEnterpriseLayeredConfig{}, parseErr
	}
	if hasOrgPath {
		parseOrgConfig, parseLoadErr := loadEnterpriseConfigFromPath(parseOrgPath)
		if parseLoadErr != nil {
			return launcherEnterpriseLayeredConfig{}, fmt.Errorf("load organization policy pack: %w", parseLoadErr)
		}
		parseLayered.Effective = mergeLauncherEnterpriseConfig(parseLayered.Effective, parseOrgConfig)
		parseLayered.Sources.OrganizationPolicyPath = parseOrgPath
		if parseFromCLI {
			parseLayered.Sources.CLIOverridePolicyPack = parseOrgPath
		}
	}

	parseProjectConfigPath := findLauncherOverrideInParents(parseCwd)
	if strings.TrimSpace(parseProjectConfigPath) != "" {
		parseProjectConfig, parseLoadErr2 := loadEnterpriseConfigFromPath(parseProjectConfigPath)
		if parseLoadErr2 != nil {
			return launcherEnterpriseLayeredConfig{}, fmt.Errorf("load project enterprise config: %w", parseLoadErr2)
		}
		parseLayered.Effective = mergeLauncherEnterpriseConfig(parseLayered.Effective, parseProjectConfig)
		parseLayered.Sources.ProjectConfigPath = parseProjectConfigPath
	}

	if parseCli.DisableHooks {
		parseLayered.Effective.Hooks = map[string][]launcherExecutableHook{}
	}
	if parseCli.DisablePlugins {
		parseLayered.Effective.Plugins = []launcherExecutablePlugin{}
	}

	return parseLayered, nil
}

func resolveOrganizationPolicyPackPath(parseCwd string, parseCliPolicyPath string) (string, bool, bool, error) {
	if strings.TrimSpace(parseCliPolicyPath) != "" {
		parseResolved, parseErr := resolveEnterpriseConfigPath(parseCwd, parseCliPolicyPath)
		if parseErr != nil {
			return "", false, false, fmt.Errorf("resolve CLI policy pack path: %w", parseErr)
		}
		if !launcherEnterprisePathExists(parseResolved) {
			return "", false, false, fmt.Errorf("CLI policy pack path does not exist: %s", parseResolved)
		}
		return parseResolved, true, true, nil
	}

	if parseEnvPath := strings.TrimSpace(launcherEnterpriseGetenv(launcherPolicyPackEnvVar)); parseEnvPath != "" {
		parseResolved2, parseErr2 := resolveEnterpriseConfigPath(parseCwd, parseEnvPath)
		if parseErr2 != nil {
			return "", false, false, fmt.Errorf("resolve %s path: %w", launcherPolicyPackEnvVar, parseErr2)
		}
		if !launcherEnterprisePathExists(parseResolved2) {
			return "", false, false, fmt.Errorf("%s path does not exist: %s", launcherPolicyPackEnvVar, parseResolved2)
		}
		return parseResolved2, true, false, nil
	}

	parseHomeDir, parseErr3 := launcherConfigUserHomeDir()
	if parseErr3 != nil {
		return "", false, false, nil
	}
	parseDefaultPath := filepath.Join(parseHomeDir, ".gwc", "policy-pack.json")
	if !launcherEnterprisePathExists(parseDefaultPath) {
		return "", false, false, nil
	}
	return parseDefaultPath, true, false, nil
}

func resolveEnterpriseConfigPath(parseBaseDir string, parseRawPath string) (string, error) {
	parseRawPath = strings.TrimSpace(parseRawPath)
	if parseRawPath == "" {
		return "", nil
	}
	if filepath.IsAbs(parseRawPath) {
		return filepath.Clean(parseRawPath), nil
	}
	if strings.TrimSpace(parseBaseDir) == "" {
		parseBaseDir = "."
	}
	return filepath.Abs(filepath.Join(parseBaseDir, parseRawPath))
}

func launcherEnterprisePathExists(parsePath string) bool {
	if strings.TrimSpace(parsePath) == "" {
		return false
	}
	_, parseErr := launcherEnterpriseStat(parsePath)
	return parseErr == nil
}

func loadEnterpriseConfigFromPath(parsePath string) (launcherEnterpriseConfig, error) {
	parseContent, parseErr := launcherEnterpriseReadFile(parsePath)
	if parseErr != nil {
		return launcherEnterpriseConfig{}, fmt.Errorf("read config %s: %w", parsePath, parseErr)
	}

	var parseWrapped launcherEnterpriseConfigFile
	if parseErr2 := json.Unmarshal(parseContent, &parseWrapped); parseErr2 != nil {
		return launcherEnterpriseConfig{}, fmt.Errorf("parse config %s: %w", parsePath, parseErr2)
	}
	if hasLauncherEnterprisePayload(parseWrapped.Enterprise) {
		parseNormalized, parseNormalizeErr := normalizeLauncherEnterpriseConfig(parsePath, parseWrapped.Enterprise)
		if parseNormalizeErr != nil {
			return launcherEnterpriseConfig{}, parseNormalizeErr
		}
		return parseNormalized, nil
	}

	var parseDirect launcherEnterpriseConfig
	if parseErr3 := json.Unmarshal(parseContent, &parseDirect); parseErr3 != nil {
		return launcherEnterpriseConfig{}, fmt.Errorf("parse config %s: %w", parsePath, parseErr3)
	}
	parseNormalized2, parseNormalizeErr2 := normalizeLauncherEnterpriseConfig(parsePath, parseDirect)
	if parseNormalizeErr2 != nil {
		return launcherEnterpriseConfig{}, parseNormalizeErr2
	}
	return parseNormalized2, nil
}

func hasLauncherEnterprisePayload(parseConfig launcherEnterpriseConfig) bool {
	return len(parseConfig.Hooks) > 0 ||
		len(parseConfig.Plugins) > 0 ||
		hasLauncherEnterprisePolicyPayload(parseConfig.Policy) ||
		hasLauncherEnterpriseSecurityPayload(parseConfig.Security)
}

func hasLauncherEnterprisePolicyPayload(parsePolicy launcherEnterprisePolicy) bool {
	return len(parsePolicy.RequiredTestLanes) > 0 ||
		parsePolicy.RequireReleaseBudgets != nil ||
		strings.TrimSpace(parsePolicy.ReleaseBinaryPattern) != "" ||
		strings.TrimSpace(parsePolicy.ReleaseManifestPattern) != "" ||
		strings.TrimSpace(parsePolicy.RequiredReleaseCompression) != "" ||
		len(parsePolicy.ApprovedGoToolchains) > 0
}

func hasLauncherEnterpriseSecurityPayload(parseSecurity launcherEnterpriseSecurityPolicy) bool {
	return parseSecurity.AllowUntrustedExtensions != nil ||
		len(parseSecurity.AllowedExecutableRoots) > 0 ||
		len(parseSecurity.InheritedEnvAllowlist) > 0 ||
		len(parseSecurity.InheritedEnvDenylist) > 0
}

func normalizeLauncherEnterpriseConfig(parseConfigPath string, parseConfig launcherEnterpriseConfig) (launcherEnterpriseConfig, error) {
	parseNormalized := defaultLauncherEnterpriseConfig()
	parseNormalized.Policy = parseConfig.Policy

	for parseHookPoint, parseHooks := range parseConfig.Hooks {
		parsePoint := normalizeHookPoint(parseHookPoint)
		if parsePoint == "" {
			continue
		}
		for _, parseHook := range parseHooks {
			parsePath, parseErr := resolveLauncherOverrideValue(parseConfigPath, parseHook.Path)
			if parseErr != nil {
				return launcherEnterpriseConfig{}, fmt.Errorf("resolve hook path %q at %s: %w", parseHook.Path, parseConfigPath, parseErr)
			}
			parseHook.Path = parsePath
			parseHook.Name = strings.TrimSpace(parseHook.Name)
			if parseHook.Name == "" {
				parseHook.Name = filepath.Base(parsePath)
			}
			parseNormalized.Hooks[parsePoint] = append(parseNormalized.Hooks[parsePoint], parseHook)
		}
	}

	for _, parsePlugin := range parseConfig.Plugins {
		parsePath2, parseErr2 := resolveLauncherOverrideValue(parseConfigPath, parsePlugin.Path)
		if parseErr2 != nil {
			return launcherEnterpriseConfig{}, fmt.Errorf("resolve plugin path %q at %s: %w", parsePlugin.Path, parseConfigPath, parseErr2)
		}
		parsePlugin.Path = parsePath2
		parsePlugin.Name = strings.TrimSpace(parsePlugin.Name)
		if parsePlugin.Name == "" {
			parsePlugin.Name = filepath.Base(parsePath2)
		}
		parseNormalizedCapabilities, parseCapabilityErr := normalizePluginCapabilities(parsePlugin.Capabilities)
		if parseCapabilityErr != nil {
			return launcherEnterpriseConfig{}, fmt.Errorf("normalize plugin capabilities for %q: %w", parsePlugin.Name, parseCapabilityErr)
		}
		parsePlugin.Capabilities = parseNormalizedCapabilities
		parseNormalized.Plugins = append(parseNormalized.Plugins, parsePlugin)
	}

	parseNormalized.Policy.RequiredTestLanes = normalizeStringList(parseConfig.Policy.RequiredTestLanes)
	if len(parseNormalized.Policy.RequiredTestLanes) > 0 {
		parseLaneValues, parseErr3 := normalizeTestLanes(parseNormalized.Policy.RequiredTestLanes)
		if parseErr3 != nil {
			return launcherEnterpriseConfig{}, fmt.Errorf("normalize required test lanes at %s: %w", parseConfigPath, parseErr3)
		}
		parseNormalized.Policy.RequiredTestLanes = parseLaneValues
	}
	parseNormalized.Policy.ReleaseBinaryPattern = strings.TrimSpace(parseConfig.Policy.ReleaseBinaryPattern)
	if parseNormalized.Policy.ReleaseBinaryPattern != "" {
		if _, parseErr4 := regexp.Compile(parseNormalized.Policy.ReleaseBinaryPattern); parseErr4 != nil {
			return launcherEnterpriseConfig{}, fmt.Errorf("compile release binary pattern at %s: %w", parseConfigPath, parseErr4)
		}
	}
	parseNormalized.Policy.ReleaseManifestPattern = strings.TrimSpace(parseConfig.Policy.ReleaseManifestPattern)
	if parseNormalized.Policy.ReleaseManifestPattern != "" {
		if _, parseErr5 := regexp.Compile(parseNormalized.Policy.ReleaseManifestPattern); parseErr5 != nil {
			return launcherEnterpriseConfig{}, fmt.Errorf("compile release manifest pattern at %s: %w", parseConfigPath, parseErr5)
		}
	}
	parseNormalized.Policy.RequiredReleaseCompression = strings.TrimSpace(strings.ToLower(parseConfig.Policy.RequiredReleaseCompression))
	if parseNormalized.Policy.RequiredReleaseCompression != "" {
		parseCompression, parseErr6 := normalizeReleaseCompressionPolicy(parseNormalized.Policy.RequiredReleaseCompression)
		if parseErr6 != nil {
			return launcherEnterpriseConfig{}, fmt.Errorf("normalize required release compression at %s: %w", parseConfigPath, parseErr6)
		}
		parseNormalized.Policy.RequiredReleaseCompression = parseCompression
	}
	parseNormalized.Policy.ApprovedGoToolchains = normalizeStringList(parseConfig.Policy.ApprovedGoToolchains)
	for parseIndex := range parseNormalized.Policy.ApprovedGoToolchains {
		parseNormalized.Policy.ApprovedGoToolchains[parseIndex] = strings.ToLower(strings.TrimSpace(parseNormalized.Policy.ApprovedGoToolchains[parseIndex]))
	}

	parseNormalized.Security = launcherEnterpriseSecurityPolicy{
		AllowUntrustedExtensions: parseConfig.Security.AllowUntrustedExtensions,
		AllowedExecutableRoots:   []string{},
		InheritedEnvAllowlist:    normalizeEnvNameList(parseConfig.Security.InheritedEnvAllowlist),
		InheritedEnvDenylist:     normalizeEnvNameList(parseConfig.Security.InheritedEnvDenylist),
	}
	for _, parseRoot := range parseConfig.Security.AllowedExecutableRoots {
		parseResolvedRoot, parseErr7 := resolveLauncherOverrideValue(parseConfigPath, parseRoot)
		if parseErr7 != nil {
			return launcherEnterpriseConfig{}, fmt.Errorf("resolve security allowed executable root %q at %s: %w", parseRoot, parseConfigPath, parseErr7)
		}
		parseNormalized.Security.AllowedExecutableRoots = append(parseNormalized.Security.AllowedExecutableRoots, parseResolvedRoot)
	}
	parseNormalized.Security.AllowedExecutableRoots = normalizeStringList(parseNormalized.Security.AllowedExecutableRoots)
	return parseNormalized, nil
}

func normalizeHookPoint(parsePoint string) string {
	parsePoint = strings.TrimSpace(strings.ToLower(parsePoint))
	if parsePoint == "" {
		return ""
	}
	return parsePoint
}

func normalizePluginCapabilities(parseCapabilities []string) ([]string, error) {
	parseNormalized := normalizeStringList(parseCapabilities)
	for _, parseCapability := range parseNormalized {
		parseKey := strings.TrimSpace(strings.ToLower(parseCapability))
		if _, parseOk := launcherPluginCapabilityAllowlist[parseKey]; !parseOk {
			return nil, fmt.Errorf("unsupported capability %q", parseCapability)
		}
	}
	slices.Sort(parseNormalized)
	return parseNormalized, nil
}

func normalizeStringList(parseValues []string) []string {
	parseSeen := make(map[string]struct{}, len(parseValues))
	parseNormalized := make([]string, 0, len(parseValues))
	for _, parseRaw := range parseValues {
		parseValue := strings.TrimSpace(parseRaw)
		if parseValue == "" {
			continue
		}
		parseKey := strings.ToLower(parseValue)
		if _, parseExists := parseSeen[parseKey]; parseExists {
			continue
		}
		parseSeen[parseKey] = struct{}{}
		parseNormalized = append(parseNormalized, parseValue)
	}
	return parseNormalized
}

func normalizeEnvNameList(parseValues []string) []string {
	parseSeen := make(map[string]struct{}, len(parseValues))
	parseNormalized := make([]string, 0, len(parseValues))
	for _, parseRaw := range parseValues {
		parseValue := strings.ToUpper(strings.TrimSpace(parseRaw))
		if parseValue == "" {
			continue
		}
		if _, parseExists := parseSeen[parseValue]; parseExists {
			continue
		}
		parseSeen[parseValue] = struct{}{}
		parseNormalized = append(parseNormalized, parseValue)
	}
	return parseNormalized
}

func mergeLauncherEnterpriseConfig(parseBase launcherEnterpriseConfig, parseOverlay launcherEnterpriseConfig) launcherEnterpriseConfig {
	parseMerged := defaultLauncherEnterpriseConfig()
	for parseKey, parseHooks := range parseBase.Hooks {
		parseMerged.Hooks[parseKey] = append(parseMerged.Hooks[parseKey], parseHooks...)
	}
	for parseKey2, parseHooks2 := range parseOverlay.Hooks {
		parseMerged.Hooks[parseKey2] = append(parseMerged.Hooks[parseKey2], parseHooks2...)
	}

	parseMerged.Plugins = append(parseMerged.Plugins, parseBase.Plugins...)
	parsePluginIndex := make(map[string]int, len(parseMerged.Plugins))
	for parseIndex, parsePlugin := range parseMerged.Plugins {
		parsePluginIndex[strings.ToLower(strings.TrimSpace(parsePlugin.Name))] = parseIndex
	}
	for _, parsePlugin2 := range parseOverlay.Plugins {
		parseKey3 := strings.ToLower(strings.TrimSpace(parsePlugin2.Name))
		if parseExisting, parseOk := parsePluginIndex[parseKey3]; parseOk {
			parseMerged.Plugins[parseExisting] = parsePlugin2
			continue
		}
		parsePluginIndex[parseKey3] = len(parseMerged.Plugins)
		parseMerged.Plugins = append(parseMerged.Plugins, parsePlugin2)
	}

	parseMerged.Policy = parseBase.Policy
	parseMerged.Policy.RequiredTestLanes = normalizeStringList(append(parseBase.Policy.RequiredTestLanes, parseOverlay.Policy.RequiredTestLanes...))
	if parseOverlay.Policy.RequireReleaseBudgets != nil {
		parseMerged.Policy.RequireReleaseBudgets = parseOverlay.Policy.RequireReleaseBudgets
	}
	if strings.TrimSpace(parseOverlay.Policy.ReleaseBinaryPattern) != "" {
		parseMerged.Policy.ReleaseBinaryPattern = parseOverlay.Policy.ReleaseBinaryPattern
	}
	if strings.TrimSpace(parseOverlay.Policy.ReleaseManifestPattern) != "" {
		parseMerged.Policy.ReleaseManifestPattern = parseOverlay.Policy.ReleaseManifestPattern
	}
	if strings.TrimSpace(parseOverlay.Policy.RequiredReleaseCompression) != "" {
		parseMerged.Policy.RequiredReleaseCompression = parseOverlay.Policy.RequiredReleaseCompression
	}
	parseMerged.Policy.ApprovedGoToolchains = normalizeStringList(append(parseBase.Policy.ApprovedGoToolchains, parseOverlay.Policy.ApprovedGoToolchains...))

	parseMerged.Security = parseBase.Security
	if parseOverlay.Security.AllowUntrustedExtensions != nil {
		parseMerged.Security.AllowUntrustedExtensions = parseOverlay.Security.AllowUntrustedExtensions
	}
	parseMerged.Security.AllowedExecutableRoots = normalizeStringList(append(parseBase.Security.AllowedExecutableRoots, parseOverlay.Security.AllowedExecutableRoots...))
	parseMerged.Security.InheritedEnvAllowlist = normalizeEnvNameList(append(parseBase.Security.InheritedEnvAllowlist, parseOverlay.Security.InheritedEnvAllowlist...))
	parseMerged.Security.InheritedEnvDenylist = normalizeEnvNameList(append(parseBase.Security.InheritedEnvDenylist, parseOverlay.Security.InheritedEnvDenylist...))

	return parseMerged
}
