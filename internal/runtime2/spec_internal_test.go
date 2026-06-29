package runtime2

import (
	"reflect"
	"strings"
	"testing"
)

// TestSerializablePropsFlatShapeHelpersMatchAndRejectUnsupportedTypes verifies flat-shape helper functions classify scalar types and reject mismatched cached prop shapes.
func TestSerializablePropsFlatShapeHelpersMatchAndRejectUnsupportedTypes(parseT *testing.T) {
	parseTypeMarkerTests := []struct {
		name           string
		parseValue     any
		wantTypeMarker uint64
		wantOK         bool
	}{
		{name: "nil", parseValue: nil, wantTypeMarker: 1, wantOK: true},
		{name: "bool", parseValue: true, wantTypeMarker: 2, wantOK: true},
		{name: "int", parseValue: int(1), wantTypeMarker: 3, wantOK: true},
		{name: "int8", parseValue: int8(1), wantTypeMarker: 4, wantOK: true},
		{name: "int16", parseValue: int16(1), wantTypeMarker: 5, wantOK: true},
		{name: "int32", parseValue: int32(1), wantTypeMarker: 6, wantOK: true},
		{name: "int64", parseValue: int64(1), wantTypeMarker: 7, wantOK: true},
		{name: "uint", parseValue: uint(1), wantTypeMarker: 8, wantOK: true},
		{name: "uint8", parseValue: uint8(1), wantTypeMarker: 9, wantOK: true},
		{name: "uint16", parseValue: uint16(1), wantTypeMarker: 10, wantOK: true},
		{name: "uint32", parseValue: uint32(1), wantTypeMarker: 11, wantOK: true},
		{name: "uint64", parseValue: uint64(1), wantTypeMarker: 12, wantOK: true},
		{name: "uintptr", parseValue: uintptr(1), wantTypeMarker: 13, wantOK: true},
		{name: "float32", parseValue: float32(1), wantTypeMarker: 14, wantOK: true},
		{name: "float64", parseValue: float64(1), wantTypeMarker: 15, wantOK: true},
		{name: "string", parseValue: "orders", wantTypeMarker: 16, wantOK: true},
		{name: "unsupported", parseValue: []int{1}, wantTypeMarker: 0, wantOK: false},
	}
	for _, parseTypeMarkerTest := range parseTypeMarkerTests {
		getTypeMarker, hasTypeMarker := buildSerializablePropsFlatTypeMarker(parseTypeMarkerTest.parseValue)
		if hasTypeMarker != parseTypeMarkerTest.wantOK || getTypeMarker != parseTypeMarkerTest.wantTypeMarker {
			parseT.Fatalf("%s type marker = (%d, %t), want (%d, %t)", parseTypeMarkerTest.name, getTypeMarker, hasTypeMarker, parseTypeMarkerTest.wantTypeMarker, parseTypeMarkerTest.wantOK)
		}
	}

	getKeyCount, _, _, getScratchKeys, getScratchTypeMarkers, hasFingerprint := buildSerializablePropsFlatShapeFingerprintWithTypeScratch(
		map[string]any{
			"count": 5,
			"title": "Orders",
		},
		nil,
		nil,
	)
	if !hasFingerprint {
		parseT.Fatal("expected flat props fingerprint")
	}
	if getKeyCount != 2 || len(getScratchKeys) != 2 || getScratchKeys[0] != "count" || getScratchKeys[1] != "title" || len(getScratchTypeMarkers) != 2 {
		parseT.Fatalf("flat props fingerprint scratch state = (%d, %+v, %+v), want sorted keys [count title]", getKeyCount, getScratchKeys, getScratchTypeMarkers)
	}
	if !hasSerializablePropsFlatShapeFingerprintMatch(map[string]any{"count": 9, "title": "Invoices"}, getKeyCount, getScratchKeys, getScratchTypeMarkers) {
		parseT.Fatal("expected matching flat props shape fingerprint")
	}
	if hasSerializablePropsFlatShapeFingerprintMatch(map[string]any{"count": "9", "title": "Invoices"}, getKeyCount, getScratchKeys, getScratchTypeMarkers) {
		parseT.Fatal("expected type-changed flat props shape fingerprint to fail")
	}
	if hasSerializablePropsFlatShapeFingerprintMatch(map[string]any{"count": 9}, getKeyCount, getScratchKeys, getScratchTypeMarkers) {
		parseT.Fatal("expected key-count mismatch to fail flat props fingerprint match")
	}
	if hasSerializablePropsFlatShapeFingerprintMatch(map[string]any{"count": 9, "title": map[string]any{"nested": true}}, getKeyCount, getScratchKeys, getScratchTypeMarkers) {
		parseT.Fatal("expected unsupported nested prop value to fail flat props fingerprint match")
	}
}

// TestSerializableAnyFastHandlesTypedMapsClosuresAndUnsafeKeys verifies the fast serializable classifier accepts supported containers and rejects guarded map markers and unsupported runtime values.
func TestSerializableAnyFastHandlesTypedMapsClosuresAndUnsafeKeys(parseT *testing.T) {
	parseSerializableTests := []struct {
		name       string
		parseValue any
		wantOK     bool
	}{
		{name: "nil", parseValue: nil, wantOK: true},
		{name: "bool-slice", parseValue: []bool{true, false}, wantOK: true},
		{name: "string-slice", parseValue: []string{"a", "b"}, wantOK: true},
		{name: "nested-any", parseValue: []any{"orders", map[string]any{"count": 3}}, wantOK: true},
		{name: "nested-any-unsupported", parseValue: []any{"orders", make(chan int)}, wantOK: false},
		{name: "typed-bool-map", parseValue: map[string]bool{"ready": true}, wantOK: true},
		{name: "typed-int-map", parseValue: map[string]int{"count": 3}, wantOK: true},
		{name: "typed-uint-map", parseValue: map[string]uint64{"count": 3}, wantOK: true},
		{name: "typed-string-map", parseValue: map[string]string{"title": "Orders"}, wantOK: true},
		{name: "typed-string-map-ref", parseValue: map[string]string{"ref": "node-1"}, wantOK: false},
		{name: "map-dom-interop", parseValue: map[string]any{"dom_ref": "node-1"}, wantOK: false},
		{name: "map-event-closure", parseValue: map[string]any{"onClick": func() {}}, wantOK: false},
		{name: "unsupported-kind", parseValue: make(chan int), wantOK: false},
	}
	for _, parseSerializableTest := range parseSerializableTests {
		if isSerializableAnyFast(parseSerializableTest.parseValue) != parseSerializableTest.wantOK {
			parseT.Fatalf("%s serializable fast result mismatch", parseSerializableTest.name)
		}
	}

	if !isSerializableScalarMapFast(map[string]string{"title": "Orders"}) {
		parseT.Fatal("expected safe typed scalar map to pass fast validation")
	}
	if isSerializableScalarMapFast(map[string]string{"ref": "node-1"}) {
		parseT.Fatal("expected typed scalar map ref marker to fail fast validation")
	}
}

// TestSerializableHelperFunctionsClassifyKeysAndClosures verifies helper classifiers recognize safe keys, normalize guarded names, and follow interface and pointer-wrapped event closures.
func TestSerializableHelperFunctionsClassifyKeysAndClosures(parseT *testing.T) {
	if hasSerializableFunctionValueAnyFast(nil) {
		parseT.Fatal("expected nil function-any fast path to report false")
	}
	if !hasSerializableFunctionValueAnyFast(func() {}) {
		parseT.Fatal("expected function-any fast path to report true")
	}
	if hasSerializableFunctionValueAnyFast(3) {
		parseT.Fatal("expected non-function any fast path to report false")
	}

	if !hasSerializableSafeMapKey("title1") {
		parseT.Fatal("expected lowercase alphanumeric key to be safe")
	}
	if hasSerializableSafeMapKey("ref") {
		parseT.Fatal("expected reserved ref key to be unsafe")
	}
	if hasSerializableSafeMapKey("onclick") {
		parseT.Fatal("expected reserved event-like key to be unsafe")
	}
	if hasSerializableSafeMapKey("Title") {
		parseT.Fatal("expected uppercase key to be unsafe")
	}
	if hasSerializableSafeMapKey("bad-key") {
		parseT.Fatal("expected punctuation key to be unsafe")
	}

	if getNormalizedName := getSerializableNormalizedName(" OnClick "); getNormalizedName != "onclick" {
		parseT.Fatalf("normalized event key = %q, want %q", getNormalizedName, "onclick")
	}
	if getNormalizedName := getSerializableNormalizedName("title"); getNormalizedName != "title" {
		parseT.Fatalf("normalized lowercase key = %q, want %q", getNormalizedName, "title")
	}

	getEventClosure := func() {}
	getEventClosurePointer := &getEventClosure
	var getNilEventClosurePointer *func()
	if !isEventClosureValue(reflect.ValueOf(getEventClosure)) {
		parseT.Fatal("expected direct function value to be treated as event closure")
	}
	if !isEventClosureValue(reflect.ValueOf(getEventClosurePointer)) {
		parseT.Fatal("expected pointer-wrapped function value to be treated as event closure")
	}
	if isEventClosureValue(reflect.ValueOf(getNilEventClosurePointer)) {
		parseT.Fatal("expected nil function pointer not to be treated as event closure")
	}
	if isEventClosureValue(reflect.Value{}) {
		parseT.Fatal("expected invalid reflect value not to be treated as event closure")
	}
}

// TestValidateSerializableValueReportsDetailedInternalFailures verifies the recursive validator keeps precise error paths for unsupported internal helper cases.
func TestValidateSerializableValueReportsDetailedInternalFailures(parseT *testing.T) {
	if parseErr := validateSerializableValue(reflect.Value{}, "props"); parseErr != nil {
		parseT.Fatalf("validateSerializableValue(invalid) returned error: %v", parseErr)
	}
	if parseErr := validateSerializableValue(reflect.ValueOf((*int)(nil)), "props"); parseErr != nil {
		parseT.Fatalf("validateSerializableValue(nil pointer) returned error: %v", parseErr)
	}

	parseMapKeyErr := validateSerializableValue(reflect.ValueOf(map[int]string{1: "bad"}), "props")
	if parseMapKeyErr == nil || !strings.Contains(parseMapKeyErr.Error(), "unsupported map key kind") {
		parseT.Fatalf("expected unsupported map key kind error, got %v", parseMapKeyErr)
	}

	parseRefErr := validateSerializableValue(reflect.ValueOf(map[string]any{"ref": "node-1"}), "props")
	if parseRefErr == nil || !strings.Contains(parseRefErr.Error(), "props.ref") || !strings.Contains(parseRefErr.Error(), "ref marker") {
		parseT.Fatalf("expected ref-marker error path, got %v", parseRefErr)
	}

	parseDOMErr := validateSerializableValue(reflect.ValueOf(struct {
		DOMNode string
	}{
		DOMNode: "node-1",
	}), "props")
	if parseDOMErr == nil || !strings.Contains(parseDOMErr.Error(), "props.DOMNode") || !strings.Contains(parseDOMErr.Error(), "direct DOM interop marker") {
		parseT.Fatalf("expected DOM interop field error path, got %v", parseDOMErr)
	}

	parseEventClosureErr := validateSerializableValue(reflect.ValueOf(map[string]any{"onClick": func() {}}), "props")
	if parseEventClosureErr == nil || !strings.Contains(parseEventClosureErr.Error(), "props.onClick") || !strings.Contains(parseEventClosureErr.Error(), "event-closure prop") {
		parseT.Fatalf("expected event-closure map error path, got %v", parseEventClosureErr)
	}

	parseNestedKindErr := validateSerializableValue(reflect.ValueOf(map[string]any{
		"items": []any{map[string]any{"channel": make(chan int)}},
	}), "props")
	if parseNestedKindErr == nil || !strings.Contains(parseNestedKindErr.Error(), "props.items[0].channel") || !strings.Contains(parseNestedKindErr.Error(), "unsupported kind chan") {
		parseT.Fatalf("expected nested unsupported-kind path, got %v", parseNestedKindErr)
	}
}
