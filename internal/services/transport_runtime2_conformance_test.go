package services_test

import (
	"encoding/json"
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime2"
	"github.com/monstercameron/GoWebComponents/v4/internal/services"
)

// v5 P3.3 — runtime2's real payload types must satisfy the extracted substrate.
//
// This is what makes internal/services an EXTRACTION rather than a second
// implementation of the same idea. If runtime2's encoders cannot drive it, the
// substrate has invented its own contract and the two will drift — which is the
// usual way "extractions" quietly become forks.
//
// External test package on purpose: it uses only what a real consumer can see.

// patchStreamCodec adapts runtime2's patch-stream encoders to services.Codec.
//
// It is an adapter, not a reimplementation: every method delegates. If
// runtime2's encoding changes, this follows automatically.
type patchStreamCodec struct{}

func (patchStreamCodec) EncodeBinary(parseStream runtime2.PatchStreamRaw) ([]byte, error) {
	return runtime2.BuildBinaryPatchPayload(parseStream)
}

// EncodeJSON uses the type's own JSON tags. runtime2's structured-clone
// ENVELOPE carries already-encoded bytes rather than the stream itself, so the
// structured-clone tier for this payload is the stream's own JSON form.
func (patchStreamCodec) EncodeJSON(parseStream runtime2.PatchStreamRaw) ([]byte, error) {
	return json.Marshal(parseStream)
}

func (patchStreamCodec) DecodeBinary(parseBytes []byte) (runtime2.PatchStreamRaw, error) {
	return runtime2.ParseBinaryPatchPayload(parseBytes)
}

func (patchStreamCodec) DecodeJSON(parseBytes []byte) (runtime2.PatchStreamRaw, error) {
	var parseStream runtime2.PatchStreamRaw
	return parseStream, json.Unmarshal(parseBytes, &parseStream)
}

// buildConformancePatchStream builds a minimal but non-empty patch stream.
func buildConformancePatchStream() runtime2.PatchStreamRaw {
	return runtime2.PatchStreamRaw{
		GetHeader: runtime2.PatchStreamHeaderRaw{
			ProtocolVersion: runtime2.PatchStreamProtocolVersion,
			RegionID:        "conformance-region",
			Epoch:           1,
			InputVersion:    1,
			PatchVersion:    2,
		},
	}
}

// TestRuntime2PatchStreamSatisfiesCodec is a compile-time-ish assertion: if the
// adapter does not satisfy services.Codec, this file does not build.
func TestRuntime2PatchStreamSatisfiesCodec(parseT *testing.T) {
	var parseCodec services.Codec[runtime2.PatchStreamRaw] = patchStreamCodec{}
	if parseCodec == nil {
		parseT.Fatal("runtime2's patch stream must satisfy the extracted codec interface")
	}
}

// TestRuntime2PatchStreamRoundTripsThroughSubstrate drives runtime2's real
// encoders through the generic Encode/Decode path at each reachable tier.
func TestRuntime2PatchStreamRoundTripsThroughSubstrate(parseT *testing.T) {
	parseStream := buildConformancePatchStream()

	for _, parseCase := range []struct {
		label        string
		capabilities services.Capabilities
		wantTier     services.Tier
	}{
		{"binary preferred", services.Capabilities{HasBinary: true, HasStructuredClone: true}, services.TierBinary},
		{"structured-clone only", services.Capabilities{HasStructuredClone: true}, services.TierStructuredClone},
	} {
		parseDecoded, parseTier, parseErr := services.RoundTrip(parseStream, patchStreamCodec{}, parseCase.capabilities)
		if parseErr != nil {
			parseT.Errorf("%s: RoundTrip: %v", parseCase.label, parseErr)
			continue
		}
		if parseTier != parseCase.wantTier {
			parseT.Errorf("%s: tier = %q, want %q", parseCase.label, parseTier, parseCase.wantTier)
		}
		if parseDecoded.GetHeader.RegionID != parseStream.GetHeader.RegionID {
			parseT.Errorf("%s: region = %q, want %q", parseCase.label,
				parseDecoded.GetHeader.RegionID, parseStream.GetHeader.RegionID)
		}
		if parseDecoded.GetHeader.PatchVersion != parseStream.GetHeader.PatchVersion {
			parseT.Errorf("%s: patch version = %d, want %d", parseCase.label,
				parseDecoded.GetHeader.PatchVersion, parseStream.GetHeader.PatchVersion)
		}
	}
}

// TestSubstrateSelectionMatchesRuntime2Selection pins that the extracted tier
// policy agrees with the one it was extracted from.
//
// Divergence here would be the quiet kind: both would work, they would simply
// choose differently, and a payload encoded by one path would be decoded by the
// other's assumption.
func TestSubstrateSelectionMatchesRuntime2Selection(parseT *testing.T) {
	for _, parseCase := range []struct {
		label   string
		runtime runtime2.CapabilityReport
		service services.Capabilities
	}{
		{
			label:   "binary and structured clone",
			runtime: runtime2.CapabilityReport{HasBinaryTransportSupport: true, HasStructuredCloneSupport: true},
			service: services.Capabilities{HasBinary: true, HasStructuredClone: true},
		},
		{
			label:   "structured clone only",
			runtime: runtime2.CapabilityReport{HasStructuredCloneSupport: true},
			service: services.Capabilities{HasStructuredClone: true},
		},
	} {
		parseRuntimeTier, parseRuntimeErr := runtime2.SelectSnapshotTransportTier(parseCase.runtime)
		parseServiceTier, parseServiceErr := services.SelectTier(parseCase.service)

		if (parseRuntimeErr == nil) != (parseServiceErr == nil) {
			parseT.Errorf("%s: runtime2 err=%v but services err=%v", parseCase.label, parseRuntimeErr, parseServiceErr)
			continue
		}
		if parseRuntimeErr != nil {
			continue
		}
		if string(parseRuntimeTier) != string(parseServiceTier) {
			parseT.Errorf("%s: runtime2 chose %q, services chose %q — the extracted policy diverged from its source",
				parseCase.label, parseRuntimeTier, parseServiceTier)
		}
	}

	// Both must refuse when nothing is supported.
	if _, parseErr := runtime2.SelectSnapshotTransportTier(runtime2.CapabilityReport{}); parseErr == nil {
		parseT.Error("runtime2 accepted an empty capability report")
	}
	if _, parseErr := services.SelectTier(services.Capabilities{}); parseErr == nil {
		parseT.Error("services accepted an empty capability report")
	}
}
