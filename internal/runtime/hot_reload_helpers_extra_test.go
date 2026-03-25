package runtime

import (
	"reflect"
	"testing"
)

func newHotReloadTestComponent(name string) *ComponentType {
	return NewComponentType("example/"+name, name, "example/"+name, nil, nil)
}

func TestHotReloadNormalizeHelpers(t *testing.T) {
	normalized := normalizeHotReloadFetchState(FetchState{
		Data: map[string]interface{}{
			"count": float64(3),
			"items": []interface{}{float64(1), float64(2.5)},
		},
		Error: "   ",
	})
	if normalized.Error != "" {
		t.Fatalf("expected normalized empty error, got %q", normalized.Error)
	}
	data, ok := normalized.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected normalized map data, got %T", normalized.Data)
	}
	if _, ok := data["count"].(int); !ok {
		t.Fatalf("expected integer normalization for whole float values, got %#v", data["count"])
	}

	if normalizeHotReloadDeps(nil) != nil {
		t.Fatalf("expected nil deps normalization for empty input")
	}
	deps := normalizeHotReloadDeps([]interface{}{float64(4), map[string]interface{}{"n": float64(5)}})
	if len(deps) != 2 {
		t.Fatalf("expected deps normalization output")
	}
	if _, ok := deps[0].(int); !ok {
		t.Fatalf("expected integer normalization for dep, got %T", deps[0])
	}

	if _, ok := NormalizeHotReloadValue(float64(7)).(int); !ok {
		t.Fatalf("expected exported normalizer to coerce whole floats to int")
	}
}

func TestCoerceHotReloadValue(t *testing.T) {
	if value, ok := coerceHotReloadValue("hello", reflect.TypeOf("")); !ok || value.(string) != "hello" {
		t.Fatalf("expected assignable coercion, got %#v ok=%t", value, ok)
	}
	if value, ok := coerceHotReloadValue(float64(9), reflect.TypeOf(int(0))); !ok || value.(int) != 9 {
		t.Fatalf("expected convertible coercion, got %#v ok=%t", value, ok)
	}
	if value, ok := coerceHotReloadValue(123, reflect.TypeOf((*interface{})(nil)).Elem()); !ok || value.(int) != 123 {
		t.Fatalf("expected empty interface coercion, got %#v ok=%t", value, ok)
	}

	type payload struct {
		Name string `json:"name"`
	}
	if value, ok := coerceHotReloadValue(map[string]interface{}{"name": "cam"}, reflect.TypeOf(payload{})); !ok || value.(payload).Name != "cam" {
		t.Fatalf("expected JSON struct coercion, got %#v ok=%t", value, ok)
	}
	if value, ok := coerceHotReloadValue(map[string]interface{}{"name": "cam"}, reflect.TypeOf(&payload{})); !ok || value.(*payload).Name != "cam" {
		t.Fatalf("expected JSON pointer coercion, got %#v ok=%t", value, ok)
	}

	if _, ok := coerceHotReloadValue(func() {}, reflect.TypeOf(payload{})); ok {
		t.Fatalf("expected non-serializable value coercion to fail")
	}
}

func TestHotReloadSnapshotCaptureAndRestoreHelpers(t *testing.T) {
	rt := &Runtime{}
	if snapshot := rt.CaptureHotReloadSnapshot(); len(snapshot.Components) != 0 {
		t.Fatalf("expected empty snapshot for nil current root, got %+v", snapshot)
	}
	if rt.HasPendingHotReloadSnapshot() {
		t.Fatalf("expected no pending snapshot initially")
	}

	componentFiber := &Fiber{
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
	root := &Fiber{typeOf: "ROOT", child: componentFiber}
	componentFiber.parent = root
	rt.currentRoot = root

	snapshot := rt.CaptureHotReloadSnapshot()
	if len(snapshot.Components) != 1 {
		t.Fatalf("expected one captured component snapshot, got %+v", snapshot)
	}
	captured := snapshot.Components[0]
	if captured.Path == "" || len(captured.IdentityTrail) == 0 {
		t.Fatalf("expected snapshot path and identity trail, got %+v", captured)
	}
	if len(captured.States) != 1 || len(captured.Memos) != 1 || len(captured.Refs) != 1 || len(captured.IDs) != 1 || len(captured.Fetches) != 1 {
		t.Fatalf("expected captured hook snapshot values, got %+v", captured)
	}

	rt.RestoreHotReloadSnapshot(snapshot)
	if !rt.HasPendingHotReloadSnapshot() {
		t.Fatalf("expected pending snapshot after restore")
	}
	if got := rt.nextHotReloadComponentSnapshot(); got == nil {
		t.Fatalf("expected next pending component snapshot")
	}
	if rt.HasPendingHotReloadSnapshot() {
		t.Fatalf("expected pending snapshots consumed after next call")
	}
}

func TestHotReloadHooksRestoreAndCompatibilityHelpers(t *testing.T) {
	restore := &HotReloadComponentSnapshot{
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
	hooks := &Hooks{hotReloadRestore: restore}
	if state, ok := hooks.restoreStateValue(0); !ok || state.(int) != 2 {
		t.Fatalf("restore state value failed: %#v ok=%t", state, ok)
	}
	if memoValue, deps, ok := hooks.restoreMemoValue(0); !ok || memoValue.(map[string]interface{})["n"].(int) != 7 || deps[0].(int) != 9 {
		t.Fatalf("restore memo value failed: value=%#v deps=%#v ok=%t", memoValue, deps, ok)
	}
	if refValue, ok := hooks.restoreRefValue(0); !ok || refValue.(int) != 10 {
		t.Fatalf("restore ref value failed: %#v ok=%t", refValue, ok)
	}
	if _, ok := hooks.restoreIDValue(0); ok {
		t.Fatalf("expected blank restored id to be rejected")
	}
	if id, ok := hooks.restoreIDValue(1); !ok || id != "id-restored" {
		t.Fatalf("restore id value failed: %q ok=%t", id, ok)
	}
	if fetchState, ok := hooks.restoreFetchValue(0, "/api/items"); !ok || fetchState.Data.(map[string]interface{})["count"].(int) != 3 {
		t.Fatalf("restore fetch value failed: %#v ok=%t", fetchState, ok)
	}
	if _, ok := hooks.restoreFetchValue(0, "/api/other"); ok {
		t.Fatalf("expected url mismatch to skip fetch restore")
	}

	componentFiber := &Fiber{
		typeOf: newHotReloadTestComponent("Counter"),
		props:  map[string]interface{}{"key": "counter-key"},
		hooks:  &Hooks{signature: []string{"state", "memo", "ref", "id", "fetch"}},
	}
	if !componentSnapshotCompatible(restore, componentFiber) {
		t.Fatalf("expected base snapshot compatibility")
	}
	if !componentSnapshotFullyCompatible(restore, componentFiber) {
		t.Fatalf("expected full snapshot compatibility")
	}
	if !componentSnapshotSerializableCompatible(restore, componentFiber) {
		t.Fatalf("expected serializable snapshot compatibility")
	}
	componentFiber.hooks.signature = []string{"state", "effect"}
	if componentSnapshotFullyCompatible(restore, componentFiber) {
		t.Fatalf("expected full compatibility to fail after hook signature change")
	}
}
