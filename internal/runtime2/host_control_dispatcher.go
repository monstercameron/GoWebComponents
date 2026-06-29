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

// handleHostControlEnvelopePreflight validates host-dispatch envelope fields with a focused hot path for host-supported kinds.
func handleHostControlEnvelopePreflight(parseEnvelope ControlEnvelope) error {
	if parseErr := validateControlProtocolVersion(parseEnvelope.ProtocolVersion); parseErr != nil {
		return parseErr
	}
	switch parseEnvelope.Kind {
	case ControlKindPatchReady:
		return validateControlPatchReadyEnvelope(parseEnvelope)
	case ControlKindDiagnostic:
		return validateControlDiagnosticEnvelope(parseEnvelope)
	case ControlKindRestart:
		if _, parseErr := ParseRegionInstanceID(string(parseEnvelope.RegionInstanceID)); parseErr != nil {
			return parseErr
		}
		if parseEnvelope.Epoch == 0 {
			return fmt.Errorf("runtime2: restart epoch is required")
		}
		return nil
	case ControlKindPong:
		if !parseRuntimeHasTrimmedNonWhitespaceText(string(parseEnvelope.PongShardID)) {
			return fmt.Errorf("runtime2: pong shard ID is required")
		}
		if parseEnvelope.PongSequence == 0 {
			return fmt.Errorf("runtime2: pong sequence is required")
		}
		return nil
	default:
		if parseErr := ValidateControlEnvelope(parseEnvelope); parseErr != nil {
			return parseErr
		}
		return fmt.Errorf("runtime2: control kind %q is unsupported for host dispatch", parseEnvelope.Kind)
	}
}

// HandleHostControlEnvelope validates and routes one host-side control envelope.
func HandleHostControlEnvelope(
	parseHostRegionAdapter *HostRegionAdapter,
	parseEnvelope ControlEnvelope,
) (HostControlDispatchResult, error) {
	if parseHostRegionAdapter == nil {
		return HostControlDispatchResult{}, fmt.Errorf("runtime2: host region adapter is nil")
	}
	if parseErr := handleHostControlEnvelopePreflight(parseEnvelope); parseErr != nil {
		return HostControlDispatchResult{}, parseErr
	}
	getRegionInstanceID := parseHostRegionAdapter.storeRegionInstanceID
	if parseEnvelope.Kind != ControlKindPong && parseEnvelope.RegionInstanceID != getRegionInstanceID {
		return HostControlDispatchResult{}, fmt.Errorf(
			"runtime2: host region adapter mounted for %q cannot handle control envelope for region %q",
			getRegionInstanceID,
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
		applyControlDiagnosticRedaction(&parseEnvelope)
		getDiagnosticResult, parseDiagnosticErr := parseHostRegionAdapter.handleHostRegionDiagnosticEnvelopeValidated(parseEnvelope)
		if parseDiagnosticErr != nil {
			return HostControlDispatchResult{}, parseDiagnosticErr
		}
		return HostControlDispatchResult{
			HasDiagnosticResult:  true,
			GetDiagnosticType:    DiagnosticEventKind(getDiagnosticResult.GetDiagnostic.DiagnosticType),
			HasDiagnosticIgnored: getDiagnosticResult.HasIgnored,
			GetDiagnosticIgnore:  getDiagnosticResult.GetIgnoreReason,
			GetDiagnostic:        getDiagnosticResult.GetDiagnostic,
		}, nil
	case ControlKindRestart:
		if parseRestartErr := parseHostRegionAdapter.storeCoordinator.handleCoordinatorRestartRegionTrusted(getRegionInstanceID, parseEnvelope.Epoch); parseRestartErr != nil {
			return HostControlDispatchResult{}, parseRestartErr
		}
		parseRegionID := string(getRegionInstanceID)
		parseHostRegionAdapter.storeScheduler.ClearSchedulerFallbackOwnership(parseRegionID)
		parseHostRegionAdapter.storeRecoveryCoordinator.ClearRegionLocalFallback(parseRegionID)
		parseHostRegionAdapter.storeHostRegionDeferredDispatch = hostRegionDeferredDispatch{}
		parseHostRegionAdapter.hasHostRegionDeferredDispatch = false
		parseHostRegionAdapter.storeHostRegionSnapshotFingerprint = ""
		parseHostRegionAdapter.storeHostRegionSnapshotHash = [32]byte{}
		parseHostRegionAdapter.hasHostRegionSnapshotHash = false
		parseHostRegionAdapter.storeHostRegionDispatchHash = [32]byte{}
		parseHostRegionAdapter.hasHostRegionDispatchHash = false
		parseHostRegionAdapter.storeHostRegionDispatchBytes = nil
		parseHostRegionAdapter.hasHostRegionDispatchBytes = false
		parseHostRegionAdapter.storeHostRegionDispatchFastHash = 0
		parseHostRegionAdapter.hasHostRegionDispatchFastHash = false
		parseHostRegionAdapter.storeHostRegionDispatchRendererID = ""
		parseHostRegionAdapter.storeHostRegionDispatchEpoch = 0
		parseHostRegionAdapter.storeHostRegionDispatchInputVersion = 0
		parseHostRegionAdapter.storeHostRegionDispatchSourceVersion = 0
		parseHostRegionAdapter.storeHostRegionDispatchSourceVersionTuple = nil
		parseHostRegionAdapter.storeHostRegionDispatchSourceVersionScratch = nil
		parseHostRegionAdapter.hasHostRegionDispatchSourceVersionTuple = false
		parseHostRegionAdapter.hasHostRegionDispatchVersionVector = false
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
