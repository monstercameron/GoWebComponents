package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime2"
)

// TestParseBinaryEnvelopeHeaderAcceptsValidHeader verifies valid binary envelope headers decode.
func TestParseBinaryEnvelopeHeaderAcceptsValidHeader(parseT *testing.T) {
	parseHeaderBytes, parseErr := runtime2.BuildBinaryEnvelopeHeader(runtime2.BinaryEnvelopeKindSnapshot, 12, 99)
	if parseErr != nil {
		parseT.Fatalf("BuildBinaryEnvelopeHeader returned error: %v", parseErr)
	}
	parseHeader, parseErr := runtime2.ParseBinaryEnvelopeHeader(parseHeaderBytes)
	if parseErr != nil {
		parseT.Fatalf("ParseBinaryEnvelopeHeader returned error: %v", parseErr)
	}
	if parseHeader.PayloadLength != 12 {
		parseT.Fatalf("expected payload length 12, got %d", parseHeader.PayloadLength)
	}
	if parseHeader.SectionCount != 1 {
		parseT.Fatalf("expected section count 1, got %d", parseHeader.SectionCount)
	}
}

// TestParseBinaryEnvelopeHeaderRejectsBadMagic verifies invalid binary header magic fails.
func TestParseBinaryEnvelopeHeaderRejectsBadMagic(parseT *testing.T) {
	parseHeaderBytes := []byte("BAD!\x01\x00\x01\x00\x0c\x00\x00\x00\x63\x00\x00\x00")
	if _, parseErr := runtime2.ParseBinaryEnvelopeHeader(parseHeaderBytes); parseErr == nil {
		parseT.Fatal("expected bad binary magic to fail")
	}
}

// TestParseBinaryEnvelopeHeaderRejectsUnsupportedVersion verifies unsupported binary header versions fail.
func TestParseBinaryEnvelopeHeaderRejectsUnsupportedVersion(parseT *testing.T) {
	parseHeaderBytes := []byte("GWB1\x02\x00\x01\x00\x0c\x00\x00\x00\x63\x00\x00\x00")
	if _, parseErr := runtime2.ParseBinaryEnvelopeHeader(parseHeaderBytes); parseErr == nil {
		parseT.Fatal("expected unsupported binary header version to fail")
	}
}

// TestParseBinaryEnvelopeHeaderRejectsUnsupportedKind verifies unsupported binary header kinds fail.
func TestParseBinaryEnvelopeHeaderRejectsUnsupportedKind(parseT *testing.T) {
	parseHeaderBytes := []byte("GWB1\x01\x00\x09\x01\x0c\x00\x00\x00\x63\x00\x00\x00")
	if _, parseErr := runtime2.ParseBinaryEnvelopeHeader(parseHeaderBytes); parseErr == nil {
		parseT.Fatal("expected unsupported binary header kind to fail")
	}
}

// TestParseBinaryEnvelopeHeaderRejectsZeroSectionCount verifies binary headers reject unsupported section counts.
func TestParseBinaryEnvelopeHeaderRejectsZeroSectionCount(parseT *testing.T) {
	parseHeaderBytes := []byte("GWB1\x01\x00\x01\x00\x0c\x00\x00\x00\x63\x00\x00\x00")
	if _, parseErr := runtime2.ParseBinaryEnvelopeHeader(parseHeaderBytes); parseErr == nil {
		parseT.Fatal("expected zero binary header section count to fail")
	}
}

// TestParseBinaryEnvelopeHeaderRejectsUnsupportedSectionCount verifies non-current binary header section counts fail.
func TestParseBinaryEnvelopeHeaderRejectsUnsupportedSectionCount(parseT *testing.T) {
	parseHeaderBytes := []byte("GWB1\x01\x00\x01\x02\x0c\x00\x00\x00\x63\x00\x00\x00")
	if _, parseErr := runtime2.ParseBinaryEnvelopeHeader(parseHeaderBytes); parseErr == nil {
		parseT.Fatal("expected unsupported binary header section count to fail")
	}
}
