package devtools

import (
	"fmt"
	"sort"
	"strings"

	"github.com/monstercameron/GoWebComponents/v5/internal/pluginruntime"
)

func init() {
	_ = pluginruntime.RegisterBuiltinPlugin(pluginruntime.PluginRegistration{
		Factory: func() pluginruntime.Plugin {
			return builtInDevtoolsPlugin{}
		},
	})
}

type builtInDevtoolsPlugin struct{}

// Manifest returns the built-in devtools plugin manifest.
func (builtInDevtoolsPlugin) Manifest() pluginruntime.Manifest {
	return pluginruntime.Manifest{
		ID:          "devtools.kernel",
		Version:     "1.0.0",
		Description: "kernel-backed devtools sections",
		// Declared so the runtime2-summary section may resolve it; Optional because
		// the section degrades gracefully when runtime2 metadata is absent.
		OptionalServices: []pluginruntime.ServiceKey{pluginruntime.ServiceKeyRuntime2Meta},
		ActivationPolicy: pluginruntime.ActivationPolicyBoot,
	}
}

// StartPlugin registers built-in kernel devtools contributions.
func (builtInDevtoolsPlugin) StartPlugin(parseContext pluginruntime.Context) (pluginruntime.Handle, error) {
	parseErr := parseContext.RegisterContribution(pluginruntime.ContributionRegistration{
		Metadata: pluginruntime.ContributionMetadata{
			ID:               "kernel-health",
			Kind:             pluginruntime.ContributionKindDevtoolsSection,
			ExecutionClass:   pluginruntime.ExecutionClassWarm,
			ActivationPolicy: pluginruntime.ActivationPolicyBoot,
			Order:            100,
		},
		Value: pluginruntime.DevtoolsSectionProviderFunc(func() ([]pluginruntime.DevtoolsSection, error) {
			getKernel := pluginruntime.GetGlobalKernel()
			if getKernel == nil {
				return nil, nil
			}
			getReports := getKernel.HealthReports()
			getEvents := getKernel.Diagnostics()
			buildLines := []string{
				fmt.Sprintf("api version: %s", getKernel.Info().APIVersion),
				fmt.Sprintf("plugins: %d", len(getReports)),
				fmt.Sprintf("events: %d", len(getEvents)),
			}
			buildSummary := map[string]string{
				"api":     getKernel.Info().APIVersion,
				"plugins": fmt.Sprintf("%d", len(getReports)),
				"events":  fmt.Sprintf("%d", len(getEvents)),
			}
			for _, getReport := range getReports {
				buildLines = append(buildLines, fmt.Sprintf("%s => %s", getReport.PluginID, getReport.State))
			}
			return []pluginruntime.DevtoolsSection{{
				Name:    "Plugin Kernel",
				Summary: buildSummary,
				Lines:   buildLines,
			}}, nil
		}),
	})
	if parseErr != nil {
		return nil, parseErr
	}
	parseErr = parseContext.RegisterContribution(pluginruntime.ContributionRegistration{
		Metadata: pluginruntime.ContributionMetadata{
			ID:               "runtime2-summary",
			Kind:             pluginruntime.ContributionKindDevtoolsSection,
			ExecutionClass:   pluginruntime.ExecutionClassWarm,
			ActivationPolicy: pluginruntime.ActivationPolicyBoot,
			Order:            110,
		},
		Value: pluginruntime.DevtoolsSectionProviderFunc(func() ([]pluginruntime.DevtoolsSection, error) {
			getRuntime2MetaService, hasRuntime2MetaService := parseContext.ResolveService(pluginruntime.ServiceKeyRuntime2Meta)
			if !hasRuntime2MetaService {
				return nil, nil
			}
			getRuntime2Meta, hasTypedRuntime2Meta := getRuntime2MetaService.(pluginruntime.Runtime2MetaService)
			if !hasTypedRuntime2Meta {
				return nil, nil
			}
			getRuntime2Snapshot, parseRuntime2Err := getRuntime2Meta.GetRuntime2MetaSnapshot(pluginruntime.QueryBudget{MaxItems: 8})
			if parseRuntime2Err != nil {
				return []pluginruntime.DevtoolsSection{{
					Name:    "Runtime2",
					Summary: map[string]string{"error": parseRuntime2Err.Error()},
					Lines:   []string{"runtime2 metadata lookup failed"},
				}}, nil
			}
			if len(getRuntime2Snapshot.Capabilities) == 0 && len(getRuntime2Snapshot.Regions) == 0 && len(getRuntime2Snapshot.Diagnostics) == 0 {
				return nil, nil
			}
			buildEnabledCapabilities := make([]string, 0, len(getRuntime2Snapshot.Capabilities))
			for parseCapability, isEnabled := range getRuntime2Snapshot.Capabilities {
				if !isEnabled {
					continue
				}
				buildEnabledCapabilities = append(buildEnabledCapabilities, parseCapability)
			}
			sort.Strings(buildEnabledCapabilities)
			buildLines := []string{
				fmt.Sprintf("regions: %d", len(getRuntime2Snapshot.Regions)),
				fmt.Sprintf("diagnostics: %d", len(getRuntime2Snapshot.Diagnostics)),
			}
			if len(buildEnabledCapabilities) > 0 {
				buildLines = append(buildLines, "capabilities: "+strings.Join(buildEnabledCapabilities, ", "))
			}
			for _, getRegion := range getRuntime2Snapshot.Regions {
				buildLines = append(buildLines, fmt.Sprintf(
					"%s => %s, transport=%s, diagnostics=%d",
					getRegion.RegionInstanceID,
					getRegion.RegionMode,
					getRegion.TransportTier,
					getRegion.DiagnosticCount,
				))
			}
			return []pluginruntime.DevtoolsSection{{
				Name: "Runtime2",
				Summary: map[string]string{
					"regions":     fmt.Sprintf("%d", len(getRuntime2Snapshot.Regions)),
					"diagnostics": fmt.Sprintf("%d", len(getRuntime2Snapshot.Diagnostics)),
					"backend":     getRuntime2Snapshot.Meta.BackendID,
				},
				Lines: buildLines,
			}}, nil
		}),
	})
	if parseErr != nil {
		return nil, parseErr
	}
	return nil, nil
}

// inspectKernelExtensionSections resolves live kernel-owned devtools sections.
func inspectKernelExtensionSections() []ExtensionSection {
	getKernel := pluginruntime.GetGlobalKernel()
	if getKernel == nil {
		return nil
	}
	getSections := getKernel.ListDevtoolsSections()
	buildSections := make([]ExtensionSection, 0, len(getSections))
	for _, getSection := range getSections {
		if strings.TrimSpace(getSection.Name) == "" {
			continue
		}
		buildSections = append(buildSections, ExtensionSection{
			Name:    getSection.Name,
			Summary: cloneHostStringMap(getSection.Summary),
			Lines:   append([]string(nil), getSection.Lines...),
		})
	}
	return buildSections
}

// inspectKernelOverlayActions resolves live kernel-owned overlay actions.
func inspectKernelOverlayActions() []ErrorOverlayAction {
	getKernel := pluginruntime.GetGlobalKernel()
	if getKernel == nil {
		return nil
	}
	getActions := getKernel.ListDevtoolsActions()
	buildActions := make([]ErrorOverlayAction, 0, len(getActions))
	for _, getAction := range getActions {
		getCurrent := getAction
		if strings.TrimSpace(getCurrent.Label) == "" {
			continue
		}
		buildActions = append(buildActions, ErrorOverlayAction{
			Label:        getCurrent.Label,
			MatchCodes:   append([]string(nil), getCurrent.MatchCodes...),
			MatchSources: append([]string(nil), getCurrent.MatchSources...),
			Run: func(parseContext ErrorOverlayActionContext) {
				if getCurrent.Run != nil {
					_ = getCurrent.Run(pluginruntime.DevtoolsActionContext{
						Issue: pluginruntime.DevtoolsIssue{
							Source:   parseContext.Issue.Source,
							Code:     parseContext.Issue.Code,
							Message:  parseContext.Issue.Message,
							Path:     parseContext.Issue.Path,
							Docs:     parseContext.Issue.Docs,
							TopFrame: parseContext.Issue.TopFrame,
						},
					})
				}
			},
		})
	}
	return buildActions
}

// snapshotKernelState returns the current kernel state summary for devtools snapshots.
func snapshotKernelState() KernelSnapshot {
	getKernel := pluginruntime.GetGlobalKernel()
	if getKernel == nil {
		return KernelSnapshot{}
	}
	buildSnapshot := KernelSnapshot{
		APIVersion: getKernel.Info().APIVersion,
	}
	for _, getReport := range getKernel.HealthReports() {
		buildSnapshot.Plugins = append(buildSnapshot.Plugins, KernelPluginHealth{
			ID:         getReport.PluginID,
			State:      string(getReport.State),
			Reason:     string(getReport.Reason),
			Message:    getReport.Message,
			ErrorCount: getReport.ErrorCount,
		})
	}
	for _, getEvent := range getKernel.Diagnostics() {
		buildSnapshot.Events = append(buildSnapshot.Events, KernelPluginEvent{
			PluginID:  getEvent.PluginID,
			Operation: getEvent.Operation,
			Reason:    string(getEvent.Reason),
			Message:   getEvent.Message,
			When:      getEvent.When,
		})
	}
	return buildSnapshot
}
