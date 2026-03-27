package runtime2

import "fmt"

const (
	// PatchStreamProtocolVersion is the supported patch-stream protocol version.
	PatchStreamProtocolVersion = "gwc.parallel.v1"
)

// PatchStreamHeaderRaw stores the wire-format header fields for one patch stream.
type PatchStreamHeaderRaw struct {
	ProtocolVersion string
	RegionID        string
	Epoch           uint64
	InputVersion    uint64
	PatchVersion    uint64
}

// PatchStreamHeader stores one validated patch-stream header.
type PatchStreamHeader struct {
	ProtocolVersion string
	RegionID        string
	Epoch           uint64
	InputVersion    uint64
	PatchVersion    uint64
}

// ParsePatchStreamHeader decodes and validates one patch-stream header.
func ParsePatchStreamHeader(parseRaw PatchStreamHeaderRaw, parseExpectedRegionID string) (PatchStreamHeader, error) {
	if parseRaw.ProtocolVersion != PatchStreamProtocolVersion {
		return PatchStreamHeader{}, fmt.Errorf("runtime2: patch stream version %q is unsupported", parseRaw.ProtocolVersion)
	}
	if parseRaw.RegionID == "" {
		return PatchStreamHeader{}, fmt.Errorf("runtime2: patch stream region id is required")
	}
	if parseExpectedRegionID != "" && parseRaw.RegionID != parseExpectedRegionID {
		return PatchStreamHeader{}, fmt.Errorf("runtime2: patch stream region id %q does not match expected %q", parseRaw.RegionID, parseExpectedRegionID)
	}
	return PatchStreamHeader{
		ProtocolVersion: parseRaw.ProtocolVersion,
		RegionID:        parseRaw.RegionID,
		Epoch:           parseRaw.Epoch,
		InputVersion:    parseRaw.InputVersion,
		PatchVersion:    parseRaw.PatchVersion,
	}, nil
}
