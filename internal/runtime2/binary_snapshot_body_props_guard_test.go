package runtime2

import (
	"strings"
	"testing"
)

// TestParseBinarySnapshotBodyRejectsRefLikePropsKey verifies decoded snapshot props still reject unsupported ref markers.
func TestParseBinarySnapshotBodyRejectsRefLikePropsKey(parseT *testing.T) {
	parsePayload, parseBuildErr := appendBinarySnapshotBody(
		make([]byte, 0, 256),
		SnapshotEnvelope{
			RegionInstanceID: RegionInstanceID("region-props-ref"),
			Epoch:            1,
			InputVersion:     1,
			Props: map[string]any{
				"ref": true,
			},
		},
	)
	if parseBuildErr != nil {
		parseT.Fatalf("appendBinarySnapshotBody returned error: %v", parseBuildErr)
	}
	if _, parseErr := ParseBinarySnapshotBody(parsePayload); parseErr == nil {
		parseT.Fatal("expected ref-like snapshot props key to fail")
	} else if !strings.Contains(parseErr.Error(), "unsupported ref marker") {
		parseT.Fatalf("expected unsupported ref marker error, got %v", parseErr)
	}
}

// TestParseBinarySnapshotBodyRejectsDOMInteropPropsKey verifies decoded snapshot props still reject direct DOM interop markers.
func TestParseBinarySnapshotBodyRejectsDOMInteropPropsKey(parseT *testing.T) {
	parsePayload, parseBuildErr := appendBinarySnapshotBody(
		make([]byte, 0, 256),
		SnapshotEnvelope{
			RegionInstanceID: RegionInstanceID("region-props-dom"),
			Epoch:            1,
			InputVersion:     1,
			Props: map[string]any{
				"domNode": true,
			},
		},
	)
	if parseBuildErr != nil {
		parseT.Fatalf("appendBinarySnapshotBody returned error: %v", parseBuildErr)
	}
	if _, parseErr := ParseBinarySnapshotBody(parsePayload); parseErr == nil {
		parseT.Fatal("expected direct DOM interop snapshot props key to fail")
	} else if !strings.Contains(parseErr.Error(), "direct DOM interop marker") {
		parseT.Fatalf("expected direct DOM interop marker error, got %v", parseErr)
	}
}
