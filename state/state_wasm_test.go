//go:build js && wasm
// +build js,wasm

package state

import (
	"context"
	"fmt"
	"syscall/js"
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
	"github.com/monstercameron/GoWebComponents/ui"
)

type noOpScheduler struct{}

func (noOpScheduler) RequestIdleCallback(callback func(runtime.Deadline)) {}

func (noOpScheduler) SetTimeout(callback func(), delay int) {}

type queuedScheduler struct {
	timeouts []func()
}

func setGlobalValue(name string, value interface{}) func() {
	global := js.Global()
	prev := global.Get(name)
	global.Set(name, value)
	return func() {
		global.Set(name, prev)
	}
}

func (s *queuedScheduler) RequestIdleCallback(callback func(runtime.Deadline)) {}

func (s *queuedScheduler) SetTimeout(callback func(), delay int) {
	s.timeouts = append(s.timeouts, callback)
}

func (s *queuedScheduler) Flush() {
	for len(s.timeouts) > 0 {
		pending := append([]func(){}, s.timeouts...)
		s.timeouts = s.timeouts[:0]
		for _, callback := range pending {
			callback()
		}
	}
}

func installStateHookContext(t *testing.T) {
	t.Helper()
	runtime.InitGlobalRuntime(runtime.Config{Scheduler: noOpScheduler{}})
	runtime.SetCurrentFiber(&runtime.Fiber{})
	t.Cleanup(func() {
		runtime.SetCurrentFiber(nil)
	})
}

func TestUseAtomSharesGlobalState(t *testing.T) {
	runtime.InitGlobalRuntime(runtime.Config{Scheduler: noOpScheduler{}})

	atomID := "state-test-shared"

	runtime.SetCurrentFiber(&runtime.Fiber{})
	first := UseAtom(atomID, "Guest")
	if first.Get() != "Guest" {
		t.Fatalf("expected initial atom value, got %q", first.Get())
	}
	first.Set("Alice")
	if first.Get() != "Alice" {
		t.Fatalf("expected updated atom value, got %q", first.Get())
	}

	runtime.SetCurrentFiber(&runtime.Fiber{})
	second := UseAtom(atomID, "Ignored")
	if second.Get() != "Alice" {
		t.Fatalf("expected second hook to reuse shared atom state, got %q", second.Get())
	}
	second.Update(func(prev string) string { return prev + " Smith" })
	if second.Get() != "Alice Smith" {
		t.Fatalf("expected updater to modify shared atom state, got %q", second.Get())
	}

	runtime.SetCurrentFiber(nil)
}

func TestUseAtomWrapper(t *testing.T) {
	installStateHookContext(t)

	atom := UseAtom("state-test-wrapper", 1)
	if atom.Get() != 1 {
		t.Fatalf("expected initial wrapper atom value, got %d", atom.Get())
	}
	atom.Set(3)
	atom.Update(func(prev int) int { return prev * 2 })
	if atom.Get() != 6 {
		t.Fatalf("expected wrapper methods to update atom value, got %d", atom.Get())
	}
}

func TestUseComputedWrapper(t *testing.T) {
	installStateHookContext(t)

	count := UseAtom("state-test-computed", 2)
	doubled := UseComputed(func() int {
		return count.Get() * 2
	}, count.Get())

	if doubled.Get() != 4 {
		t.Fatalf("expected computed value 4, got %d", doubled.Get())
	}
}

func TestComputedZeroValue(t *testing.T) {
	var computed Computed[string]
	if computed.Get() != "" {
		t.Fatalf("expected zero-value computed handle to return empty string, got %q", computed.Get())
	}
}

func TestUseDerivedTracksSharedDerivedAtom(t *testing.T) {
	installStateHookContext(t)
	count := UseAtom("state-test-derived-count", 2)
	double := UseDerived("state-test-derived-double", func() int {
		return count.Get() * 2
	}, "state-test-derived-count")

	if double.Get() != 4 {
		t.Fatalf("expected initial derived value 4, got %d", double.Get())
	}
	count.Set(5)
	if double.Get() != 10 {
		t.Fatalf("expected derived value 10 after atom update, got %d", double.Get())
	}
}

func TestSelectProjectsAtomValues(t *testing.T) {
	installStateHookContext(t)
	count := UseAtom("state-test-select-count", 2)
	parity := Select("state-test-select-parity", count, func(value int) string {
		if value%2 == 0 {
			return "even"
		}
		return "odd"
	})

	if parity.Get() != "even" {
		t.Fatalf("expected initial selector value even, got %q", parity.Get())
	}
	count.Set(5)
	if parity.Get() != "odd" {
		t.Fatalf("expected selector value odd after atom update, got %q", parity.Get())
	}
}

func TestSelectProjectsDerivedValues(t *testing.T) {
	installStateHookContext(t)
	count := UseAtom("state-test-select-derived-count", 3)
	double := UseDerived("state-test-select-double", func() int {
		return count.Get() * 2
	}, "state-test-select-derived-count")
	label := Select("state-test-select-label", double, func(value int) string {
		return fmt.Sprintf("value:%d", value)
	})

	if label.Get() != "value:6" {
		t.Fatalf("expected initial projected derived value value:6, got %q", label.Get())
	}
	count.Set(4)
	if label.Get() != "value:8" {
		t.Fatalf("expected projected derived value value:8 after source update, got %q", label.Get())
	}
}

func TestSelectScopesRequestedIDsPerHookContext(t *testing.T) {
	runtime.InitGlobalRuntime(runtime.Config{Scheduler: noOpScheduler{}})
	countAID := "state-test-select-scope-a"
	countBID := "state-test-select-scope-b"

	runtime.SetCurrentFiber(&runtime.Fiber{})
	countA := UseAtom(countAID, 1)
	selectorA := Select("shared-selector", countA, func(value int) string {
		return fmt.Sprintf("a:%d", value)
	})

	runtime.SetCurrentFiber(&runtime.Fiber{})
	countB := UseAtom(countBID, 10)
	selectorB := Select("shared-selector", countB, func(value int) string {
		return fmt.Sprintf("b:%d", value)
	})

	t.Cleanup(func() {
		runtime.SetCurrentFiber(nil)
	})

	if selectorA.Get() != "a:1" {
		t.Fatalf("expected first selector value a:1, got %q", selectorA.Get())
	}
	if selectorB.Get() != "b:10" {
		t.Fatalf("expected second selector value b:10, got %q", selectorB.Get())
	}

	countA.Set(2)
	if selectorA.Get() != "a:2" {
		t.Fatalf("expected first selector to remain scoped to atom A, got %q", selectorA.Get())
	}
	if selectorB.Get() != "b:10" {
		t.Fatalf("expected second selector to remain isolated after atom A update, got %q", selectorB.Get())
	}

	countB.Set(11)
	if selectorA.Get() != "a:2" {
		t.Fatalf("expected first selector to remain isolated after atom B update, got %q", selectorA.Get())
	}
	if selectorB.Get() != "b:11" {
		t.Fatalf("expected second selector to remain scoped to atom B, got %q", selectorB.Get())
	}
}

func TestAtomTextBuildsReactiveTextElement(t *testing.T) {
	installStateHookContext(t)
	count := UseAtom("state-test-text-atom", 2)
	node := count.Text(func(value int) string {
		return fmt.Sprintf("count:%d", value)
	})
	if node == nil {
		t.Fatal("expected reactive text element")
	}
	if node.Type != runtime.ReactiveTextNodeType {
		t.Fatalf("expected reactive text node type, got %#v", node.Type)
	}
	if got, _ := node.Props["__gwc_reactive_text_atom_id"].(string); got != "state-test-text-atom" {
		t.Fatalf("expected reactive text atom id to round-trip, got %q", got)
	}
	getter, _ := node.Props["__gwc_reactive_text_getter"].(func() string)
	if getter == nil {
		t.Fatal("expected reactive text getter")
	}
	if got := getter(); got != "count:2" {
		t.Fatalf("expected reactive text getter to render count:2, got %q", got)
	}
	count.Set(4)
	if got := getter(); got != "count:4" {
		t.Fatalf("expected reactive text getter to observe latest atom value, got %q", got)
	}
}

func TestSelectTextBuildsReactiveTextElement(t *testing.T) {
	installStateHookContext(t)
	count := UseAtom("state-test-text-select-count", 1)
	parity := Select("state-test-text-select-parity", count, func(value int) string {
		if value%2 == 0 {
			return "even"
		}
		return "odd"
	})
	node := parity.Text(func(value string) string {
		return "parity:" + value
	})
	if node == nil {
		t.Fatal("expected selector-backed reactive text element")
	}
	getter, _ := node.Props["__gwc_reactive_text_getter"].(func() string)
	if getter == nil {
		t.Fatal("expected selector-backed reactive text getter")
	}
	if got := getter(); got != "parity:odd" {
		t.Fatalf("expected initial selector text parity:odd, got %q", got)
	}
	count.Set(2)
	if got := getter(); got != "parity:even" {
		t.Fatalf("expected selector text parity:even after source update, got %q", got)
	}
}

func TestImportSnapshotTransitionViaUIPublicWrapperDefersUntilTimeout(t *testing.T) {
	scheduler := &queuedScheduler{}
	runtime.InitGlobalRuntime(runtime.Config{Scheduler: scheduler})
	fiber := &runtime.Fiber{}
	runtime.SetCurrentFiber(fiber)
	t.Cleanup(func() {
		runtime.SetCurrentFiber(nil)
	})

	theme := UseAtom("state-test-public-transition-theme", "light")

	ui.StartTransition(func() {
		if err := ImportSnapshot(Snapshot{"state-test-public-transition-theme": "dark"}); err != nil {
			t.Fatalf("unexpected import snapshot error inside transition: %v", err)
		}
	})

	if theme.Get() != "light" {
		t.Fatalf("expected shared state update to stay deferred before timeout, got %q", theme.Get())
	}
	if len(scheduler.timeouts) != 1 {
		t.Fatalf("expected one deferred timeout for public transition-wrapped snapshot import, got %d", len(scheduler.timeouts))
	}

	scheduler.Flush()

	if theme.Get() != "dark" {
		t.Fatalf("expected deferred shared state update after timeout, got %q", theme.Get())
	}
}

func TestDerivedZeroValue(t *testing.T) {
	var derived Derived[string]
	if derived.Get() != "" {
		t.Fatalf("expected zero-value derived handle to return empty string, got %q", derived.Get())
	}
}

func installMockStorage(t *testing.T) {
	t.Helper()
	global := js.Global()
	objectCtor := global.Get("Object")
	reflectObj := global.Get("Reflect")

	prevLocal := global.Get("localStorage")
	prevSession := global.Get("sessionStorage")

	makeStorage := func() (js.Value, []js.Func) {
		data := objectCtor.New()
		setItem := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			data.Set(args[0].String(), args[1].String())
			return nil
		})
		getItem := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			value := data.Get(args[0].String())
			if value.IsUndefined() {
				return js.Null()
			}
			return value
		})
		removeItem := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			reflectObj.Call("deleteProperty", data, args[0].String())
			return nil
		})
		storage := objectCtor.New()
		storage.Set("setItem", setItem)
		storage.Set("getItem", getItem)
		storage.Set("removeItem", removeItem)
		return storage, []js.Func{setItem, getItem, removeItem}
	}

	local, localFns := makeStorage()
	session, sessionFns := makeStorage()
	global.Set("localStorage", local)
	global.Set("sessionStorage", session)

	t.Cleanup(func() {
		global.Set("localStorage", prevLocal)
		global.Set("sessionStorage", prevSession)
		for _, fn := range localFns {
			fn.Release()
		}
		for _, fn := range sessionFns {
			fn.Release()
		}
	})
}

func TestSnapshotExportImportAndSelect(t *testing.T) {
	runtime.InitGlobalRuntime(runtime.Config{Scheduler: noOpScheduler{}})
	_ = runtime.GetGlobalRuntime().SetAtomValue("persist-theme", "dark")
	_ = runtime.GetGlobalRuntime().SetAtomValue("persist-flag", true)

	snapshot, err := ExportSnapshot()
	if err != nil {
		t.Fatalf("expected snapshot export, got %v", err)
	}
	if snapshot["persist-theme"] != "dark" {
		t.Fatalf("expected exported theme atom, got %#v", snapshot["persist-theme"])
	}

	selected := snapshot.Select("persist-flag")
	if len(selected) != 1 || selected["persist-flag"] != true {
		t.Fatalf("expected selected snapshot to keep one key, got %#v", selected)
	}

	if err := ImportSnapshot(Snapshot{"persist-theme": "light"}); err != nil {
		t.Fatalf("unexpected import error: %v", err)
	}
	value, ok := runtime.GetGlobalRuntime().GetAtomValue("persist-theme")
	if !ok || value != "light" {
		t.Fatalf("expected imported snapshot to restore atom, got %#v ok=%t", value, ok)
	}
}

func TestSnapshotJSONStorageRoundTrip(t *testing.T) {
	installMockStorage(t)
	runtime.InitGlobalRuntime(runtime.Config{Scheduler: noOpScheduler{}})

	snapshot := Snapshot{
		"persist-user":  "alice",
		"persist-ready": true,
	}
	if err := SaveSnapshot("app-state", snapshot, LocalStorage); err != nil {
		t.Fatalf("unexpected save error: %v", err)
	}

	loaded, ok, err := LoadSnapshot("app-state", LocalStorage)
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}
	if !ok {
		t.Fatal("expected snapshot to be present in storage")
	}
	if loaded["persist-user"] != "alice" || loaded["persist-ready"] != true {
		t.Fatalf("unexpected loaded snapshot: %#v", loaded)
	}

	if restored, err := RestoreSnapshot("app-state", LocalStorage); err != nil || !restored {
		t.Fatalf("expected restore from storage to succeed, restored=%t err=%v", restored, err)
	}
	value, _ := runtime.GetGlobalRuntime().GetAtomValue("persist-user")
	if value != "alice" {
		t.Fatalf("expected restored user atom, got %#v", value)
	}
}

func TestPersistentSnapshotRoundTrip(t *testing.T) {
	installMockStorage(t)
	restoreIndexedDB := setGlobalValue("indexedDB", js.Undefined())
	defer restoreIndexedDB()
	runtime.InitGlobalRuntime(runtime.Config{Scheduler: noOpScheduler{}})

	snapshot := Snapshot{
		"persist-user":  "atlas",
		"persist-ready": true,
	}
	if err := SavePersistentSnapshot(context.Background(), "app-state-persistent", snapshot); err != nil {
		t.Fatalf("unexpected persistent save error: %v", err)
	}

	loaded, ok, err := LoadPersistentSnapshot(context.Background(), "app-state-persistent")
	if err != nil {
		t.Fatalf("unexpected persistent load error: %v", err)
	}
	if !ok {
		t.Fatal("expected persistent snapshot to be present")
	}
	if loaded["persist-user"] != "atlas" || loaded["persist-ready"] != true {
		t.Fatalf("unexpected persistent loaded snapshot: %#v", loaded)
	}

	if restored, err := RestorePersistentSnapshot(context.Background(), "app-state-persistent"); err != nil || !restored {
		t.Fatalf("expected persistent restore to succeed, restored=%t err=%v", restored, err)
	}
	value, _ := runtime.GetGlobalRuntime().GetAtomValue("persist-user")
	if value != "atlas" {
		t.Fatalf("expected persistent restored user atom, got %#v", value)
	}
}

func TestUnmarshalSnapshotJSONNormalizesWholeNumbers(t *testing.T) {
	snapshot, err := UnmarshalSnapshotJSON([]byte(`{"count":1,"ratio":1.5,"nested":{"items":[2,2.5]}}`))
	if err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	count, ok := snapshot["count"].(int)
	if !ok || count != 1 {
		t.Fatalf("expected whole number to normalize to int, got %#v", snapshot["count"])
	}

	ratio, ok := snapshot["ratio"].(float64)
	if !ok || ratio != 1.5 {
		t.Fatalf("expected non-whole number to remain float64, got %#v", snapshot["ratio"])
	}

	nested := snapshot["nested"].(map[string]interface{})
	items := nested["items"].([]interface{})
	if first, ok := items[0].(int); !ok || first != 2 {
		t.Fatalf("expected nested whole number to normalize to int, got %#v", items[0])
	}
	if second, ok := items[1].(float64); !ok || second != 2.5 {
		t.Fatalf("expected nested non-whole number to remain float64, got %#v", items[1])
	}
}
