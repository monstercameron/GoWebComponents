package runtime

import (
	"reflect"
	"testing"
)

func newHotReloadTestComponent(parseName string) *ComponentType {
	return NewComponentType("example/"+parseName, parseName, "example/"+parseName, nil, nil)
}

func TestHotReloadNormalizeHelpers(parseT *testing.T) {
	parseNormalized := normalizeHotReloadFetchState(FetchState{
		Data: map[string]interface{}{
			"count": float64(3),
			"items": []interface{}{float64(1), float64(2.5)},
		},
		Error: "   ",
	})
	if parseNormalized.Error != "" {
		parseT.Fatalf("expected normalized empty error, got %q", parseNormalized.Error)
	}
	parseData, parseOk := parseNormalized.Data.(map[string]interface{})
	if !parseOk {
		parseT.Fatalf("expected normalized map data, got %T", parseNormalized.Data)
	}
	if _, parseOk2 := parseData["count"].(int); !parseOk2 {
		parseT.Fatalf("expected integer normalization for whole float values, got %#v", parseData["count"])
	}

	if normalizeHotReloadDeps(nil) != nil {
		parseT.Fatalf("expected nil deps normalization for empty input")
	}
	parseDeps := normalizeHotReloadDeps([]interface{}{float64(4), map[string]interface{}{"n": float64(5)}})
	if len(parseDeps) != 2 {
		parseT.Fatalf("expected deps normalization output")
	}
	if _, parseOk3 := parseDeps[0].(int); !parseOk3 {
		parseT.Fatalf("expected integer normalization for dep, got %T", parseDeps[0])
	}

	if _, parseOk4 := NormalizeHotReloadValue(float64(7)).(int); !parseOk4 {
		parseT.Fatalf("expected exported normalizer to coerce whole floats to int")
	}
}

func TestCoerceHotReloadValue(parseT *testing.T) {
	if parseValue, parseOk := coerceHotReloadValue("hello", reflect.TypeOf("")); !parseOk || parseValue.(string) != "hello" {
		parseT.Fatalf("expected assignable coercion, got %#v ok=%t", parseValue, parseOk)
	}
	if parseValue2, parseOk2 := coerceHotReloadValue(float64(9), reflect.TypeOf(int(0))); !parseOk2 || parseValue2.(int) != 9 {
		parseT.Fatalf("expected convertible coercion, got %#v ok=%t", parseValue2, parseOk2)
	}
	if parseValue3, parseOk3 := coerceHotReloadValue(123, reflect.TypeOf((*interface{})(nil)).Elem()); !parseOk3 || parseValue3.(int) != 123 {
		parseT.Fatalf("expected empty interface coercion, got %#v ok=%t", parseValue3, parseOk3)
	}

	type payload struct {
		Name string `json:"name"`
	}
	if parseValue4, parseOk4 := coerceHotReloadValue(map[string]interface{}{"name": "cam"}, reflect.TypeOf(payload{})); !parseOk4 || parseValue4.(payload).Name != "cam" {
		parseT.Fatalf("expected JSON struct coercion, got %#v ok=%t", parseValue4, parseOk4)
	}
	if parseValue5, parseOk5 := coerceHotReloadValue(map[string]interface{}{"name": "cam"}, reflect.TypeOf(&payload{})); !parseOk5 || parseValue5.(*payload).Name != "cam" {
		parseT.Fatalf("expected JSON pointer coercion, got %#v ok=%t", parseValue5, parseOk5)
	}

	if _, parseOk6 := coerceHotReloadValue(func() {}, reflect.TypeOf(payload{})); parseOk6 {
		parseT.Fatalf("expected non-serializable value coercion to fail")
	}
}

func TestHotReloadSnapshotCaptureAndRestoreHelpers(parseT *testing.T) {
	parseRt := &Runtime{}
	if parseSnapshot := parseRt.CaptureHotReloadSnapshot(); len(parseSnapshot.Components) != 0 {
		parseT.Fatalf("expected empty snapshot for nil current root, got %+v", parseSnapshot)
	}
	if parseRt.HasPendingHotReloadSnapshot() {
		parseT.Fatalf("expected no pending snapshot initially")
	}

	parseComponentFiber := &Fiber{
		typeOf: newHotReloadTestComponent("Counter"),
		props:  map[string]interface{}{"key": "counter-key"},
		hooks: &Hooks{
			signature: []string{"state", "memo", "ref", "id", "fetch"},
			states:    []interface{}{float64(11), nil},
			memos:     []memoizedValue{{value: map[string]interface{}{"n": float64(3)}, deps: []interface{}{float64(5)}}},
			refs:      []*RefValue{{Current: float64(4)}},
			ids:       []string{"id-1"},
			fetches:   []fetchValue{{url: "/api/items", state: FetchState{Loading: true, Data: map[string]interface{}{"total": float64(6)}}}},
		},
	}
	parseRoot := &Fiber{typeOf: "ROOT", child: parseComponentFiber}
	parseComponentFiber.parent = parseRoot
	parseRt.currentRoot = parseRoot

	parseSnapshot2 := parseRt.CaptureHotReloadSnapshot()
	if len(parseSnapshot2.Components) != 1 {
		parseT.Fatalf("expected one captured component snapshot, got %+v", parseSnapshot2)
	}
	parseCaptured := parseSnapshot2.Components[0]
	if parseCaptured.Path == "" || len(parseCaptured.IdentityTrail) == 0 {
		parseT.Fatalf("expected snapshot path and identity trail, got %+v", parseCaptured)
	}
	if len(parseCaptured.States) != 1 || len(parseCaptured.Memos) != 1 || len(parseCaptured.Refs) != 1 || len(parseCaptured.IDs) != 1 || len(parseCaptured.Fetches) != 1 {
		parseT.Fatalf("expected captured hook snapshot values, got %+v", parseCaptured)
	}

	parseRt.RestoreHotReloadSnapshot(parseSnapshot2)
	if !parseRt.HasPendingHotReloadSnapshot() {
		parseT.Fatalf("expected pending snapshot after restore")
	}
	if parseGot := parseRt.nextHotReloadComponentSnapshot(); parseGot == nil {
		parseT.Fatalf("expected next pending component snapshot")
	}
	if parseRt.HasPendingHotReloadSnapshot() {
		parseT.Fatalf("expected pending snapshots consumed after next call")
	}
}

func TestHotReloadHooksRestoreAndCompatibilityHelpers(parseT *testing.T) {
	parseRestore := &HotReloadComponentSnapshot{
		Signature: ComponentSignature{
			Kind:          "component",
			Name:          "Counter",
			QualifiedName: "example/Counter",
			Key:           "counter-key",
			HookKinds:     []string{"state", "memo", "ref", "id", "fetch"},
		},
		States: []interface{}{float64(2)},
		Memos:  []HotReloadMemoSnapshot{{Value: map[string]interface{}{"n": float64(7)}, Deps: []interface{}{float64(9)}}},
		Refs:   []interface{}{float64(10)},
		IDs:    []string{"", "id-restored"},
		Fetches: []HotReloadFetchSnapshot{{
			URL: "/api/items",
			State: FetchState{
				Loading: true,
				Data:    map[string]interface{}{"count": float64(3)},
			},
		}},
	}
	parseHooks := &Hooks{hotReloadRestore: parseRestore}
	if parseState, parseOk := parseHooks.restoreStateValue(0); !parseOk || parseState.(int) != 2 {
		parseT.Fatalf("restore state value failed: %#v ok=%t", parseState, parseOk)
	}
	if parseMemoValue, parseDeps, parseOk2 := parseHooks.restoreMemoValue(0); !parseOk2 || parseMemoValue.(map[string]interface{})["n"].(int) != 7 || parseDeps[0].(int) != 9 {
		parseT.Fatalf("restore memo value failed: value=%#v deps=%#v ok=%t", parseMemoValue, parseDeps, parseOk2)
	}
	if parseRefValue, parseOk3 := parseHooks.restoreRefValue(0); !parseOk3 || parseRefValue.(int) != 10 {
		parseT.Fatalf("restore ref value failed: %#v ok=%t", parseRefValue, parseOk3)
	}
	if _, parseOk4 := parseHooks.restoreIDValue(0); parseOk4 {
		parseT.Fatalf("expected blank restored id to be rejected")
	}
	if parseId, parseOk5 := parseHooks.restoreIDValue(1); !parseOk5 || parseId != "id-restored" {
		parseT.Fatalf("restore id value failed: %q ok=%t", parseId, parseOk5)
	}
	if parseFetchState, parseOk6 := parseHooks.restoreFetchValue(0, "/api/items"); !parseOk6 || parseFetchState.Data.(map[string]interface{})["count"].(int) != 3 {
		parseT.Fatalf("restore fetch value failed: %#v ok=%t", parseFetchState, parseOk6)
	}
	if _, parseOk7 := parseHooks.restoreFetchValue(0, "/api/other"); parseOk7 {
		parseT.Fatalf("expected url mismatch to skip fetch restore")
	}

	parseComponentFiber := &Fiber{
		typeOf: newHotReloadTestComponent("Counter"),
		props:  map[string]interface{}{"key": "counter-key"},
		hooks:  &Hooks{signature: []string{"state", "memo", "ref", "id", "fetch"}},
	}
	if !componentSnapshotCompatible(parseRestore, parseComponentFiber) {
		parseT.Fatalf("expected base snapshot compatibility")
	}
	if !componentSnapshotFullyCompatible(parseRestore, parseComponentFiber) {
		parseT.Fatalf("expected full snapshot compatibility")
	}
	if !componentSnapshotSerializableCompatible(parseRestore, parseComponentFiber) {
		parseT.Fatalf("expected serializable snapshot compatibility")
	}
	parseComponentFiber.hooks.signature = []string{"state", "effect"}
	if componentSnapshotFullyCompatible(parseRestore, parseComponentFiber) {
		parseT.Fatalf("expected full compatibility to fail after hook signature change")
	}
}
