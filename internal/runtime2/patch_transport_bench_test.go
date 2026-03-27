package runtime2

import "testing"

// buildPatchTransportBenchmarkPatchStream builds one minimal valid patch stream for transport microbenchmarks.
func buildPatchTransportBenchmarkPatchStream(parseB *testing.B) PatchStreamRaw {
	parseB.Helper()
	return PatchStreamRaw{
		GetHeader: PatchStreamHeaderRaw{
			ProtocolVersion: PatchStreamProtocolVersion,
			RegionID:        "region-1",
			Epoch:           1,
			InputVersion:    2,
			PatchVersion:    2,
		},
	}
}

// buildPatchTransportBenchmarkSharedPayload builds one structured-clone patch-envelope payload for shared-page publish or read benchmarks.
func buildPatchTransportBenchmarkSharedPayload(parseB *testing.B) []byte {
	parseB.Helper()
	parsePatchStream := buildPatchTransportBenchmarkPatchStream(parseB)
	parsePatchPayload, parsePatchPayloadErr := BuildStructuredClonePatchPayloadJSON(parsePatchStream)
	if parsePatchPayloadErr != nil {
		parseB.Fatalf("BuildStructuredClonePatchPayloadJSON returned error: %v", parsePatchPayloadErr)
	}
	parseEnvelopePayload, parseEnvelopePayloadErr := BuildStructuredClonePatchEnvelopeJSON(StructuredClonePatchEnvelope{
		RegionInstanceID: "region-1",
		Epoch:            1,
		PatchVersion:     parsePatchStream.GetHeader.PatchVersion,
		PatchPayload:     parsePatchPayload,
	})
	if parseEnvelopePayloadErr != nil {
		parseB.Fatalf("BuildStructuredClonePatchEnvelopeJSON returned error: %v", parseEnvelopePayloadErr)
	}
	return parseEnvelopePayload
}

// BenchmarkBuildBinaryPatchPayload measures binary patch payload encoding cost.
func BenchmarkBuildBinaryPatchPayload(parseB *testing.B) {
	parsePatchStream := buildPatchTransportBenchmarkPatchStream(parseB)
	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
		if _, parsePayloadErr := BuildBinaryPatchPayload(parsePatchStream); parsePayloadErr != nil {
			parseB.Fatalf("BuildBinaryPatchPayload returned error: %v", parsePayloadErr)
		}
	}
}

// BenchmarkParseBinaryPatchPayload measures binary patch payload decode cost.
func BenchmarkParseBinaryPatchPayload(parseB *testing.B) {
	parsePatchPayload, parsePatchPayloadErr := BuildBinaryPatchPayload(buildPatchTransportBenchmarkPatchStream(parseB))
	if parsePatchPayloadErr != nil {
		parseB.Fatalf("BuildBinaryPatchPayload(seed) returned error: %v", parsePatchPayloadErr)
	}
	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
		if _, parseDecodeErr := ParseBinaryPatchPayload(parsePatchPayload); parseDecodeErr != nil {
			parseB.Fatalf("ParseBinaryPatchPayload returned error: %v", parseDecodeErr)
		}
	}
}

// BenchmarkHandleSharedPatchPublishPayload measures shared-page patch publication cost.
func BenchmarkHandleSharedPatchPublishPayload(parseB *testing.B) {
	parseSharedPatchPage, parseSharedPatchPageErr := BuildSharedPatchPage(64 * 1024)
	if parseSharedPatchPageErr != nil {
		parseB.Fatalf("BuildSharedPatchPage returned error: %v", parseSharedPatchPageErr)
	}
	parsePayload := buildPatchTransportBenchmarkSharedPayload(parseB)
	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
		if _, parsePublishErr := parseSharedPatchPage.HandleSharedPatchPublishPayload(parsePayload); parsePublishErr != nil {
			parseB.Fatalf("HandleSharedPatchPublishPayload returned error: %v", parsePublishErr)
		}
	}
}

// BenchmarkGetSharedPatchReadPayload measures shared-page patch payload read cost.
func BenchmarkGetSharedPatchReadPayload(parseB *testing.B) {
	parseSharedPatchPage, parseSharedPatchPageErr := BuildSharedPatchPage(64 * 1024)
	if parseSharedPatchPageErr != nil {
		parseB.Fatalf("BuildSharedPatchPage returned error: %v", parseSharedPatchPageErr)
	}
	parsePayload := buildPatchTransportBenchmarkSharedPayload(parseB)
	if _, parsePublishErr := parseSharedPatchPage.HandleSharedPatchPublishPayload(parsePayload); parsePublishErr != nil {
		parseB.Fatalf("HandleSharedPatchPublishPayload(seed) returned error: %v", parsePublishErr)
	}
	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
		if _, parseReadErr := parseSharedPatchPage.GetSharedPatchReadPayload(); parseReadErr != nil {
			parseB.Fatalf("GetSharedPatchReadPayload returned error: %v", parseReadErr)
		}
	}
}
