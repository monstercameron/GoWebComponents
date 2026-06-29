package runtime2

import (
	"encoding/json"
	"fmt"
	"hash"
	"hash/fnv"
	"strconv"
	"testing"
)

var (
	storeRuntime2PatchIdentityBenchmarkSink string
)

// buildPatchStreamIdentityBenchmarkCurrent calls BuildPatchStreamIdentity through a noinline wrapper to keep compare benches honest.
//
//go:noinline
func buildPatchStreamIdentityBenchmarkCurrent(parseRaw PatchStreamRaw) (string, error) {
	return BuildPatchStreamIdentity(parseRaw)
}

// buildPatchStreamIdentityBenchmarkLegacy calls the legacy marshal-based patch identity path through a noinline wrapper for compare benches.
//
//go:noinline
func buildPatchStreamIdentityBenchmarkLegacy(parseRaw PatchStreamRaw) (string, error) {
	return buildRuntime2LegacyPatchStreamIdentity(parseRaw)
}

// buildPatchStreamIdentityBenchmarkLegacyStreaming calls the previous streaming-writer patch identity path through a noinline wrapper for compare benches.
//
//go:noinline
func buildPatchStreamIdentityBenchmarkLegacyStreaming(parseRaw PatchStreamRaw) (string, error) {
	return buildPatchStreamIdentityLegacyStreamingBenchmark(parseRaw)
}

// buildPatchStreamIdentitySmallBenchmarkList builds one deterministic small patch-stream set to keep identity benches from collapsing to one constant input.
func buildPatchStreamIdentitySmallBenchmarkList(parseB *testing.B) []PatchStreamRaw {
	parseB.Helper()
	buildPatchStreams := make([]PatchStreamRaw, 0, 8)
	for parseIndex := range 8 {
		buildPatchStreams = append(buildPatchStreams, buildPerfHotspotPatchStream(
			parseB,
			parseBuildAgent3BenchRenderOutput("before-"+strconv.Itoa(parseIndex), "active-"+strconv.Itoa(parseIndex)),
			parseBuildAgent3BenchRenderOutput("after-"+strconv.Itoa(parseIndex), "idle-"+strconv.Itoa(parseIndex)),
		))
	}
	return buildPatchStreams
}

// buildPatchStreamIdentityLargeBenchmarkList builds one deterministic large patch-stream set with varying keyed rotations for identity benches.
func buildPatchStreamIdentityLargeBenchmarkList(parseB *testing.B) []PatchStreamRaw {
	parseB.Helper()
	buildPatchStreams := make([]PatchStreamRaw, 0, 8)
	for parseIndex := range 8 {
		buildPatchStreams = append(buildPatchStreams, buildPerfHotspotPatchStream(
			parseB,
			buildPerfHotspotKeyedListRenderOutput(64, parseIndex, "active"),
			buildPerfHotspotKeyedListRenderOutput(64, parseIndex+1, "idle"),
		))
	}
	return buildPatchStreams
}

type buildJSONHashWriterLegacyBenchmark struct {
	getHasher        hash.Hash64
	getPendingByte   byte
	hasPendingByte   bool
	getPendingBuffer [1]byte
}

// buildRuntime2LegacyPatchStreamIdentity computes patch identity using the pre-optimization JSON marshal path.
func buildRuntime2LegacyPatchStreamIdentity(parseRaw PatchStreamRaw) (string, error) {
	parsePayload, parsePayloadErr := json.Marshal(struct {
		GetHeader      PatchStreamHeaderRaw
		GetStringTable []string
		GetOps         []PatchStreamOpRaw
	}{
		GetHeader:      parseRaw.GetHeader,
		GetStringTable: parseRaw.GetStringTable,
		GetOps:         parseRaw.GetOps,
	})
	if parsePayloadErr != nil {
		return "", fmt.Errorf("runtime2: encode patch identity payload: %w", parsePayloadErr)
	}
	parseHasher := fnv.New64a()
	_, _ = parseHasher.Write(parsePayload)
	return fmt.Sprintf("%x", parseHasher.Sum64()), nil
}

// Write preserves the previous JSON hash writer behavior for compare benchmarks.
func (parseWriter *buildJSONHashWriterLegacyBenchmark) Write(parseBytes []byte) (int, error) {
	if parseWriter == nil {
		return 0, fmt.Errorf("runtime2: json hash benchmark writer is nil")
	}
	if len(parseBytes) == 0 {
		return 0, nil
	}
	if parseWriter.hasPendingByte {
		parseWriter.getPendingBuffer[0] = parseWriter.getPendingByte
		if _, parseErr := parseWriter.getHasher.Write(parseWriter.getPendingBuffer[:]); parseErr != nil {
			return 0, parseErr
		}
		parseWriter.hasPendingByte = false
	}
	if len(parseBytes) > 1 {
		if _, parseErr := parseWriter.getHasher.Write(parseBytes[:len(parseBytes)-1]); parseErr != nil {
			return 0, parseErr
		}
	}
	parseWriter.getPendingByte = parseBytes[len(parseBytes)-1]
	parseWriter.hasPendingByte = true
	return len(parseBytes), nil
}

// Flush preserves the previous JSON hash writer flush behavior for compare benchmarks.
func (parseWriter *buildJSONHashWriterLegacyBenchmark) Flush() error {
	if parseWriter == nil || !parseWriter.hasPendingByte {
		return nil
	}
	parsePendingByte := parseWriter.getPendingByte
	parseWriter.hasPendingByte = false
	if parsePendingByte == '\n' {
		return nil
	}
	parseWriter.getPendingBuffer[0] = parsePendingByte
	_, parseErr := parseWriter.getHasher.Write(parseWriter.getPendingBuffer[:])
	return parseErr
}

// buildJSONHashDigestLegacyBenchmark encodes one payload with the previous JSON streaming hash writer behavior.
func buildJSONHashDigestLegacyBenchmark(parseHasher hash.Hash64, parseValue any) error {
	if parseHasher == nil {
		return fmt.Errorf("runtime2: json hash benchmark digest hasher is nil")
	}
	parseWriter := buildJSONHashWriterLegacyBenchmark{
		getHasher: parseHasher,
	}
	parseEncoder := json.NewEncoder(&parseWriter)
	if parseErr := parseEncoder.Encode(parseValue); parseErr != nil {
		return fmt.Errorf("runtime2: encode json hash benchmark payload: %w", parseErr)
	}
	if parseErr := parseWriter.Flush(); parseErr != nil {
		return fmt.Errorf("runtime2: flush json hash benchmark payload: %w", parseErr)
	}
	return nil
}

// buildPatchStreamIdentityLegacyStreamingBenchmark computes patch identity using the previous JSON streaming hash writer behavior.
func buildPatchStreamIdentityLegacyStreamingBenchmark(parseRaw PatchStreamRaw) (string, error) {
	parseHasher := fnv.New64a()
	if parseTokenErr := writeJSONHashLiteral(parseHasher, writePatchStreamIdentityTokenHeader); parseTokenErr != nil {
		return "", parseTokenErr
	}
	if parseHeaderErr := buildJSONHashDigestLegacyBenchmark(parseHasher, parseRaw.GetHeader); parseHeaderErr != nil {
		return "", parseHeaderErr
	}
	if parseTokenErr := writeJSONHashLiteral(parseHasher, writePatchStreamIdentityTokenStringTable); parseTokenErr != nil {
		return "", parseTokenErr
	}
	if parseStringTableErr := buildJSONHashDigestLegacyBenchmark(parseHasher, parseRaw.GetStringTable); parseStringTableErr != nil {
		return "", parseStringTableErr
	}
	if parseTokenErr := writeJSONHashLiteral(parseHasher, writePatchStreamIdentityTokenOps); parseTokenErr != nil {
		return "", parseTokenErr
	}
	if parseRaw.GetOps == nil {
		if parseTokenErr := writeJSONHashLiteral(parseHasher, writePatchStreamIdentityTokenNullClose); parseTokenErr != nil {
			return "", parseTokenErr
		}
		return fmt.Sprintf("%x", parseHasher.Sum64()), nil
	}
	if parseTokenErr := writeJSONHashLiteral(parseHasher, writePatchStreamIdentityTokenArrayOpen); parseTokenErr != nil {
		return "", parseTokenErr
	}
	for parseIndex, parseOp := range parseRaw.GetOps {
		if parseIndex > 0 {
			if parseTokenErr := writeJSONHashLiteral(parseHasher, writePatchStreamIdentityTokenComma); parseTokenErr != nil {
				return "", parseTokenErr
			}
		}
		if parseOpErr := buildJSONHashDigestLegacyBenchmark(parseHasher, parseOp); parseOpErr != nil {
			return "", parseOpErr
		}
	}
	if parseTokenErr := writeJSONHashLiteral(parseHasher, writePatchStreamIdentityTokenArrayClose); parseTokenErr != nil {
		return "", parseTokenErr
	}
	return fmt.Sprintf("%x", parseHasher.Sum64()), nil
}

// BenchmarkBuildPatchStreamIdentityCurrentVsLegacy compares current patch identity hashing against the legacy marshal-based implementation.
func BenchmarkBuildPatchStreamIdentityCurrentVsLegacy(parseB *testing.B) {
	parseB.Run("small/current", func(parseB *testing.B) {
		parsePatchStreams := buildPatchStreamIdentitySmallBenchmarkList(parseB)
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			parseIdentity, parseIdentityErr := buildPatchStreamIdentityBenchmarkCurrent(parsePatchStreams[parseIndex%len(parsePatchStreams)])
			if parseIdentityErr != nil {
				parseB.Fatalf("BuildPatchStreamIdentity returned error: %v", parseIdentityErr)
			}
			storeRuntime2PatchIdentityBenchmarkSink = parseIdentity
		}
	})
	parseB.Run("small/legacy", func(parseB *testing.B) {
		parsePatchStreams := buildPatchStreamIdentitySmallBenchmarkList(parseB)
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			parseIdentity, parseIdentityErr := buildPatchStreamIdentityBenchmarkLegacy(parsePatchStreams[parseIndex%len(parsePatchStreams)])
			if parseIdentityErr != nil {
				parseB.Fatalf("buildRuntime2LegacyPatchStreamIdentity returned error: %v", parseIdentityErr)
			}
			storeRuntime2PatchIdentityBenchmarkSink = parseIdentity
		}
	})
	parseB.Run("large/current", func(parseB *testing.B) {
		parsePatchStreams := buildPatchStreamIdentityLargeBenchmarkList(parseB)
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			parseIdentity, parseIdentityErr := buildPatchStreamIdentityBenchmarkCurrent(parsePatchStreams[parseIndex%len(parsePatchStreams)])
			if parseIdentityErr != nil {
				parseB.Fatalf("BuildPatchStreamIdentity returned error: %v", parseIdentityErr)
			}
			storeRuntime2PatchIdentityBenchmarkSink = parseIdentity
		}
	})
	parseB.Run("large/legacy", func(parseB *testing.B) {
		parsePatchStreams := buildPatchStreamIdentityLargeBenchmarkList(parseB)
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			parseIdentity, parseIdentityErr := buildPatchStreamIdentityBenchmarkLegacy(parsePatchStreams[parseIndex%len(parsePatchStreams)])
			if parseIdentityErr != nil {
				parseB.Fatalf("buildRuntime2LegacyPatchStreamIdentity returned error: %v", parseIdentityErr)
			}
			storeRuntime2PatchIdentityBenchmarkSink = parseIdentity
		}
	})
}

// BenchmarkBuildPatchStreamIdentityStreamingCurrentVsLegacy compares the current patch identity stream against the previous JSON writer behavior.
func BenchmarkBuildPatchStreamIdentityStreamingCurrentVsLegacy(parseB *testing.B) {
	parseB.Run("small/current", func(parseB *testing.B) {
		parsePatchStreams := buildPatchStreamIdentitySmallBenchmarkList(parseB)
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			parseIdentity, parseIdentityErr := buildPatchStreamIdentityBenchmarkCurrent(parsePatchStreams[parseIndex%len(parsePatchStreams)])
			if parseIdentityErr != nil {
				parseB.Fatalf("BuildPatchStreamIdentity returned error: %v", parseIdentityErr)
			}
			storeRuntime2PatchIdentityBenchmarkSink = parseIdentity
		}
	})
	parseB.Run("small/legacy-stream-writer", func(parseB *testing.B) {
		parsePatchStreams := buildPatchStreamIdentitySmallBenchmarkList(parseB)
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			parseIdentity, parseIdentityErr := buildPatchStreamIdentityBenchmarkLegacyStreaming(parsePatchStreams[parseIndex%len(parsePatchStreams)])
			if parseIdentityErr != nil {
				parseB.Fatalf("buildPatchStreamIdentityLegacyStreamingBenchmark returned error: %v", parseIdentityErr)
			}
			storeRuntime2PatchIdentityBenchmarkSink = parseIdentity
		}
	})
	parseB.Run("large/current", func(parseB *testing.B) {
		parsePatchStreams := buildPatchStreamIdentityLargeBenchmarkList(parseB)
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			parseIdentity, parseIdentityErr := buildPatchStreamIdentityBenchmarkCurrent(parsePatchStreams[parseIndex%len(parsePatchStreams)])
			if parseIdentityErr != nil {
				parseB.Fatalf("BuildPatchStreamIdentity returned error: %v", parseIdentityErr)
			}
			storeRuntime2PatchIdentityBenchmarkSink = parseIdentity
		}
	})
	parseB.Run("large/legacy-stream-writer", func(parseB *testing.B) {
		parsePatchStreams := buildPatchStreamIdentityLargeBenchmarkList(parseB)
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			parseIdentity, parseIdentityErr := buildPatchStreamIdentityBenchmarkLegacyStreaming(parsePatchStreams[parseIndex%len(parsePatchStreams)])
			if parseIdentityErr != nil {
				parseB.Fatalf("buildPatchStreamIdentityLegacyStreamingBenchmark returned error: %v", parseIdentityErr)
			}
			storeRuntime2PatchIdentityBenchmarkSink = parseIdentity
		}
	})
}
