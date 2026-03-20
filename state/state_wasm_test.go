//go:build js && wasm
// +build js,wasm

package state

import (
	"syscall/js"
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

type noOpScheduler struct{}

func (noOpScheduler) RequestIdleCallback(callback func(runtime.Deadline)) {}

func (noOpScheduler) SetTimeout(callback func(), delay int) {}

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

	snapshot := ExportSnapshot()
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
