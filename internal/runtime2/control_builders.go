package runtime2

// BuildControlReadyEnvelope builds one validated ready control envelope.
func BuildControlReadyEnvelope() ControlEnvelope {
	parseEnvelope := ControlEnvelope{
		ProtocolVersion: ProtocolVersionParallelV1,
		Kind:            ControlKindReady,
	}
	return parseEnvelope
}

// BuildControlCapabilitiesEnvelope builds one validated capabilities control envelope.
func BuildControlCapabilitiesEnvelope(parseCapabilityReport CapabilityReport) (ControlEnvelope, error) {
	parseEnvelope := ControlEnvelope{
		ProtocolVersion: ProtocolVersionParallelV1,
		Kind:            ControlKindCapabilities,
		Capabilities:    &parseCapabilityReport,
	}
	if parseErr := ValidateControlEnvelope(parseEnvelope); parseErr != nil {
		return ControlEnvelope{}, parseErr
	}
	return parseEnvelope, nil
}

// ControlDiagnosticEnvelopeSpec stores one diagnostic envelope payload contract.
type ControlDiagnosticEnvelopeSpec struct {
	DiagnosticType      DiagnosticEventKind
	DiagnosticText      string
	DiagnosticTiming    *DiagnosticTimingMetrics
	DiagnosticSize      *DiagnosticSizeMetrics
	DiagnosticFallback  *DiagnosticFallbackReason
	DiagnosticTrace     *DiagnosticTraceMetadata
	DiagnosticShardID   SchedulerShardID
	TransportTier       TransportTier
	DiagnosticDowngrade *DiagnosticDowngradeReason
}

// BuildControlPatchReadyEnvelope builds one validated patch-ready control envelope.
func BuildControlPatchReadyEnvelope(
	parseRegionInstanceID RegionInstanceID,
	parsePatchVersion uint64,
	parseInputVersion uint64,
	parseTransportTier TransportTier,
) (ControlEnvelope, error) {
	parseEnvelope := ControlEnvelope{
		ProtocolVersion:  ProtocolVersionParallelV1,
		Kind:             ControlKindPatchReady,
		RegionInstanceID: parseRegionInstanceID,
		PatchVersion:     parsePatchVersion,
		InputVersion:     parseInputVersion,
		TransportTier:    parseTransportTier,
	}
	if parseErr := ValidateControlEnvelope(parseEnvelope); parseErr != nil {
		return ControlEnvelope{}, parseErr
	}
	return parseEnvelope, nil
}

// BuildControlDiagnosticEnvelope builds one validated diagnostic control envelope.
func BuildControlDiagnosticEnvelope(
	parseRegionInstanceID RegionInstanceID,
	parseDiagnosticSpec ControlDiagnosticEnvelopeSpec,
) (ControlEnvelope, error) {
	parseEnvelope := ControlEnvelope{
		ProtocolVersion:     ProtocolVersionParallelV1,
		Kind:                ControlKindDiagnostic,
		RegionInstanceID:    parseRegionInstanceID,
		DiagnosticType:      string(parseDiagnosticSpec.DiagnosticType),
		DiagnosticText:      parseDiagnosticSpec.DiagnosticText,
		DiagnosticTiming:    parseDiagnosticSpec.DiagnosticTiming,
		DiagnosticSize:      parseDiagnosticSpec.DiagnosticSize,
		DiagnosticFallback:  parseDiagnosticSpec.DiagnosticFallback,
		DiagnosticTrace:     parseDiagnosticSpec.DiagnosticTrace,
		DiagnosticShardID:   parseDiagnosticSpec.DiagnosticShardID,
		TransportTier:       parseDiagnosticSpec.TransportTier,
		DiagnosticDowngrade: parseDiagnosticSpec.DiagnosticDowngrade,
	}
	parseEnvelope = RedactControlDiagnosticEnvelope(parseEnvelope)
	if parseErr := ValidateControlEnvelope(parseEnvelope); parseErr != nil {
		return ControlEnvelope{}, parseErr
	}
	return parseEnvelope, nil
}

// BuildControlMountEnvelope builds one validated mount control envelope.
func BuildControlMountEnvelope(parseRendererID RendererID, parseSnapshot SnapshotEnvelope) (ControlEnvelope, error) {
	parseEnvelope := ControlEnvelope{
		ProtocolVersion:  ProtocolVersionParallelV1,
		Kind:             ControlKindMount,
		RegionInstanceID: parseSnapshot.RegionInstanceID,
		RendererID:       parseRendererID,
		Snapshot:         &parseSnapshot,
	}
	if parseErr := ValidateControlEnvelope(parseEnvelope); parseErr != nil {
		return ControlEnvelope{}, parseErr
	}
	return parseEnvelope, nil
}

// BuildControlUpdateEnvelope builds one validated update control envelope.
func BuildControlUpdateEnvelope(parseSnapshot SnapshotEnvelope) (ControlEnvelope, error) {
	parseEnvelope := ControlEnvelope{
		ProtocolVersion:  ProtocolVersionParallelV1,
		Kind:             ControlKindUpdate,
		RegionInstanceID: parseSnapshot.RegionInstanceID,
		InputVersion:     parseSnapshot.InputVersion,
		Snapshot:         &parseSnapshot,
	}
	if parseErr := ValidateControlEnvelope(parseEnvelope); parseErr != nil {
		return ControlEnvelope{}, parseErr
	}
	return parseEnvelope, nil
}

// BuildControlCancelEnvelope builds one validated cancel control envelope.
func BuildControlCancelEnvelope(parseRegionInstanceID RegionInstanceID) (ControlEnvelope, error) {
	parseEnvelope := ControlEnvelope{
		ProtocolVersion:  ProtocolVersionParallelV1,
		Kind:             ControlKindCancel,
		RegionInstanceID: parseRegionInstanceID,
	}
	if parseErr := ValidateControlEnvelope(parseEnvelope); parseErr != nil {
		return ControlEnvelope{}, parseErr
	}
	return parseEnvelope, nil
}

// BuildControlDisposeEnvelope builds one validated dispose control envelope.
func BuildControlDisposeEnvelope(parseRegionInstanceID RegionInstanceID) (ControlEnvelope, error) {
	parseEnvelope := ControlEnvelope{
		ProtocolVersion:  ProtocolVersionParallelV1,
		Kind:             ControlKindDispose,
		RegionInstanceID: parseRegionInstanceID,
	}
	if parseErr := ValidateControlEnvelope(parseEnvelope); parseErr != nil {
		return ControlEnvelope{}, parseErr
	}
	return parseEnvelope, nil
}

// BuildControlRestartEnvelope builds one validated restart control envelope.
func BuildControlRestartEnvelope(parseRegionInstanceID RegionInstanceID, parseEpoch uint64) (ControlEnvelope, error) {
	parseEnvelope := ControlEnvelope{
		ProtocolVersion:  ProtocolVersionParallelV1,
		Kind:             ControlKindRestart,
		RegionInstanceID: parseRegionInstanceID,
		Epoch:            parseEpoch,
	}
	if parseErr := ValidateControlEnvelope(parseEnvelope); parseErr != nil {
		return ControlEnvelope{}, parseErr
	}
	return parseEnvelope, nil
}

// BuildControlPongEnvelope builds one validated pong control envelope.
func BuildControlPongEnvelope(parseShardID SchedulerShardID, parseSequence uint64) (ControlEnvelope, error) {
	parseEnvelope := ControlEnvelope{
		ProtocolVersion: ProtocolVersionParallelV1,
		Kind:            ControlKindPong,
		PongShardID:     parseShardID,
		PongSequence:    parseSequence,
	}
	if parseErr := ValidateControlEnvelope(parseEnvelope); parseErr != nil {
		return ControlEnvelope{}, parseErr
	}
	return parseEnvelope, nil
}
