package runtime2

import (
	"encoding/binary"
	"strings"
	"testing"
)

// buildBinarySnapshotBodyTestPayload builds one valid snapshot-body payload for edge-case tests.
func buildBinarySnapshotBodyTestPayload(parseT *testing.T, parseProps any, parseSources map[string]any) []byte {
	parseT.Helper()
	parsePayload, parseErr := BuildBinarySnapshotBody(SnapshotEnvelope{
		RegionInstanceID: RegionInstanceID("region-1"),
		Epoch:            7,
		InputVersion:     19,
		Props:            parseProps,
		Sources:          parseSources,
	})
	if parseErr != nil {
		parseT.Fatalf("BuildBinarySnapshotBody returned error: %v", parseErr)
	}
	return parsePayload
}

// TestBuildBinaryCanonicalSourceIDsIntoAndWhitespaceGuards verifies source IDs are validated and sorted before encoding.
func TestBuildBinaryCanonicalSourceIDsIntoAndWhitespaceGuards(parseT *testing.T) {
	parseIDs, parseErr := buildBinaryCanonicalSourceIDsInto(nil, map[string]any{
		"beta":  1,
		"alpha": 2,
	})
	if parseErr != nil {
		parseT.Fatalf("buildBinaryCanonicalSourceIDsInto returned error: %v", parseErr)
	}
	if len(parseIDs) != 2 || parseIDs[0] != "alpha" || parseIDs[1] != "beta" {
		parseT.Fatalf("expected sorted source IDs [alpha beta], got %v", parseIDs)
	}

	parseCases := []struct {
		name      string
		sourceID  string
		wantError string
	}{
		{
			name:      "empty",
			sourceID:  "",
			wantError: "source ID is required",
		},
		{
			name:      "ascii-leading-whitespace",
			sourceID:  " alpha",
			wantError: "surrounding whitespace",
		},
		{
			name:      "ascii-trailing-whitespace",
			sourceID:  "alpha ",
			wantError: "surrounding whitespace",
		},
		{
			name:      "unicode-leading-whitespace",
			sourceID:  "\u2003alpha",
			wantError: "surrounding whitespace",
		},
		{
			name:      "unicode-trailing-whitespace",
			sourceID:  "alpha\u2003",
			wantError: "surrounding whitespace",
		},
		{
			name:      "unsupported-character",
			sourceID:  "alpha!",
			wantError: "unsupported character",
		},
	}
	for _, parseCase := range parseCases {
		parseT.Run(parseCase.name, func(parseT *testing.T) {
			parseT.Helper()
			_, parseErr := buildBinaryCanonicalSourceIDsInto(nil, map[string]any{
				parseCase.sourceID: true,
			})
			if parseErr == nil {
				parseT.Fatal("expected buildBinaryCanonicalSourceIDsInto to fail")
			}
			if !strings.Contains(parseErr.Error(), parseCase.wantError) {
				parseT.Fatalf("expected %q, got %v", parseCase.wantError, parseErr)
			}
		})
	}
}

// TestAppendBinarySourceValuesSectionAndParseBinarySourceValuesSection verifies source values round-trip and fail fast on malformed payloads.
func TestAppendBinarySourceValuesSectionAndParseBinarySourceValuesSection(parseT *testing.T) {
	parseSourceIDs := []string{"alpha", "beta"}
	parseSourceValues := map[string]any{
		"alpha": true,
		"beta":  "ok",
	}

	parsePayload, parseErr := appendBinarySourceValuesSection(nil, parseSourceIDs, parseSourceValues)
	if parseErr != nil {
		parseT.Fatalf("appendBinarySourceValuesSection returned error: %v", parseErr)
	}
	parseDecodedValues, parseErr := parseBinarySourceValuesSection(parseSourceIDs, parsePayload)
	if parseErr != nil {
		parseT.Fatalf("parseBinarySourceValuesSection returned error: %v", parseErr)
	}
	if len(parseDecodedValues) != 2 || parseDecodedValues["alpha"] != true || parseDecodedValues["beta"] != "ok" {
		parseT.Fatalf("unexpected decoded source values: %v", parseDecodedValues)
	}

	parseT.Run("missing-source-value", func(parseT *testing.T) {
		parseT.Helper()
		if _, parseErr := appendBinarySourceValuesSection(nil, []string{"alpha"}, map[string]any{}); parseErr == nil {
			parseT.Fatal("expected appendBinarySourceValuesSection to fail for missing source value")
		} else if !strings.Contains(parseErr.Error(), "missing source value") {
			parseT.Fatalf("expected missing source value error, got %v", parseErr)
		}
	})

	parseT.Run("no-source-ids-with-payload", func(parseT *testing.T) {
		parseT.Helper()
		if _, parseErr := parseBinarySourceValuesSection(nil, []byte{1}); parseErr == nil {
			parseT.Fatal("expected parseBinarySourceValuesSection to fail for payload with no source IDs")
		} else if !strings.Contains(parseErr.Error(), "no source IDs") {
			parseT.Fatalf("expected no-source-ids error, got %v", parseErr)
		}
	})

	parseT.Run("truncated-length-prefix", func(parseT *testing.T) {
		parseT.Helper()
		if _, parseErr := parseBinarySourceValuesSection([]string{"alpha"}, []byte{1, 2, 3}); parseErr == nil {
			parseT.Fatal("expected parseBinarySourceValuesSection to fail for truncated length prefix")
		} else if !strings.Contains(parseErr.Error(), "payload is truncated") {
			parseT.Fatalf("expected truncated-length error, got %v", parseErr)
		}
	})

	parseT.Run("length-exceeds-payload", func(parseT *testing.T) {
		parseT.Helper()
		if _, parseErr := parseBinarySourceValuesSection([]string{"alpha"}, []byte{4, 0, 0, 0}); parseErr == nil {
			parseT.Fatal("expected parseBinarySourceValuesSection to fail for oversized payload length")
		} else if !strings.Contains(parseErr.Error(), "exceeds payload size") {
			parseT.Fatalf("expected length-exceeds error, got %v", parseErr)
		}
	})

	parseT.Run("unsupported-kind", func(parseT *testing.T) {
		parseT.Helper()
		parseMalformedPayload := []byte{1, 0, 0, 0, 99}
		if _, parseErr := parseBinarySourceValuesSection([]string{"alpha"}, parseMalformedPayload); parseErr == nil {
			parseT.Fatal("expected parseBinarySourceValuesSection to fail for unsupported source value kind")
		} else if !strings.Contains(parseErr.Error(), "unsupported") {
			parseT.Fatalf("expected unsupported-kind error, got %v", parseErr)
		}
	})

	parseT.Run("trailing-bytes", func(parseT *testing.T) {
		parseT.Helper()
		parseMalformedPayload := []byte{1, 0, 0, 0, binarySourceValueKindBoolTrue, 0}
		if _, parseErr := parseBinarySourceValuesSection([]string{"alpha"}, parseMalformedPayload); parseErr == nil {
			parseT.Fatal("expected parseBinarySourceValuesSection to fail for trailing bytes")
		} else if !strings.Contains(parseErr.Error(), "trailing bytes") {
			parseT.Fatalf("expected trailing-bytes error, got %v", parseErr)
		}
	})
}

// TestParseBinarySnapshotBodyEdgeCases verifies malformed snapshot bodies fail at the expected decode stage.
func TestParseBinarySnapshotBodyEdgeCases(parseT *testing.T) {
	parseValidMinimalPayload := buildBinarySnapshotBodyTestPayload(parseT, nil, nil)
	parseDecodedEnvelope, parseErr := ParseBinarySnapshotBody(parseValidMinimalPayload)
	if parseErr != nil {
		parseT.Fatalf("ParseBinarySnapshotBody returned error: %v", parseErr)
	}
	if parseDecodedEnvelope.RegionInstanceID != RegionInstanceID("region-1") || parseDecodedEnvelope.Epoch != 7 || parseDecodedEnvelope.InputVersion != 19 {
		parseT.Fatalf("unexpected decoded envelope: %+v", parseDecodedEnvelope)
	}
	if parseDecodedEnvelope.Props != nil || len(parseDecodedEnvelope.Sources) != 0 {
		parseT.Fatalf("expected nil props and no sources, got %+v", parseDecodedEnvelope)
	}

	const (
		parseRegionIDLengthFieldSize = 2
		parseUint64FieldSize         = 8
		parseUint32FieldSize         = 4
		parseRegionIDLengthValue     = len("region-1")
		parsePropsBodyLength         = 1
		parseSourceIDTableLength     = 2
		parseSourceValuesLength      = 0
	)
	parsePropsLengthOffset := parseRegionIDLengthFieldSize + parseRegionIDLengthValue + 3*parseUint64FieldSize
	parsePropsBodyOffset := parsePropsLengthOffset + parseUint32FieldSize
	parseSourceIDTableLengthOffset := parsePropsBodyOffset + parsePropsBodyLength
	parseSourceIDTableBodyOffset := parseSourceIDTableLengthOffset + parseUint32FieldSize
	parseSourceValuesLengthOffset := parseSourceIDTableBodyOffset + parseSourceIDTableLength

	parseT.Run("region-length-truncated", func(parseT *testing.T) {
		parseT.Helper()
		if _, parseErr := ParseBinarySnapshotBody(parseValidMinimalPayload[:1]); parseErr == nil {
			parseT.Fatal("expected ParseBinarySnapshotBody to fail for truncated region length")
		} else if !strings.Contains(parseErr.Error(), "region_instance_id: length is truncated") {
			parseT.Fatalf("expected truncated region-length error, got %v", parseErr)
		}
	})

	parseT.Run("region-length-exceeds-payload", func(parseT *testing.T) {
		parseT.Helper()
		parseMalformedPayload := append([]byte(nil), parseValidMinimalPayload...)
		binary.LittleEndian.PutUint16(parseMalformedPayload[:2], 0xFFFF)
		if _, parseErr := ParseBinarySnapshotBody(parseMalformedPayload); parseErr == nil {
			parseT.Fatal("expected ParseBinarySnapshotBody to fail for oversized region length")
		} else if !strings.Contains(parseErr.Error(), "region_instance_id: length 65535 exceeds payload size") {
			parseT.Fatalf("expected oversized region-length error, got %v", parseErr)
		}
	})

	parseT.Run("props-decode-fails", func(parseT *testing.T) {
		parseT.Helper()
		parseMalformedPayload := append([]byte(nil), parseValidMinimalPayload...)
		parseMalformedPayload[parsePropsBodyOffset] = 99
		if _, parseErr := ParseBinarySnapshotBody(parseMalformedPayload); parseErr == nil {
			parseT.Fatal("expected ParseBinarySnapshotBody to fail for malformed props payload")
		} else if !strings.Contains(parseErr.Error(), "decode snapshot props") {
			parseT.Fatalf("expected props decode error, got %v", parseErr)
		}
	})

	parseT.Run("props-length-exceeds-payload", func(parseT *testing.T) {
		parseT.Helper()
		parseMalformedPayload := append([]byte(nil), parseValidMinimalPayload...)
		parseRemainingPayload := len(parseMalformedPayload) - parsePropsBodyOffset
		binary.LittleEndian.PutUint32(parseMalformedPayload[parsePropsLengthOffset:], uint32(parseRemainingPayload+1))
		if _, parseErr := ParseBinarySnapshotBody(parseMalformedPayload); parseErr == nil {
			parseT.Fatal("expected ParseBinarySnapshotBody to fail for oversized props length")
		} else if !strings.Contains(parseErr.Error(), "props payload: length") {
			parseT.Fatalf("expected oversized props-length error, got %v", parseErr)
		}
	})

	parseT.Run("source-id-table-length-truncated", func(parseT *testing.T) {
		parseT.Helper()
		if _, parseErr := ParseBinarySnapshotBody(parseValidMinimalPayload[:parseSourceIDTableLengthOffset]); parseErr == nil {
			parseT.Fatal("expected ParseBinarySnapshotBody to fail for truncated source-id-table length")
		} else if !strings.Contains(parseErr.Error(), "source-id-table length: payload is truncated") {
			parseT.Fatalf("expected truncated source-id-table-length error, got %v", parseErr)
		}
	})

	parseT.Run("source-id-table-length-exceeds-payload", func(parseT *testing.T) {
		parseT.Helper()
		parseMalformedPayload := append([]byte(nil), parseValidMinimalPayload...)
		parseRemainingPayload := len(parseMalformedPayload) - parseSourceIDTableBodyOffset
		binary.LittleEndian.PutUint32(parseMalformedPayload[parseSourceIDTableLengthOffset:], uint32(parseRemainingPayload+1))
		if _, parseErr := ParseBinarySnapshotBody(parseMalformedPayload); parseErr == nil {
			parseT.Fatal("expected ParseBinarySnapshotBody to fail for oversized source-id-table length")
		} else if !strings.Contains(parseErr.Error(), "source-id-table payload: length") {
			parseT.Fatalf("expected oversized source-id-table-length error, got %v", parseErr)
		}
	})

	parseT.Run("source-values-length-truncated", func(parseT *testing.T) {
		parseT.Helper()
		if _, parseErr := ParseBinarySnapshotBody(parseValidMinimalPayload[:parseSourceValuesLengthOffset]); parseErr == nil {
			parseT.Fatal("expected ParseBinarySnapshotBody to fail for truncated source-values length")
		} else if !strings.Contains(parseErr.Error(), "source-values length: payload is truncated") {
			parseT.Fatalf("expected truncated source-values-length error, got %v", parseErr)
		}
	})

	parseT.Run("source-values-length-exceeds-payload", func(parseT *testing.T) {
		parseT.Helper()
		parseMalformedPayload := append([]byte(nil), parseValidMinimalPayload...)
		binary.LittleEndian.PutUint32(parseMalformedPayload[parseSourceValuesLengthOffset:], 1)
		if _, parseErr := ParseBinarySnapshotBody(parseMalformedPayload); parseErr == nil {
			parseT.Fatal("expected ParseBinarySnapshotBody to fail for oversized source-values length")
		} else if !strings.Contains(parseErr.Error(), "source-values payload: length") {
			parseT.Fatalf("expected oversized source-values-length error, got %v", parseErr)
		}
	})

	parseT.Run("source-values-decode-fails", func(parseT *testing.T) {
		parseT.Helper()
		parseOneSourcePayload := buildBinarySnapshotBodyTestPayload(parseT, nil, map[string]any{
			"alpha": true,
		})
		parseOneSourcePropsLengthOffset := parseRegionIDLengthFieldSize + parseRegionIDLengthValue + 3*parseUint64FieldSize
		parseOneSourcePropsBodyOffset := parseOneSourcePropsLengthOffset + parseUint32FieldSize
		parseOneSourceSourceIDTableLengthOffset := parseOneSourcePropsBodyOffset + parsePropsBodyLength
		parseOneSourceSourceIDTableBodyOffset := parseOneSourceSourceIDTableLengthOffset + parseUint32FieldSize
		parseOneSourceSourceValuesLengthOffset := parseOneSourceSourceIDTableBodyOffset + (parseSourceIDTableLength + 7)
		parseMalformedPayload := append([]byte(nil), parseOneSourcePayload...)
		parseMalformedPayload[parseOneSourceSourceValuesLengthOffset+4] = 99
		if _, parseErr := ParseBinarySnapshotBody(parseMalformedPayload); parseErr == nil {
			parseT.Fatal("expected ParseBinarySnapshotBody to fail for malformed source-values payload")
		} else if !strings.Contains(parseErr.Error(), "decode snapshot source-values") {
			parseT.Fatalf("expected source-values decode error, got %v", parseErr)
		}
	})

	parseT.Run("source-id-table-decode-fails", func(parseT *testing.T) {
		parseT.Helper()
		parseOneSourcePayload := buildBinarySnapshotBodyTestPayload(parseT, nil, map[string]any{
			"alpha": true,
		})
		parseOneSourcePropsLengthOffset := parseRegionIDLengthFieldSize + parseRegionIDLengthValue + 3*parseUint64FieldSize
		parseOneSourcePropsBodyOffset := parseOneSourcePropsLengthOffset + parseUint32FieldSize
		parseOneSourceSourceIDTableLengthOffset := parseOneSourcePropsBodyOffset + parsePropsBodyLength
		parseOneSourceSourceIDTableBodyOffset := parseOneSourceSourceIDTableLengthOffset + parseUint32FieldSize
		parseOneSourceSourceIDKeyOffset := parseOneSourceSourceIDTableBodyOffset + 4
		parseMalformedPayload := append([]byte(nil), parseOneSourcePayload...)
		parseMalformedPayload[parseOneSourceSourceIDKeyOffset] = ' '
		if _, parseErr := ParseBinarySnapshotBody(parseMalformedPayload); parseErr == nil {
			parseT.Fatal("expected ParseBinarySnapshotBody to fail for malformed source-id table payload")
		} else if !strings.Contains(parseErr.Error(), "decode snapshot source-id table") {
			parseT.Fatalf("expected source-id-table decode error, got %v", parseErr)
		}
	})

	parseT.Run("trailing-bytes", func(parseT *testing.T) {
		parseT.Helper()
		parseMalformedPayload := append(append([]byte(nil), parseValidMinimalPayload...), 0)
		if _, parseErr := ParseBinarySnapshotBody(parseMalformedPayload); parseErr == nil {
			parseT.Fatal("expected ParseBinarySnapshotBody to fail for trailing bytes")
		} else if !strings.Contains(parseErr.Error(), "trailing bytes") {
			parseT.Fatalf("expected trailing-bytes error, got %v", parseErr)
		}
	})
}

// TestAppendBinaryLengthPrefixedStringRejectsOversizedValue verifies oversized strings are rejected before encoding.
func TestAppendBinaryLengthPrefixedStringRejectsOversizedValue(parseT *testing.T) {
	parseOversizedValue := strings.Repeat("a", 0x10000)
	if _, parseErr := appendBinaryLengthPrefixedString(nil, parseOversizedValue); parseErr == nil {
		parseT.Fatal("expected appendBinaryLengthPrefixedString to fail for oversized string")
	} else if !strings.Contains(parseErr.Error(), "too large") {
		parseT.Fatalf("expected oversized-string error, got %v", parseErr)
	}
}
