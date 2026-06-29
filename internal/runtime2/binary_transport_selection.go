package runtime2

import (
	"encoding/binary"
	"fmt"
)

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
	if parseCapabilities.HasBinaryTransportSupport {
		parsePayload, parseErr := BuildBinarySnapshotEnvelope(parseEnvelope)
		if parseErr == nil {
			return TransportTierBinary, parsePayload, nil
		}
		if !parseCapabilities.HasStructuredCloneSupport {
			return "", nil, parseErr
		}
		parseStructuredClonePayload, parseStructuredCloneErr := BuildStructuredCloneSnapshotEnvelopeJSON(parseEnvelope)
		if parseStructuredCloneErr != nil {
			return "", nil, fmt.Errorf("runtime2: binary encode failed: %v; structured-clone fallback failed: %w", parseErr, parseStructuredCloneErr)
		}
		return TransportTierStructuredClone, parseStructuredClonePayload, nil
	}
	if parseCapabilities.HasStructuredCloneSupport {
		parsePayload, parseErr := BuildStructuredCloneSnapshotEnvelopeJSON(parseEnvelope)
		if parseErr != nil {
			return "", nil, parseErr
		}
		return TransportTierStructuredClone, parsePayload, nil
	}
	return "", nil, fmt.Errorf("runtime2: no supported snapshot transport tier is available")
}

// ParseSnapshotTransportPayloadWithFallback decodes one snapshot payload and downgrades to structured-clone when binary decode fails.
func ParseSnapshotTransportPayloadWithFallback(parseTransportTier TransportTier, parsePayload []byte) (TransportTier, SnapshotEnvelope, error) {
	switch parseTransportTier {
	case TransportTierBinary:
		if !hasSnapshotTransportBinaryHeader(parsePayload) {
			parseStructuredCloneEnvelope, parseStructuredCloneErr := ParseStructuredCloneSnapshotEnvelopeJSON(parsePayload)
			if parseStructuredCloneErr != nil {
				return "", SnapshotEnvelope{}, fmt.Errorf("runtime2: binary decode failed: runtime2: binary envelope header marker is invalid; structured-clone fallback failed: %w", parseStructuredCloneErr)
			}
			return TransportTierStructuredClone, parseStructuredCloneEnvelope, nil
		}
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

// hasSnapshotTransportBinaryHeader reports whether one payload has a snapshot-binary envelope header marker.
func hasSnapshotTransportBinaryHeader(parsePayload []byte) bool {
	if len(parsePayload) < binaryEnvelopeHeaderSize {
		return false
	}
	if parsePayload[0] != binaryEnvelopeHeaderMagic[0] ||
		parsePayload[1] != binaryEnvelopeHeaderMagic[1] ||
		parsePayload[2] != binaryEnvelopeHeaderMagic[2] ||
		parsePayload[3] != binaryEnvelopeHeaderMagic[3] {
		return false
	}
	if binary.LittleEndian.Uint16(parsePayload[4:6]) != binaryEnvelopeHeaderVersion {
		return false
	}
	if parsePayload[6] != byte(BinaryEnvelopeKindSnapshot) {
		return false
	}
	return parsePayload[7] == byte(binaryEnvelopeSectionCount)
}
