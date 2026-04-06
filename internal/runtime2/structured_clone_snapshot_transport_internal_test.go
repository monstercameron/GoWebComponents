package runtime2

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestStructuredCloneSnapshotTransportJSONHelpers verifies the structured-clone snapshot builder and parser hit both the header-only and extended-section paths.
func TestStructuredCloneSnapshotTransportJSONHelpers(parseT *testing.T) {
	parseHeaderEnvelope := SnapshotEnvelope{
		RegionInstanceID: RegionInstanceID("region-1"),
		Epoch:            7,
		InputVersion:     9,
	}
	parseHeaderPayload, parseErr := BuildStructuredCloneSnapshotEnvelopeJSON(parseHeaderEnvelope)
	if parseErr != nil {
		parseT.Fatalf("BuildStructuredCloneSnapshotEnvelopeJSON(header-only) returned error: %v", parseErr)
	}
	const parseHeaderWant = `{"region_instance_id":"region-1","epoch":7,"input_version":9}`
	if string(parseHeaderPayload) != parseHeaderWant {
		parseT.Fatalf("expected header-only payload %s, got %s", parseHeaderWant, string(parseHeaderPayload))
	}
	if parseLen := getStructuredCloneSnapshotEnvelopeJSONLength(parseHeaderEnvelope, 0, 0); parseLen < len(parseHeaderPayload) {
		parseT.Fatalf("expected header-only length hint at least %d, got %d", len(parseHeaderPayload), parseLen)
	}
	parseHeaderDecoded, parseErr := ParseStructuredCloneSnapshotEnvelopeJSON(parseHeaderPayload)
	if parseErr != nil {
		parseT.Fatalf("ParseStructuredCloneSnapshotEnvelopeJSON(header-only) returned error: %v", parseErr)
	}
	if parseHeaderDecoded.RegionInstanceID != parseHeaderEnvelope.RegionInstanceID ||
		parseHeaderDecoded.Epoch != parseHeaderEnvelope.Epoch ||
		parseHeaderDecoded.InputVersion != parseHeaderEnvelope.InputVersion ||
		parseHeaderDecoded.SourceVersion != parseHeaderEnvelope.SourceVersion ||
		parseHeaderDecoded.Props != nil ||
		parseHeaderDecoded.Sources != nil {
		parseT.Fatalf("expected header-only round-trip %+v, got %+v", parseHeaderEnvelope, parseHeaderDecoded)
	}
	parseHeaderHelperDecoded, parseOk := parseStructuredCloneSnapshotEnvelopeHeaderOnly(parseHeaderPayload)
	if !parseOk {
		parseT.Fatal("expected parseStructuredCloneSnapshotEnvelopeHeaderOnly to accept header-only payload")
	}
	if parseHeaderHelperDecoded.RegionInstanceID != parseHeaderEnvelope.RegionInstanceID ||
		parseHeaderHelperDecoded.Epoch != parseHeaderEnvelope.Epoch ||
		parseHeaderHelperDecoded.InputVersion != parseHeaderEnvelope.InputVersion ||
		parseHeaderHelperDecoded.SourceVersion != parseHeaderEnvelope.SourceVersion {
		parseT.Fatalf("expected header-only helper decode %+v, got %+v", parseHeaderEnvelope, parseHeaderHelperDecoded)
	}

	parseRichEnvelope := SnapshotEnvelope{
		RegionInstanceID: RegionInstanceID(`region-"1`),
		Epoch:            11,
		InputVersion:     13,
		SourceVersion:    17,
		Props:            map[string]any{"title": "Orders"},
		Sources:          map[string]any{"status": "healthy"},
	}
	parsePropsPayload := []byte(`{"title":"Orders"}`)
	parseSourcesPayload := []byte(`{"status":"healthy"}`)
	parseRichPayload, parseErr := BuildStructuredCloneSnapshotEnvelopeJSON(parseRichEnvelope)
	if parseErr != nil {
		parseT.Fatalf("BuildStructuredCloneSnapshotEnvelopeJSON(rich) returned error: %v", parseErr)
	}
	parseRichDecoded, parseErr := ParseStructuredCloneSnapshotEnvelopeJSON(parseRichPayload)
	if parseErr != nil {
		parseT.Fatalf("ParseStructuredCloneSnapshotEnvelopeJSON(rich) returned error: %v", parseErr)
	}
	if parseRichDecoded.RegionInstanceID != parseRichEnvelope.RegionInstanceID ||
		parseRichDecoded.Epoch != parseRichEnvelope.Epoch ||
		parseRichDecoded.InputVersion != parseRichEnvelope.InputVersion ||
		parseRichDecoded.SourceVersion != parseRichEnvelope.SourceVersion {
		parseT.Fatalf("expected rich round-trip %+v, got %+v", parseRichEnvelope, parseRichDecoded)
	}
	parseAppendPayload := appendStructuredCloneSnapshotEnvelopeJSON(nil, parseRichEnvelope, parsePropsPayload, parseSourcesPayload)
	parseAppendWant := `{"region_instance_id":"region-\"1","epoch":11,"input_version":13,"source_version":17,"props":{"title":"Orders"},"sources":{"status":"healthy"}}`
	if string(parseAppendPayload) != parseAppendWant {
		parseT.Fatalf("expected rich append payload %s, got %s", parseAppendWant, string(parseAppendPayload))
	}
	if parseLen := getStructuredCloneSnapshotEnvelopeJSONLength(parseRichEnvelope, len(parsePropsPayload), len(parseSourcesPayload)); parseLen < len(parseRichPayload) {
		parseT.Fatalf("expected rich length hint at least %d, got %d", len(parseRichPayload), parseLen)
	}
	parseRichHelperDecoded, parseOk := parseStructuredCloneSnapshotEnvelopeHeaderOnly(appendStructuredCloneSnapshotEnvelopeJSON(nil, SnapshotEnvelope{
		RegionInstanceID: RegionInstanceID(`region-\test`),
		Epoch:            21,
		InputVersion:     23,
		SourceVersion:    29,
	}, nil, nil))
	if !parseOk {
		parseT.Fatal("expected parseStructuredCloneSnapshotEnvelopeHeaderOnly to accept escaped payload")
	}
	if parseRichHelperDecoded.RegionInstanceID != RegionInstanceID(`region-\test`) || parseRichHelperDecoded.SourceVersion != 29 {
		parseT.Fatalf("expected escaped header decode to preserve values, got %+v", parseRichHelperDecoded)
	}
}

// TestStructuredCloneSnapshotTransportParseAndValidationHelpers verifies the low-level structured-clone JSON scanners and field guards.
func TestStructuredCloneSnapshotTransportParseAndValidationHelpers(parseT *testing.T) {
	parseT.Run("quoted-text-span", func(parseT *testing.T) {
		parseStart, parseEnd, parseOffset, parseHasEscape, parseOK := parseStructuredCloneSnapshotQuotedTextSpan([]byte(`"abc"`), 0)
		if !parseOK || parseStart != 1 || parseEnd != 4 || parseOffset != 5 || parseHasEscape {
			parseT.Fatalf("unexpected plain quoted span: start=%d end=%d offset=%d escape=%v ok=%v", parseStart, parseEnd, parseOffset, parseHasEscape, parseOK)
		}

		parseEscapedPayload := []byte{'"', 'a', '\\', '\\', '\\', '"', 'b', '"'}
		parseStart, parseEnd, parseOffset, parseHasEscape, parseOK = parseStructuredCloneSnapshotQuotedTextSpan(parseEscapedPayload, 0)
		if !parseOK || parseStart != 1 || parseEnd != 7 || parseOffset != 8 || !parseHasEscape {
			parseT.Fatalf("unexpected escaped quoted span: start=%d end=%d offset=%d escape=%v ok=%v", parseStart, parseEnd, parseOffset, parseHasEscape, parseOK)
		}

		if _, _, _, _, parseOK = parseStructuredCloneSnapshotQuotedTextSpan([]byte{'"', 'a', '\n', 'b', '"'}, 0); parseOK {
			parseT.Fatal("expected control characters inside quoted text to fail")
		}
		if _, _, _, _, parseOK = parseStructuredCloneSnapshotQuotedTextSpan([]byte(`"abc`), 0); parseOK {
			parseT.Fatal("expected unterminated quoted text to fail")
		}
	})

	parseT.Run("uint", func(parseT *testing.T) {
		parseValue, parseOffset, parseOK := parseStructuredCloneSnapshotUint([]byte("123,rest"), 0)
		if !parseOK || parseValue != 123 || parseOffset != 3 {
			parseT.Fatalf("unexpected parsed uint: value=%d offset=%d ok=%v", parseValue, parseOffset, parseOK)
		}
		if _, _, parseOK := parseStructuredCloneSnapshotUint([]byte("-1"), 0); parseOK {
			parseT.Fatal("expected negative-looking input to fail")
		}
		if _, _, parseOK := parseStructuredCloneSnapshotUint([]byte(strings.Repeat("9", 21)), 0); parseOK {
			parseT.Fatal("expected uint overflow to fail")
		}
	})

	parseT.Run("object", func(parseT *testing.T) {
		parseObject, parseErr := parseStructuredCloneObject([]byte(`{"title":"Orders"}`))
		if parseErr != nil {
			parseT.Fatalf("parseStructuredCloneObject returned error: %v", parseErr)
		}
		if parseObject["title"][0] != '"' {
			parseT.Fatalf("expected parsed object value to remain raw JSON, got %q", string(parseObject["title"]))
		}
		if _, parseErr := parseStructuredCloneObject([]byte(`[]`)); parseErr == nil {
			parseT.Fatal("expected non-object JSON to fail")
		}
	})

	parseT.Run("field-type-guards", func(parseT *testing.T) {
		parseEnvelopeRaw := map[string]json.RawMessage{
			"string_field": json.RawMessage(`"value"`),
			"list_field":   json.RawMessage(`["value"]`),
			"object_field": json.RawMessage(`{"value":true}`),
		}
		if parseErr := validateStructuredCloneFieldType(parseEnvelopeRaw, "string_field", structuredCloneFieldString, false); parseErr != nil {
			parseT.Fatalf("expected string field to validate, got %v", parseErr)
		}
		if parseErr := validateStructuredCloneFieldType(parseEnvelopeRaw, "list_field", structuredCloneFieldList, false); parseErr != nil {
			parseT.Fatalf("expected list field to validate, got %v", parseErr)
		}
		if parseErr := validateStructuredCloneFieldType(parseEnvelopeRaw, "object_field", structuredCloneFieldObject, false); parseErr != nil {
			parseT.Fatalf("expected object field to validate, got %v", parseErr)
		}
		if parseErr := validateStructuredCloneFieldType(parseEnvelopeRaw, "missing_optional", structuredCloneFieldString, true); parseErr != nil {
			parseT.Fatalf("expected optional missing field to validate, got %v", parseErr)
		}
		if parseErr := validateStructuredCloneFieldType(parseEnvelopeRaw, "missing_required", structuredCloneFieldString, false); parseErr == nil {
			parseT.Fatal("expected required missing field to fail")
		}
		if parseErr := validateStructuredCloneFieldType(parseEnvelopeRaw, "list_field", structuredCloneFieldString, false); parseErr == nil {
			parseT.Fatal("expected list field where string required to fail")
		}
		if parseErr := validateStructuredCloneFieldType(parseEnvelopeRaw, "string_field", structuredCloneFieldList, false); parseErr == nil {
			parseT.Fatal("expected string field where list required to fail")
		}
		if parseErr := validateStructuredCloneFieldType(parseEnvelopeRaw, "string_field", structuredCloneFieldObject, false); parseErr == nil {
			parseT.Fatal("expected string field where object required to fail")
		}
		if parseErr := validateStructuredCloneFieldType(parseEnvelopeRaw, "string_field", structuredCloneFieldType(99), false); parseErr == nil {
			parseT.Fatal("expected unsupported field guard to fail")
		}
	})
}

// TestStructuredCloneSnapshotTransportMountAndUpdateValidation verifies mount and update envelopes enforce their own validation rules.
func TestStructuredCloneSnapshotTransportMountAndUpdateValidation(parseT *testing.T) {
	parseMountEnvelope := StructuredCloneMountEnvelope{
		RegionInstanceID: RegionInstanceID("region-1"),
		RendererID:       RendererID("dashboard.hot-panel"),
		SourceIDs:        []string{"beta", "alpha"},
		Snapshot: SnapshotEnvelope{
			RegionInstanceID: RegionInstanceID("region-1"),
			Epoch:            1,
			InputVersion:     2,
			Props:            map[string]any{"title": "Orders"},
		},
	}
	parseMountPayload, parseErr := BuildStructuredCloneMountEnvelopeJSON(parseMountEnvelope)
	if parseErr != nil {
		parseT.Fatalf("BuildStructuredCloneMountEnvelopeJSON returned error: %v", parseErr)
	}
	parseMountDecoded, parseErr := ParseStructuredCloneMountEnvelopeJSON(parseMountPayload)
	if parseErr != nil {
		parseT.Fatalf("ParseStructuredCloneMountEnvelopeJSON returned error: %v", parseErr)
	}
	if len(parseMountDecoded.SourceIDs) != 2 || parseMountDecoded.SourceIDs[0] != "alpha" || parseMountDecoded.SourceIDs[1] != "beta" {
		parseT.Fatalf("expected normalized source IDs [alpha beta], got %v", parseMountDecoded.SourceIDs)
	}
	parseMountMismatch := parseMountEnvelope
	parseMountMismatch.Snapshot.RegionInstanceID = RegionInstanceID("region-2")
	if _, parseErr := BuildStructuredCloneMountEnvelopeJSON(parseMountMismatch); parseErr == nil {
		parseT.Fatal("expected mount snapshot region mismatch to fail")
	}

	parseUpdateEnvelope := StructuredCloneUpdateEnvelope{
		RegionInstanceID: RegionInstanceID("region-1"),
		InputVersion:     2,
		Snapshot: SnapshotEnvelope{
			RegionInstanceID: RegionInstanceID("region-1"),
			Epoch:            1,
			InputVersion:     2,
			Props:            map[string]any{"title": "Orders"},
		},
	}
	parseUpdatePayload, parseErr := BuildStructuredCloneUpdateEnvelopeJSON(parseUpdateEnvelope)
	if parseErr != nil {
		parseT.Fatalf("BuildStructuredCloneUpdateEnvelopeJSON returned error: %v", parseErr)
	}
	parseUpdateDecoded, parseErr := ParseStructuredCloneUpdateEnvelopeJSON(parseUpdatePayload)
	if parseErr != nil {
		parseT.Fatalf("ParseStructuredCloneUpdateEnvelopeJSON returned error: %v", parseErr)
	}
	if parseUpdateDecoded.InputVersion != parseUpdateEnvelope.InputVersion {
		parseT.Fatalf("expected update input version %d, got %d", parseUpdateEnvelope.InputVersion, parseUpdateDecoded.InputVersion)
	}

	parseUpdateZero := parseUpdateEnvelope
	parseUpdateZero.InputVersion = 0
	if _, parseErr := BuildStructuredCloneUpdateEnvelopeJSON(parseUpdateZero); parseErr == nil {
		parseT.Fatal("expected zero update input version to fail")
	}
	parseUpdateRegionMismatch := parseUpdateEnvelope
	parseUpdateRegionMismatch.Snapshot.RegionInstanceID = RegionInstanceID("region-2")
	if _, parseErr := BuildStructuredCloneUpdateEnvelopeJSON(parseUpdateRegionMismatch); parseErr == nil {
		parseT.Fatal("expected update snapshot region mismatch to fail")
	}
	parseUpdateVersionMismatch := parseUpdateEnvelope
	parseUpdateVersionMismatch.Snapshot.InputVersion = 3
	if _, parseErr := BuildStructuredCloneUpdateEnvelopeJSON(parseUpdateVersionMismatch); parseErr == nil {
		parseT.Fatal("expected update snapshot input-version mismatch to fail")
	}

	parseUpdateParsePayload := []byte(`{"region_instance_id":"region-1","input_version":2,"snapshot":{"region_instance_id":"region-1","epoch":1,"input_version":2}}`)
	if _, parseErr := ParseStructuredCloneUpdateEnvelopeJSON(parseUpdateParsePayload); parseErr != nil {
		parseT.Fatalf("ParseStructuredCloneUpdateEnvelopeJSON(valid) returned error: %v", parseErr)
	}
	if _, parseErr := ParseStructuredCloneUpdateEnvelopeJSON([]byte(`{"region_instance_id":123,"input_version":2,"snapshot":{"region_instance_id":"region-1","epoch":1,"input_version":2}}`)); parseErr == nil {
		parseT.Fatal("expected numeric region_instance_id to fail")
	}
	if _, parseErr := ParseStructuredCloneUpdateEnvelopeJSON([]byte(`{"region_instance_id":"region-1","input_version":2,"snapshot":"bad"}`)); parseErr == nil {
		parseT.Fatal("expected non-object snapshot to fail")
	}
}
