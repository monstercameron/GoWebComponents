package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime2"
)

// TestParseSharedSnapshotPageHeaderAcceptsValidHeader verifies valid shared-page headers decode.
func TestParseSharedSnapshotPageHeaderAcceptsValidHeader(parseT *testing.T) {
	parseHeaderBytes, parseErr := runtime2.BuildSharedSnapshotPageHeader(runtime2.SharedSnapshotPageKindSnapshot, 3, 12, runtime2.SharedSnapshotPageStatusComplete)
	if parseErr != nil {
		parseT.Fatalf("BuildSharedSnapshotPageHeader returned error: %v", parseErr)
	}
	parseHeader, parseErr := runtime2.ParseSharedSnapshotPageHeader(parseHeaderBytes)
	if parseErr != nil {
		parseT.Fatalf("ParseSharedSnapshotPageHeader returned error: %v", parseErr)
	}
	if parseHeader.Generation != 3 {
		parseT.Fatalf("expected generation 3, got %d", parseHeader.Generation)
	}
}

// TestParseSharedSnapshotPageHeaderRejectsBadMagic verifies invalid shared-page magic fails.
func TestParseSharedSnapshotPageHeaderRejectsBadMagic(parseT *testing.T) {
	parseHeaderBytes := []byte("BAD!\x01\x00\x01\x00\x01\x00\x00\x00\x00\x00\x00\x00\x0c\x00\x00\x00\x01\x00\x00\x00")
	if _, parseErr := runtime2.ParseSharedSnapshotPageHeader(parseHeaderBytes); parseErr == nil {
		parseT.Fatal("expected bad shared-page magic to fail")
	}
}

// TestParseSharedSnapshotPageHeaderRejectsUnsupportedVersion verifies unsupported shared-page versions fail.
func TestParseSharedSnapshotPageHeaderRejectsUnsupportedVersion(parseT *testing.T) {
	parseHeaderBytes := []byte("GSP1\x02\x00\x01\x00\x01\x00\x00\x00\x00\x00\x00\x00\x0c\x00\x00\x00\x01\x00\x00\x00")
	if _, parseErr := runtime2.ParseSharedSnapshotPageHeader(parseHeaderBytes); parseErr == nil {
		parseT.Fatal("expected unsupported shared-page version to fail")
	}
}

// TestParseSharedSnapshotPageHeaderRejectsTruncatedHeader verifies truncated shared-page headers fail.
func TestParseSharedSnapshotPageHeaderRejectsTruncatedHeader(parseT *testing.T) {
	if _, parseErr := runtime2.ParseSharedSnapshotPageHeader([]byte("GSP1")); parseErr == nil {
		parseT.Fatal("expected truncated shared-page header to fail")
	}
}

// TestParseSharedSnapshotPageHeaderRejectsUnsupportedKind verifies unknown shared-page kinds fail.
func TestParseSharedSnapshotPageHeaderRejectsUnsupportedKind(parseT *testing.T) {
	parseHeaderBytes, parseErr := runtime2.BuildSharedSnapshotPageHeader(runtime2.SharedSnapshotPageKindSnapshot, 3, 12, runtime2.SharedSnapshotPageStatusComplete)
	if parseErr != nil {
		parseT.Fatalf("BuildSharedSnapshotPageHeader returned error: %v", parseErr)
	}
	parseHeaderBytes[6] = 99
	parseHeaderBytes[7] = 0
	if _, parseErr := runtime2.ParseSharedSnapshotPageHeader(parseHeaderBytes); parseErr == nil {
		parseT.Fatal("expected unsupported shared-page kind to fail")
	}
}

// TestParseSharedSnapshotPageHeaderRejectsUnsupportedStatus verifies unsupported shared-page status values fail.
func TestParseSharedSnapshotPageHeaderRejectsUnsupportedStatus(parseT *testing.T) {
	parseHeaderBytes, parseErr := runtime2.BuildSharedSnapshotPageHeader(runtime2.SharedSnapshotPageKindSnapshot, 3, 12, runtime2.SharedSnapshotPageStatusComplete)
	if parseErr != nil {
		parseT.Fatalf("BuildSharedSnapshotPageHeader returned error: %v", parseErr)
	}
	parseHeaderBytes[20] = 9
	parseHeaderBytes[21] = 0
	parseHeaderBytes[22] = 0
	parseHeaderBytes[23] = 0
	if _, parseErr := runtime2.ParseSharedSnapshotPageHeader(parseHeaderBytes); parseErr == nil {
		parseT.Fatal("expected unsupported shared-page status to fail")
	}
}
