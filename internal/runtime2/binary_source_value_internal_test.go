package runtime2

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// TestBuildBinarySourceValueCapacityHints verifies the coarse size hints hit every branch and stay non-zero for supported shapes.
func TestBuildBinarySourceValueCapacityHints(parseT *testing.T) {
	type parseExample struct {
		Name  string
		Count int
		hidden bool
	}

	parseCases := []struct {
		name   string
		value  any
		hintFn func(any) int
	}{
		{name: "nil-any", value: nil, hintFn: buildBinarySourceValueAppendCapacityHint},
		{name: "bool", value: true, hintFn: buildBinarySourceValueAppendCapacityHint},
		{name: "int", value: int(-3), hintFn: buildBinarySourceValueAppendCapacityHint},
		{name: "uint", value: uint(7), hintFn: buildBinarySourceValueAppendCapacityHint},
		{name: "float", value: float64(1.5), hintFn: buildBinarySourceValueAppendCapacityHint},
		{name: "string", value: "abc", hintFn: buildBinarySourceValueAppendCapacityHint},
		{name: "list", value: []any{true, "x"}, hintFn: buildBinarySourceValueAppendCapacityHint},
		{name: "map", value: map[string]any{"b": 1, "a": 2}, hintFn: buildBinarySourceValueAppendCapacityHint},
		{name: "default", value: complex(1, 2), hintFn: buildBinarySourceValueAppendCapacityHint},
	}
	for _, parseCase := range parseCases {
		parseT.Run(parseCase.name, func(parseT *testing.T) {
			if parseHint := parseCase.hintFn(parseCase.value); parseHint <= 0 {
				parseT.Fatalf("expected positive hint, got %d", parseHint)
			}
		})
	}

	parseReflectCases := []struct {
		name   string
		value  reflect.Value
		expect int
	}{
		{name: "invalid", value: reflect.Value{}, expect: 1},
		{name: "nil-interface", value: func() reflect.Value { var parseValue any; return reflect.ValueOf(&parseValue).Elem() }(), expect: 1},
		{name: "non-nil-interface", value: func() reflect.Value { var parseValue any = []any{1}; return reflect.ValueOf(&parseValue).Elem() }(), expect: 2 + 1*14},
		{name: "nil-pointer", value: reflect.ValueOf((*int)(nil)), expect: 1},
		{name: "bool", value: reflect.ValueOf(true), expect: 1},
		{name: "int", value: reflect.ValueOf(int64(7)), expect: 9},
		{name: "uint", value: reflect.ValueOf(uint64(7)), expect: 9},
		{name: "float", value: reflect.ValueOf(float64(7)), expect: 9},
		{name: "string", value: reflect.ValueOf("abc"), expect: 8},
		{name: "slice", value: reflect.ValueOf([]any{1, 2}), expect: 2 + 2*14},
		{name: "array", value: reflect.ValueOf([2]any{1, 2}), expect: 2 + 2*14},
		{name: "map", value: reflect.ValueOf(map[string]any{"a": 1, "bb": 2}), expect: 2 + 2*18},
	}
	for _, parseCase := range parseReflectCases {
		parseT.Run(parseCase.name, func(parseT *testing.T) {
			if parseHint := buildBinarySourceReflectValueAppendCapacityHint(parseCase.value); parseHint != parseCase.expect {
				parseT.Fatalf("expected hint %d, got %d", parseCase.expect, parseHint)
			}
		})
	}

	parseStructHint := buildBinarySourceReflectValueAppendCapacityHint(reflect.ValueOf(parseExample{}))
	if parseStructHint != 9 {
		parseT.Fatalf("expected struct hint 9, got %d", parseStructHint)
	}
}

// TestBuildBinarySourceValueIntoAndParseBinarySourceValue verifies all concrete source encodings round-trip and the low-level failure cases fail cleanly.
func TestBuildBinarySourceValueIntoAndParseBinarySourceValue(parseT *testing.T) {
	parseCases := []struct {
		name  string
		value any
		want  any
	}{
		{name: "nil", value: nil, want: nil},
		{name: "bool-true", value: true, want: true},
		{name: "bool-false", value: false, want: false},
		{name: "int", value: int(-3), want: float64(-3)},
		{name: "int8", value: int8(-4), want: float64(-4)},
		{name: "int16", value: int16(-5), want: float64(-5)},
		{name: "int32", value: int32(-6), want: float64(-6)},
		{name: "int64", value: int64(-7), want: float64(-7)},
		{name: "uint", value: uint(8), want: float64(8)},
		{name: "uint8", value: uint8(9), want: float64(9)},
		{name: "uint16", value: uint16(10), want: float64(10)},
		{name: "uint32", value: uint32(11), want: float64(11)},
		{name: "uint64", value: uint64(12), want: float64(12)},
		{name: "uintptr", value: uintptr(13), want: float64(13)},
		{name: "float32", value: float32(14.5), want: float64(14.5)},
		{name: "float64", value: float64(15.5), want: float64(15.5)},
		{name: "string", value: "ok", want: "ok"},
		{name: "list", value: []any{true, float64(2), "x"}, want: []any{true, float64(2), "x"}},
		{name: "map", value: map[string]any{"b": 2, "a": 1}, want: map[string]any{"a": float64(1), "b": float64(2)}},
	}
	for _, parseCase := range parseCases {
		parseT.Run(parseCase.name, func(parseT *testing.T) {
			parsePayload, parseErr := BuildBinarySourceValue(parseCase.value)
			if parseErr != nil {
				parseT.Fatalf("BuildBinarySourceValue returned error: %v", parseErr)
			}
			parseDecoded, parseErr := ParseBinarySourceValue(parsePayload)
			if parseErr != nil {
				parseT.Fatalf("ParseBinarySourceValue returned error: %v", parseErr)
			}
			if !reflect.DeepEqual(parseDecoded, parseCase.want) {
				parseT.Fatalf("expected %#v, got %#v", parseCase.want, parseDecoded)
			}
		})
	}

	type parseExported struct {
		Title string
		Count int
	}
	parseStructPayload, parseErr := buildBinarySourceValueReflectInto(nil, reflect.ValueOf(parseExported{Title: "A", Count: 3}))
	if parseErr != nil {
		parseT.Fatalf("buildBinarySourceValueReflectInto(struct) returned error: %v", parseErr)
	}
	parseStructDecoded, parseErr := ParseBinarySourceValue(parseStructPayload)
	if parseErr != nil {
		parseT.Fatalf("ParseBinarySourceValue(struct) returned error: %v", parseErr)
	}
	parseStructMap, parseOk := parseStructDecoded.(map[string]any)
	if !parseOk || parseStructMap["Title"] != "A" || parseStructMap["Count"] != float64(3) {
		parseT.Fatalf("expected struct to encode as exported-field map, got %#v", parseStructDecoded)
	}

	parseCasesErr := []struct {
		name string
		raw  []byte
		want string
	}{
		{name: "empty", raw: nil, want: "empty"},
		{name: "unsupported-kind", raw: []byte{99}, want: "unsupported"},
		{name: "number-truncated", raw: []byte{binarySourceValueKindNumber}, want: "invalid"},
		{name: "string-truncated", raw: []byte{binarySourceValueKindString, 1}, want: "truncated"},
		{name: "string-length-exceeds", raw: []byte{binarySourceValueKindString, 4, 0, 0, 0, 'o'}, want: "exceeds payload size"},
		{name: "string-trailing-bytes", raw: []byte{binarySourceValueKindString, 1, 0, 0, 0, 'o', '!'}, want: "trailing bytes"},
	}
	for _, parseCase := range parseCasesErr {
		parseT.Run(parseCase.name, func(parseT *testing.T) {
			_, parseErr := ParseBinarySourceValue(parseCase.raw)
			if parseErr == nil || !strings.Contains(parseErr.Error(), parseCase.want) {
				parseT.Fatalf("expected error containing %q, got %v", parseCase.want, parseErr)
			}
		})
	}

	if _, parseErr := buildBinarySourceValueInto(nil, complex(1, 2)); parseErr == nil {
		parseT.Fatal("expected unsupported complex source value to fail")
	}
}

// TestBuildBinarySourceAnyListIntoAndParseBinarySourceListValue verifies list encoding, nested recursion, and parse guards.
func TestBuildBinarySourceAnyListIntoAndParseBinarySourceListValue(parseT *testing.T) {
	parsePayload, parseErr := buildBinarySourceAnyListInto(nil, []any{true, []any{"x"}, map[string]any{"a": 1}})
	if parseErr != nil {
		parseT.Fatalf("buildBinarySourceAnyListInto returned error: %v", parseErr)
	}
	parseDecoded, parseErr := parseBinarySourceListValue(parsePayload)
	if parseErr != nil {
		parseT.Fatalf("parseBinarySourceListValue returned error: %v", parseErr)
	}
	if len(parseDecoded) != 3 {
		parseT.Fatalf("expected list length 3, got %d", len(parseDecoded))
	}

	if parseNilPayload, parseErr := buildBinarySourceAnyListInto(nil, nil); parseErr != nil || len(parseNilPayload) != 1 || parseNilPayload[0] != binarySourceValueKindNil {
		parseT.Fatalf("expected nil list to encode as nil sentinel, got payload=%v err=%v", parseNilPayload, parseErr)
	}
	if _, parseErr := buildBinarySourceAnyListInto(nil, make([]any, binarySourceValueListLimit+1)); parseErr == nil {
		parseT.Fatal("expected oversized list to fail")
	}
	if _, parseErr := buildBinarySourceAnyListInto(nil, []any{complex(1, 2)}); parseErr == nil {
		parseT.Fatal("expected nested unsupported list item to fail")
	}

	if _, parseErr := parseBinarySourceListValue([]byte{binarySourceValueKindList}); parseErr == nil {
		parseT.Fatal("expected truncated list payload to fail")
	}
	if _, parseErr := parseBinarySourceListValue([]byte{binarySourceValueKindList, 1, 4, 0, 0, 0, 1}); parseErr == nil {
		parseT.Fatal("expected oversized list item length to fail")
	}
	if _, parseErr := parseBinarySourceListValue([]byte{binarySourceValueKindList, 1, 1, 0, 0, 0, binarySourceValueKindBoolTrue, 0}); parseErr == nil {
		parseT.Fatal("expected trailing list bytes to fail")
	}
}

// TestBuildBinarySourceAnyMapIntoAndParseBinarySourceMapValue verifies canonical map encoding, pooling helpers, and parse guards.
func TestBuildBinarySourceAnyMapIntoAndParseBinarySourceMapValue(parseT *testing.T) {
	parsePayload, parseErr := buildBinarySourceAnyMapInto(nil, map[string]any{"b": 2, "a": 1})
	if parseErr != nil {
		parseT.Fatalf("buildBinarySourceAnyMapInto returned error: %v", parseErr)
	}
	parseDecoded, parseErr := parseBinarySourceMapValue(parsePayload)
	if parseErr != nil {
		parseT.Fatalf("parseBinarySourceMapValue returned error: %v", parseErr)
	}
	if !reflect.DeepEqual(parseDecoded, map[string]any{"a": float64(1), "b": float64(2)}) {
		parseT.Fatalf("expected canonical map decode, got %#v", parseDecoded)
	}

	if parseNilPayload, parseErr := buildBinarySourceAnyMapInto(nil, nil); parseErr != nil || len(parseNilPayload) != 1 || parseNilPayload[0] != binarySourceValueKindNil {
		parseT.Fatalf("expected nil map to encode as nil sentinel, got payload=%v err=%v", parseNilPayload, parseErr)
	}
	parseOversizedMap := make(map[string]any, binarySourceValueMapLimit+1)
	for parseIndex := 0; parseIndex <= binarySourceValueMapLimit; parseIndex++ {
		parseOversizedMap[fmt.Sprintf("k%d", parseIndex)] = true
	}
	if _, parseErr := buildBinarySourceAnyMapInto(nil, parseOversizedMap); parseErr == nil {
		parseT.Fatal("expected oversized map to fail")
	}
	parseTooLongKey := strings.Repeat("a", 0x10000)
	if _, parseErr := buildBinarySourceAnyMapInto(nil, map[string]any{parseTooLongKey: true}); parseErr == nil {
		parseT.Fatal("expected oversized map key to fail")
	}

	if _, parseErr := parseBinarySourceMapValue([]byte{binarySourceValueKindMap}); parseErr == nil {
		parseT.Fatal("expected truncated map payload to fail")
	}
	if _, parseErr := parseBinarySourceMapValue([]byte{binarySourceValueKindMap, 1, 1}); parseErr == nil {
		parseT.Fatal("expected truncated map key length to fail")
	}
	if _, parseErr := parseBinarySourceMapValue([]byte{binarySourceValueKindMap, 1, 4, 0, 'a'}); parseErr == nil {
		parseT.Fatal("expected oversized map key to fail")
	}
	if _, parseErr := parseBinarySourceMapValue([]byte{binarySourceValueKindMap, 1, 1, 0, 'a'}); parseErr == nil {
		parseT.Fatal("expected truncated map value length to fail")
	}
	if _, parseErr := parseBinarySourceMapValue([]byte{binarySourceValueKindMap, 1, 1, 0, 'a', 2, 0, 0, 0, binarySourceValueKindBoolTrue}); parseErr == nil {
		parseT.Fatal("expected oversize map value to fail")
	}
	if _, parseErr := parseBinarySourceMapValue([]byte{binarySourceValueKindMap, 1, 1, 0, 'a', 1, 0, 0, 0, binarySourceValueKindBoolTrue, 0}); parseErr == nil {
		parseT.Fatal("expected trailing map bytes to fail")
	}
	if _, parseErr := parseBinarySourceMapValue([]byte{binarySourceValueKindMap, 2, 1, 0, 'b', 1, 0, 0, 0, binarySourceValueKindBoolTrue, 1, 0, 'a', 1, 0, 0, 0, binarySourceValueKindBoolTrue}); parseErr == nil {
		parseT.Fatal("expected non-canonical map ordering to fail")
	}
}

// TestBuildBinarySourceAnyMapEntryAndKeyBuffers verifies the reusable buffers are sized and cleared correctly.
func TestBuildBinarySourceAnyMapEntryAndKeyBuffers(parseT *testing.T) {
	parseEntries, parseEntryCache := buildBinarySourceAnyMapEntryBuffer(4)
	if cap(parseEntries) < 4 {
		parseT.Fatalf("expected entry buffer capacity >= 4, got %d", cap(parseEntries))
	}
	parseEntries = append(parseEntries, buildBinarySourceAnyMapEntry{getKey: "a", getValue: true})
	storeBinarySourceAnyMapEntryBuffer(parseEntries, parseEntryCache)
	if parseEntryCache.getEntries == nil || len(parseEntryCache.getEntries) != 0 {
		parseT.Fatalf("expected entry cache to be cleared, got %#v", parseEntryCache.getEntries)
	}
	storeBinarySourceAnyMapEntryBuffer([]buildBinarySourceAnyMapEntry{{getKey: "x"}}, nil)

	parseKeys, parseKeyCache := buildBinarySourceMapKeyBuffer(4)
	if cap(parseKeys) < 4 {
		parseT.Fatalf("expected key buffer capacity >= 4, got %d", cap(parseKeys))
	}
	parseKeys = append(parseKeys, "a", "b")
	storeBinarySourceMapKeyBuffer(parseKeys, parseKeyCache)
	if parseKeyCache.getKeys == nil || len(parseKeyCache.getKeys) != 0 {
		parseT.Fatalf("expected key cache to be cleared, got %#v", parseKeyCache.getKeys)
	}
	storeBinarySourceMapKeyBuffer([]string{"x"}, nil)
}

// TestSortBinarySourceAnyMapEntries verifies small and large entry sorts use the expected ordering paths.
func TestSortBinarySourceAnyMapEntries(parseT *testing.T) {
	parseSmallEntries := []buildBinarySourceAnyMapEntry{
		{getKey: "c"},
		{getKey: "a"},
		{getKey: "b"},
	}
	sortBinarySourceAnyMapEntries(parseSmallEntries)
	if parseSmallEntries[0].getKey != "a" || parseSmallEntries[1].getKey != "b" || parseSmallEntries[2].getKey != "c" {
		parseT.Fatalf("expected insertion sort ordering, got %#v", parseSmallEntries)
	}

	parseLargeEntries := []buildBinarySourceAnyMapEntry{
		{getKey: "i"}, {getKey: "h"}, {getKey: "g"}, {getKey: "f"},
		{getKey: "e"}, {getKey: "d"}, {getKey: "c"}, {getKey: "b"}, {getKey: "a"},
	}
	sortBinarySourceAnyMapEntries(parseLargeEntries)
	for parseIndex, parseEntry := range parseLargeEntries {
		if parseEntry.getKey != string(rune('a'+parseIndex)) {
			parseT.Fatalf("expected canonical sort at index %d, got %#v", parseIndex, parseLargeEntries)
		}
	}

	sortBinarySourceAnyMapEntries([]buildBinarySourceAnyMapEntry{{getKey: "z"}})
}

// TestBuildBinarySourceValueReflectInto verifies reflect-based encoding handles pointers, interfaces, slices, arrays, maps, structs, and unsupported kinds.
func TestBuildBinarySourceValueReflectInto(parseT *testing.T) {
	type parseExample struct {
		Title  string
		Count  int
		hidden string
	}

	var parseNilInterface any
	parseInterfaceValue := "x"
	parsePointerValue := 7
	parseCases := []struct {
		name  string
		value reflect.Value
		want  any
	}{
		{name: "invalid", value: reflect.Value{}, want: nil},
		{name: "nil-interface", value: reflect.ValueOf(&parseNilInterface).Elem(), want: nil},
		{name: "non-nil-interface", value: reflect.ValueOf(&parseInterfaceValue).Elem(), want: "x"},
		{name: "nil-pointer", value: reflect.ValueOf((*int)(nil)), want: nil},
		{name: "bool", value: reflect.ValueOf(true), want: true},
		{name: "int", value: reflect.ValueOf(int(2)), want: float64(2)},
		{name: "uint", value: reflect.ValueOf(uint(3)), want: float64(3)},
		{name: "float", value: reflect.ValueOf(float32(4.5)), want: float64(4.5)},
		{name: "string", value: reflect.ValueOf("ok"), want: "ok"},
		{name: "slice-nil", value: reflect.ValueOf([]any(nil)), want: nil},
		{name: "slice", value: reflect.ValueOf([]any{true, "x"}), want: []any{true, "x"}},
		{name: "array", value: reflect.ValueOf([2]any{true, "x"}), want: []any{true, "x"}},
		{name: "map-nil", value: reflect.ValueOf(map[string]any(nil)), want: nil},
		{name: "map", value: reflect.ValueOf(map[string]any{"b": 2, "a": 1}), want: map[string]any{"a": float64(1), "b": float64(2)}},
		{name: "struct", value: reflect.ValueOf(parseExample{Title: "A", Count: 3, hidden: "skip"}), want: map[string]any{"Count": float64(3), "Title": "A"}},
		{name: "pointer", value: reflect.ValueOf(&parsePointerValue), want: float64(7)},
	}
	for _, parseCase := range parseCases {
		parseT.Run(parseCase.name, func(parseT *testing.T) {
			parsePayload, parseErr := buildBinarySourceValueReflectInto(nil, parseCase.value)
			if parseErr != nil {
				parseT.Fatalf("buildBinarySourceValueReflectInto returned error: %v", parseErr)
			}
			parseDecoded, parseErr := ParseBinarySourceValue(parsePayload)
			if parseErr != nil {
				parseT.Fatalf("ParseBinarySourceValue returned error: %v", parseErr)
			}
			if !reflect.DeepEqual(parseDecoded, parseCase.want) {
				parseT.Fatalf("expected %#v, got %#v", parseCase.want, parseDecoded)
			}
		})
	}

	if _, parseErr := buildBinarySourceValueReflectInto(nil, reflect.ValueOf(make(chan int))); parseErr == nil {
		parseT.Fatal("expected unsupported channel kind to fail")
	}

	if _, parseErr := buildBinarySourceListInto(nil, reflect.ValueOf([]any{true, "x"})); parseErr != nil {
		parseT.Fatalf("buildBinarySourceListInto returned error: %v", parseErr)
	}
	if _, parseErr := buildBinarySourceListInto(nil, reflect.ValueOf([binarySourceValueListLimit + 1]any{})); parseErr == nil {
		parseT.Fatal("expected oversized reflect list to fail")
	}

	if _, parseErr := buildBinarySourceMapInto(nil, reflect.ValueOf(map[int]any{1: true})); parseErr == nil {
		parseT.Fatal("expected non-string map key kind to fail")
	}
	if _, parseErr := buildBinarySourceMapInto(nil, reflect.ValueOf(map[string]any{})); parseErr != nil {
		parseT.Fatalf("buildBinarySourceMapInto empty map returned error: %v", parseErr)
	}
	if _, parseErr := buildBinarySourceMapInto(nil, reflect.ValueOf(map[string]any{strings.Repeat("a", 0x10000): true})); parseErr == nil {
		parseT.Fatal("expected oversized reflect map key to fail")
	}
	if _, parseErr := buildBinarySourceMapInto(nil, reflect.ValueOf(map[string]any{"a": complex(1, 2)})); parseErr == nil {
		parseT.Fatal("expected unsupported reflect map value to fail")
	}

	parseStructPayload, parseErr := buildBinarySourceStructInto(nil, reflect.ValueOf(parseExample{Title: "A", Count: 3, hidden: "skip"}))
	if parseErr != nil {
		parseT.Fatalf("buildBinarySourceStructInto returned error: %v", parseErr)
	}
	parseStructDecoded, parseErr := ParseBinarySourceValue(parseStructPayload)
	if parseErr != nil {
		parseT.Fatalf("ParseBinarySourceValue(struct) returned error: %v", parseErr)
	}
	if parseStructMap, parseOk := parseStructDecoded.(map[string]any); !parseOk || len(parseStructMap) != 2 {
		parseT.Fatalf("expected exported fields only, got %#v", parseStructDecoded)
	}

	parseType := reflect.TypeOf(parseExample{})
	if parseType.NumField() != 3 {
		parseT.Fatal("unexpected test type shape")
	}
}
