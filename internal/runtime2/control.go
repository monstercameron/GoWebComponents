package runtime2

import (
	"encoding/json"
	"fmt"
)

// ControlKind identifies one control-plane message kind.
type ControlKind string

const (
	// ControlKindReady reports worker readiness.
	ControlKindReady ControlKind = "ready"
	// ControlKindCapabilities reports worker capabilities.
	ControlKindCapabilities ControlKind = "capabilities"
	// ControlKindMount requests region mount work.
	ControlKindMount ControlKind = "mount"
	// ControlKindUpdate requests region update work.
	ControlKindUpdate ControlKind = "update"
	// ControlKindEvent requests one semantic event-slot dispatch against one mounted region.
	ControlKindEvent ControlKind = "event"
	// ControlKindCancel requests cancellation of in-flight region work.
	ControlKindCancel ControlKind = "cancel"
	// ControlKindDispose requests region disposal.
	ControlKindDispose ControlKind = "dispose"
	// ControlKindPatchReady reports that a patch is available.
	ControlKindPatchReady ControlKind = "patch-ready"
	// ControlKindDiagnostic reports diagnostics.
	ControlKindDiagnostic ControlKind = "diagnostic"
	// ControlKindRestart coordinates restart handling.
	ControlKindRestart ControlKind = "restart"
	// ControlKindPong reports shard liveness keepalive.
	ControlKindPong ControlKind = "pong"
)

// TransportTier identifies the patch or snapshot transport tier.
type TransportTier string

const (
	// TransportTierStructuredClone identifies structured-clone payload transport.
	TransportTierStructuredClone TransportTier = "structured-clone"
	// TransportTierBinary identifies binary payload transport.
	TransportTierBinary TransportTier = "binary"
	// TransportTierSharedBuffer identifies shared-buffer transport.
	TransportTierSharedBuffer TransportTier = "shared-buffer"
)

// ControlEnvelope stores a validated control-plane message.
type ControlEnvelope struct {
	ProtocolVersion     ProtocolVersion            `json:"protocol_version"`
	Kind                ControlKind                `json:"kind"`
	RegionInstanceID    RegionInstanceID           `json:"region_instance_id,omitempty"`
	RendererID          RendererID                 `json:"renderer_id,omitempty"`
	Epoch               uint64                     `json:"epoch,omitempty"`
	InputVersion        uint64                     `json:"input_version,omitempty"`
	PatchVersion        uint64                     `json:"patch_version,omitempty"`
	TransportTier       TransportTier              `json:"transport_tier,omitempty"`
	EventSlot           *EventSlotDispatch         `json:"event_slot,omitempty"`
	Capabilities        *CapabilityReport          `json:"capabilities,omitempty"`
	Snapshot            *SnapshotEnvelope          `json:"snapshot,omitempty"`
	DiagnosticType      string                     `json:"diagnostic_type,omitempty"`
	DiagnosticText      string                     `json:"diagnostic_text,omitempty"`
	DiagnosticTiming    *DiagnosticTimingMetrics   `json:"diagnostic_timing,omitempty"`
	DiagnosticSize      *DiagnosticSizeMetrics     `json:"diagnostic_size,omitempty"`
	DiagnosticFallback  *DiagnosticFallbackReason  `json:"diagnostic_fallback,omitempty"`
	DiagnosticTrace     *DiagnosticTraceMetadata   `json:"diagnostic_trace,omitempty"`
	DiagnosticShardID   SchedulerShardID           `json:"diagnostic_shard_id,omitempty"`
	DiagnosticDowngrade *DiagnosticDowngradeReason `json:"diagnostic_downgrade,omitempty"`
	PongShardID         SchedulerShardID           `json:"pong_shard_id,omitempty"`
	PongSequence        uint64                     `json:"pong_sequence,omitempty"`
}

// ParseControlKind validates a raw control-plane kind value.
func ParseControlKind(parseRaw string) (ControlKind, error) {
	parseKind := ControlKind(parseRaw)
	switch parseKind {
	case ControlKindReady,
		ControlKindCapabilities,
		ControlKindMount,
		ControlKindUpdate,
		ControlKindEvent,
		ControlKindCancel,
		ControlKindDispose,
		ControlKindPatchReady,
		ControlKindDiagnostic,
		ControlKindRestart,
		ControlKindPong:
		return parseKind, nil
	case "":
		return "", fmt.Errorf("runtime2: control kind is required")
	default:
		return "", fmt.Errorf("runtime2: control kind %q is unsupported", parseRaw)
	}
}

// ParseTransportTier validates a raw transport-tier value.
func ParseTransportTier(parseRaw string) (TransportTier, error) {
	parseTier := TransportTier(parseRaw)
	switch parseTier {
	case TransportTierStructuredClone, TransportTierBinary, TransportTierSharedBuffer:
		return parseTier, nil
	case "":
		return "", fmt.Errorf("runtime2: transport tier is required")
	default:
		return "", fmt.Errorf("runtime2: transport tier %q is unsupported", parseRaw)
	}
}

// validateControlProtocolVersion verifies one control-envelope protocol version with a direct hot-path check.
func validateControlProtocolVersion(parseProtocolVersion ProtocolVersion) error {
	switch parseProtocolVersion {
	case ProtocolVersionParallelV1:
		return nil
	case "":
		return fmt.Errorf("runtime2: protocol version is required")
	default:
		return fmt.Errorf("runtime2: protocol version %q is unsupported", parseProtocolVersion)
	}
}

// validateControlTransportTier verifies one control-envelope transport tier with a direct hot-path check.
func validateControlTransportTier(parseTransportTier TransportTier) error {
	switch parseTransportTier {
	case TransportTierStructuredClone, TransportTierBinary, TransportTierSharedBuffer:
		return nil
	case "":
		return fmt.Errorf("runtime2: transport tier is required")
	default:
		return fmt.Errorf("runtime2: transport tier %q is unsupported", parseTransportTier)
	}
}

// validateControlPatchReadyEnvelope verifies the patch-ready-specific control-envelope fields.
func validateControlPatchReadyEnvelope(parseEnvelope ControlEnvelope) error {
	if _, parseErr := ParseRegionInstanceID(string(parseEnvelope.RegionInstanceID)); parseErr != nil {
		return parseErr
	}
	if parseEnvelope.PatchVersion == 0 {
		return fmt.Errorf("runtime2: patch version is required")
	}
	if parseEnvelope.InputVersion == 0 {
		return fmt.Errorf("runtime2: patch-ready input version is required")
	}
	return validateControlTransportTier(parseEnvelope.TransportTier)
}

// validateControlDiagnosticEnvelope verifies the diagnostic-specific control-envelope fields.
func validateControlDiagnosticEnvelope(parseEnvelope ControlEnvelope) error {
	if _, parseErr := ParseRegionInstanceID(string(parseEnvelope.RegionInstanceID)); parseErr != nil {
		return parseErr
	}
	if _, parseErr := ParseDiagnosticEventKind(parseEnvelope.DiagnosticType); parseErr != nil {
		return parseErr
	}
	if parseEnvelope.DiagnosticTiming != nil {
		if parseErr := ValidateDiagnosticTimingMetrics(*parseEnvelope.DiagnosticTiming); parseErr != nil {
			return parseErr
		}
	}
	if parseEnvelope.DiagnosticSize != nil {
		if parseErr := ValidateDiagnosticSizeMetrics(*parseEnvelope.DiagnosticSize); parseErr != nil {
			return parseErr
		}
	}
	if parseEnvelope.DiagnosticFallback != nil {
		if parseErr := ValidateDiagnosticFallbackReason(*parseEnvelope.DiagnosticFallback); parseErr != nil {
			return parseErr
		}
	}
	if parseEnvelope.DiagnosticTrace != nil {
		if parseErr := ValidateDiagnosticTraceMetadata(*parseEnvelope.DiagnosticTrace); parseErr != nil {
			return parseErr
		}
	}
	if parseEnvelope.DiagnosticShardID != "" {
		if parseErr := ValidateDiagnosticShardID(parseEnvelope.DiagnosticShardID); parseErr != nil {
			return parseErr
		}
	}
	if parseEnvelope.TransportTier != "" {
		if parseErr := validateControlTransportTier(parseEnvelope.TransportTier); parseErr != nil {
			return parseErr
		}
	}
	if parseEnvelope.DiagnosticDowngrade != nil {
		if parseErr := ValidateDiagnosticDowngradeReason(*parseEnvelope.DiagnosticDowngrade); parseErr != nil {
			return parseErr
		}
	}
	return nil
}

// ValidateControlEnvelope verifies one control-plane message is internally consistent.
func ValidateControlEnvelope(parseEnvelope ControlEnvelope) error {
	if parseErr := validateControlProtocolVersion(parseEnvelope.ProtocolVersion); parseErr != nil {
		return parseErr
	}
	switch parseEnvelope.Kind {
	case ControlKindReady:
		return nil
	case ControlKindPatchReady:
		return validateControlPatchReadyEnvelope(parseEnvelope)
	case ControlKindDiagnostic:
		return validateControlDiagnosticEnvelope(parseEnvelope)
	case ControlKindCapabilities:
		if parseEnvelope.Capabilities == nil {
			return fmt.Errorf("runtime2: capabilities payload is required")
		}
		return ValidateCapabilityReport(*parseEnvelope.Capabilities)
	case ControlKindMount:
		if _, parseErr := ParseRendererID(string(parseEnvelope.RendererID)); parseErr != nil {
			return parseErr
		}
		if _, parseErr := ParseRegionInstanceID(string(parseEnvelope.RegionInstanceID)); parseErr != nil {
			return parseErr
		}
		if parseEnvelope.Snapshot == nil {
			return fmt.Errorf("runtime2: mount snapshot is required")
		}
		if parseErr := ValidateSnapshotEnvelope(*parseEnvelope.Snapshot); parseErr != nil {
			return parseErr
		}
		if parseEnvelope.Snapshot.RegionInstanceID != parseEnvelope.RegionInstanceID {
			return fmt.Errorf("runtime2: mount snapshot region instance ID mismatch")
		}
		return nil
	case ControlKindUpdate:
		if _, parseErr := ParseRegionInstanceID(string(parseEnvelope.RegionInstanceID)); parseErr != nil {
			return parseErr
		}
		if parseEnvelope.InputVersion == 0 {
			return fmt.Errorf("runtime2: update input version is required")
		}
		if parseEnvelope.Snapshot == nil {
			return fmt.Errorf("runtime2: update snapshot is required")
		}
		if parseErr := ValidateSnapshotEnvelope(*parseEnvelope.Snapshot); parseErr != nil {
			return parseErr
		}
		if parseEnvelope.Snapshot.RegionInstanceID != parseEnvelope.RegionInstanceID {
			return fmt.Errorf("runtime2: update snapshot region instance ID mismatch")
		}
		if parseEnvelope.Snapshot.InputVersion != parseEnvelope.InputVersion {
			return fmt.Errorf("runtime2: update snapshot input version mismatch")
		}
		return nil
	case ControlKindEvent:
		if _, parseErr := ParseRegionInstanceID(string(parseEnvelope.RegionInstanceID)); parseErr != nil {
			return parseErr
		}
		if parseEnvelope.EventSlot == nil {
			return fmt.Errorf("runtime2: event-slot payload is required")
		}
		return ValidateEventSlotDispatch(*parseEnvelope.EventSlot)
	case ControlKindCancel, ControlKindDispose:
		if _, parseErr := ParseRegionInstanceID(string(parseEnvelope.RegionInstanceID)); parseErr != nil {
			return parseErr
		}
		return nil
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
	case "":
		return fmt.Errorf("runtime2: control kind is required")
	default:
		return fmt.Errorf("runtime2: control kind %q is unsupported", parseEnvelope.Kind)
	}
}

// hasControlDiagnosticText reports whether one control envelope carries redactable diagnostic text.
func hasControlDiagnosticText(parseEnvelope ControlEnvelope) bool {
	return parseEnvelope.Kind == ControlKindDiagnostic && parseEnvelope.DiagnosticText != ""
}

// BuildControlEnvelopeJSON encodes a validated control-plane envelope.
func BuildControlEnvelopeJSON(parseEnvelope ControlEnvelope) ([]byte, error) {
	if parseErr := ValidateControlEnvelope(parseEnvelope); parseErr != nil {
		return nil, parseErr
	}
	if !hasControlDiagnosticText(parseEnvelope) {
		return json.Marshal(parseEnvelope)
	}
	applyControlDiagnosticRedaction(&parseEnvelope)
	return json.Marshal(parseEnvelope)
}

// ParseControlEnvelopeJSON decodes and validates a control-plane envelope.
func ParseControlEnvelopeJSON(parseValue []byte) (ControlEnvelope, error) {
	var parseEnvelope ControlEnvelope
	if parseErr := json.Unmarshal(parseValue, &parseEnvelope); parseErr != nil {
		return ControlEnvelope{}, fmt.Errorf("runtime2: decode control envelope: %w", parseErr)
	}
	if parseErr := ValidateControlEnvelope(parseEnvelope); parseErr != nil {
		return ControlEnvelope{}, parseErr
	}
	if !hasControlDiagnosticText(parseEnvelope) {
		return parseEnvelope, nil
	}
	applyControlDiagnosticRedaction(&parseEnvelope)
	return parseEnvelope, nil
}
