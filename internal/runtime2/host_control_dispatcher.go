package runtime2

import "fmt"

// HostControlDispatchResult reports one host-side control-envelope dispatch outcome.
type HostControlDispatchResult struct {
	HasPatchReadyResult bool
	HasDiagnosticResult bool
	HasRestartResult    bool

	GetPatchReadyResult HostRegionPatchReadyResult
	GetDiagnosticType   DiagnosticEventKind
	GetRestartEpoch     uint64
}

// HandleHostControlEnvelope validates and routes one host-side control envelope.
func HandleHostControlEnvelope(
	parseHostRegionAdapter *HostRegionAdapter,
	parseEnvelope ControlEnvelope,
) (HostControlDispatchResult, error) {
	if parseHostRegionAdapter == nil {
		return HostControlDispatchResult{}, fmt.Errorf("runtime2: host region adapter is nil")
	}
	if parseErr := ValidateControlEnvelope(parseEnvelope); parseErr != nil {
		return HostControlDispatchResult{}, parseErr
	}
	if parseEnvelope.RegionInstanceID != parseHostRegionAdapter.storeRegionInstanceID {
		return HostControlDispatchResult{}, fmt.Errorf(
			"runtime2: host region adapter mounted for %q cannot handle control envelope for region %q",
			parseHostRegionAdapter.storeRegionInstanceID,
			parseEnvelope.RegionInstanceID,
		)
	}
	switch parseEnvelope.Kind {
	case ControlKindPatchReady:
		parsePatchReadyResult, parsePatchReadyErr := parseHostRegionAdapter.HandleHostRegionPatchReady(parseEnvelope.PatchVersion)
		if parsePatchReadyErr != nil {
			return HostControlDispatchResult{}, parsePatchReadyErr
		}
		return HostControlDispatchResult{
			HasPatchReadyResult: true,
			GetPatchReadyResult: parsePatchReadyResult,
		}, nil
	case ControlKindDiagnostic:
		if _, parseHasEntry := parseHostRegionAdapter.storeCoordinator.GetEntry(parseHostRegionAdapter.storeRegionInstanceID); !parseHasEntry {
			return HostControlDispatchResult{}, fmt.Errorf("runtime2: host region %q is not mounted", parseHostRegionAdapter.storeRegionInstanceID)
		}
		parseDiagnosticType, parseDiagnosticTypeErr := ParseDiagnosticEventKind(parseEnvelope.DiagnosticType)
		if parseDiagnosticTypeErr != nil {
			return HostControlDispatchResult{}, parseDiagnosticTypeErr
		}
		return HostControlDispatchResult{
			HasDiagnosticResult: true,
			GetDiagnosticType:   parseDiagnosticType,
		}, nil
	case ControlKindRestart:
		if _, parseHasEntry := parseHostRegionAdapter.storeCoordinator.GetEntry(parseHostRegionAdapter.storeRegionInstanceID); !parseHasEntry {
			return HostControlDispatchResult{}, fmt.Errorf("runtime2: host region %q is not mounted", parseHostRegionAdapter.storeRegionInstanceID)
		}
		if parseRestartErr := parseHostRegionAdapter.storeCoordinator.RestartRegion(parseHostRegionAdapter.storeRegionInstanceID, parseEnvelope.Epoch); parseRestartErr != nil {
			return HostControlDispatchResult{}, parseRestartErr
		}
		parseRegionID := string(parseHostRegionAdapter.storeRegionInstanceID)
		parseHostRegionAdapter.storeScheduler.ClearSchedulerFallbackOwnership(parseRegionID)
		parseHostRegionAdapter.storeRecoveryCoordinator.ClearRegionLocalFallback(parseRegionID)
		parseHostRegionAdapter.storeHostRegionDeferredDispatch = nil
		parseHostRegionAdapter.storeHostRegionSnapshotFingerprint = ""
		parseHostRegionAdapter.storeHostRegionRepairRemountEpoch = 0
		parseHostRegionAdapter.storeHostRegionRepairVersionFloor = 0
		parseHostRegionAdapter.isHostRegionFallbackPending = false
		parseHostRegionAdapter.isHostRegionFallbackActive = false
		parseHostRegionAdapter.isHostRegionRepairPending = false
		parseHostRegionAdapter.isHostRegionRemoved = false
		parseHostRegionAdapter.hasHostRegionPostHydrationAttached = false
		return HostControlDispatchResult{
			HasRestartResult: true,
			GetRestartEpoch:  parseEnvelope.Epoch,
		}, nil
	default:
		return HostControlDispatchResult{}, fmt.Errorf("runtime2: control kind %q is unsupported for host dispatch", parseEnvelope.Kind)
	}
}
