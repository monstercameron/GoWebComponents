package runtime2

import (
	"crypto/sha256"
	"fmt"
	"math"
	"reflect"
	"testing"
)

// TestSnapshotDispatchHashComparatorAndPoolHelpers verifies the map-entry comparators, sort fast paths, buffer pools, string views, and hashers.
func TestSnapshotDispatchHashComparatorAndPoolHelpers(parseT *testing.T) {
	if got := compareSnapshotDispatchMapEntryByKey(buildSnapshotDispatchMapEntry{getKey: "a"}, buildSnapshotDispatchMapEntry{getKey: "b"}); got != -1 {
		parseT.Fatalf("expected a<b comparator result -1, got %d", got)
	}
	if got := compareSnapshotDispatchMapEntryByKey(buildSnapshotDispatchMapEntry{getKey: "b"}, buildSnapshotDispatchMapEntry{getKey: "b"}); got != 0 {
		parseT.Fatalf("expected equal comparator result 0, got %d", got)
	}
	if got := compareSnapshotDispatchMapEntryByKey(buildSnapshotDispatchMapEntry{getKey: "c"}, buildSnapshotDispatchMapEntry{getKey: "b"}); got != 1 {
		parseT.Fatalf("expected c>b comparator result 1, got %d", got)
	}

	parseSmallEntries := []buildSnapshotDispatchMapEntry{
		{getKey: "c"},
		{getKey: "a"},
		{getKey: "b"},
	}
	sortSnapshotDispatchMapEntriesByKey(parseSmallEntries)
	if parseSmallEntries[0].getKey != "a" || parseSmallEntries[1].getKey != "b" || parseSmallEntries[2].getKey != "c" {
		parseT.Fatalf("expected insertion-sort ordering, got %#v", parseSmallEntries)
	}

	parseLargeEntries := make([]buildSnapshotDispatchMapEntry, 0, 24)
	for parseIndex := 23; parseIndex >= 0; parseIndex-- {
		parseLargeEntries = append(parseLargeEntries, buildSnapshotDispatchMapEntry{getKey: fmt.Sprintf("k%02d", parseIndex)})
	}
	sortSnapshotDispatchMapEntriesByKey(parseLargeEntries)
	for parseIndex, parseEntry := range parseLargeEntries {
		if parseEntry.getKey != fmt.Sprintf("k%02d", parseIndex) {
			parseT.Fatalf("expected sorted key k%02d at index %d, got %#v", parseIndex, parseIndex, parseLargeEntries)
		}
	}

	if got := compareSnapshotDispatchReflectMapEntryByKey(buildSnapshotDispatchReflectMapEntry{getKey: "a"}, buildSnapshotDispatchReflectMapEntry{getKey: "b"}); got != -1 {
		parseT.Fatalf("expected reflect a<b comparator result -1, got %d", got)
	}
	if got := compareSnapshotDispatchReflectMapEntryByKey(buildSnapshotDispatchReflectMapEntry{getKey: "b"}, buildSnapshotDispatchReflectMapEntry{getKey: "b"}); got != 0 {
		parseT.Fatalf("expected reflect equal comparator result 0, got %d", got)
	}
	if got := compareSnapshotDispatchReflectMapEntryByKey(buildSnapshotDispatchReflectMapEntry{getKey: "c"}, buildSnapshotDispatchReflectMapEntry{getKey: "b"}); got != 1 {
		parseT.Fatalf("expected reflect c>b comparator result 1, got %d", got)
	}

	parseSmallReflectEntries := []buildSnapshotDispatchReflectMapEntry{
		{getKey: "c", getValue: reflect.ValueOf(3)},
		{getKey: "a", getValue: reflect.ValueOf(1)},
		{getKey: "b", getValue: reflect.ValueOf(2)},
	}
	sortSnapshotDispatchReflectMapEntriesByKey(parseSmallReflectEntries)
	if parseSmallReflectEntries[0].getKey != "a" || parseSmallReflectEntries[1].getKey != "b" || parseSmallReflectEntries[2].getKey != "c" {
		parseT.Fatalf("expected reflect insertion-sort ordering, got %#v", parseSmallReflectEntries)
	}

	parseLargeReflectEntries := make([]buildSnapshotDispatchReflectMapEntry, 0, 24)
	for parseIndex := 23; parseIndex >= 0; parseIndex-- {
		parseLargeReflectEntries = append(parseLargeReflectEntries, buildSnapshotDispatchReflectMapEntry{
			getKey:   fmt.Sprintf("k%02d", parseIndex),
			getValue: reflect.ValueOf(parseIndex),
		})
	}
	sortSnapshotDispatchReflectMapEntriesByKey(parseLargeReflectEntries)
	for parseIndex, parseEntry := range parseLargeReflectEntries {
		if parseEntry.getKey != fmt.Sprintf("k%02d", parseIndex) {
			parseT.Fatalf("expected sorted reflect key k%02d at index %d, got %#v", parseIndex, parseIndex, parseLargeReflectEntries)
		}
	}

	parseEntries, parseCache := buildSnapshotDispatchMapEntryBuffer(16)
	if cap(parseEntries) < 16 {
		parseT.Fatalf("expected map-entry buffer capacity >= 16, got %d", cap(parseEntries))
	}
	parseEntries = append(parseEntries, buildSnapshotDispatchMapEntry{getKey: "alpha", getValue: true})
	storeSnapshotDispatchMapEntryBuffer(parseEntries, parseCache)
	if len(parseCache.getEntries) != 0 {
		parseT.Fatalf("expected cleared map-entry cache, got len=%d", len(parseCache.getEntries))
	}
	storeSnapshotDispatchMapEntryBuffer(nil, nil)

	parseOriginalMapEntryPoolNew := storeSnapshotDispatchMapEntryPool.New
	storeSnapshotDispatchMapEntryPool.New = func() any { return nil }
	parseFallbackEntries, parseFallbackCache := buildSnapshotDispatchMapEntryBuffer(4)
	storeSnapshotDispatchMapEntryPool.New = parseOriginalMapEntryPoolNew
	if cap(parseFallbackEntries) < 4 || parseFallbackCache == nil {
		parseT.Fatalf("expected fallback map-entry buffer and cache, got cap=%d cache=%#v", cap(parseFallbackEntries), parseFallbackCache)
	}
	storeSnapshotDispatchMapEntryBuffer(parseFallbackEntries, parseFallbackCache)

	parseHasher := buildSnapshotDispatchHasher()
	if parseHasher == nil {
		parseT.Fatal("expected snapshot dispatch hasher")
	}
	storeSnapshotDispatchHasher(parseHasher)
	storeSnapshotDispatchHasher(nil)

	parseOriginalHasherPoolNew := storeSnapshotDispatchHasherPool.New
	storeSnapshotDispatchHasherPool.New = func() any { return nil }
	parseFallbackHasher := buildSnapshotDispatchHasher()
	storeSnapshotDispatchHasherPool.New = parseOriginalHasherPoolNew
	if parseFallbackHasher == nil {
		parseT.Fatal("expected fallback snapshot dispatch hasher")
	}
	storeSnapshotDispatchHasher(parseFallbackHasher)

	if got := getSnapshotDispatchStringBytes(""); got != nil {
		parseT.Fatalf("expected empty string view to be nil, got %#v", got)
	}
	if got := getSnapshotDispatchStringBytes("abc"); string(got) != "abc" {
		parseT.Fatalf("expected string view abc, got %q", string(got))
	}
}

// TestSnapshotDispatchHashValueSwitches verifies the appendSnapshotDispatchValue type switch covers scalar, list, map, and fallback paths.
func TestSnapshotDispatchHashValueSwitches(parseT *testing.T) {
	parseCases := []struct {
		name       string
		value      any
		wantMarker byte
		wantErr    bool
	}{
		{name: "nil", value: nil, wantMarker: getSnapshotDispatchHashMarkerNil},
		{name: "bool-true", value: true, wantMarker: getSnapshotDispatchHashMarkerBool},
		{name: "bool-false", value: false, wantMarker: getSnapshotDispatchHashMarkerBool},
		{name: "int", value: int(-3), wantMarker: getSnapshotDispatchHashMarkerNumber},
		{name: "int8", value: int8(-4), wantMarker: getSnapshotDispatchHashMarkerNumber},
		{name: "int16", value: int16(-5), wantMarker: getSnapshotDispatchHashMarkerNumber},
		{name: "int32", value: int32(-6), wantMarker: getSnapshotDispatchHashMarkerNumber},
		{name: "int64", value: int64(-7), wantMarker: getSnapshotDispatchHashMarkerNumber},
		{name: "uint", value: uint(8), wantMarker: getSnapshotDispatchHashMarkerNumber},
		{name: "uint8", value: uint8(9), wantMarker: getSnapshotDispatchHashMarkerNumber},
		{name: "uint16", value: uint16(10), wantMarker: getSnapshotDispatchHashMarkerNumber},
		{name: "uint32", value: uint32(11), wantMarker: getSnapshotDispatchHashMarkerNumber},
		{name: "uint64", value: uint64(12), wantMarker: getSnapshotDispatchHashMarkerNumber},
		{name: "uintptr", value: uintptr(13), wantMarker: getSnapshotDispatchHashMarkerNumber},
		{name: "float32", value: float32(14.5), wantMarker: getSnapshotDispatchHashMarkerNumber},
		{name: "float64", value: float64(15.5), wantMarker: getSnapshotDispatchHashMarkerNumber},
		{name: "string", value: "ok", wantMarker: getSnapshotDispatchHashMarkerString},
		{name: "empty-bool-slice", value: []bool{}, wantMarker: getSnapshotDispatchHashMarkerList},
		{name: "bool-slice", value: []bool{true, false}, wantMarker: getSnapshotDispatchHashMarkerList},
		{name: "int-slice", value: []int{1, 2}, wantMarker: getSnapshotDispatchHashMarkerList},
		{name: "int8-slice", value: []int8{1, 2}, wantMarker: getSnapshotDispatchHashMarkerList},
		{name: "int16-slice", value: []int16{1, 2}, wantMarker: getSnapshotDispatchHashMarkerList},
		{name: "int32-slice", value: []int32{1, 2}, wantMarker: getSnapshotDispatchHashMarkerList},
		{name: "int64-slice", value: []int64{1, 2}, wantMarker: getSnapshotDispatchHashMarkerList},
		{name: "uint-slice", value: []uint{1, 2}, wantMarker: getSnapshotDispatchHashMarkerList},
		{name: "uint8-slice", value: []uint8{1, 2}, wantMarker: getSnapshotDispatchHashMarkerList},
		{name: "uint16-slice", value: []uint16{1, 2}, wantMarker: getSnapshotDispatchHashMarkerList},
		{name: "uint32-slice", value: []uint32{1, 2}, wantMarker: getSnapshotDispatchHashMarkerList},
		{name: "uint64-slice", value: []uint64{1, 2}, wantMarker: getSnapshotDispatchHashMarkerList},
		{name: "uintptr-slice", value: []uintptr{1, 2}, wantMarker: getSnapshotDispatchHashMarkerList},
		{name: "float32-slice", value: []float32{1.5, 2.5}, wantMarker: getSnapshotDispatchHashMarkerList},
		{name: "float64-slice", value: []float64{3.5, 4.5}, wantMarker: getSnapshotDispatchHashMarkerList},
		{name: "string-slice", value: []string{"a", "b"}, wantMarker: getSnapshotDispatchHashMarkerList},
		{name: "any-list", value: []any{true, 1, "x"}, wantMarker: getSnapshotDispatchHashMarkerList},
		{name: "empty-bool-map", value: map[string]bool{}, wantMarker: getSnapshotDispatchHashMarkerMap},
		{name: "bool-map", value: map[string]bool{"b": true, "a": false}, wantMarker: getSnapshotDispatchHashMarkerMap},
		{name: "int-map", value: map[string]int{"b": 2, "a": 1}, wantMarker: getSnapshotDispatchHashMarkerMap},
		{name: "int8-map", value: map[string]int8{"b": 2, "a": 1}, wantMarker: getSnapshotDispatchHashMarkerMap},
		{name: "int16-map", value: map[string]int16{"b": 2, "a": 1}, wantMarker: getSnapshotDispatchHashMarkerMap},
		{name: "int32-map", value: map[string]int32{"b": 2, "a": 1}, wantMarker: getSnapshotDispatchHashMarkerMap},
		{name: "int64-map", value: map[string]int64{"b": 2, "a": 1}, wantMarker: getSnapshotDispatchHashMarkerMap},
		{name: "uint-map", value: map[string]uint{"b": 2, "a": 1}, wantMarker: getSnapshotDispatchHashMarkerMap},
		{name: "uint8-map", value: map[string]uint8{"b": 2, "a": 1}, wantMarker: getSnapshotDispatchHashMarkerMap},
		{name: "uint16-map", value: map[string]uint16{"b": 2, "a": 1}, wantMarker: getSnapshotDispatchHashMarkerMap},
		{name: "uint32-map", value: map[string]uint32{"b": 2, "a": 1}, wantMarker: getSnapshotDispatchHashMarkerMap},
		{name: "uint64-map", value: map[string]uint64{"b": 2, "a": 1}, wantMarker: getSnapshotDispatchHashMarkerMap},
		{name: "uintptr-map", value: map[string]uintptr{"b": 2, "a": 1}, wantMarker: getSnapshotDispatchHashMarkerMap},
		{name: "float32-map", value: map[string]float32{"b": 2, "a": 1}, wantMarker: getSnapshotDispatchHashMarkerMap},
		{name: "float64-map", value: map[string]float64{"b": 2, "a": 1}, wantMarker: getSnapshotDispatchHashMarkerMap},
		{name: "string-map", value: map[string]string{"b": "2", "a": "1"}, wantMarker: getSnapshotDispatchHashMarkerMap},
		{name: "any-map-empty", value: map[string]any{}, wantMarker: getSnapshotDispatchHashMarkerMap},
		{name: "any-map", value: map[string]any{"b": 2, "a": 1}, wantMarker: getSnapshotDispatchHashMarkerMap},
		{name: "unsupported-complex", value: complex(1, 2), wantErr: true},
	}
	for _, parseCase := range parseCases {
		parseT.Run(parseCase.name, func(parseT *testing.T) {
			parsePayload, parseErr := appendSnapshotDispatchValue(nil, parseCase.value)
			if parseCase.wantErr {
				if parseErr == nil {
					parseT.Fatal("expected appendSnapshotDispatchValue to fail")
				}
				return
			}
			if parseErr != nil {
				parseT.Fatalf("appendSnapshotDispatchValue returned error: %v", parseErr)
			}
			if len(parsePayload) == 0 {
				parseT.Fatal("expected non-empty dispatch payload")
			}
			if parsePayload[0] != parseCase.wantMarker {
				parseT.Fatalf("expected marker %d, got %d", parseCase.wantMarker, parsePayload[0])
			}
		})
	}

	if _, parseErr := appendSnapshotDispatchValue(nil, math.NaN()); parseErr == nil {
		parseT.Fatal("expected NaN to fail")
	}
	if _, parseErr := appendSnapshotDispatchValue(nil, math.Inf(1)); parseErr == nil {
		parseT.Fatal("expected infinity to fail")
	}

	parseScalarMapLarge := map[string]int{"c": 3, "a": 1, "b": 2}
	parseFirst, parseErr := appendSnapshotDispatchValue(nil, parseScalarMapLarge)
	if parseErr != nil {
		parseT.Fatalf("appendSnapshotDispatchValue(scalar map) returned error: %v", parseErr)
	}
	parseSecond, parseErr := appendSnapshotDispatchValue(nil, map[string]int{"b": 2, "c": 3, "a": 1})
	if parseErr != nil {
		parseT.Fatalf("appendSnapshotDispatchValue(scalar map reordered) returned error: %v", parseErr)
	}
	if string(parseFirst) != string(parseSecond) {
		parseT.Fatalf("expected canonical scalar map encoding, got %x and %x", parseFirst, parseSecond)
	}

	parseAnyMapLarge := map[string]any{"c": 3, "a": 1, "b": 2}
	parseFirstAny, parseErr := appendSnapshotDispatchValue(nil, parseAnyMapLarge)
	if parseErr != nil {
		parseT.Fatalf("appendSnapshotDispatchValue(any map) returned error: %v", parseErr)
	}
	parseSecondAny, parseErr := appendSnapshotDispatchValue(nil, map[string]any{"b": 2, "c": 3, "a": 1})
	if parseErr != nil {
		parseT.Fatalf("appendSnapshotDispatchValue(any map reordered) returned error: %v", parseErr)
	}
	if string(parseFirstAny) != string(parseSecondAny) {
		parseT.Fatalf("expected canonical any-map encoding, got %x and %x", parseFirstAny, parseSecondAny)
	}
}

// TestSnapshotDispatchHashReflectAndOrderedKeyHelpers verifies the reflect append paths, ordered-key helpers, and direct envelope hash writers.
func TestSnapshotDispatchHashReflectAndOrderedKeyHelpers(parseT *testing.T) {
	type parseExample struct {
		Name   string
		Count  int
		hidden bool
	}

	var parseNilInterface any
	parsePointerValue := 7
	parseCases := []struct {
		name       string
		value      reflect.Value
		wantMarker byte
		wantErr    bool
	}{
		{name: "invalid", value: reflect.Value{}, wantMarker: getSnapshotDispatchHashMarkerNil},
		{name: "nil-interface", value: reflect.ValueOf(&parseNilInterface).Elem(), wantMarker: getSnapshotDispatchHashMarkerNil},
		{name: "pointer-nil", value: reflect.ValueOf((*int)(nil)), wantMarker: getSnapshotDispatchHashMarkerNil},
		{name: "pointer", value: reflect.ValueOf(&parsePointerValue), wantMarker: getSnapshotDispatchHashMarkerNumber},
		{name: "bool", value: reflect.ValueOf(true), wantMarker: getSnapshotDispatchHashMarkerBool},
		{name: "int", value: reflect.ValueOf(int(-3)), wantMarker: getSnapshotDispatchHashMarkerNumber},
		{name: "uint", value: reflect.ValueOf(uint(3)), wantMarker: getSnapshotDispatchHashMarkerNumber},
		{name: "float32", value: reflect.ValueOf(float32(1.5)), wantMarker: getSnapshotDispatchHashMarkerNumber},
		{name: "float64", value: reflect.ValueOf(float64(2.5)), wantMarker: getSnapshotDispatchHashMarkerNumber},
		{name: "string", value: reflect.ValueOf("ok"), wantMarker: getSnapshotDispatchHashMarkerString},
		{name: "slice-nil", value: reflect.ValueOf([]any(nil)), wantMarker: getSnapshotDispatchHashMarkerNil},
		{name: "slice", value: reflect.ValueOf([]any{true, 1}), wantMarker: getSnapshotDispatchHashMarkerList},
		{name: "array", value: reflect.ValueOf([2]any{true, 1}), wantMarker: getSnapshotDispatchHashMarkerList},
		{name: "map-nil", value: reflect.ValueOf(map[string]any(nil)), wantMarker: getSnapshotDispatchHashMarkerNil},
		{name: "map-empty", value: reflect.ValueOf(map[string]any{}), wantMarker: getSnapshotDispatchHashMarkerMap},
		{name: "map-single", value: reflect.ValueOf(map[string]any{"a": 1}), wantMarker: getSnapshotDispatchHashMarkerMap},
		{name: "map-pair", value: reflect.ValueOf(map[string]any{"b": 2, "a": 1}), wantMarker: getSnapshotDispatchHashMarkerMap},
		{name: "map-large", value: reflect.ValueOf(map[string]any{"c": 3, "b": 2, "a": 1}), wantMarker: getSnapshotDispatchHashMarkerMap},
		{name: "struct", value: reflect.ValueOf(parseExample{Name: "A", Count: 3, hidden: true}), wantMarker: getSnapshotDispatchHashMarkerStruct},
		{name: "unsupported", value: reflect.ValueOf(make(chan int)), wantErr: true},
	}
	for _, parseCase := range parseCases {
		parseT.Run(parseCase.name, func(parseT *testing.T) {
			parsePayload, parseErr := appendSnapshotDispatchReflect(nil, parseCase.value)
			if parseCase.wantErr {
				if parseErr == nil {
					parseT.Fatal("expected appendSnapshotDispatchReflect to fail")
				}
				return
			}
			if parseErr != nil {
				parseT.Fatalf("appendSnapshotDispatchReflect returned error: %v", parseErr)
			}
			if len(parsePayload) == 0 {
				parseT.Fatal("expected non-empty reflect payload")
			}
			if parsePayload[0] != parseCase.wantMarker {
				parseT.Fatalf("expected marker %d, got %d", parseCase.wantMarker, parsePayload[0])
			}
		})
	}

	parseMapPayload, parseErr := appendSnapshotDispatchReflectMap(nil, reflect.ValueOf(map[int]any{1: true}))
	if parseErr == nil || parseMapPayload != nil {
		parseT.Fatal("expected non-string reflect map keys to fail")
	}

	parseSingleReflect, parseErr := appendSnapshotDispatchReflectMapSingle(nil, reflect.ValueOf(map[string]any{"alpha": 1}))
	if parseErr != nil || len(parseSingleReflect) == 0 || parseSingleReflect[0] != getSnapshotDispatchHashMarkerMap {
		parseT.Fatalf("expected single-entry reflect map payload, got payload=%v err=%v", parseSingleReflect, parseErr)
	}

	parsePairReflect, parseErr := appendSnapshotDispatchReflectMapPair(nil, reflect.ValueOf(map[string]any{"b": 2, "a": 1}))
	if parseErr != nil || len(parsePairReflect) == 0 || parsePairReflect[0] != getSnapshotDispatchHashMarkerMap {
		parseT.Fatalf("expected two-entry reflect map payload, got payload=%v err=%v", parsePairReflect, parseErr)
	}

	parseStructPayload, parseErr := appendSnapshotDispatchReflectStruct(nil, reflect.ValueOf(parseExample{Name: "A", Count: 3, hidden: true}))
	if parseErr != nil || len(parseStructPayload) == 0 || parseStructPayload[0] != getSnapshotDispatchHashMarkerStruct {
		parseT.Fatalf("expected struct payload, got payload=%v err=%v", parseStructPayload, parseErr)
	}

	parseAnyOrderedPayload, parseHasOrdered, parseErr := appendSnapshotDispatchAnyMapWithOrderedKeys(
		nil,
		map[string]any{"b": 2, "a": 1},
		[]string{"a", "b"},
	)
	if parseErr != nil || !parseHasOrdered || len(parseAnyOrderedPayload) == 0 {
		parseT.Fatalf("expected ordered any-map payload, got payload=%v hasOrdered=%v err=%v", parseAnyOrderedPayload, parseHasOrdered, parseErr)
	}
	parseAnyFallbackPayload, parseHasOrdered, parseErr := appendSnapshotDispatchAnyMapWithOrderedKeys(
		nil,
		map[string]any{"b": 2, "a": 1},
		[]string{"a"},
	)
	if parseErr != nil || parseHasOrdered || parseAnyFallbackPayload != nil {
		parseT.Fatalf("expected ordered any-map fallback, got payload=%v hasOrdered=%v err=%v", parseAnyFallbackPayload, parseHasOrdered, parseErr)
	}

	parseSourceOrderedPayload, parseErr := appendSnapshotDispatchSourceMap(nil, map[string]any{"b": 2, "a": 1}, []string{"a", "b"})
	if parseErr != nil || len(parseSourceOrderedPayload) == 0 {
		parseT.Fatalf("expected ordered source-map payload, got payload=%v err=%v", parseSourceOrderedPayload, parseErr)
	}
	parseSourceFallbackPayload, parseErr := appendSnapshotDispatchSourceMap(nil, map[string]any{"b": 2, "a": 1}, []string{"a"})
	if parseErr != nil || len(parseSourceFallbackPayload) == 0 {
		parseT.Fatalf("expected fallback source-map payload, got payload=%v err=%v", parseSourceFallbackPayload, parseErr)
	}

	parsePropsOrderedPayload, parseErr := appendSnapshotDispatchPropsValue(nil, map[string]any{"b": 2, "a": 1}, []string{"a", "b"})
	if parseErr != nil || len(parsePropsOrderedPayload) == 0 {
		parseT.Fatalf("expected ordered props payload, got payload=%v err=%v", parsePropsOrderedPayload, parseErr)
	}
	parsePropsFallbackPayload, parseErr := appendSnapshotDispatchPropsValue(nil, map[string]any{"b": 2, "a": 1}, []string{"a"})
	if parseErr != nil || len(parsePropsFallbackPayload) == 0 {
		parseT.Fatalf("expected fallback props payload, got payload=%v err=%v", parsePropsFallbackPayload, parseErr)
	}

	parseEnvelope := SnapshotEnvelope{
		RegionInstanceID: RegionInstanceID("region-dispatch"),
		Epoch:            7,
		InputVersion:     11,
		SourceVersion:    13,
		Props: map[string]any{
			"b": 2,
			"a": 1,
		},
		Sources: map[string]any{
			"c": 3,
			"a": 1,
			"b": 2,
		},
	}
	parseHash, parseScratch, parseErr := buildSnapshotDispatchHashIntoWithSourceAndPropsKeys(
		parseEnvelope,
		[]string{"a", "b", "c"},
		[]string{"a", "b"},
		nil,
	)
	if parseErr != nil {
		parseT.Fatalf("buildSnapshotDispatchHashIntoWithSourceAndPropsKeys returned error: %v", parseErr)
	}
	if len(parseScratch) != 0 {
		parseT.Fatalf("expected reused scratch to be reset, got len=%d", len(parseScratch))
	}
	if parseHash == [sha256.Size]byte{} {
		parseT.Fatal("expected non-zero dispatch hash")
	}

	parseBufferedHash, parseErr := buildSnapshotDispatchHash(parseEnvelope)
	if parseErr != nil {
		parseT.Fatalf("buildSnapshotDispatchHash returned error: %v", parseErr)
	}
	parseStreamedHash, parseErr := buildSnapshotDispatchHashStreamed(parseEnvelope)
	if parseErr != nil {
		parseT.Fatalf("buildSnapshotDispatchHashStreamed returned error: %v", parseErr)
	}
	if parseBufferedHash != parseStreamedHash {
		parseT.Fatal("expected buffered and streamed hash outputs to match")
	}

	parseHasher := sha256.New()
	parseScratchBytes, parseErr := writeSnapshotDispatchEnvelopeHash(parseHasher, nil, parseEnvelope)
	if parseErr != nil {
		parseT.Fatalf("writeSnapshotDispatchEnvelopeHash returned error: %v", parseErr)
	}
	if len(parseScratchBytes) == 0 {
		parseT.Fatal("expected scratch bytes from envelope writer")
	}

	parseEncodedOrdered, parseErr := appendSnapshotDispatchEnvelopeWithSourceIDs(nil, parseEnvelope, []string{"a", "b", "c"})
	if parseErr != nil {
		parseT.Fatalf("appendSnapshotDispatchEnvelopeWithSourceIDs returned error: %v", parseErr)
	}
	parseEncodedFallback, parseErr := appendSnapshotDispatchEnvelopeWithSourceAndPropsKeys(nil, parseEnvelope, []string{"a"}, []string{"a"})
	if parseErr != nil {
		parseT.Fatalf("appendSnapshotDispatchEnvelopeWithSourceAndPropsKeys returned error: %v", parseErr)
	}
	if len(parseEncodedOrdered) == 0 || len(parseEncodedFallback) == 0 {
		parseT.Fatal("expected envelope payloads to be non-empty")
	}

	parseOrderedEntryA := []buildSnapshotDispatchMapEntry{{getKey: "b", getValue: 2}, {getKey: "a", getValue: 1}}
	parseOrderedEntryB := []buildSnapshotDispatchMapEntry{{getKey: "a", getValue: 1}, {getKey: "b", getValue: 2}}
	if got, hasOrdered, err := appendSnapshotDispatchAnyMapWithOrderedKeys(nil, map[string]any{"b": 2, "a": 1}, []string{"b", "missing"}); err != nil || hasOrdered || len(got) != 0 {
		parseT.Fatalf("expected missing ordered key fallback, got payload=%v hasOrdered=%v err=%v", got, hasOrdered, err)
	}
	if got, err := appendSnapshotDispatchAnyMapSingle(nil, map[string]any{"a": 1}); err != nil || len(got) == 0 {
		parseT.Fatalf("expected single any-map payload, got payload=%v err=%v", got, err)
	}
	if got, err := appendSnapshotDispatchAnyMapPair(nil, map[string]any{"b": 2, "a": 1}); err != nil || len(got) == 0 {
		parseT.Fatalf("expected pair any-map payload, got payload=%v err=%v", got, err)
	}
	if got, err := appendSnapshotDispatchScalarMapSingle(nil, map[string]int{"a": 1}); err != nil || len(got) == 0 {
		parseT.Fatalf("expected single scalar-map payload, got payload=%v err=%v", got, err)
	}
	if got, err := appendSnapshotDispatchScalarMapPair(nil, map[string]int{"b": 2, "a": 1}); err != nil || len(got) == 0 {
		parseT.Fatalf("expected pair scalar-map payload, got payload=%v err=%v", got, err)
	}

	parseReflectScalarMapSingle, parseErr := appendSnapshotDispatchReflectMapSingle(nil, reflect.ValueOf(map[string]any{"a": 1}))
	if parseErr != nil || len(parseReflectScalarMapSingle) == 0 {
		parseT.Fatalf("expected single reflect-map payload, got payload=%v err=%v", parseReflectScalarMapSingle, parseErr)
	}
	parseReflectScalarMapPair, parseErr := appendSnapshotDispatchReflectMapPair(nil, reflect.ValueOf(map[string]any{"b": 2, "a": 1}))
	if parseErr != nil || len(parseReflectScalarMapPair) == 0 {
		parseT.Fatalf("expected pair reflect-map payload, got payload=%v err=%v", parseReflectScalarMapPair, parseErr)
	}

	for parseIndex := range 64 {
		if _, parseErr := appendSnapshotDispatchAnyMapPair(nil, map[string]any{"b": 2, "a": 1}); parseErr != nil {
			parseT.Fatalf("appendSnapshotDispatchAnyMapPair iteration %d returned error: %v", parseIndex, parseErr)
		}
		if _, parseErr := appendSnapshotDispatchScalarMapPair(nil, map[string]int{"b": 2, "a": 1}); parseErr != nil {
			parseT.Fatalf("appendSnapshotDispatchScalarMapPair iteration %d returned error: %v", parseIndex, parseErr)
		}
		if _, parseErr := appendSnapshotDispatchReflectMapPair(nil, reflect.ValueOf(map[string]any{"b": 2, "a": 1})); parseErr != nil {
			parseT.Fatalf("appendSnapshotDispatchReflectMapPair iteration %d returned error: %v", parseIndex, parseErr)
		}
	}

	_ = parseOrderedEntryA
	_ = parseOrderedEntryB
}

// TestSnapshotDispatchHashIntoEntryPoints verifies the direct hash-into entry points reuse scratch and match the canonical hash output.
func TestSnapshotDispatchHashIntoEntryPoints(parseT *testing.T) {
	getEnvelope := SnapshotEnvelope{
		RegionInstanceID: RegionInstanceID("region-hash-into"),
		Epoch:            4,
		InputVersion:     12,
		SourceVersion:    18,
		Props: map[string]any{
			"title": "Orders",
			"count": 7,
		},
		Sources: map[string]any{
			"filters": map[string]any{
				"status": "open",
			},
		},
	}

	getHash, getScratch, parseErr := buildSnapshotDispatchHashInto(getEnvelope, []byte("seed"))
	if parseErr != nil {
		parseT.Fatalf("buildSnapshotDispatchHashInto returned error: %v", parseErr)
	}
	if len(getScratch) != 0 {
		parseT.Fatalf("expected reused scratch to be reset, got len=%d", len(getScratch))
	}

	getCanonicalHash, parseErr := buildSnapshotDispatchHash(getEnvelope)
	if parseErr != nil {
		parseT.Fatalf("buildSnapshotDispatchHash returned error: %v", parseErr)
	}
	if getHash != getCanonicalHash {
		parseT.Fatal("expected hash-into output to match the canonical dispatch hash")
	}

	getOrderedHash, getOrderedScratch, parseErr := buildSnapshotDispatchHashIntoWithSourceIDs(
		getEnvelope,
		[]string{"filters"},
		nil,
	)
	if parseErr != nil {
		parseT.Fatalf("buildSnapshotDispatchHashIntoWithSourceIDs returned error: %v", parseErr)
	}
	if len(getOrderedScratch) != 0 {
		parseT.Fatalf("expected ordered scratch to stay reset, got len=%d", len(getOrderedScratch))
	}
	if getOrderedHash == [sha256.Size]byte{} {
		parseT.Fatal("expected ordered hash output to be non-zero")
	}
}
