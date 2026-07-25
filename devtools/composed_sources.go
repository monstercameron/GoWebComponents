package devtools

import (
	"sync"

	"github.com/monstercameron/GoWebComponents/v5/plugin"
)

var hostExtensionSources struct {
	getMu        sync.RWMutex
	getNextID    int
	getHosts     map[int]*plugin.Host
	getHostIDs   map[*plugin.Host]int
	getRefCounts map[int]int
}

// InspectComposedExtensionSections returns app-owned, host-owned, and kernel-owned extension sections.
func InspectComposedExtensionSections() []ExtensionSection {
	buildSections := append([]ExtensionSection(nil), InspectExtensionSections()...)
	buildSections = append(buildSections, inspectHostExtensionSections()...)
	buildSections = append(buildSections, inspectKernelExtensionSections()...)
	return cloneExtensionSections(buildSections)
}

// InspectComposedErrorOverlayActions returns app-owned, host-owned, and kernel-owned overlay actions.
func InspectComposedErrorOverlayActions() []ErrorOverlayAction {
	buildActions := append([]ErrorOverlayAction(nil), InspectErrorOverlayActions()...)
	buildActions = append(buildActions, inspectHostOverlayActions()...)
	buildActions = append(buildActions, inspectKernelOverlayActions()...)
	return cloneErrorOverlayActions(buildActions)
}

// registerHostExtensionSource stores one compatibility host source and returns its token.
func registerHostExtensionSource(parseHost *plugin.Host) int {
	if parseHost == nil {
		return 0
	}
	hostExtensionSources.getMu.Lock()
	defer hostExtensionSources.getMu.Unlock()
	if hostExtensionSources.getHosts == nil {
		hostExtensionSources.getHosts = map[int]*plugin.Host{}
	}
	if hostExtensionSources.getHostIDs == nil {
		hostExtensionSources.getHostIDs = map[*plugin.Host]int{}
	}
	if hostExtensionSources.getRefCounts == nil {
		hostExtensionSources.getRefCounts = map[int]int{}
	}
	if getExistingID, hasExistingID := hostExtensionSources.getHostIDs[parseHost]; hasExistingID {
		hostExtensionSources.getRefCounts[getExistingID]++
		return getExistingID
	}
	hostExtensionSources.getNextID++
	getID := hostExtensionSources.getNextID
	hostExtensionSources.getHosts[getID] = parseHost
	hostExtensionSources.getHostIDs[parseHost] = getID
	hostExtensionSources.getRefCounts[getID] = 1
	return getID
}

// unregisterHostExtensionSource removes one compatibility host source.
func unregisterHostExtensionSource(parseID int) {
	if parseID == 0 {
		return
	}
	hostExtensionSources.getMu.Lock()
	defer hostExtensionSources.getMu.Unlock()
	getHost, hasHost := hostExtensionSources.getHosts[parseID]
	if !hasHost {
		return
	}
	if getRefCount := hostExtensionSources.getRefCounts[parseID]; getRefCount > 1 {
		hostExtensionSources.getRefCounts[parseID] = getRefCount - 1
		return
	}
	delete(hostExtensionSources.getHosts, parseID)
	delete(hostExtensionSources.getRefCounts, parseID)
	delete(hostExtensionSources.getHostIDs, getHost)
}

// inspectHostExtensionSections resolves current host-owned sections from all registered compatibility sources.
func inspectHostExtensionSections() []ExtensionSection {
	hostExtensionSources.getMu.RLock()
	defer hostExtensionSources.getMu.RUnlock()
	buildSections := make([]ExtensionSection, 0)
	for _, getHost := range hostExtensionSources.getHosts {
		buildSections = append(buildSections, buildHostExtensionSections(getHost)...)
	}
	return cloneExtensionSections(buildSections)
}

// inspectHostOverlayActions resolves current host-owned actions from all registered compatibility sources.
func inspectHostOverlayActions() []ErrorOverlayAction {
	hostExtensionSources.getMu.RLock()
	defer hostExtensionSources.getMu.RUnlock()
	buildActions := make([]ErrorOverlayAction, 0)
	for _, getHost := range hostExtensionSources.getHosts {
		buildActions = append(buildActions, buildHostOverlayActions(getHost)...)
	}
	return cloneErrorOverlayActions(buildActions)
}
