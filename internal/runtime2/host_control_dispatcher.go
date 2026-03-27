package runtime2

import "fmt"

// HostControlDispatchResult reports one host-side control-envelope dispatch outcome.
type HostControlDispatchResult struct {
	HasPatchReadyResult bool
	HasDiagnosticResult bool
	HasRestartResult    bool
	HasPongResult       bool

	GetPatchReadyResult  HostRegionPatchReadyResult
	GetDiagnosticType    DiagnosticEventKind
	HasDiagnosticIgnored bool
	GetDiagnosticIgnore  string
	GetDiagnostic        ControlEnvelope
	GetRestartEpoch      uint64
	GetPongShardID       SchedulerShardID
	GetPongSequence      uint64
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
	if parseEnvelope.Kind != ControlKindPong && parseEnvelope.RegionInstanceID != parseHostRegionAdapter.storeRegionInstanceID {
		return HostControlDispatchResult{}, fmt.Errorf(
			"runtime2: host region adapter mounted for %q cannot handle control envelope for region %q",
			parseHostRegionAdapter.storeRegionInstanceID,
			parseEnvelope.RegionInstanceID,
		)
	}
	switch parseEnvelope.Kind {
	case ControlKindPong:
		if parsePongErr := parseHostRegionAdapter.storeScheduler.HandleSchedulerKeepalivePong(parseEnvelope.PongShardID, parseEnvelope.PongSequence); parsePongErr != nil {
			return HostControlDispatchResult{}, parsePongErr
		}
		return HostControlDispatchResult{
			HasPongResult:   true,
			GetPongShardID:  parseEnvelope.PongShardID,
			GetPongSequence: parseEnvelope.PongSequence,
		}, nil
	case ControlKindPatchReady:
		if parseTierErr := parseHostRegionAdapter.SetHostRegionPatchTransportTier(parseEnvelope.TransportTier); parseTierErr != nil {
			return HostControlDispatchResult{}, parseTierErr
		}
		parsePatchReadyResult, parsePatchReadyErr := parseHostRegionAdapter.HandleHostRegionPatchReadyWithVersion(
			parseEnvelope.PatchVersion,
			parseEnvelope.InputVersion,
		)
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
		getDiagnosticResult, parseDiagnosticErr := parseHostRegionAdapter.HandleHostRegionDiagnosticEnvelope(parseEnvelope)
		if parseDiagnosticErr != nil {
			return HostControlDispatchResult{}, parseDiagnosticErr
		}
		parseDiagnosticType, parseDiagnosticTypeErr := ParseDiagnosticEventKind(getDiagnosticResult.GetDiagnostic.DiagnosticType)
		if parseDiagnosticTypeErr != nil {
			return HostControlDispatchResult{}, parseDiagnosticTypeErr
		}
		return HostControlDispatchResult{
			HasDiagnosticResult:  true,
			GetDiagnosticType:    parseDiagnosticType,
			HasDiagnosticIgnored: getDiagnosticResult.HasIgnored,
			GetDiagnosticIgnore:  getDiagnosticResult.GetIgnoreReason,
			GetDiagnostic:        getDiagnosticResult.GetDiagnostic,
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
		parseHostRegionAdapter.storeHostRegionTransportTier = TransportTierStructuredClone
		parseHostRegionAdapter.storeHostRegionSnapshotTier = TransportTierStructuredClone
		parseHostRegionAdapter.storeHostRegionPatchTier = TransportTierStructuredClone
		parseHostRegionAdapter.storeHostRegionSnapshotDowngrade = DiagnosticDowngradeReason{}
		parseHostRegionAdapter.storeHostRegionPatchDowngrade = DiagnosticDowngradeReason{}
		parseHostRegionAdapter.hasHostRegionSnapshotDowngrade = false
		parseHostRegionAdapter.hasHostRegionPatchDowngrade = false
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
