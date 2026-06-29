package runtime2

import (
	"encoding/json"
	"fmt"
	"testing"
)

var storeHostPatchStructuredParseBenchmarkSink PatchStreamRaw

// buildHostPatchStructuredParseBenchFixture stores one reusable structured-clone patch parse benchmark fixture.
type buildHostPatchStructuredParseBenchFixture struct {
	getRawPatchPayload        []byte
	getStructuredEnvelopeJSON []byte
}

// buildHostPatchStructuredParseBenchmarkFixture builds one fixture for raw and envelope structured-clone patch parse benchmarks.
func buildHostPatchStructuredParseBenchmarkFixture(parseB *testing.B) buildHostPatchStructuredParseBenchFixture {
	parseB.Helper()
	parsePatchStream := buildPatchTransportBenchmarkPatchStream(parseB)
	parseRawPatchPayload, parseRawPatchPayloadErr := BuildStructuredClonePatchPayloadJSON(parsePatchStream)
	if parseRawPatchPayloadErr != nil {
		parseB.Fatalf("BuildStructuredClonePatchPayloadJSON returned error: %v", parseRawPatchPayloadErr)
	}
	parseStructuredEnvelopeJSON, parseStructuredEnvelopeJSONErr := BuildStructuredClonePatchEnvelopeJSON(StructuredClonePatchEnvelope{
		RegionInstanceID: RegionInstanceID(parsePatchStream.GetHeader.RegionID),
		Epoch:            parsePatchStream.GetHeader.Epoch,
		PatchVersion:     parsePatchStream.GetHeader.PatchVersion,
		PatchPayload:     parseRawPatchPayload,
	})
	if parseStructuredEnvelopeJSONErr != nil {
		parseB.Fatalf("BuildStructuredClonePatchEnvelopeJSON returned error: %v", parseStructuredEnvelopeJSONErr)
	}
	return buildHostPatchStructuredParseBenchFixture{
		getRawPatchPayload:        parseRawPatchPayload,
		getStructuredEnvelopeJSON: parseStructuredEnvelopeJSON,
	}
}

// parsePatchStreamFromStructuredClonePayloadLegacy preserves the previous decode order that always attempted envelope decode before raw patch decode.
func parsePatchStreamFromStructuredClonePayloadLegacy(parsePayload []byte) (PatchStreamRaw, error) {
	parseStructuredEnvelope, parseStructuredEnvelopeErr := ParseStructuredClonePatchEnvelopeJSON(parsePayload)
	if parseStructuredEnvelopeErr == nil {
		var parsePatchStream PatchStreamRaw
		if parsePatchStreamErr := json.Unmarshal(parseStructuredEnvelope.PatchPayload, &parsePatchStream); parsePatchStreamErr != nil {
			return PatchStreamRaw{}, fmt.Errorf("runtime2: decode structured-clone patch envelope payload: %w", parsePatchStreamErr)
		}
		if _, parseHeaderErr := ParsePatchStreamHeader(parsePatchStream.GetHeader, parsePatchStream.GetHeader.RegionID); parseHeaderErr != nil {
			return PatchStreamRaw{}, parseHeaderErr
		}
		if parsePatchStream.GetHeader.RegionID != string(parseStructuredEnvelope.RegionInstanceID) {
			return PatchStreamRaw{}, fmt.Errorf("runtime2: structured-clone patch envelope region mismatch")
		}
		if parsePatchStream.GetHeader.PatchVersion != parseStructuredEnvelope.PatchVersion {
			return PatchStreamRaw{}, fmt.Errorf("runtime2: structured-clone patch envelope patch version mismatch")
		}
		return parsePatchStream, nil
	}
	var parsePatchStream PatchStreamRaw
	if parsePatchStreamErr := json.Unmarshal(parsePayload, &parsePatchStream); parsePatchStreamErr != nil {
		return PatchStreamRaw{}, fmt.Errorf("runtime2: decode structured-clone patch payload: %w", parsePatchStreamErr)
	}
	if _, parseHeaderErr := ParsePatchStreamHeader(parsePatchStream.GetHeader, parsePatchStream.GetHeader.RegionID); parseHeaderErr != nil {
		return PatchStreamRaw{}, parseHeaderErr
	}
	return parsePatchStream, nil
}

// BenchmarkParsePatchStreamFromStructuredClonePayloadCurrentVsLegacy compares current payload-shape-gated parse against the previous always-envelope-first parse order.
func BenchmarkParsePatchStreamFromStructuredClonePayloadCurrentVsLegacy(parseB *testing.B) {
	getFixture := buildHostPatchStructuredParseBenchmarkFixture(parseB)
	parseB.Run("legacy_raw_payload", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			parsePatchStream, parsePatchStreamErr := parsePatchStreamFromStructuredClonePayloadLegacy(getFixture.getRawPatchPayload)
			if parsePatchStreamErr != nil {
				parseB.Fatalf("parsePatchStreamFromStructuredClonePayloadLegacy returned error: %v", parsePatchStreamErr)
			}
			storeHostPatchStructuredParseBenchmarkSink = parsePatchStream
		}
	})
	parseB.Run("current_raw_payload", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			parsePatchStream, parsePatchStreamErr := parsePatchStreamFromStructuredClonePayload(getFixture.getRawPatchPayload)
			if parsePatchStreamErr != nil {
				parseB.Fatalf("parsePatchStreamFromStructuredClonePayload returned error: %v", parsePatchStreamErr)
			}
			storeHostPatchStructuredParseBenchmarkSink = parsePatchStream
		}
	})
	parseB.Run("legacy_envelope_payload", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			parsePatchStream, parsePatchStreamErr := parsePatchStreamFromStructuredClonePayloadLegacy(getFixture.getStructuredEnvelopeJSON)
			if parsePatchStreamErr != nil {
				parseB.Fatalf("parsePatchStreamFromStructuredClonePayloadLegacy returned error: %v", parsePatchStreamErr)
			}
			storeHostPatchStructuredParseBenchmarkSink = parsePatchStream
		}
	})
	parseB.Run("current_envelope_payload", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			parsePatchStream, parsePatchStreamErr := parsePatchStreamFromStructuredClonePayload(getFixture.getStructuredEnvelopeJSON)
			if parsePatchStreamErr != nil {
				parseB.Fatalf("parsePatchStreamFromStructuredClonePayload returned error: %v", parsePatchStreamErr)
			}
			storeHostPatchStructuredParseBenchmarkSink = parsePatchStream
		}
	})
}
