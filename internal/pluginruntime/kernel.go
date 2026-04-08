package pluginruntime

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/monstercameron/GoWebComponents/internal/diagnostics"
)

// PluginDiagnostic stores one kernel-attributed plugin event.
type PluginDiagnostic struct {
	PluginID  string
	Operation string
	Reason    HealthReason
	Message   string
	When      time.Time
	Report    diagnostics.Report
}

type pluginState struct {
	getManifest      Manifest
	getHandle        Handle
	getCleanups      []CleanupFunc
	getContributions []registeredContribution
}

// Kernel owns framework-managed plugin lifecycle, services, and contributions.
type Kernel struct {
	getMu         sync.RWMutex
	getInfo       KernelInfo
	getServices   map[ServiceKey]any
	getPlugins    map[string]*pluginState
	getStartOrder []string
	getHealth     *healthStateStore
	getEvents     []PluginDiagnostic
	getClosed     bool
}

// NewKernel constructs and starts one kernel from bootstrap options.
func NewKernel(parseOptions BootstrapOptions) (*Kernel, error) {
	buildKernel := &Kernel{
		getInfo:     KernelInfo{APIVersion: "v1alpha1"},
		getServices: map[ServiceKey]any{},
		getPlugins:  map[string]*pluginState{},
		getHealth:   newHealthStateStore(),
	}
	for _, parseService := range cloneServiceRegistrations(parseOptions.Services) {
		if parseService.Key == "" || parseService.Value == nil {
			continue
		}
		buildKernel.getServices[parseService.Key] = parseService.Value
	}
	buildRegistrations, parseErr := mergeBootstrapRegistrations(parseOptions.Registrations)
	if parseErr != nil {
		return nil, parseErr
	}
	buildDisabled := normalizeDisabledPluginIDs(parseOptions.DisabledPluginIDs)
	buildDisabledSet := make(map[string]struct{}, len(buildDisabled))
	for _, parseID := range buildDisabled {
		buildDisabledSet[parseID] = struct{}{}
	}
	for _, parseRegistration := range buildRegistrations {
		if _, hasDisabled := buildDisabledSet[parseRegistration.getManifest.ID]; hasDisabled {
			continue
		}
		if parseErr2 := buildKernel.startPlugin(parseRegistration); parseErr2 != nil {
			return nil, parseErr2
		}
	}
	return buildKernel, nil
}

// Info reports stable kernel identity metadata.
func (parseKernel *Kernel) Info() KernelInfo {
	if parseKernel == nil {
		return KernelInfo{}
	}
	return parseKernel.getInfo
}

// ResolveService returns one registered service value.
func (parseKernel *Kernel) ResolveService(parseKey ServiceKey) (any, bool) {
	if parseKernel == nil {
		return nil, false
	}
	parseKernel.getMu.RLock()
	defer parseKernel.getMu.RUnlock()
	if parseKernel.getClosed {
		return nil, false
	}
	getService, hasService := parseKernel.getServices[parseKey]
	return getService, hasService
}

// SetService stores or replaces one service on the kernel.
func (parseKernel *Kernel) SetService(parseKey ServiceKey, parseValue any) {
	if parseKernel == nil || parseKey == "" || parseValue == nil {
		return
	}
	parseKernel.getMu.Lock()
	defer parseKernel.getMu.Unlock()
	if parseKernel.getClosed {
		return
	}
	parseKernel.getServices[parseKey] = parseValue
}

// HealthReports returns stable health snapshots for every tracked plugin.
func (parseKernel *Kernel) HealthReports() []HealthReport {
	if parseKernel == nil {
		return nil
	}
	return parseKernel.getHealth.listHealthReports()
}

// Diagnostics returns one cloned plugin diagnostic event slice.
func (parseKernel *Kernel) Diagnostics() []PluginDiagnostic {
	if parseKernel == nil {
		return nil
	}
	parseKernel.getMu.RLock()
	defer parseKernel.getMu.RUnlock()
	return append([]PluginDiagnostic(nil), parseKernel.getEvents...)
}

// Close stops all started plugins in reverse start order.
func (parseKernel *Kernel) Close() error {
	if parseKernel == nil {
		return nil
	}
	parseKernel.getMu.Lock()
	if parseKernel.getClosed {
		parseKernel.getMu.Unlock()
		return nil
	}
	buildOrder := append([]string(nil), parseKernel.getStartOrder...)
	parseKernel.getMu.Unlock()
	var buildJoined error
	for parseIndex := len(buildOrder) - 1; parseIndex >= 0; parseIndex-- {
		if parseErr := parseKernel.stopPlugin(buildOrder[parseIndex]); parseErr != nil {
			buildJoined = errors.Join(buildJoined, parseErr)
		}
	}
	parseKernel.getMu.Lock()
	parseKernel.getClosed = true
	parseKernel.getStartOrder = nil
	parseKernel.getPlugins = map[string]*pluginState{}
	parseKernel.getMu.Unlock()
	return buildJoined
}

// ListDevtoolsSections resolves all active devtools section contributions.
func (parseKernel *Kernel) ListDevtoolsSections() []DevtoolsSection {
	if parseKernel == nil {
		return nil
	}
	parseKernel.getMu.RLock()
	if parseKernel.getClosed {
		parseKernel.getMu.RUnlock()
		return nil
	}
	buildContributions := parseKernel.listContributionsLocked(ContributionKindDevtoolsSection)
	parseKernel.getMu.RUnlock()
	buildSections := make([]DevtoolsSection, 0)
	for _, parseContribution := range buildContributions {
		if !parseKernel.shouldActivate(parseContribution.getMetadata.ActivationPolicy) {
			continue
		}
		getProvider, hasProvider := parseContribution.getValue.(DevtoolsSectionProvider)
		if !hasProvider || getProvider == nil {
			continue
		}
		getSections, parseErr := parseKernel.runDevtoolsSectionProvider(parseContribution.getPluginID, parseContribution.getMetadata.ID, getProvider)
		if parseErr != nil {
			continue
		}
		buildSections = append(buildSections, getSections...)
	}
	return cloneDevtoolsSections(buildSections)
}

// ListDevtoolsActions resolves all active devtools action contributions.
func (parseKernel *Kernel) ListDevtoolsActions() []DevtoolsAction {
	if parseKernel == nil {
		return nil
	}
	parseKernel.getMu.RLock()
	if parseKernel.getClosed {
		parseKernel.getMu.RUnlock()
		return nil
	}
	buildContributions := parseKernel.listContributionsLocked(ContributionKindDevtoolsAction)
	parseKernel.getMu.RUnlock()
	buildActions := make([]DevtoolsAction, 0)
	for _, parseContribution := range buildContributions {
		if !parseKernel.shouldActivate(parseContribution.getMetadata.ActivationPolicy) {
			continue
		}
		getProvider, hasProvider := parseContribution.getValue.(DevtoolsActionProvider)
		if !hasProvider || getProvider == nil {
			continue
		}
		getActions, parseErr := parseKernel.runDevtoolsActionProvider(parseContribution.getPluginID, parseContribution.getMetadata.ID, getProvider)
		if parseErr != nil {
			continue
		}
		buildActions = append(buildActions, getActions...)
	}
	return cloneDevtoolsActions(buildActions)
}

// registerContribution stores one plugin-owned contribution.
func (parseKernel *Kernel) registerContribution(parsePluginID string, parseRegistration ContributionRegistration) error {
	if parseErr := validateContributionRegistration(parseRegistration); parseErr != nil {
		return parseErr
	}
	parseKernel.getMu.Lock()
	defer parseKernel.getMu.Unlock()
	getState := parseKernel.getPlugins[parsePluginID]
	if getState == nil {
		getState = &pluginState{getManifest: Manifest{ID: parsePluginID}}
		parseKernel.getPlugins[parsePluginID] = getState
	}
	getState.getContributions = append(getState.getContributions, registeredContribution{
		getPluginID: parsePluginID,
		getMetadata: parseRegistration.Metadata,
		getValue:    parseRegistration.Value,
	})
	sortRegisteredContributions(getState.getContributions)
	return nil
}

// registerCleanup stores one plugin-owned cleanup callback.
func (parseKernel *Kernel) registerCleanup(parsePluginID string, parseCleanup CleanupFunc) {
	parseKernel.getMu.Lock()
	defer parseKernel.getMu.Unlock()
	getState := parseKernel.getPlugins[parsePluginID]
	if getState == nil {
		getState = &pluginState{getManifest: Manifest{ID: parsePluginID}}
		parseKernel.getPlugins[parsePluginID] = getState
	}
	getState.getCleanups = append(getState.getCleanups, parseCleanup)
}

// startPlugin starts one plugin registration and stores its lifecycle state.
func (parseKernel *Kernel) startPlugin(parseRegistration mergedRegistration) error {
	if parseKernel == nil {
		return fmt.Errorf("pluginruntime: kernel is required")
	}
	parseKernel.getMu.RLock()
	if parseKernel.getClosed {
		parseKernel.getMu.RUnlock()
		return fmt.Errorf("pluginruntime: kernel is closed")
	}
	_, hasPlugin := parseKernel.getPlugins[parseRegistration.getManifest.ID]
	parseKernel.getMu.RUnlock()
	if hasPlugin {
		return fmt.Errorf("pluginruntime: plugin %q is already started", parseRegistration.getManifest.ID)
	}
	getMissing := parseKernel.getMissingServices(parseRegistration.getManifest.RequiredServices)
	if len(getMissing) > 0 {
		buildErr := fmt.Errorf("pluginruntime: plugin %q requires missing services: %s", parseRegistration.getManifest.ID, strings.Join(getMissing, ", "))
		parseKernel.storeHealth(parseRegistration.getManifest.ID, HealthStateQuarantined, HealthReasonCompatibilityFailure, buildErr.Error(), 1)
		parseKernel.appendDiagnostic(parseRegistration.getManifest.ID, "start", HealthReasonCompatibilityFailure, buildErr)
		return buildErr
	}
	parseKernel.storeHealth(parseRegistration.getManifest.ID, HealthStateStarting, "", "", 0)
	parseKernel.getMu.Lock()
	parseKernel.getPlugins[parseRegistration.getManifest.ID] = &pluginState{
		getManifest: parseRegistration.getManifest,
	}
	parseKernel.getMu.Unlock()
	buildPlugin := parseRegistration.getFactory()
	if buildPlugin == nil {
		buildErr := fmt.Errorf("pluginruntime: plugin factory returned nil plugin for %q", parseRegistration.getManifest.ID)
		parseKernel.storeHealth(parseRegistration.getManifest.ID, HealthStateQuarantined, HealthReasonStartupError, buildErr.Error(), 1)
		parseKernel.appendDiagnostic(parseRegistration.getManifest.ID, "start", HealthReasonStartupError, buildErr)
		return buildErr
	}
	buildContext := pluginContext{getKernel: parseKernel, getPluginID: parseRegistration.getManifest.ID}
	getResult := runGuardedCall("plugin start", func() error {
		getHandle, parseErr := buildPlugin.StartPlugin(buildContext)
		if parseErr != nil {
			return parseErr
		}
		parseKernel.getMu.Lock()
		if getState := parseKernel.getPlugins[parseRegistration.getManifest.ID]; getState != nil {
			getState.getHandle = getHandle
		} else {
			parseKernel.getPlugins[parseRegistration.getManifest.ID] = &pluginState{
				getManifest: parseRegistration.getManifest,
				getHandle:   getHandle,
			}
		}
		parseKernel.getStartOrder = append(parseKernel.getStartOrder, parseRegistration.getManifest.ID)
		parseKernel.getMu.Unlock()
		return nil
	})
	if getResult.getErr != nil {
		parseKernel.getMu.Lock()
		delete(parseKernel.getPlugins, parseRegistration.getManifest.ID)
		parseKernel.getMu.Unlock()
		buildReason := HealthReasonStartupError
		if getResult.getRecovered != nil {
			buildReason = HealthReasonPanic
		}
		parseKernel.storeHealth(parseRegistration.getManifest.ID, HealthStateQuarantined, buildReason, getResult.getErr.Error(), 1)
		parseKernel.appendDiagnostic(parseRegistration.getManifest.ID, "start", buildReason, getResult.getErr)
		return getResult.getErr
	}
	parseKernel.storeHealth(parseRegistration.getManifest.ID, HealthStateHealthy, "", "", 0)
	return nil
}

// stopPlugin stops one started plugin and runs its cleanups.
func (parseKernel *Kernel) stopPlugin(parsePluginID string) error {
	parseKernel.getMu.RLock()
	getState := parseKernel.getPlugins[parsePluginID]
	parseKernel.getMu.RUnlock()
	if getState == nil {
		return nil
	}
	var buildJoined error
	if getState.getHandle != nil {
		getResult := runGuardedCall("plugin stop", getState.getHandle.StopPlugin)
		if getResult.getErr != nil {
			buildJoined = errors.Join(buildJoined, getResult.getErr)
			parseKernel.appendDiagnostic(parsePluginID, "stop", HealthReasonStopped, getResult.getErr)
		}
	}
	for parseIndex := len(getState.getCleanups) - 1; parseIndex >= 0; parseIndex-- {
		if parseCleanup := getState.getCleanups[parseIndex]; parseCleanup != nil {
			getResult := runGuardedCall("plugin cleanup", parseCleanup)
			if getResult.getErr != nil {
				buildJoined = errors.Join(buildJoined, getResult.getErr)
				parseKernel.appendDiagnostic(parsePluginID, "cleanup", HealthReasonStopped, getResult.getErr)
			}
		}
	}
	parseKernel.storeHealth(parsePluginID, HealthStateStopped, HealthReasonStopped, "", 0)
	parseKernel.getMu.Lock()
	delete(parseKernel.getPlugins, parsePluginID)
	for parseIndex, getStartedPluginID := range parseKernel.getStartOrder {
		if getStartedPluginID != parsePluginID {
			continue
		}
		copy(parseKernel.getStartOrder[parseIndex:], parseKernel.getStartOrder[parseIndex+1:])
		parseKernel.getStartOrder[len(parseKernel.getStartOrder)-1] = ""
		parseKernel.getStartOrder = parseKernel.getStartOrder[:len(parseKernel.getStartOrder)-1]
		break
	}
	parseKernel.getMu.Unlock()
	return buildJoined
}

// isClosed reports whether the kernel has been closed and must be rebooted before reuse.
func (parseKernel *Kernel) isClosed() bool {
	if parseKernel == nil {
		return true
	}
	parseKernel.getMu.RLock()
	defer parseKernel.getMu.RUnlock()
	return parseKernel.getClosed
}

// listContributionsLocked collects contributions of one kind under the kernel lock.
func (parseKernel *Kernel) listContributionsLocked(parseKind ContributionKind) []registeredContribution {
	buildContributions := make([]registeredContribution, 0)
	for _, getState := range parseKernel.getPlugins {
		if getState == nil {
			continue
		}
		for _, getContribution := range getState.getContributions {
			if getContribution.getMetadata.Kind != parseKind {
				continue
			}
			if getHealth, hasHealth := parseKernel.getHealth.getHealthReport(getContribution.getPluginID); hasHealth && getHealth.State == HealthStateQuarantined {
				continue
			}
			buildContributions = append(buildContributions, getContribution)
		}
	}
	sortRegisteredContributions(buildContributions)
	return buildContributions
}

// runDevtoolsSectionProvider resolves one section provider under guarded execution.
func (parseKernel *Kernel) runDevtoolsSectionProvider(parsePluginID string, parseContributionID string, parseProvider DevtoolsSectionProvider) ([]DevtoolsSection, error) {
	var buildSections []DevtoolsSection
	getResult := runGuardedCall("devtools section provider", func() error {
		getSections, parseErr := parseProvider.GetDevtoolsSections()
		if parseErr != nil {
			return parseErr
		}
		buildSections = cloneDevtoolsSections(getSections)
		return nil
	})
	if getResult.getErr != nil {
		parseKernel.handlePluginFailure(parsePluginID, parseContributionID, "devtools-section", getResult)
		return nil, getResult.getErr
	}
	return buildSections, nil
}

// runDevtoolsActionProvider resolves one action provider under guarded execution.
func (parseKernel *Kernel) runDevtoolsActionProvider(parsePluginID string, parseContributionID string, parseProvider DevtoolsActionProvider) ([]DevtoolsAction, error) {
	var buildActions []DevtoolsAction
	getResult := runGuardedCall("devtools action provider", func() error {
		getActions, parseErr := parseProvider.GetDevtoolsActions()
		if parseErr != nil {
			return parseErr
		}
		buildActions = cloneDevtoolsActions(getActions)
		return nil
	})
	if getResult.getErr != nil {
		parseKernel.handlePluginFailure(parsePluginID, parseContributionID, "devtools-action", getResult)
		return nil, getResult.getErr
	}
	for parseIndex := range buildActions {
		getAction := buildActions[parseIndex]
		buildActions[parseIndex].Run = func(parseContext DevtoolsActionContext) error {
			getActionResult := runGuardedCall("devtools action run", func() error {
				if getAction.Run == nil {
					return nil
				}
				return getAction.Run(parseContext)
			})
			if getActionResult.getErr != nil {
				parseKernel.handlePluginFailure(parsePluginID, parseContributionID, "devtools-action-run", getActionResult)
			}
			return getActionResult.getErr
		}
	}
	return buildActions, nil
}

// handlePluginFailure applies health transitions and diagnostics for one plugin failure.
func (parseKernel *Kernel) handlePluginFailure(parsePluginID string, parseContributionID string, parseOperation string, parseResult guardedResult) {
	getHealth, hasHealth := parseKernel.getHealth.getHealthReport(parsePluginID)
	buildCount := 1
	if hasHealth {
		buildCount = getHealth.ErrorCount + 1
	}
	buildState := HealthStateDegraded
	buildReason := HealthReasonRepeatedErrors
	if parseResult.getRecovered != nil {
		buildState = HealthStateQuarantined
		buildReason = HealthReasonPanic
	} else if buildCount > 1 {
		buildState = HealthStateQuarantined
	}
	parseKernel.storeHealth(parsePluginID, buildState, buildReason, fmt.Sprintf("%s failed: %v", parseOperation, parseResult.getErr), buildCount)
	parseKernel.appendDiagnostic(parsePluginID, parseOperation+":"+parseContributionID, buildReason, parseResult.getErr)
}

// shouldActivate reports whether one activation policy is active for the current kernel pass.
func (parseKernel *Kernel) shouldActivate(parsePolicy ActivationPolicy) bool {
	switch parsePolicy {
	case "", ActivationPolicyBoot, ActivationPolicySession, ActivationPolicyView, ActivationPolicyOpportunistic:
		return true
	default:
		return false
	}
}

// getMissingServices returns sorted required services not currently registered.
func (parseKernel *Kernel) getMissingServices(parseRequired []ServiceKey) []string {
	if len(parseRequired) == 0 {
		return nil
	}
	parseKernel.getMu.RLock()
	defer parseKernel.getMu.RUnlock()
	buildMissing := make([]string, 0)
	for _, parseKey := range parseRequired {
		if _, hasService := parseKernel.getServices[parseKey]; !hasService {
			buildMissing = append(buildMissing, string(parseKey))
		}
	}
	sort.Strings(buildMissing)
	return buildMissing
}

// appendDiagnostic stores one plugin diagnostic event.
func (parseKernel *Kernel) appendDiagnostic(parsePluginID string, parseOperation string, parseReason HealthReason, parseErr error) {
	if parseKernel == nil || parseErr == nil {
		return
	}
	buildReport := diagnostics.Build(diagnostics.Options{
		Summary:  parseErr.Error(),
		Code:     "GWC-PLUGIN-KERNEL",
		Headline: "plugin kernel failure",
		Path:     parsePluginID,
		Runtime:  parseKernel.getInfo.APIVersion,
		Next:     "Inspect the plugin health report and quarantined diagnostics.",
		Docs:     "docs/REFERENCE_MANUAL/15-design-notes-and-boundaries.md#internal-plugin-kernel",
	})
	parseKernel.getMu.Lock()
	defer parseKernel.getMu.Unlock()
	parseKernel.getEvents = append(parseKernel.getEvents, PluginDiagnostic{
		PluginID:  parsePluginID,
		Operation: parseOperation,
		Reason:    parseReason,
		Message:   parseErr.Error(),
		When:      time.Now().UTC(),
		Report:    buildReport,
	})
}

// storeHealth writes one health report snapshot.
func (parseKernel *Kernel) storeHealth(parsePluginID string, parseState HealthState, parseReason HealthReason, parseMessage string, parseErrorCount int) {
	parseKernel.getHealth.setHealthReport(HealthReport{
		PluginID:   parsePluginID,
		State:      parseState,
		Reason:     parseReason,
		Message:    parseMessage,
		ErrorCount: parseErrorCount,
		UpdatedAt:  time.Now().UTC(),
	})
}
