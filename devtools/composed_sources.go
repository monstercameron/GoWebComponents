package devtools

import (
	"sync"

	"github.com/monstercameron/GoWebComponents/plugin"
)

var hostExtensionSources struct {
	getMu     sync.RWMutex
	getNextID int
	getHosts  map[int]*plugin.Host
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
	hostExtensionSources.getNextID++
	getID := hostExtensionSources.getNextID
	hostExtensionSources.getHosts[getID] = parseHost
	return getID
}

// unregisterHostExtensionSource removes one compatibility host source.
func unregisterHostExtensionSource(parseID int) {
	if parseID == 0 {
		return
	}
	hostExtensionSources.getMu.Lock()
	defer hostExtensionSources.getMu.Unlock()
	delete(hostExtensionSources.getHosts, parseID)
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
