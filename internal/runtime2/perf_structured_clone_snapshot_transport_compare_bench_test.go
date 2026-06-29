package runtime2

import (
	"encoding/json"
	"fmt"
	"testing"
)

var storeStructuredCloneSnapshotBenchmarkPayloadSink []byte
var storeStructuredCloneSnapshotBenchmarkEnvelopeSink SnapshotEnvelope

// buildStructuredCloneSnapshotEnvelopeJSONLegacyBenchmark preserves the legacy multi-step snapshot envelope encoding path for compare benchmarks.
func buildStructuredCloneSnapshotEnvelopeJSONLegacyBenchmark(parseEnvelope SnapshotEnvelope) ([]byte, error) {
	if parseErr := ValidateSnapshotEnvelope(parseEnvelope); parseErr != nil {
		return nil, parseErr
	}
	var buildPropsPayload []byte
	if parseEnvelope.Props != nil {
		parsePropsPayload, parsePropsPayloadErr := json.Marshal(parseEnvelope.Props)
		if parsePropsPayloadErr != nil {
			return nil, fmt.Errorf("runtime2: encode structured-clone snapshot props: %w", parsePropsPayloadErr)
		}
		buildPropsPayload = parsePropsPayload
	}
	var buildSourcesPayload []byte
	if len(parseEnvelope.Sources) > 0 {
		parseSourcesPayload, parseSourcesPayloadErr := json.Marshal(parseEnvelope.Sources)
		if parseSourcesPayloadErr != nil {
			return nil, fmt.Errorf("runtime2: encode structured-clone snapshot sources: %w", parseSourcesPayloadErr)
		}
		buildSourcesPayload = parseSourcesPayload
	}
	buildPayload := make(
		[]byte,
		0,
		getStructuredCloneSnapshotEnvelopeJSONLength(parseEnvelope, len(buildPropsPayload), len(buildSourcesPayload)),
	)
	buildPayload = appendStructuredCloneSnapshotEnvelopeJSON(buildPayload, parseEnvelope, buildPropsPayload, buildSourcesPayload)
	return buildPayload, nil
}

// parseStructuredCloneSnapshotEnvelopeJSONLegacyBenchmark preserves the legacy decode path that always attempted header-only parse before full decode.
func parseStructuredCloneSnapshotEnvelopeJSONLegacyBenchmark(parsePayload []byte) (SnapshotEnvelope, error) {
	if parseEnvelopeFast, hasParseEnvelopeFast := parseStructuredCloneSnapshotEnvelopeHeaderOnly(parsePayload); hasParseEnvelopeFast {
		if parseErr := ValidateSnapshotEnvelope(parseEnvelopeFast); parseErr != nil {
			return SnapshotEnvelope{}, parseErr
		}
		return parseEnvelopeFast, nil
	}
	var parseEnvelope SnapshotEnvelope
	if parseErr := json.Unmarshal(parsePayload, &parseEnvelope); parseErr != nil {
		return SnapshotEnvelope{}, fmt.Errorf("runtime2: decode structured-clone snapshot envelope: %w", parseErr)
	}
	if parseErr := ValidateSnapshotEnvelope(parseEnvelope); parseErr != nil {
		return SnapshotEnvelope{}, parseErr
	}
	return parseEnvelope, nil
}

// BenchmarkStructuredCloneSnapshotEnvelopeCurrentVsLegacy compares current structured-clone snapshot build/parse paths against legacy behavior.
func BenchmarkStructuredCloneSnapshotEnvelopeCurrentVsLegacy(parseB *testing.B) {
	parseRichEnvelope := SnapshotEnvelope{
		RegionInstanceID: RegionInstanceID("region-1"),
		Epoch:            3,
		InputVersion:     42,
		SourceVersion:    7,
		Props: map[string]any{
			"title":  "Orders",
			"status": "healthy",
			"count":  128,
		},
		Sources: map[string]any{
			"filter": "active",
			"page":   2,
		},
	}
	parseHeaderOnlyEnvelope := SnapshotEnvelope{
		RegionInstanceID: RegionInstanceID("region-2"),
		Epoch:            4,
		InputVersion:     9,
	}
	parseRichPayload, parseRichPayloadErr := BuildStructuredCloneSnapshotEnvelopeJSON(parseRichEnvelope)
	if parseRichPayloadErr != nil {
		parseB.Fatalf("BuildStructuredCloneSnapshotEnvelopeJSON(rich) returned error: %v", parseRichPayloadErr)
	}
	parseHeaderOnlyPayload, parseHeaderOnlyPayloadErr := BuildStructuredCloneSnapshotEnvelopeJSON(parseHeaderOnlyEnvelope)
	if parseHeaderOnlyPayloadErr != nil {
		parseB.Fatalf("BuildStructuredCloneSnapshotEnvelopeJSON(header-only) returned error: %v", parseHeaderOnlyPayloadErr)
	}

	parseB.Run("build_rich/legacy", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			parsePayload, parseErr := buildStructuredCloneSnapshotEnvelopeJSONLegacyBenchmark(parseRichEnvelope)
			if parseErr != nil {
				parseB.Fatalf("buildStructuredCloneSnapshotEnvelopeJSONLegacyBenchmark returned error: %v", parseErr)
			}
			storeStructuredCloneSnapshotBenchmarkPayloadSink = parsePayload
		}
	})
	parseB.Run("build_rich/current", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			parsePayload, parseErr := BuildStructuredCloneSnapshotEnvelopeJSON(parseRichEnvelope)
			if parseErr != nil {
				parseB.Fatalf("BuildStructuredCloneSnapshotEnvelopeJSON returned error: %v", parseErr)
			}
			storeStructuredCloneSnapshotBenchmarkPayloadSink = parsePayload
		}
	})
	parseB.Run("build_header_only/legacy", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			parsePayload, parseErr := buildStructuredCloneSnapshotEnvelopeJSONLegacyBenchmark(parseHeaderOnlyEnvelope)
			if parseErr != nil {
				parseB.Fatalf("buildStructuredCloneSnapshotEnvelopeJSONLegacyBenchmark(header-only) returned error: %v", parseErr)
			}
			storeStructuredCloneSnapshotBenchmarkPayloadSink = parsePayload
		}
	})
	parseB.Run("build_header_only/current", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			parsePayload, parseErr := BuildStructuredCloneSnapshotEnvelopeJSON(parseHeaderOnlyEnvelope)
			if parseErr != nil {
				parseB.Fatalf("BuildStructuredCloneSnapshotEnvelopeJSON(header-only) returned error: %v", parseErr)
			}
			storeStructuredCloneSnapshotBenchmarkPayloadSink = parsePayload
		}
	})

	parseB.Run("parse_header_only/legacy", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			parseEnvelope, parseErr := parseStructuredCloneSnapshotEnvelopeJSONLegacyBenchmark(parseHeaderOnlyPayload)
			if parseErr != nil {
				parseB.Fatalf("parseStructuredCloneSnapshotEnvelopeJSONLegacyBenchmark returned error: %v", parseErr)
			}
			storeStructuredCloneSnapshotBenchmarkEnvelopeSink = parseEnvelope
		}
	})
	parseB.Run("parse_header_only/current", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			parseEnvelope, parseErr := ParseStructuredCloneSnapshotEnvelopeJSON(parseHeaderOnlyPayload)
			if parseErr != nil {
				parseB.Fatalf("ParseStructuredCloneSnapshotEnvelopeJSON returned error: %v", parseErr)
			}
			storeStructuredCloneSnapshotBenchmarkEnvelopeSink = parseEnvelope
		}
	})

	parseB.Run("parse_rich/legacy", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			parseEnvelope, parseErr := parseStructuredCloneSnapshotEnvelopeJSONLegacyBenchmark(parseRichPayload)
			if parseErr != nil {
				parseB.Fatalf("parseStructuredCloneSnapshotEnvelopeJSONLegacyBenchmark returned error: %v", parseErr)
			}
			storeStructuredCloneSnapshotBenchmarkEnvelopeSink = parseEnvelope
		}
	})
	parseB.Run("parse_rich/current", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			parseEnvelope, parseErr := ParseStructuredCloneSnapshotEnvelopeJSON(parseRichPayload)
			if parseErr != nil {
				parseB.Fatalf("ParseStructuredCloneSnapshotEnvelopeJSON returned error: %v", parseErr)
			}
			storeStructuredCloneSnapshotBenchmarkEnvelopeSink = parseEnvelope
		}
	})
}
