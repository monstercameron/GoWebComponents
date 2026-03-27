package runtime2

import (
	"fmt"
	"testing"
)

// buildPerfHotspotKeyedListRenderOutput builds one keyed-list render output with optional sibling rotation.
func buildPerfHotspotKeyedListRenderOutput(parseItemCount int, parseRotation int, parseClass string) map[string]any {
	if parseItemCount <= 0 {
		parseItemCount = 1
	}
	buildChildren := make([]any, 0, parseItemCount)
	for parseIndex := 0; parseIndex < parseItemCount; parseIndex++ {
		getItemIndex := (parseIndex + parseRotation) % parseItemCount
		buildChildren = append(buildChildren, map[string]any{
			"kind": "host-element",
			"tag":  "li",
			"key":  fmt.Sprintf("item-%d", getItemIndex),
			"children": []any{
				map[string]any{
					"kind": "text",
					"text": fmt.Sprintf("value-%d", getItemIndex),
				},
			},
		})
	}
	return map[string]any{
		"kind": "host-element",
		"tag":  "div",
		"props": map[string]any{
			"class": parseClass,
		},
		"children": []any{
			map[string]any{
				"kind":     "host-element",
				"tag":      "ul",
				"children": buildChildren,
			},
		},
	}
}

// buildPerfHotspotPatchStream builds one canonical patch stream from two render outputs.
func buildPerfHotspotPatchStream(parseB *testing.B, parsePreviousRenderOutput any, parseNextRenderOutput any) PatchStreamRaw {
	parseB.Helper()
	parsePreviousIR, parsePreviousErr := BuildCanonicalRenderIR(parsePreviousRenderOutput)
	if parsePreviousErr != nil {
		parseB.Fatalf("BuildCanonicalRenderIR(previous) returned error: %v", parsePreviousErr)
	}
	parseNextIR, parseNextErr := BuildCanonicalRenderIR(parseNextRenderOutput)
	if parseNextErr != nil {
		parseB.Fatalf("BuildCanonicalRenderIR(next) returned error: %v", parseNextErr)
	}
	parsePatchStream, hasNoOp, parsePatchErr := BuildCanonicalPatchStream("bench-region", 1, 2, 2, parsePreviousIR, parseNextIR)
	if parsePatchErr != nil {
		parseB.Fatalf("BuildCanonicalPatchStream returned error: %v", parsePatchErr)
	}
	if hasNoOp {
		parseB.Fatal("BuildCanonicalPatchStream unexpectedly returned no-op for differing payloads")
	}
	return parsePatchStream
}

// buildPerfHotspotMountedHostRegionAdapter builds and mounts one host-region adapter for dispatch benchmarks.
func buildPerfHotspotMountedHostRegionAdapter(parseB *testing.B, parseRegionID RegionInstanceID) *HostRegionAdapter {
	parseB.Helper()
	buildHostRegionAdapter, parseBuildErr := BuildHostRegionAdapter(parseRegionID, []SchedulerShardID{"shard-a"})
	if parseBuildErr != nil {
		parseB.Fatalf("BuildHostRegionAdapter returned error: %v", parseBuildErr)
	}
	_, parseMountErr := buildHostRegionAdapter.HandleHostRegionMount(
		ParallelRegionSpec{
			RendererID:       RendererID("dashboard.hot-panel"),
			RegionInstanceID: parseRegionID,
		},
		1,
	)
	if parseMountErr != nil {
		parseB.Fatalf("HandleHostRegionMount returned error: %v", parseMountErr)
	}
	return buildHostRegionAdapter
}

// BenchmarkParseCanonicalRenderTree benchmarks canonical tree decode cost from one prebuilt render IR.
func BenchmarkParseCanonicalRenderTree(parseB *testing.B) {
	parseRenderOutput := buildPerfHotspotKeyedListRenderOutput(64, 0, "active")
	parseCanonicalIR, parseCanonicalErr := BuildCanonicalRenderIR(parseRenderOutput)
	if parseCanonicalErr != nil {
		parseB.Fatalf("BuildCanonicalRenderIR returned error: %v", parseCanonicalErr)
	}
	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
		if _, parseTreeErr := ParseCanonicalRenderTree(parseCanonicalIR); parseTreeErr != nil {
			parseB.Fatalf("ParseCanonicalRenderTree returned error: %v", parseTreeErr)
		}
	}
}

// BenchmarkBuildPatchStreamIdentity benchmarks patch-identity hashing over representative small and large patch streams.
func BenchmarkBuildPatchStreamIdentity(parseB *testing.B) {
	parseB.Run("small", func(parseB *testing.B) {
		parsePatchStream := buildPerfHotspotPatchStream(
			parseB,
			parseBuildAgent3BenchRenderOutput("before", "active"),
			parseBuildAgent3BenchRenderOutput("after", "idle"),
		)
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			if _, parseIdentityErr := BuildPatchStreamIdentity(parsePatchStream); parseIdentityErr != nil {
				parseB.Fatalf("BuildPatchStreamIdentity returned error: %v", parseIdentityErr)
			}
		}
	})
	parseB.Run("large-keyed-rotate", func(parseB *testing.B) {
		parsePatchStream := buildPerfHotspotPatchStream(
			parseB,
			buildPerfHotspotKeyedListRenderOutput(64, 0, "active"),
			buildPerfHotspotKeyedListRenderOutput(64, 1, "idle"),
		)
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			if _, parseIdentityErr := BuildPatchStreamIdentity(parsePatchStream); parseIdentityErr != nil {
				parseB.Fatalf("BuildPatchStreamIdentity returned error: %v", parseIdentityErr)
			}
		}
	})
}

// BenchmarkHandleHostRegionUpdateDispatchWithPriorityDeferred benchmarks deferred dispatch no-change versus changed snapshot handling.
func BenchmarkHandleHostRegionUpdateDispatchWithPriorityDeferred(parseB *testing.B) {
	parseB.Run("no-change", func(parseB *testing.B) {
		parseHostRegionAdapter := buildPerfHotspotMountedHostRegionAdapter(parseB, RegionInstanceID("region-deferred-no-change"))
		parseSpec := ParallelRegionSpec{
			RendererID:       RendererID("dashboard.hot-panel"),
			RegionInstanceID: RegionInstanceID("region-deferred-no-change"),
			Props: map[string]any{
				"title": "Orders",
			},
		}
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			if _, parseDispatchErr := parseHostRegionAdapter.HandleHostRegionUpdateDispatchWithPriority(
				parseSpec,
				uint64(parseIndex+1),
				HostRegionDispatchPriorityDeferred,
			); parseDispatchErr != nil {
				parseB.Fatalf("HandleHostRegionUpdateDispatchWithPriority(no-change) returned error: %v", parseDispatchErr)
			}
		}
	})
	parseB.Run("changed", func(parseB *testing.B) {
		parseHostRegionAdapter := buildPerfHotspotMountedHostRegionAdapter(parseB, RegionInstanceID("region-deferred-changed"))
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			parseSpec := ParallelRegionSpec{
				RendererID:       RendererID("dashboard.hot-panel"),
				RegionInstanceID: RegionInstanceID("region-deferred-changed"),
				Props: map[string]any{
					"title": "Orders",
					"tick":  parseIndex,
				},
			}
			if _, parseDispatchErr := parseHostRegionAdapter.HandleHostRegionUpdateDispatchWithPriority(
				parseSpec,
				uint64(parseIndex+1),
				HostRegionDispatchPriorityDeferred,
			); parseDispatchErr != nil {
				parseB.Fatalf("HandleHostRegionUpdateDispatchWithPriority(changed) returned error: %v", parseDispatchErr)
			}
		}
	})
	parseB.Run("changed-reused-spec", func(parseB *testing.B) {
		parseHostRegionAdapter := buildPerfHotspotMountedHostRegionAdapter(parseB, RegionInstanceID("region-deferred-changed-reused-spec"))
		parseProps := map[string]any{
			"title": "Orders",
			"tick":  0,
		}
		parseSpec := ParallelRegionSpec{
			RendererID:       RendererID("dashboard.hot-panel"),
			RegionInstanceID: RegionInstanceID("region-deferred-changed-reused-spec"),
			Props:            parseProps,
		}
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			parseProps["tick"] = parseIndex
			if _, parseDispatchErr := parseHostRegionAdapter.HandleHostRegionUpdateDispatchWithPriority(
				parseSpec,
				uint64(parseIndex+1),
				HostRegionDispatchPriorityDeferred,
			); parseDispatchErr != nil {
				parseB.Fatalf("HandleHostRegionUpdateDispatchWithPriority(changed-reused-spec) returned error: %v", parseDispatchErr)
			}
		}
	})
}

// BenchmarkHandleHostRegionSnapshotFingerprint benchmarks no-change versus changed direct snapshot fingerprint handling.
func BenchmarkHandleHostRegionSnapshotFingerprint(parseB *testing.B) {
	parseB.Run("no-change", func(parseB *testing.B) {
		parseHostRegionAdapter, parseBuildErr := BuildHostRegionAdapter(
			RegionInstanceID("region-fingerprint-no-change"),
			[]SchedulerShardID{"shard-a"},
		)
		if parseBuildErr != nil {
			parseB.Fatalf("BuildHostRegionAdapter returned error: %v", parseBuildErr)
		}
		parseSnapshotEnvelope := SnapshotEnvelope{
			RegionInstanceID: RegionInstanceID("region-fingerprint-no-change"),
			Epoch:            1,
			InputVersion:     1,
			Props: map[string]any{
				"title": "Orders",
			},
		}
		if _, parseFingerprintErr := parseHostRegionAdapter.HandleHostRegionSnapshotFingerprint(parseSnapshotEnvelope); parseFingerprintErr != nil {
			parseB.Fatalf("HandleHostRegionSnapshotFingerprint(warm-up) returned error: %v", parseFingerprintErr)
		}
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			parseSnapshotEnvelope.InputVersion = uint64(parseIndex + 2)
			if _, parseFingerprintErr := parseHostRegionAdapter.HandleHostRegionSnapshotFingerprint(parseSnapshotEnvelope); parseFingerprintErr != nil {
				parseB.Fatalf("HandleHostRegionSnapshotFingerprint(no-change) returned error: %v", parseFingerprintErr)
			}
		}
	})
	parseB.Run("changed", func(parseB *testing.B) {
		parseHostRegionAdapter, parseBuildErr := BuildHostRegionAdapter(
			RegionInstanceID("region-fingerprint-changed"),
			[]SchedulerShardID{"shard-a"},
		)
		if parseBuildErr != nil {
			parseB.Fatalf("BuildHostRegionAdapter returned error: %v", parseBuildErr)
		}
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			parseSnapshotEnvelope := SnapshotEnvelope{
				RegionInstanceID: RegionInstanceID("region-fingerprint-changed"),
				Epoch:            1,
				InputVersion:     uint64(parseIndex + 1),
				Props: map[string]any{
					"title": "Orders",
					"tick":  parseIndex,
				},
			}
			if _, parseFingerprintErr := parseHostRegionAdapter.HandleHostRegionSnapshotFingerprint(parseSnapshotEnvelope); parseFingerprintErr != nil {
				parseB.Fatalf("HandleHostRegionSnapshotFingerprint(changed) returned error: %v", parseFingerprintErr)
			}
		}
	})
}
