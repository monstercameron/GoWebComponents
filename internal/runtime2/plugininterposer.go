package runtime2

import (
	"sync"

	"github.com/monstercameron/GoWebComponents/v4/internal/pluginruntime"
)

func init() {
	_ = pluginruntime.RegisterBuiltinService(pluginruntime.ServiceRegistration{
		Key:   pluginruntime.ServiceKeyRuntime2Meta,
		Value: BuildRuntime2MetaService(),
	})
}

type buildRuntime2MetaService struct{}

// Runtime2MetaProvider returns one normalized runtime2 metadata snapshot.
type Runtime2MetaProvider func(pluginruntime.QueryBudget) (pluginruntime.Runtime2MetaSnapshot, error)

var storeRuntime2MetaState struct {
	getMu       sync.RWMutex
	getProvider Runtime2MetaProvider
}

// BuildRuntime2MetaService returns one runtime2 capability service.
func BuildRuntime2MetaService() pluginruntime.Runtime2MetaService {
	return buildRuntime2MetaService{}
}

// SetRuntime2MetaProvider stores one runtime2 metadata provider for plugin inspection.
func SetRuntime2MetaProvider(parseProvider Runtime2MetaProvider) {
	storeRuntime2MetaState.getMu.Lock()
	defer storeRuntime2MetaState.getMu.Unlock()
	storeRuntime2MetaState.getProvider = parseProvider
}

// ResetRuntime2MetaProvider clears the current runtime2 metadata provider.
func ResetRuntime2MetaProvider() {
	SetRuntime2MetaProvider(nil)
}

// GetRuntime2MetaSnapshot returns one normalized runtime2 capability snapshot.
func (buildRuntime2MetaService) GetRuntime2MetaSnapshot(parseBudget pluginruntime.QueryBudget) (pluginruntime.Runtime2MetaSnapshot, error) {
	buildSnapshot := pluginruntime.Runtime2MetaSnapshot{
		Meta:         pluginruntime.BuildSnapshotMeta(pluginruntime.BackendIDRuntime2, false),
		Capabilities: buildRuntime2CapabilityMap(),
	}
	getProvider := getRuntime2MetaProvider()
	if getProvider == nil {
		return buildSnapshot, nil
	}
	getSnapshot, parseErr := getProvider(parseBudget)
	if parseErr != nil {
		return pluginruntime.Runtime2MetaSnapshot{}, parseErr
	}
	if len(getSnapshot.Capabilities) == 0 {
		getSnapshot.Capabilities = buildSnapshot.Capabilities
	}
	if getSnapshot.Meta.BackendID == "" {
		getSnapshot.Meta.BackendID = string(pluginruntime.BackendIDRuntime2)
	}
	return getSnapshot, nil
}

// buildRuntime2CapabilityMap builds one cloned capability map for runtime2 inspection.
func buildRuntime2CapabilityMap() map[string]bool {
	getReport := GetCapabilityReport()
	return map[string]bool{
		"hasWorkerSupport":                getReport.HasWorkerSupport,
		"hasMessagePortSupport":           getReport.HasMessagePortSupport,
		"hasStructuredCloneSupport":       getReport.HasStructuredCloneSupport,
		"hasBinaryTransportSupport":       getReport.HasBinaryTransportSupport,
		"hasSharedBufferSupport":          getReport.HasSharedBufferSupport,
		"hasSharedMemoryTransportSupport": getReport.HasSharedMemoryTransportSupport,
	}
}

// getRuntime2MetaProvider returns the currently registered runtime2 metadata provider.
func getRuntime2MetaProvider() Runtime2MetaProvider {
	storeRuntime2MetaState.getMu.RLock()
	defer storeRuntime2MetaState.getMu.RUnlock()
	return storeRuntime2MetaState.getProvider
}
