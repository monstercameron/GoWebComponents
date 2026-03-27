package runtime2

import (
	"fmt"
	"testing"
)

var storeHostPatchTransportParseBenchmarkTier TransportTier
var storeHostPatchTransportParseBenchmarkRaw PatchStreamRaw

// buildHostPatchTransportParseBenchmarkFixture builds one reusable patch payload fixture for host fallback-parse compare benchmarks.
func buildHostPatchTransportParseBenchmarkFixture(parseB *testing.B) (ControlEnvelope, []byte, []byte, *SharedPatchPage) {
	parseB.Helper()
	buildPatchStream := PatchStreamRaw{
		GetHeader: PatchStreamHeaderRaw{
			ProtocolVersion: PatchStreamProtocolVersion,
			RegionID:        "bench-region",
			Epoch:           1,
			InputVersion:    2,
			PatchVersion:    2,
		},
		GetPatchIdentity: "bench-id",
	}
	buildBinaryPayload, parseBinaryPayloadErr := BuildBinaryPatchPayload(buildPatchStream)
	if parseBinaryPayloadErr != nil {
		parseB.Fatalf("BuildBinaryPatchPayload returned error: %v", parseBinaryPayloadErr)
	}
	buildStructuredPayload, parseStructuredPayloadErr := BuildStructuredClonePatchPayloadJSON(buildPatchStream)
	if parseStructuredPayloadErr != nil {
		parseB.Fatalf("BuildStructuredClonePatchPayloadJSON returned error: %v", parseStructuredPayloadErr)
	}
	buildStructuredEnvelopePayload, parseStructuredEnvelopePayloadErr := BuildStructuredClonePatchEnvelopeJSON(StructuredClonePatchEnvelope{
		RegionInstanceID: RegionInstanceID(buildPatchStream.GetHeader.RegionID),
		Epoch:            buildPatchStream.GetHeader.Epoch,
		PatchVersion:     buildPatchStream.GetHeader.PatchVersion,
		PatchPayload:     buildStructuredPayload,
	})
	if parseStructuredEnvelopePayloadErr != nil {
		parseB.Fatalf("BuildStructuredClonePatchEnvelopeJSON returned error: %v", parseStructuredEnvelopePayloadErr)
	}
	buildSharedPatchPage, parseSharedPatchPageErr := BuildSharedPatchPage(64 * 1024)
	if parseSharedPatchPageErr != nil {
		parseB.Fatalf("BuildSharedPatchPage returned error: %v", parseSharedPatchPageErr)
	}
	if _, parsePublishErr := buildSharedPatchPage.HandleSharedPatchPublishPayload(buildStructuredEnvelopePayload); parsePublishErr != nil {
		parseB.Fatalf("HandleSharedPatchPublishPayload returned error: %v", parsePublishErr)
	}
	buildPatchReadyEnvelope, parsePatchReadyEnvelopeErr := BuildControlPatchReadyEnvelope(
		RegionInstanceID(buildPatchStream.GetHeader.RegionID),
		buildPatchStream.GetHeader.PatchVersion,
		buildPatchStream.GetHeader.InputVersion,
		TransportTierBinary,
	)
	if parsePatchReadyEnvelopeErr != nil {
		parseB.Fatalf("BuildControlPatchReadyEnvelope returned error: %v", parsePatchReadyEnvelopeErr)
	}
	return buildPatchReadyEnvelope, buildBinaryPayload, buildStructuredEnvelopePayload, buildSharedPatchPage
}

// parseHostPatchPayloadWithFallbackLegacyBenchmark preserves the previous fallback parser path that revalidated parsed patch headers.
func parseHostPatchPayloadWithFallbackLegacyBenchmark(
	parsePatchReadyEnvelope ControlEnvelope,
	parseMessagePayload []byte,
	parseSharedPatchPage *SharedPatchPage,
) (TransportTier, PatchStreamRaw, error) {
	if parsePatchReadyEnvelope.Kind != ControlKindPatchReady {
		return "", PatchStreamRaw{}, fmt.Errorf("runtime2: patch-ready control envelope is required")
	}
	if parseEnvelopeErr := ValidateControlEnvelope(parsePatchReadyEnvelope); parseEnvelopeErr != nil {
		return "", PatchStreamRaw{}, parseEnvelopeErr
	}
	parseExpectedRegionID := string(parsePatchReadyEnvelope.RegionInstanceID)
	parseExpectedPatchVersion := parsePatchReadyEnvelope.PatchVersion
	parseValidatePatchStream := func(parseTier TransportTier, parsePatchStream PatchStreamRaw) (TransportTier, PatchStreamRaw, error) {
		parseHeader, parseHeaderErr := ParsePatchStreamHeader(parsePatchStream.GetHeader, parsePatchStream.GetHeader.RegionID)
		if parseHeaderErr != nil {
			return "", PatchStreamRaw{}, parseHeaderErr
		}
		if parseHeader.RegionID != parseExpectedRegionID {
			return "", PatchStreamRaw{}, fmt.Errorf("runtime2: patch payload region mismatch expected=%q actual=%q", parseExpectedRegionID, parseHeader.RegionID)
		}
		if parseHeader.PatchVersion != parseExpectedPatchVersion {
			return "", PatchStreamRaw{}, fmt.Errorf("runtime2: patch payload version mismatch expected=%d actual=%d", parseExpectedPatchVersion, parseHeader.PatchVersion)
		}
		return parseTier, parsePatchStream, nil
	}
	switch parsePatchReadyEnvelope.TransportTier {
	case TransportTierSharedBuffer:
		if parseSharedPatchPage != nil {
			parseSharedEnvelope, parseSharedEnvelopeErr := ParseSharedPatchPayloadFromPage(parseSharedPatchPage)
			if parseSharedEnvelopeErr == nil {
				parseStructuredPatchStream, parseStructuredPatchStreamErr := parsePatchStreamFromStructuredClonePayload(parseSharedEnvelope.PatchPayload)
				if parseStructuredPatchStreamErr == nil {
					return parseValidatePatchStream(TransportTierSharedBuffer, parseStructuredPatchStream)
				}
			}
		}
		parseBinaryPatchStream, parseBinaryPatchStreamErr := ParseBinaryPatchPayload(parseMessagePayload)
		if parseBinaryPatchStreamErr == nil {
			return parseValidatePatchStream(TransportTierBinary, parseBinaryPatchStream)
		}
		parseStructuredPatchStream, parseStructuredPatchStreamErr := parsePatchStreamFromStructuredClonePayload(parseMessagePayload)
		if parseStructuredPatchStreamErr != nil {
			return "", PatchStreamRaw{}, fmt.Errorf("runtime2: shared patch decode failed and fallback decode failed: %v; %w", parseBinaryPatchStreamErr, parseStructuredPatchStreamErr)
		}
		return parseValidatePatchStream(TransportTierStructuredClone, parseStructuredPatchStream)
	case TransportTierBinary:
		parseBinaryPatchStream, parseBinaryPatchStreamErr := ParseBinaryPatchPayload(parseMessagePayload)
		if parseBinaryPatchStreamErr == nil {
			return parseValidatePatchStream(TransportTierBinary, parseBinaryPatchStream)
		}
		parseStructuredPatchStream, parseStructuredPatchStreamErr := parsePatchStreamFromStructuredClonePayload(parseMessagePayload)
		if parseStructuredPatchStreamErr != nil {
			return "", PatchStreamRaw{}, fmt.Errorf("runtime2: binary patch decode failed and structured fallback failed: %v; %w", parseBinaryPatchStreamErr, parseStructuredPatchStreamErr)
		}
		return parseValidatePatchStream(TransportTierStructuredClone, parseStructuredPatchStream)
	case TransportTierStructuredClone:
		parseStructuredPatchStream, parseStructuredPatchStreamErr := parsePatchStreamFromStructuredClonePayload(parseMessagePayload)
		if parseStructuredPatchStreamErr != nil {
			return "", PatchStreamRaw{}, parseStructuredPatchStreamErr
		}
		return parseValidatePatchStream(TransportTierStructuredClone, parseStructuredPatchStream)
	default:
		return "", PatchStreamRaw{}, fmt.Errorf("runtime2: patch-ready transport tier %q is unsupported for host patch parse", parsePatchReadyEnvelope.TransportTier)
	}
}

// BenchmarkParseHostPatchPayloadWithFallbackCurrentVsLegacy compares current host fallback parse path against legacy revalidation behavior.
func BenchmarkParseHostPatchPayloadWithFallbackCurrentVsLegacy(parseB *testing.B) {
	buildPatchReadyEnvelope, buildBinaryPayload, buildStructuredEnvelopePayload, buildSharedPatchPage := buildHostPatchTransportParseBenchmarkFixture(parseB)

	parseB.Run("binary_primary/legacy", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			getTier, getPatchStream, parseErr := parseHostPatchPayloadWithFallbackLegacyBenchmark(
				buildPatchReadyEnvelope,
				buildBinaryPayload,
				nil,
			)
			if parseErr != nil {
				parseB.Fatalf("parseHostPatchPayloadWithFallbackLegacyBenchmark(binary_primary) returned error: %v", parseErr)
			}
			storeHostPatchTransportParseBenchmarkTier = getTier
			storeHostPatchTransportParseBenchmarkRaw = getPatchStream
		}
	})
	parseB.Run("binary_primary/current", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			getTier, getPatchStream, parseErr := ParseHostPatchPayloadWithFallback(
				buildPatchReadyEnvelope,
				buildBinaryPayload,
				nil,
			)
			if parseErr != nil {
				parseB.Fatalf("ParseHostPatchPayloadWithFallback(binary_primary) returned error: %v", parseErr)
			}
			storeHostPatchTransportParseBenchmarkTier = getTier
			storeHostPatchTransportParseBenchmarkRaw = getPatchStream
		}
	})

	parseB.Run("binary_fallback_structured/legacy", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			getTier, getPatchStream, parseErr := parseHostPatchPayloadWithFallbackLegacyBenchmark(
				buildPatchReadyEnvelope,
				buildStructuredEnvelopePayload,
				nil,
			)
			if parseErr != nil {
				parseB.Fatalf("parseHostPatchPayloadWithFallbackLegacyBenchmark(binary_fallback_structured) returned error: %v", parseErr)
			}
			storeHostPatchTransportParseBenchmarkTier = getTier
			storeHostPatchTransportParseBenchmarkRaw = getPatchStream
		}
	})
	parseB.Run("binary_fallback_structured/current", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			getTier, getPatchStream, parseErr := ParseHostPatchPayloadWithFallback(
				buildPatchReadyEnvelope,
				buildStructuredEnvelopePayload,
				nil,
			)
			if parseErr != nil {
				parseB.Fatalf("ParseHostPatchPayloadWithFallback(binary_fallback_structured) returned error: %v", parseErr)
			}
			storeHostPatchTransportParseBenchmarkTier = getTier
			storeHostPatchTransportParseBenchmarkRaw = getPatchStream
		}
	})

	parseSharedPatchReadyEnvelope := buildPatchReadyEnvelope
	parseSharedPatchReadyEnvelope.TransportTier = TransportTierSharedBuffer
	parseB.Run("shared_buffer_fallback_structured/legacy", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			getTier, getPatchStream, parseErr := parseHostPatchPayloadWithFallbackLegacyBenchmark(
				parseSharedPatchReadyEnvelope,
				buildStructuredEnvelopePayload,
				nil,
			)
			if parseErr != nil {
				parseB.Fatalf("parseHostPatchPayloadWithFallbackLegacyBenchmark(shared_buffer_fallback_structured) returned error: %v", parseErr)
			}
			storeHostPatchTransportParseBenchmarkTier = getTier
			storeHostPatchTransportParseBenchmarkRaw = getPatchStream
		}
	})
	parseB.Run("shared_buffer_fallback_structured/current", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			getTier, getPatchStream, parseErr := ParseHostPatchPayloadWithFallback(
				parseSharedPatchReadyEnvelope,
				buildStructuredEnvelopePayload,
				nil,
			)
			if parseErr != nil {
				parseB.Fatalf("ParseHostPatchPayloadWithFallback(shared_buffer_fallback_structured) returned error: %v", parseErr)
			}
			storeHostPatchTransportParseBenchmarkTier = getTier
			storeHostPatchTransportParseBenchmarkRaw = getPatchStream
		}
	})
	parseB.Run("shared_buffer_primary/legacy", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			getTier, getPatchStream, parseErr := parseHostPatchPayloadWithFallbackLegacyBenchmark(
				parseSharedPatchReadyEnvelope,
				nil,
				buildSharedPatchPage,
			)
			if parseErr != nil {
				parseB.Fatalf("parseHostPatchPayloadWithFallbackLegacyBenchmark(shared_buffer_primary) returned error: %v", parseErr)
			}
			storeHostPatchTransportParseBenchmarkTier = getTier
			storeHostPatchTransportParseBenchmarkRaw = getPatchStream
		}
	})
	parseB.Run("shared_buffer_primary/current", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			getTier, getPatchStream, parseErr := ParseHostPatchPayloadWithFallback(
				parseSharedPatchReadyEnvelope,
				nil,
				buildSharedPatchPage,
			)
			if parseErr != nil {
				parseB.Fatalf("ParseHostPatchPayloadWithFallback(shared_buffer_primary) returned error: %v", parseErr)
			}
			storeHostPatchTransportParseBenchmarkTier = getTier
			storeHostPatchTransportParseBenchmarkRaw = getPatchStream
		}
	})
}
