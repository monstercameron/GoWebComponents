package runtime2

import "fmt"

// SelectSnapshotTransportTier selects the preferred snapshot transport tier from the current capability report.
func SelectSnapshotTransportTier(parseCapabilities CapabilityReport) (TransportTier, error) {
	if parseCapabilities.HasBinaryTransportSupport {
		return TransportTierBinary, nil
	}
	if parseCapabilities.HasStructuredCloneSupport {
		return TransportTierStructuredClone, nil
	}
	return "", fmt.Errorf("runtime2: no supported snapshot transport tier is available")
}

// BuildSnapshotTransportPayloadWithFallback encodes one snapshot payload and downgrades to structured-clone when binary encode fails.
func BuildSnapshotTransportPayloadWithFallback(parseEnvelope SnapshotEnvelope, parseCapabilities CapabilityReport) (TransportTier, []byte, error) {
	parseTransportTier, parseErr := SelectSnapshotTransportTier(parseCapabilities)
	if parseErr != nil {
		return "", nil, parseErr
	}
	if parseTransportTier == TransportTierBinary {
		parsePayload, parseErr := BuildBinarySnapshotEnvelope(parseEnvelope)
		if parseErr == nil {
			return TransportTierBinary, parsePayload, nil
		}
		if parseCapabilities.HasStructuredCloneSupport {
			parseStructuredClonePayload, parseStructuredCloneErr := BuildStructuredCloneSnapshotEnvelopeJSON(parseEnvelope)
			if parseStructuredCloneErr != nil {
				return "", nil, fmt.Errorf("runtime2: binary encode failed: %v; structured-clone fallback failed: %w", parseErr, parseStructuredCloneErr)
			}
			return TransportTierStructuredClone, parseStructuredClonePayload, nil
		}
		return "", nil, parseErr
	}
	parsePayload, parseErr := BuildStructuredCloneSnapshotEnvelopeJSON(parseEnvelope)
	if parseErr != nil {
		return "", nil, parseErr
	}
	return TransportTierStructuredClone, parsePayload, nil
}

// ParseSnapshotTransportPayloadWithFallback decodes one snapshot payload and downgrades to structured-clone when binary decode fails.
func ParseSnapshotTransportPayloadWithFallback(parseTransportTier TransportTier, parsePayload []byte) (TransportTier, SnapshotEnvelope, error) {
	switch parseTransportTier {
	case TransportTierBinary:
		parseEnvelope, parseErr := ParseBinarySnapshotEnvelope(parsePayload)
		if parseErr == nil {
			return TransportTierBinary, parseEnvelope, nil
		}
		parseStructuredCloneEnvelope, parseStructuredCloneErr := ParseStructuredCloneSnapshotEnvelopeJSON(parsePayload)
		if parseStructuredCloneErr != nil {
			return "", SnapshotEnvelope{}, fmt.Errorf("runtime2: binary decode failed: %v; structured-clone fallback failed: %w", parseErr, parseStructuredCloneErr)
		}
		return TransportTierStructuredClone, parseStructuredCloneEnvelope, nil
	case TransportTierStructuredClone:
		parseEnvelope, parseErr := ParseStructuredCloneSnapshotEnvelopeJSON(parsePayload)
		if parseErr != nil {
			return "", SnapshotEnvelope{}, parseErr
		}
		return TransportTierStructuredClone, parseEnvelope, nil
	default:
		return "", SnapshotEnvelope{}, fmt.Errorf("runtime2: transport tier %q is unsupported for snapshot decode", parseTransportTier)
	}
}
