package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime2"
)

// FuzzParseSharedSnapshotPageHeader fuzzes shared-page header parsing to ensure malformed inputs fail safely without panics.
func FuzzParseSharedSnapshotPageHeader(parseF *testing.F) {
	parseValidHeaderBytes, parseErr := runtime2.BuildSharedSnapshotPageHeader(runtime2.SharedSnapshotPageKindSnapshot, 2, 8, runtime2.SharedSnapshotPageStatusComplete)
	if parseErr != nil {
		parseF.Fatalf("BuildSharedSnapshotPageHeader returned error: %v", parseErr)
	}
	parseF.Add(parseValidHeaderBytes)
	parseF.Add([]byte{})
	parseF.Add([]byte("GSP1"))
	parseF.Add([]byte("BAD!\x01\x00\x01\x00\x01\x00\x00\x00\x00\x00\x00\x00\x04\x00\x00\x00\x01\x00\x00\x00"))

	parseF.Fuzz(func(parseT *testing.T, parsePayload []byte) {
		parseHeader, parseParseErr := runtime2.ParseSharedSnapshotPageHeader(parsePayload)
		if parseParseErr != nil {
			return
		}
		if parseHeader.Kind != runtime2.SharedSnapshotPageKindSnapshot {
			parseT.Fatalf("expected kind %d, got %d", runtime2.SharedSnapshotPageKindSnapshot, parseHeader.Kind)
		}
		if parseHeader.Status != runtime2.SharedSnapshotPageStatusWriting && parseHeader.Status != runtime2.SharedSnapshotPageStatusComplete {
			parseT.Fatalf("expected known status value, got %d", parseHeader.Status)
		}
	})
}
