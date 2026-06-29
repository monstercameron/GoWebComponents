package pluginruntime

import (
	"fmt"
	"strings"
	"sync"
)

// BootstrapOptions stores kernel bootstrap registrations and service values.
type BootstrapOptions struct {
	Registrations     []PluginRegistration
	Services          []ServiceRegistration
	DisabledPluginIDs []string
}

type mergedRegistration struct {
	getManifest Manifest
	getFactory  PluginFactory
}

var storeBootstrapState struct {
	getMu           sync.RWMutex
	getBuiltin      []PluginRegistration
	getServices     []ServiceRegistration
	getGlobalKernel *Kernel
}

// RegisterBuiltinPlugin registers one built-in internal plugin factory.
func RegisterBuiltinPlugin(parseRegistration PluginRegistration) error {
	buildManifest, parseErr := mergeRegistrationManifest(parseRegistration)
	if parseErr != nil {
		return parseErr
	}
	storeBootstrapState.getMu.Lock()
	for _, getExisting := range storeBootstrapState.getBuiltin {
		getManifest, getErr := mergeRegistrationManifest(getExisting)
		if getErr == nil && getManifest.ID == buildManifest.ID {
			storeBootstrapState.getMu.Unlock()
			return fmt.Errorf("pluginruntime: plugin %q is already registered", buildManifest.ID)
		}
	}
	storeBootstrapState.getBuiltin = append(storeBootstrapState.getBuiltin, parseRegistration)
	getKernel := storeBootstrapState.getGlobalKernel
	storeBootstrapState.getMu.Unlock()
	if getKernel == nil || getKernel.isClosed() {
		return nil
	}
	return getKernel.startPlugin(mergedRegistration{
		getManifest: buildManifest,
		getFactory:  parseRegistration.Factory,
	})
}

// RegisteredBuiltinPlugins returns cloned built-in plugin registrations.
func RegisteredBuiltinPlugins() []PluginRegistration {
	storeBootstrapState.getMu.RLock()
	defer storeBootstrapState.getMu.RUnlock()
	return append([]PluginRegistration(nil), storeBootstrapState.getBuiltin...)
}

// RegisterBuiltinService registers one built-in kernel service value.
func RegisterBuiltinService(parseRegistration ServiceRegistration) error {
	if parseRegistration.Key == "" {
		return fmt.Errorf("pluginruntime: service key is required")
	}
	if parseRegistration.Value == nil {
		return fmt.Errorf("pluginruntime: service %q value is required", parseRegistration.Key)
	}
	storeBootstrapState.getMu.Lock()
	for parseIndex, getExisting := range storeBootstrapState.getServices {
		if getExisting.Key != parseRegistration.Key {
			continue
		}
		storeBootstrapState.getServices[parseIndex] = parseRegistration
		getKernel := storeBootstrapState.getGlobalKernel
		storeBootstrapState.getMu.Unlock()
		if getKernel != nil && !getKernel.isClosed() {
			getKernel.SetService(parseRegistration.Key, parseRegistration.Value)
		}
		return nil
	}
	storeBootstrapState.getServices = append(storeBootstrapState.getServices, parseRegistration)
	getKernel := storeBootstrapState.getGlobalKernel
	storeBootstrapState.getMu.Unlock()
	if getKernel != nil && !getKernel.isClosed() {
		getKernel.SetService(parseRegistration.Key, parseRegistration.Value)
	}
	return nil
}

// RegisteredBuiltinServices returns cloned built-in service registrations.
func RegisteredBuiltinServices() []ServiceRegistration {
	storeBootstrapState.getMu.RLock()
	defer storeBootstrapState.getMu.RUnlock()
	return append([]ServiceRegistration(nil), storeBootstrapState.getServices...)
}

// BootGlobalKernel constructs and stores the process-global kernel when needed.
func BootGlobalKernel(parseOptions BootstrapOptions) (*Kernel, error) {
	storeBootstrapState.getMu.Lock()
	defer storeBootstrapState.getMu.Unlock()
	if storeBootstrapState.getGlobalKernel != nil && !storeBootstrapState.getGlobalKernel.isClosed() {
		return storeBootstrapState.getGlobalKernel, nil
	}
	buildKernel, parseErr := NewKernel(BootstrapOptions{
		Registrations: append(append([]PluginRegistration(nil), storeBootstrapState.getBuiltin...), parseOptions.Registrations...),
		Services:      mergeServiceRegistrations(append([]ServiceRegistration(nil), storeBootstrapState.getServices...), parseOptions.Services),
		DisabledPluginIDs: append([]string(nil),
			parseOptions.DisabledPluginIDs...,
		),
	})
	if parseErr != nil {
		return nil, parseErr
	}
	storeBootstrapState.getGlobalKernel = buildKernel
	return buildKernel, nil
}

// GetGlobalKernel returns the process-global kernel when booted.
func GetGlobalKernel() *Kernel {
	storeBootstrapState.getMu.RLock()
	defer storeBootstrapState.getMu.RUnlock()
	return storeBootstrapState.getGlobalKernel
}

// ResetGlobalKernelForTesting clears global kernel state for tests.
func ResetGlobalKernelForTesting() error {
	storeBootstrapState.getMu.Lock()
	defer storeBootstrapState.getMu.Unlock()
	if storeBootstrapState.getGlobalKernel != nil {
		if parseErr := storeBootstrapState.getGlobalKernel.Close(); parseErr != nil {
			return parseErr
		}
	}
	storeBootstrapState.getBuiltin = nil
	storeBootstrapState.getServices = nil
	storeBootstrapState.getGlobalKernel = nil
	return nil
}

// mergeBootstrapRegistrations validates and merges one registration list.
func mergeBootstrapRegistrations(parseRegistrations []PluginRegistration) ([]mergedRegistration, error) {
	buildMerged := make([]mergedRegistration, 0, len(parseRegistrations))
	buildSeen := map[string]struct{}{}
	for _, parseRegistration := range parseRegistrations {
		buildManifest, parseErr := mergeRegistrationManifest(parseRegistration)
		if parseErr != nil {
			return nil, parseErr
		}
		if _, hasSeen := buildSeen[buildManifest.ID]; hasSeen {
			return nil, fmt.Errorf("pluginruntime: plugin %q is already registered", buildManifest.ID)
		}
		buildSeen[buildManifest.ID] = struct{}{}
		buildMerged = append(buildMerged, mergedRegistration{
			getManifest: buildManifest,
			getFactory:  parseRegistration.Factory,
		})
	}
	return buildMerged, nil
}

// normalizeDisabledPluginIDs returns trimmed disabled plugin IDs.
func normalizeDisabledPluginIDs(parseIDs []string) []string {
	if len(parseIDs) == 0 {
		return nil
	}
	buildIDs := make([]string, 0, len(parseIDs))
	buildSeen := map[string]struct{}{}
	for _, parseID := range parseIDs {
		buildTrimmed := strings.TrimSpace(parseID)
		if buildTrimmed == "" {
			continue
		}
		if _, hasSeen := buildSeen[buildTrimmed]; hasSeen {
			continue
		}
		buildSeen[buildTrimmed] = struct{}{}
		buildIDs = append(buildIDs, buildTrimmed)
	}
	return buildIDs
}

// mergeServiceRegistrations merges base services with overrides by key.
func mergeServiceRegistrations(parseBase []ServiceRegistration, parseOverrides []ServiceRegistration) []ServiceRegistration {
	buildByKey := map[ServiceKey]ServiceRegistration{}
	for _, parseRegistration := range parseBase {
		if parseRegistration.Key == "" || parseRegistration.Value == nil {
			continue
		}
		buildByKey[parseRegistration.Key] = parseRegistration
	}
	for _, parseRegistration := range parseOverrides {
		if parseRegistration.Key == "" || parseRegistration.Value == nil {
			continue
		}
		buildByKey[parseRegistration.Key] = parseRegistration
	}
	buildMerged := make([]ServiceRegistration, 0, len(buildByKey))
	for _, getRegistration := range buildByKey {
		buildMerged = append(buildMerged, getRegistration)
	}
	return cloneServiceRegistrations(buildMerged)
}
