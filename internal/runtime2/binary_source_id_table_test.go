package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime2"
)

// TestBuildBinarySourceIDTableRoundTripsCanonicalIDs verifies canonical source-ID tables round-trip.
func TestBuildBinarySourceIDTableRoundTripsCanonicalIDs(parseT *testing.T) {
	parsePayload, parseErr := runtime2.BuildBinarySourceIDTable([]string{"status", "user.id"})
	if parseErr != nil {
		parseT.Fatalf("BuildBinarySourceIDTable returned error: %v", parseErr)
	}
	parseSourceIDs, parseErr := runtime2.ParseBinarySourceIDTable(parsePayload)
	if parseErr != nil {
		parseT.Fatalf("ParseBinarySourceIDTable returned error: %v", parseErr)
	}
	if len(parseSourceIDs) != 2 || parseSourceIDs[0] != "status" || parseSourceIDs[1] != "user.id" {
		parseT.Fatalf("unexpected source IDs %#v", parseSourceIDs)
	}
}

// TestParseBinarySourceIDTableRejectsNonCanonicalOrder verifies non-canonical source-ID tables fail decode.
func TestParseBinarySourceIDTableRejectsNonCanonicalOrder(parseT *testing.T) {
	parsePayload := []byte{2, 0, 6, 0, 'z', 'e', 't', 'a', '.', 'a', 5, 0, 'a', 'l', 'p', 'h', 'a'}
	if _, parseErr := runtime2.ParseBinarySourceIDTable(parsePayload); parseErr == nil {
		parseT.Fatal("expected non-canonical source-ID table to fail")
	}
}
