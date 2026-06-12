package runtime2

import (
	"math"
	"testing"
)

// TestSnapshotDispatchFastHashHasherHelpersWriteStableBytes verifies low-level fast-hash writer helpers accept empty and non-empty payloads and always produce a usable hash state.
func TestSnapshotDispatchFastHashHasherHelpersWriteStableBytes(parseT *testing.T) {
	if getFastHash := buildSnapshotDispatchFastHash(nil); getFastHash == 0 {
		parseT.Fatal("expected buffered fast hash to stay non-zero for empty payloads")
	}

	parseHasher := buildSnapshotDispatchFastHasher()
	if parseHasher == nil {
		parseT.Fatal("expected reusable fast hasher")
	}
	if parseErr := writeSnapshotDispatchFastHashByte(parseHasher, 1); parseErr != nil {
		parseT.Fatalf("writeSnapshotDispatchFastHashByte returned error: %v", parseErr)
	}
	if parseErr := writeSnapshotDispatchFastHashUint64(parseHasher, 7); parseErr != nil {
		parseT.Fatalf("writeSnapshotDispatchFastHashUint64 returned error: %v", parseErr)
	}
	if parseErr := writeSnapshotDispatchFastHashByteAndUint64(parseHasher, 2, 9); parseErr != nil {
		parseT.Fatalf("writeSnapshotDispatchFastHashByteAndUint64 returned error: %v", parseErr)
	}
	if parseErr := writeSnapshotDispatchFastHashByteAndByte(parseHasher, 3, 4); parseErr != nil {
		parseT.Fatalf("writeSnapshotDispatchFastHashByteAndByte returned error: %v", parseErr)
	}
	if parseErr := writeSnapshotDispatchFastHashString(parseHasher, "region-1"); parseErr != nil {
		parseT.Fatalf("writeSnapshotDispatchFastHashString(non-empty) returned error: %v", parseErr)
	}
	if parseErr := writeSnapshotDispatchFastHashString(parseHasher, ""); parseErr != nil {
		parseT.Fatalf("writeSnapshotDispatchFastHashString(empty) returned error: %v", parseErr)
	}
	if parseErr := writeSnapshotDispatchFastHashBytes(parseHasher, []byte{1, 2, 3}); parseErr != nil {
		parseT.Fatalf("writeSnapshotDispatchFastHashBytes(non-empty) returned error: %v", parseErr)
	}
	if parseErr := writeSnapshotDispatchFastHashBytes(parseHasher, nil); parseErr != nil {
		parseT.Fatalf("writeSnapshotDispatchFastHashBytes(empty) returned error: %v", parseErr)
	}
	if parseHasher.Sum64() == 0 {
		parseT.Fatal("expected helper writes to leave a non-zero fast hash state")
	}
	storeSnapshotDispatchFastHasher(parseHasher)
	storeSnapshotDispatchFastHasher(nil)
	if parseReusedHasher := buildSnapshotDispatchFastHasher(); parseReusedHasher == nil {
		parseT.Fatal("expected fast hasher to remain reusable after pool store")
	} else {
		storeSnapshotDispatchFastHasher(parseReusedHasher)
	}
}

// TestSnapshotDispatchFastHashValueCoversPrimitiveContainerAndFallbackBranches verifies the fast-hash value encoder covers primitive, container, reflect-fallback, and non-finite error branches.
func TestSnapshotDispatchFastHashValueCoversPrimitiveContainerAndFallbackBranches(parseT *testing.T) {
	parseValueTests := []struct {
		name       string
		parseValue any
		wantErr    bool
	}{
		{name: "nil", parseValue: nil},
		{name: "bool", parseValue: true},
		{name: "int", parseValue: int(-1)},
		{name: "int8", parseValue: int8(-2)},
		{name: "int16", parseValue: int16(-3)},
		{name: "int32", parseValue: int32(-4)},
		{name: "int64", parseValue: int64(-5)},
		{name: "uint", parseValue: uint(6)},
		{name: "uint8", parseValue: uint8(7)},
		{name: "uint16", parseValue: uint16(8)},
		{name: "uint32", parseValue: uint32(9)},
		{name: "uint64", parseValue: uint64(10)},
		{name: "uintptr", parseValue: uintptr(11)},
		{name: "float32", parseValue: float32(1.25)},
		{name: "float64", parseValue: 2.5},
		{name: "string", parseValue: "orders"},
		{name: "bool-list", parseValue: []bool{true, false}},
		{name: "int-list", parseValue: []int{1, 2, 3}},
		{name: "byte-list", parseValue: []byte{1, 2, 3}},
		{name: "uint16-list", parseValue: []uint16{4, 5}},
		{name: "uint32-list", parseValue: []uint32{6, 7}},
		{name: "uint64-list", parseValue: []uint64{8, 9}},
		{name: "uintptr-list", parseValue: []uintptr{10, 11}},
		{name: "float32-list", parseValue: []float32{1.5, 2.5}},
		{name: "string-list", parseValue: []string{"a", "b"}},
		{name: "any-list", parseValue: []any{1, "x", map[string]any{"ready": true}}},
		{name: "bool-map", parseValue: map[string]bool{"ready": true}},
		{name: "int-map", parseValue: map[string]int{"a": 1, "b": 2, "c": 3}},
		{name: "uint8-map", parseValue: map[string]uint8{"a": 1, "b": 2}},
		{name: "uint16-map", parseValue: map[string]uint16{"a": 3, "b": 4}},
		{name: "uint32-map", parseValue: map[string]uint32{"a": 5, "b": 6}},
		{name: "uint64-map", parseValue: map[string]uint64{"a": 7, "b": 8}},
		{name: "uintptr-map", parseValue: map[string]uintptr{"a": 9, "b": 10}},
		{name: "float32-map", parseValue: map[string]float32{"a": 1.25, "b": 2.5}},
		{name: "float64-map", parseValue: map[string]float64{"a": 3.75, "b": 4.5}},
		{name: "string-map", parseValue: map[string]string{"title": "Orders"}},
		{name: "any-map", parseValue: map[string]any{"z": 1, "a": "x"}},
		{name: "any-map-empty", parseValue: map[string]any{}},
		{name: "any-map-multi", parseValue: map[string]any{"c": 3, "a": "x", "b": true}},
		{name: "reflect-fallback", parseValue: struct{ Count int }{Count: 4}},
		{name: "non-finite", parseValue: math.NaN(), wantErr: true},
	}
	for _, parseValueTest := range parseValueTests {
		parseHasher := buildSnapshotDispatchFastHasher()
		_, parseErr := writeSnapshotDispatchFastHashValue(parseHasher, parseValueTest.parseValue, nil)
		if parseValueTest.wantErr {
			if parseErr == nil {
				parseT.Fatalf("%s expected fast-hash value encode to fail", parseValueTest.name)
			}
			storeSnapshotDispatchFastHasher(parseHasher)
			continue
		}
		if parseErr != nil {
			parseT.Fatalf("%s writeSnapshotDispatchFastHashValue returned error: %v", parseValueTest.name, parseErr)
		}
		if parseHasher.Sum64() == 0 {
			parseT.Fatalf("%s expected non-zero fast-hash state", parseValueTest.name)
		}
		storeSnapshotDispatchFastHasher(parseHasher)
	}

	parseHasher := buildSnapshotDispatchFastHasher()
	if parseErr := writeSnapshotDispatchFastHashFloat64(parseHasher, math.Inf(1)); parseErr == nil {
		parseT.Fatal("expected non-finite fast-hash float helper to fail")
	}
	storeSnapshotDispatchFastHasher(parseHasher)
}

// TestSnapshotDispatchFastHashMapHelpersCoverNilSinglePairOrderedAndEntryLayouts verifies map helpers cover nil, singleton, pair, ordered-key, and pre-built entry layouts.
func TestSnapshotDispatchFastHashMapHelpersCoverNilSinglePairOrderedAndEntryLayouts(parseT *testing.T) {
	parseHasher := buildSnapshotDispatchFastHasher()
	parseScratch, parseErr := writeSnapshotDispatchFastHashScalarMap(parseHasher, map[string]int(nil), nil)
	if parseErr != nil {
		parseT.Fatalf("writeSnapshotDispatchFastHashScalarMap(nil) returned error: %v", parseErr)
	}
	parseScratch, parseErr = writeSnapshotDispatchFastHashScalarMapSingle(parseHasher, map[string]int{"count": 3}, parseScratch)
	if parseErr != nil {
		parseT.Fatalf("writeSnapshotDispatchFastHashScalarMapSingle returned error: %v", parseErr)
	}
	parseScratch, parseErr = writeSnapshotDispatchFastHashScalarMapSingle(parseHasher, map[string]int{}, parseScratch)
	if parseErr != nil {
		parseT.Fatalf("writeSnapshotDispatchFastHashScalarMapSingle(empty) returned error: %v", parseErr)
	}
	parseScratch, parseErr = writeSnapshotDispatchFastHashScalarMapPair(parseHasher, map[string]int{"b": 2, "a": 1}, parseScratch)
	if parseErr != nil {
		parseT.Fatalf("writeSnapshotDispatchFastHashScalarMapPair returned error: %v", parseErr)
	}
	parseScratch, parseErr = writeSnapshotDispatchFastHashScalarMap(parseHasher, map[string]int{"c": 3, "a": 1, "b": 2}, parseScratch)
	if parseErr != nil {
		parseT.Fatalf("writeSnapshotDispatchFastHashScalarMap(multi) returned error: %v", parseErr)
	}
	parseScratch, parseErr = writeSnapshotDispatchFastHashAnyMap(parseHasher, nil, nil, parseScratch)
	if parseErr != nil {
		parseT.Fatalf("writeSnapshotDispatchFastHashAnyMap(nil) returned error: %v", parseErr)
	}
	parseScratch, parseErr = writeSnapshotDispatchFastHashAnyMap(parseHasher, map[string]any{"count": 3}, nil, parseScratch)
	if parseErr != nil {
		parseT.Fatalf("writeSnapshotDispatchFastHashAnyMap(single) returned error: %v", parseErr)
	}
	parseScratch, parseErr = writeSnapshotDispatchFastHashAnyMap(parseHasher, map[string]any{}, nil, parseScratch)
	if parseErr != nil {
		parseT.Fatalf("writeSnapshotDispatchFastHashAnyMap(empty) returned error: %v", parseErr)
	}
	parseScratch, parseErr = writeSnapshotDispatchFastHashAnyMap(parseHasher, map[string]any{"b": 2, "a": 1}, nil, parseScratch)
	if parseErr != nil {
		parseT.Fatalf("writeSnapshotDispatchFastHashAnyMap(pair) returned error: %v", parseErr)
	}
	parseScratch, parseErr = writeSnapshotDispatchFastHashAnyMap(parseHasher, map[string]any{"c": 3, "a": 1, "b": 2}, nil, parseScratch)
	if parseErr != nil {
		parseT.Fatalf("writeSnapshotDispatchFastHashAnyMap(multi) returned error: %v", parseErr)
	}
	parseScratch, parseErr = writeSnapshotDispatchFastHashAnyMap(parseHasher, map[string]any{"c": 3, "a": 1, "b": 2}, []string{"a", "b", "c"}, parseScratch)
	if parseErr != nil {
		parseT.Fatalf("writeSnapshotDispatchFastHashAnyMap(ordered) returned error: %v", parseErr)
	}
	parseScratch, parseErr = writeSnapshotDispatchFastHashAnyMapEntries(parseHasher, []buildSnapshotDispatchMapEntry{
		{getKey: "alpha", getValue: true},
		{getKey: "beta", getValue: "ok"},
	}, parseScratch)
	if parseErr != nil {
		parseT.Fatalf("writeSnapshotDispatchFastHashAnyMapEntries returned error: %v", parseErr)
	}
	parseScratch, parseErr = writeSnapshotDispatchFastHashAnyMapEntries(parseHasher, nil, parseScratch)
	if parseErr != nil {
		parseT.Fatalf("writeSnapshotDispatchFastHashAnyMapEntries(nil) returned error: %v", parseErr)
	}
	parseScratch, parseErr = writeSnapshotDispatchFastHashAnyMapEntry(parseHasher, "title", "Orders", parseScratch)
	if parseErr != nil {
		parseT.Fatalf("writeSnapshotDispatchFastHashAnyMapEntry returned error: %v", parseErr)
	}
	parseScratch, parseErr = writeSnapshotDispatchFastHashAnyMapEntry(parseHasher, "", true, parseScratch)
	if parseErr != nil {
		parseT.Fatalf("writeSnapshotDispatchFastHashAnyMapEntry(empty key) returned error: %v", parseErr)
	}
	if _, parseErr := writeSnapshotDispatchFastHashAnyMapOrdered(parseHasher, map[string]any{"title": "Orders"}, []string{"missing"}, parseScratch); parseErr == nil {
		parseT.Fatal("expected ordered fast-hash map helper to fail for missing key")
	}
	if parseHasher.Sum64() == 0 {
		parseT.Fatal("expected map helper writes to leave a non-zero fast-hash state")
	}
	storeSnapshotDispatchFastHasher(parseHasher)
}

// TestSnapshotDispatchFastHashEnvelopeAndPropsHelpers verifies envelope and props helpers honor pre-sorted source IDs, ordered prop keys, and pre-built prop entries.
func TestSnapshotDispatchFastHashEnvelopeAndPropsHelpers(parseT *testing.T) {
	parseEnvelope := SnapshotEnvelope{
		RegionInstanceID: RegionInstanceID("region-1"),
		Epoch:            7,
		InputVersion:     9,
		SourceVersion:    11,
		Props:            map[string]any{"title": "Orders", "count": 3},
		Sources:          map[string]any{"status": "ready", "count": 3},
	}
	parseFastHash, parseScratch, parseErr := buildSnapshotDispatchFastHashIntoWithSourceAndPropsLayout(
		parseEnvelope,
		[]string{"count", "status"},
		[]string{"count", "title"},
		nil,
		nil,
	)
	if parseErr != nil {
		parseT.Fatalf("buildSnapshotDispatchFastHashIntoWithSourceAndPropsLayout(ordered keys) returned error: %v", parseErr)
	}
	if parseFastHash == 0 || len(parseScratch) != 0 {
		parseT.Fatalf("expected non-zero fast hash with recycled scratch, got hash=%d scratch=%d", parseFastHash, len(parseScratch))
	}
	parseFastHash, parseScratch, parseErr = buildSnapshotDispatchFastHashIntoWithSourceAndPropsEntries(
		parseEnvelope,
		[]string{"count", "status"},
		[]buildSnapshotDispatchMapEntry{
			{getKey: "count", getValue: 3},
			{getKey: "title", getValue: "Orders"},
		},
		parseScratch,
	)
	if parseErr != nil {
		parseT.Fatalf("buildSnapshotDispatchFastHashIntoWithSourceAndPropsEntries returned error: %v", parseErr)
	}
	if parseFastHash == 0 || len(parseScratch) != 0 {
		parseT.Fatalf("expected non-zero fast hash with entry layout, got hash=%d scratch=%d", parseFastHash, len(parseScratch))
	}

	parseHasher := buildSnapshotDispatchFastHasher()
	parseScratch, parseErr = writeSnapshotDispatchFastHashEnvelopeWithSourceAndPropsKeys(
		parseHasher,
		parseEnvelope,
		[]string{"count", "status"},
		[]string{"count", "title"},
		nil,
		parseScratch,
	)
	if parseErr != nil {
		parseT.Fatalf("writeSnapshotDispatchFastHashEnvelopeWithSourceAndPropsKeys(ordered keys) returned error: %v", parseErr)
	}
	parseScratch, parseErr = writeSnapshotDispatchFastHashEnvelopeWithSourceAndPropsKeys(
		parseHasher,
		parseEnvelope,
		[]string{"count", "status"},
		nil,
		[]buildSnapshotDispatchMapEntry{
			{getKey: "count", getValue: 3},
			{getKey: "title", getValue: "Orders"},
		},
		parseScratch,
	)
	if parseErr != nil {
		parseT.Fatalf("writeSnapshotDispatchFastHashEnvelopeWithSourceAndPropsKeys(entries) returned error: %v", parseErr)
	}
	_, parseErr = writeSnapshotDispatchFastHashEnvelopeWithSourceAndPropsKeys(
		parseHasher,
		SnapshotEnvelope{
			RegionInstanceID: RegionInstanceID("region-2"),
			Epoch:            5,
			InputVersion:     6,
			SourceVersion:    7,
			Props:            struct{ Count int }{Count: 4},
		},
		nil,
		nil,
		nil,
		parseScratch,
	)
	if parseErr != nil {
		parseT.Fatalf("writeSnapshotDispatchFastHashEnvelopeWithSourceAndPropsKeys(fallback props) returned error: %v", parseErr)
	}
	if parseHasher.Sum64() == 0 {
		parseT.Fatal("expected envelope fast-hash writes to leave a non-zero hash state")
	}
	storeSnapshotDispatchFastHasher(parseHasher)
}
