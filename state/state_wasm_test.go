//go:build js && wasm

package state

import (
	"context"
	"fmt"
	"syscall/js"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/internal/runtime"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

type noOpScheduler struct{}

func (noOpScheduler) RequestIdleCallback(parseCallback func(runtime.Deadline)) {}

func (noOpScheduler) SetTimeout(parseCallback func(), parseDelay int) {}

type queuedScheduler struct {
	timeouts []func()
}

func setGlobalValue(parseName string, parseValue interface{}) func() {
	parseGlobal := js.Global()
	parsePrev := parseGlobal.Get(parseName)
	parseGlobal.Set(parseName, parseValue)
	return func() {
		parseGlobal.Set(parseName, parsePrev)
	}
}

func (parseS *queuedScheduler) RequestIdleCallback(parseCallback func(runtime.Deadline)) {}

func (parseS *queuedScheduler) SetTimeout(parseCallback func(), parseDelay int) {
	parseS.timeouts = append(parseS.timeouts, parseCallback)
}

func (parseS *queuedScheduler) Flush() {
	for len(parseS.timeouts) > 0 {
		parsePending := append([]func(){}, parseS.timeouts...)
		parseS.timeouts = parseS.timeouts[:0]
		for _, parseCallback := range parsePending {
			parseCallback()
		}
	}
}

func installStateHookContext(parseT *testing.T) {
	parseT.Helper()
	runtime.InitGlobalRuntime(runtime.Config{Scheduler: noOpScheduler{}})
	runtime.SetCurrentFiber(&runtime.Fiber{})
	parseT.Cleanup(func() {
		runtime.SetCurrentFiber(nil)
	})
}

func TestUseAtomSharesGlobalState(parseT *testing.T) {
	runtime.InitGlobalRuntime(runtime.Config{Scheduler: noOpScheduler{}})

	parseAtomID := "state-test-shared"

	runtime.SetCurrentFiber(&runtime.Fiber{})
	parseFirst := UseAtom(parseAtomID, "Guest")
	if parseFirst.Get() != "Guest" {
		parseT.Fatalf("expected initial atom value, got %q", parseFirst.Get())
	}
	parseFirst.Set("Alice")
	if parseFirst.Get() != "Alice" {
		parseT.Fatalf("expected updated atom value, got %q", parseFirst.Get())
	}

	runtime.SetCurrentFiber(&runtime.Fiber{})
	parseSecond := UseAtom(parseAtomID, "Ignored")
	if parseSecond.Get() != "Alice" {
		parseT.Fatalf("expected second hook to reuse shared atom state, got %q", parseSecond.Get())
	}
	parseSecond.Update(func(parsePrev string) string { return parsePrev + " Smith" })
	if parseSecond.Get() != "Alice Smith" {
		parseT.Fatalf("expected updater to modify shared atom state, got %q", parseSecond.Get())
	}

	runtime.SetCurrentFiber(nil)
}

func TestUseAtomWrapper(parseT *testing.T) {
	installStateHookContext(parseT)

	parseAtom := UseAtom("state-test-wrapper", 1)
	if parseAtom.Get() != 1 {
		parseT.Fatalf("expected initial wrapper atom value, got %d", parseAtom.Get())
	}
	parseAtom.Set(3)
	parseAtom.Update(func(parsePrev int) int { return parsePrev * 2 })
	if parseAtom.Get() != 6 {
		parseT.Fatalf("expected wrapper methods to update atom value, got %d", parseAtom.Get())
	}
}

func TestUseComputedWrapper(parseT *testing.T) {
	installStateHookContext(parseT)

	parseCount := UseAtom("state-test-computed", 2)
	parseDoubled := UseComputed(func() int {
		return parseCount.Get() * 2
	}, parseCount.Get())

	if parseDoubled.Get() != 4 {
		parseT.Fatalf("expected computed value 4, got %d", parseDoubled.Get())
	}
}

func TestComputedZeroValue(parseT *testing.T) {
	var parseComputed Computed[string]
	if parseComputed.Get() != "" {
		parseT.Fatalf("expected zero-value computed handle to return empty string, got %q", parseComputed.Get())
	}
}

func TestUseDerivedTracksSharedDerivedAtom(parseT *testing.T) {
	installStateHookContext(parseT)
	parseCount := UseAtom("state-test-derived-count", 2)
	parseDouble := UseDerived("state-test-derived-double", func() int {
		return parseCount.Get() * 2
	}, "state-test-derived-count")

	if parseDouble.Get() != 4 {
		parseT.Fatalf("expected initial derived value 4, got %d", parseDouble.Get())
	}
	parseCount.Set(5)
	if parseDouble.Get() != 10 {
		parseT.Fatalf("expected derived value 10 after atom update, got %d", parseDouble.Get())
	}
}

func TestSelectProjectsAtomValues(parseT *testing.T) {
	installStateHookContext(parseT)
	parseCount := UseAtom("state-test-select-count", 2)
	parseParity := Select("state-test-select-parity", parseCount, func(parseValue int) string {
		if parseValue%2 == 0 {
			return "even"
		}
		return "odd"
	})

	if parseParity.Get() != "even" {
		parseT.Fatalf("expected initial selector value even, got %q", parseParity.Get())
	}
	parseCount.Set(5)
	if parseParity.Get() != "odd" {
		parseT.Fatalf("expected selector value odd after atom update, got %q", parseParity.Get())
	}
}

func TestSelectProjectsDerivedValues(parseT *testing.T) {
	installStateHookContext(parseT)
	parseCount := UseAtom("state-test-select-derived-count", 3)
	parseDouble := UseDerived("state-test-select-double", func() int {
		return parseCount.Get() * 2
	}, "state-test-select-derived-count")
	parseLabel := Select("state-test-select-label", parseDouble, func(parseValue int) string {
		return fmt.Sprintf("value:%d", parseValue)
	})

	if parseLabel.Get() != "value:6" {
		parseT.Fatalf("expected initial projected derived value value:6, got %q", parseLabel.Get())
	}
	parseCount.Set(4)
	if parseLabel.Get() != "value:8" {
		parseT.Fatalf("expected projected derived value value:8 after source update, got %q", parseLabel.Get())
	}
}

func TestSelectScopesRequestedIDsPerHookContext(parseT *testing.T) {
	runtime.InitGlobalRuntime(runtime.Config{Scheduler: noOpScheduler{}})
	parseCountAID := "state-test-select-scope-a"
	parseCountBID := "state-test-select-scope-b"

	runtime.SetCurrentFiber(&runtime.Fiber{})
	parseCountA := UseAtom(parseCountAID, 1)
	parseSelectorA := Select("shared-selector", parseCountA, func(parseValue int) string {
		return fmt.Sprintf("a:%d", parseValue)
	})

	runtime.SetCurrentFiber(&runtime.Fiber{})
	parseCountB := UseAtom(parseCountBID, 10)
	parseSelectorB := Select("shared-selector", parseCountB, func(parseValue2 int) string {
		return fmt.Sprintf("b:%d", parseValue2)
	})

	parseT.Cleanup(func() {
		runtime.SetCurrentFiber(nil)
	})

	if parseSelectorA.Get() != "a:1" {
		parseT.Fatalf("expected first selector value a:1, got %q", parseSelectorA.Get())
	}
	if parseSelectorB.Get() != "b:10" {
		parseT.Fatalf("expected second selector value b:10, got %q", parseSelectorB.Get())
	}

	parseCountA.Set(2)
	if parseSelectorA.Get() != "a:2" {
		parseT.Fatalf("expected first selector to remain scoped to atom A, got %q", parseSelectorA.Get())
	}
	if parseSelectorB.Get() != "b:10" {
		parseT.Fatalf("expected second selector to remain isolated after atom A update, got %q", parseSelectorB.Get())
	}

	parseCountB.Set(11)
	if parseSelectorA.Get() != "a:2" {
		parseT.Fatalf("expected first selector to remain isolated after atom B update, got %q", parseSelectorA.Get())
	}
	if parseSelectorB.Get() != "b:11" {
		parseT.Fatalf("expected second selector to remain scoped to atom B, got %q", parseSelectorB.Get())
	}
}

func TestAtomTextBuildsReactiveTextElement(parseT *testing.T) {
	installStateHookContext(parseT)
	parseCount := UseAtom("state-test-text-atom", 2)
	parseNode := parseCount.Text(func(parseValue int) string {
		return fmt.Sprintf("count:%d", parseValue)
	})
	if parseNode == nil {
		parseT.Fatal("expected reactive text element")
	}
	if parseNode.Type != runtime.ReactiveTextNodeType {
		parseT.Fatalf("expected reactive text node type, got %#v", parseNode.Type)
	}
	if parseGot, _ := parseNode.Props["__gwc_reactive_text_atom_id"].(string); parseGot != "state-test-text-atom" {
		parseT.Fatalf("expected reactive text atom id to round-trip, got %q", parseGot)
	}
	parseGetter, _ := parseNode.Props["__gwc_reactive_text_getter"].(func() string)
	if parseGetter == nil {
		parseT.Fatal("expected reactive text getter")
	}
	if parseGot2 := parseGetter(); parseGot2 != "count:2" {
		parseT.Fatalf("expected reactive text getter to render count:2, got %q", parseGot2)
	}
	parseCount.Set(4)
	if parseGot3 := parseGetter(); parseGot3 != "count:4" {
		parseT.Fatalf("expected reactive text getter to observe latest atom value, got %q", parseGot3)
	}
}

func TestSelectTextBuildsReactiveTextElement(parseT *testing.T) {
	installStateHookContext(parseT)
	parseCount := UseAtom("state-test-text-select-count", 1)
	parseParity := Select("state-test-text-select-parity", parseCount, func(parseValue int) string {
		if parseValue%2 == 0 {
			return "even"
		}
		return "odd"
	})
	parseNode := parseParity.Text(func(parseValue2 string) string {
		return "parity:" + parseValue2
	})
	if parseNode == nil {
		parseT.Fatal("expected selector-backed reactive text element")
	}
	parseGetter, _ := parseNode.Props["__gwc_reactive_text_getter"].(func() string)
	if parseGetter == nil {
		parseT.Fatal("expected selector-backed reactive text getter")
	}
	if parseGot := parseGetter(); parseGot != "parity:odd" {
		parseT.Fatalf("expected initial selector text parity:odd, got %q", parseGot)
	}
	parseCount.Set(2)
	if parseGot2 := parseGetter(); parseGot2 != "parity:even" {
		parseT.Fatalf("expected selector text parity:even after source update, got %q", parseGot2)
	}
}

func TestImportSnapshotTransitionViaUIPublicWrapperDefersUntilTimeout(parseT *testing.T) {
	parseScheduler := &queuedScheduler{}
	runtime.InitGlobalRuntime(runtime.Config{Scheduler: parseScheduler})
	parseFiber := &runtime.Fiber{}
	runtime.SetCurrentFiber(parseFiber)
	parseT.Cleanup(func() {
		runtime.SetCurrentFiber(nil)
	})

	parseTheme := UseAtom("state-test-public-transition-theme", "light")

	ui.StartTransition(func() {
		if parseErr := ImportSnapshot(Snapshot{"state-test-public-transition-theme": "dark"}); parseErr != nil {
			parseT.Fatalf("unexpected import snapshot error inside transition: %v", parseErr)
		}
	})

	if parseTheme.Get() != "light" {
		parseT.Fatalf("expected shared state update to stay deferred before timeout, got %q", parseTheme.Get())
	}
	if len(parseScheduler.timeouts) != 1 {
		parseT.Fatalf("expected one deferred timeout for public transition-wrapped snapshot import, got %d", len(parseScheduler.timeouts))
	}

	parseScheduler.Flush()

	if parseTheme.Get() != "dark" {
		parseT.Fatalf("expected deferred shared state update after timeout, got %q", parseTheme.Get())
	}
}

func TestDerivedZeroValue(parseT *testing.T) {
	var parseDerived Derived[string]
	if parseDerived.Get() != "" {
		parseT.Fatalf("expected zero-value derived handle to return empty string, got %q", parseDerived.Get())
	}
}

func installMockStorage(parseT *testing.T) {
	parseT.Helper()
	parseGlobal := js.Global()
	parseObjectCtor := parseGlobal.Get("Object")
	parseReflectObj := parseGlobal.Get("Reflect")

	parsePrevLocal := parseGlobal.Get("localStorage")
	parsePrevSession := parseGlobal.Get("sessionStorage")

	parseMakeStorage := func() (js.Value, []js.Func) {
		parseData := parseObjectCtor.New()
		setItem := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
			parseData.Set(parseArgs[0].String(), parseArgs[1].String())
			return nil
		})
		getItem := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
			parseValue := parseData.Get(parseArgs2[0].String())
			if parseValue.IsUndefined() {
				return js.Null()
			}
			return parseValue
		})
		parseRemoveItem := js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} {
			parseReflectObj.Call("deleteProperty", parseData, parseArgs3[0].String())
			return nil
		})
		parseStorage := parseObjectCtor.New()
		parseStorage.Set("setItem", setItem)
		parseStorage.Set("getItem", getItem)
		parseStorage.Set("removeItem", parseRemoveItem)
		return parseStorage, []js.Func{setItem, getItem, parseRemoveItem}
	}

	parseLocal, parseLocalFns := parseMakeStorage()
	parseSession, parseSessionFns := parseMakeStorage()
	parseGlobal.Set("localStorage", parseLocal)
	parseGlobal.Set("sessionStorage", parseSession)

	parseT.Cleanup(func() {
		parseGlobal.Set("localStorage", parsePrevLocal)
		parseGlobal.Set("sessionStorage", parsePrevSession)
		for _, parseFn := range parseLocalFns {
			parseFn.Release()
		}
		for _, parseFn2 := range parseSessionFns {
			parseFn2.Release()
		}
	})
}

func TestSnapshotExportImportAndSelect(parseT *testing.T) {
	runtime.InitGlobalRuntime(runtime.Config{Scheduler: noOpScheduler{}})
	_ = runtime.GetGlobalRuntime().SetAtomValue("persist-theme", "dark")
	_ = runtime.GetGlobalRuntime().SetAtomValue("persist-flag", true)

	parseSnapshot, parseErr := ExportSnapshot()
	if parseErr != nil {
		parseT.Fatalf("expected snapshot export, got %v", parseErr)
	}
	if parseSnapshot["persist-theme"] != "dark" {
		parseT.Fatalf("expected exported theme atom, got %#v", parseSnapshot["persist-theme"])
	}

	parseSelected := parseSnapshot.Select("persist-flag")
	if len(parseSelected) != 1 || parseSelected["persist-flag"] != true {
		parseT.Fatalf("expected selected snapshot to keep one key, got %#v", parseSelected)
	}

	if parseErr2 := ImportSnapshot(Snapshot{"persist-theme": "light"}); parseErr2 != nil {
		parseT.Fatalf("unexpected import error: %v", parseErr2)
	}
	parseValue, parseOk := runtime.GetGlobalRuntime().GetAtomValue("persist-theme")
	if !parseOk || parseValue != "light" {
		parseT.Fatalf("expected imported snapshot to restore atom, got %#v ok=%t", parseValue, parseOk)
	}
}

func TestSnapshotJSONStorageRoundTrip(parseT *testing.T) {
	installMockStorage(parseT)
	runtime.InitGlobalRuntime(runtime.Config{Scheduler: noOpScheduler{}})

	parseSnapshot := Snapshot{
		"persist-user":  "alice",
		"persist-ready": true,
	}
	if parseErr := SaveSnapshot("app-state", parseSnapshot, LocalStorage); parseErr != nil {
		parseT.Fatalf("unexpected save error: %v", parseErr)
	}

	parseLoaded, parseOk, parseErr2 := LoadSnapshot("app-state", LocalStorage)
	if parseErr2 != nil {
		parseT.Fatalf("unexpected load error: %v", parseErr2)
	}
	if !parseOk {
		parseT.Fatal("expected snapshot to be present in storage")
	}
	if parseLoaded["persist-user"] != "alice" || parseLoaded["persist-ready"] != true {
		parseT.Fatalf("unexpected loaded snapshot: %#v", parseLoaded)
	}

	if parseRestored, parseErr3 := RestoreSnapshot("app-state", LocalStorage); parseErr3 != nil || !parseRestored {
		parseT.Fatalf("expected restore from storage to succeed, restored=%t err=%v", parseRestored, parseErr3)
	}
	parseValue, _ := runtime.GetGlobalRuntime().GetAtomValue("persist-user")
	if parseValue != "alice" {
		parseT.Fatalf("expected restored user atom, got %#v", parseValue)
	}
}

func TestPersistentSnapshotRoundTrip(parseT *testing.T) {
	installMockStorage(parseT)
	parseRestoreIndexedDB := setGlobalValue("indexedDB", js.Undefined())
	defer parseRestoreIndexedDB()
	runtime.InitGlobalRuntime(runtime.Config{Scheduler: noOpScheduler{}})

	parseSnapshot := Snapshot{
		"persist-user":  "atlas",
		"persist-ready": true,
	}
	if parseErr := SavePersistentSnapshot(context.Background(), "app-state-persistent", parseSnapshot); parseErr != nil {
		parseT.Fatalf("unexpected persistent save error: %v", parseErr)
	}

	parseLoaded, parseOk, parseErr2 := LoadPersistentSnapshot(context.Background(), "app-state-persistent")
	if parseErr2 != nil {
		parseT.Fatalf("unexpected persistent load error: %v", parseErr2)
	}
	if !parseOk {
		parseT.Fatal("expected persistent snapshot to be present")
	}
	if parseLoaded["persist-user"] != "atlas" || parseLoaded["persist-ready"] != true {
		parseT.Fatalf("unexpected persistent loaded snapshot: %#v", parseLoaded)
	}

	if parseRestored, parseErr3 := RestorePersistentSnapshot(context.Background(), "app-state-persistent"); parseErr3 != nil || !parseRestored {
		parseT.Fatalf("expected persistent restore to succeed, restored=%t err=%v", parseRestored, parseErr3)
	}
	parseValue, _ := runtime.GetGlobalRuntime().GetAtomValue("persist-user")
	if parseValue != "atlas" {
		parseT.Fatalf("expected persistent restored user atom, got %#v", parseValue)
	}
}

func TestUnmarshalSnapshotJSONNormalizesWholeNumbers(parseT *testing.T) {
	parseSnapshot, parseErr := UnmarshalSnapshotJSON([]byte(`{"count":1,"ratio":1.5,"nested":{"items":[2,2.5]}}`))
	if parseErr != nil {
		parseT.Fatalf("unexpected unmarshal error: %v", parseErr)
	}

	parseCount, parseOk := parseSnapshot["count"].(int)
	if !parseOk || parseCount != 1 {
		parseT.Fatalf("expected whole number to normalize to int, got %#v", parseSnapshot["count"])
	}

	parseRatio, parseOk := parseSnapshot["ratio"].(float64)
	if !parseOk || parseRatio != 1.5 {
		parseT.Fatalf("expected non-whole number to remain float64, got %#v", parseSnapshot["ratio"])
	}

	parseNested := parseSnapshot["nested"].(map[string]interface{})
	parseItems := parseNested["items"].([]interface{})
	if parseFirst, parseOk2 := parseItems[0].(int); !parseOk2 || parseFirst != 2 {
		parseT.Fatalf("expected nested whole number to normalize to int, got %#v", parseItems[0])
	}
	if parseSecond, parseOk3 := parseItems[1].(float64); !parseOk3 || parseSecond != 2.5 {
		parseT.Fatalf("expected nested non-whole number to remain float64, got %#v", parseItems[1])
	}
}
