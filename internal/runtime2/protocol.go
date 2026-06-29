package runtime2

import (
	"encoding/json"
	"fmt"
)

// ProtocolVersion identifies the multithreaded runtime wire-contract version.
type ProtocolVersion string

const (
	// ProtocolVersionParallelV1 is the first multithreaded runtime protocol version.
	ProtocolVersionParallelV1 ProtocolVersion = "gwc.parallel.v1"
)

// CapabilityFixture stores a deterministic capability-negotiation fixture for tests.
type CapabilityFixture struct {
	ProtocolVersion ProtocolVersion  `json:"protocol_version"`
	Capabilities    CapabilityReport `json:"capabilities"`
}

// ParseProtocolVersion validates and normalizes a protocol version string.
func ParseProtocolVersion(parseRaw string) (ProtocolVersion, error) {
	parseVersion := ProtocolVersion(parseRaw)
	switch parseVersion {
	case ProtocolVersionParallelV1:
		return parseVersion, nil
	case "":
		return "", fmt.Errorf("runtime2: protocol version is required")
	default:
		return "", fmt.Errorf("runtime2: protocol version %q is unsupported", parseRaw)
	}
}

// ValidateProtocolVersionMatch verifies two protocol versions are compatible.
func ValidateProtocolVersionMatch(parseExpected ProtocolVersion, parseActual ProtocolVersion) error {
	if parseExpected == "" {
		return fmt.Errorf("runtime2: expected protocol version is required")
	}
	if parseActual == "" {
		return fmt.Errorf("runtime2: actual protocol version is required")
	}
	if parseExpected != parseActual {
		return fmt.Errorf("runtime2: protocol version mismatch expected=%q actual=%q", parseExpected, parseActual)
	}
	return nil
}

// ValidateCapabilityFixture verifies a decoded capability-negotiation fixture is complete.
func ValidateCapabilityFixture(parseFixture CapabilityFixture) error {
	if _, parseErr := ParseProtocolVersion(string(parseFixture.ProtocolVersion)); parseErr != nil {
		return parseErr
	}
	if !parseFixture.Capabilities.HasWorkerSupport &&
		(parseFixture.Capabilities.HasMessagePortSupport ||
			parseFixture.Capabilities.HasStructuredCloneSupport ||
			parseFixture.Capabilities.HasBinaryTransportSupport ||
			parseFixture.Capabilities.HasSharedBufferSupport ||
			parseFixture.Capabilities.HasSharedMemoryTransportSupport) {
		return fmt.Errorf("runtime2: capability fixture requires worker support before dependent features")
	}
	return nil
}

// BuildCapabilityFixtureJSON encodes a validated capability-negotiation fixture.
func BuildCapabilityFixtureJSON(parseFixture CapabilityFixture) ([]byte, error) {
	if parseErr := ValidateCapabilityFixture(parseFixture); parseErr != nil {
		return nil, parseErr
	}
	return json.Marshal(parseFixture)
}

// ParseCapabilityFixtureJSON decodes and validates a capability-negotiation fixture.
func ParseCapabilityFixtureJSON(parseValue []byte) (CapabilityFixture, error) {
	var parseEnvelope map[string]json.RawMessage
	if parseErr := json.Unmarshal(parseValue, &parseEnvelope); parseErr != nil {
		return CapabilityFixture{}, fmt.Errorf("runtime2: decode capability fixture envelope: %w", parseErr)
	}
	if _, parseHasProtocolVersion := parseEnvelope["protocol_version"]; !parseHasProtocolVersion {
		return CapabilityFixture{}, fmt.Errorf("runtime2: capability fixture missing protocol_version")
	}
	if _, parseHasCapabilities := parseEnvelope["capabilities"]; !parseHasCapabilities {
		return CapabilityFixture{}, fmt.Errorf("runtime2: capability fixture missing capabilities")
	}
	var parseFixture CapabilityFixture
	if parseErr := json.Unmarshal(parseValue, &parseFixture); parseErr != nil {
		return CapabilityFixture{}, fmt.Errorf("runtime2: decode capability fixture: %w", parseErr)
	}
	if parseErr := ValidateCapabilityFixture(parseFixture); parseErr != nil {
		return CapabilityFixture{}, parseErr
	}
	return parseFixture, nil
}
