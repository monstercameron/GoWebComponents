//go:build js && wasm

package virtualization

import (
	"fmt"
	"strings"
	"syscall/js"
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/html"
	"github.com/monstercameron/GoWebComponents/v4/internal/platform/mockdom"
	"github.com/monstercameron/GoWebComponents/v4/internal/runtime"
	"github.com/monstercameron/GoWebComponents/v4/ui"
)

// buildListWASMQueuedScheduler stores queued idle and timeout callbacks so tests can advance effects deterministically.
type buildListWASMQueuedScheduler struct {
	getIdleCallbacks []func(runtime.Deadline)
	getTimeouts      []func()
}

// buildListWASMQueuedDeadline reports abundant time so virtualization effects can run without timeout pressure.
type buildListWASMQueuedDeadline struct{}

// TimeRemaining reports enough remaining time for queued test work.
func (buildListWASMQueuedDeadline) TimeRemaining() float64 { return 1000 }

// DidTimeout reports false for the synthetic queued test deadline.
func (buildListWASMQueuedDeadline) DidTimeout() bool { return false }

// RequestIdleCallback records one idle callback for later flushing.
func (parseS *buildListWASMQueuedScheduler) RequestIdleCallback(parseCallback func(runtime.Deadline)) {
	parseS.getIdleCallbacks = append(parseS.getIdleCallbacks, parseCallback)
}

// SetTimeout records one timeout callback for later flushing.
func (parseS *buildListWASMQueuedScheduler) SetTimeout(parseCallback func(), parseDelay int) {
	_ = parseDelay
	parseS.getTimeouts = append(parseS.getTimeouts, parseCallback)
}

// Flush drains queued idle and timeout callbacks until the scheduler is empty.
func (parseS *buildListWASMQueuedScheduler) Flush() {
	for len(parseS.getIdleCallbacks) > 0 {
		parsePendingIdle := append([]func(runtime.Deadline){}, parseS.getIdleCallbacks...)
		parseS.getIdleCallbacks = parseS.getIdleCallbacks[:0]
		for _, parseCallback := range parsePendingIdle {
			parseCallback(buildListWASMQueuedDeadline{})
		}
	}
	for len(parseS.getTimeouts) > 0 {
		parsePendingTimeouts := append([]func(){}, parseS.getTimeouts...)
		parseS.getTimeouts = parseS.getTimeouts[:0]
		for _, parseCallback2 := range parsePendingTimeouts {
			parseCallback2()
		}
		for len(parseS.getIdleCallbacks) > 0 {
			parsePendingIdle2 := append([]func(runtime.Deadline){}, parseS.getIdleCallbacks...)
			parseS.getIdleCallbacks = parseS.getIdleCallbacks[:0]
			for _, parseCallback3 := range parsePendingIdle2 {
				parseCallback3(buildListWASMQueuedDeadline{})
			}
		}
	}
}

// buildListWASMBrowserHarness keeps the mocked browser objects needed by virtualization list effects and exposes simple trigger helpers.
type buildListWASMBrowserHarness struct {
	getObjectCtor     js.Value
	getElement        js.Value
	getSessionStorage js.Value
	getListeners      map[string]js.Value
	getResizeCallback *js.Value
}

// getListWASMStorageItem reads one persisted sessionStorage key from the mocked browser harness.
func (parseH buildListWASMBrowserHarness) getListWASMStorageItem(parseKey string) string {
	parseValue := parseH.getSessionStorage.Call("getItem", parseKey)
	if parseValue.IsUndefined() || parseValue.IsNull() {
		return ""
	}
	return parseValue.String()
}

// setListWASMStorageItem writes one mocked sessionStorage key for restoration setup.
func (parseH buildListWASMBrowserHarness) setListWASMStorageItem(parseKey, parseValue string) {
	parseH.getSessionStorage.Call("setItem", parseKey, parseValue)
}

// triggerListWASMScroll dispatches one synthetic scroll event through the mocked element listener.
func (parseH buildListWASMBrowserHarness) triggerListWASMScroll(parseT *testing.T) {
	parseT.Helper()
	parseListener, parseOk := parseH.getListeners["scroll"]
	if !parseOk || parseListener.IsUndefined() || parseListener.IsNull() {
		parseT.Fatal("expected mocked scroll listener")
	}
	parseEvent := parseH.getObjectCtor.New()
	parseEvent.Set("type", "scroll")
	parseEvent.Set("target", parseH.getElement)
	parseEvent.Set("currentTarget", parseH.getElement)
	parseListener.Invoke(parseEvent)
}

// triggerListWASMResize dispatches one synthetic resize observer callback for the mocked list element.
func (parseH buildListWASMBrowserHarness) triggerListWASMResize(parseT *testing.T, parseViewportHeight float64) {
	parseT.Helper()
	if parseH.getResizeCallback == nil || parseH.getResizeCallback.IsUndefined() || parseH.getResizeCallback.IsNull() {
		parseT.Fatal("expected mocked resize observer callback")
	}
	parseH.getElement.Set("clientHeight", parseViewportHeight)
	parseRect := parseH.getObjectCtor.New()
	parseRect.Set("x", 0)
	parseRect.Set("y", 0)
	parseRect.Set("width", 0)
	parseRect.Set("height", parseViewportHeight)
	parseRect.Set("top", 0)
	parseRect.Set("right", 0)
	parseRect.Set("bottom", parseViewportHeight)
	parseRect.Set("left", 0)
	parseEntry := parseH.getObjectCtor.New()
	parseEntry.Set("target", parseH.getElement)
	parseEntry.Set("contentRect", parseRect)
	parseEntries := js.Global().Get("Array").New()
	parseEntries.Call("push", parseEntry)
	parseH.getResizeCallback.Invoke(parseEntries)
}

// installListWASMBrowserHarness installs mocked document, sessionStorage, and ResizeObserver globals that are sufficient for virtualization list effects.
func installListWASMBrowserHarness(parseT *testing.T, parseListID string, parseScrollTop, parseScrollHeight, parseClientHeight float64) buildListWASMBrowserHarness {
	parseT.Helper()
	parseGlobal := js.Global()
	parseObjectCtor := parseGlobal.Get("Object")
	parseReflectObj := parseGlobal.Get("Reflect")
	parsePrevDocument := parseGlobal.Get("document")
	parsePrevSession := parseGlobal.Get("sessionStorage")
	parsePrevResizeObserver := parseGlobal.Get("ResizeObserver")

	parseSessionData := parseObjectCtor.New()
	parseSetItem := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseSessionData.Set(parseArgs[0].String(), parseArgs[1].String())
		return nil
	})
	parseGetItem := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
		parseValue := parseSessionData.Get(parseArgs2[0].String())
		if parseValue.IsUndefined() {
			return js.Null()
		}
		return parseValue
	})
	parseRemoveItem := js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} {
		parseReflectObj.Call("deleteProperty", parseSessionData, parseArgs3[0].String())
		return nil
	})
	parseSessionStorage := parseObjectCtor.New()
	parseSessionStorage.Set("setItem", parseSetItem)
	parseSessionStorage.Set("getItem", parseGetItem)
	parseSessionStorage.Set("removeItem", parseRemoveItem)
	parseGlobal.Set("sessionStorage", parseSessionStorage)

	parseListeners := map[string]js.Value{}
	parseElement := parseObjectCtor.New()
	parseElement.Set("id", parseListID)
	parseElement.Set("tagName", "DIV")
	parseElement.Set("className", "virtualization-list")
	parseElement.Set("scrollTop", parseScrollTop)
	parseElement.Set("scrollHeight", parseScrollHeight)
	parseElement.Set("clientHeight", parseClientHeight)
	parseAddEventListener := js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} {
		if len(parseArgs4) >= 2 {
			parseListeners[parseArgs4[0].String()] = parseArgs4[1]
		}
		return nil
	})
	parseRemoveEventListener := js.FuncOf(func(parseThis5 js.Value, parseArgs5 []js.Value) interface{} {
		if len(parseArgs5) >= 2 {
			parseEventName := parseArgs5[0].String()
			if parseListener, parseOk := parseListeners[parseEventName]; parseOk && parseListener.Equal(parseArgs5[1]) {
				delete(parseListeners, parseEventName)
			}
		}
		return nil
	})
	parseElement.Set("addEventListener", parseAddEventListener)
	parseElement.Set("removeEventListener", parseRemoveEventListener)

	parseDocument := parseObjectCtor.New()
	parseGetElementByID := js.FuncOf(func(parseThis6 js.Value, parseArgs6 []js.Value) interface{} {
		if len(parseArgs6) > 0 && parseArgs6[0].String() == parseListID {
			return parseElement
		}
		return js.Null()
	})
	parseDocument.Set("getElementById", parseGetElementByID)
	parseGlobal.Set("document", parseDocument)

	var parseResizeCallback js.Value
	var parseResizeObserve js.Func
	var parseResizeDisconnect js.Func
	hasResizeObserve := false
	hasResizeDisconnect := false
	parseResizeObserverCtor := js.FuncOf(func(parseThis7 js.Value, parseArgs7 []js.Value) interface{} {
		if len(parseArgs7) > 0 {
			parseResizeCallback = parseArgs7[0]
		}
		parseObserver := parseObjectCtor.New()
		parseResizeObserve = js.FuncOf(func(parseThis8 js.Value, parseArgs8 []js.Value) interface{} {
			return nil
		})
		hasResizeObserve = true
		parseResizeDisconnect = js.FuncOf(func(parseThis9 js.Value, parseArgs9 []js.Value) interface{} {
			return nil
		})
		hasResizeDisconnect = true
		parseObserver.Set("observe", parseResizeObserve)
		parseObserver.Set("disconnect", parseResizeDisconnect)
		return parseObserver
	})
	parseGlobal.Set("ResizeObserver", parseResizeObserverCtor)

	parseT.Cleanup(func() {
		parseGlobal.Set("document", parsePrevDocument)
		parseGlobal.Set("sessionStorage", parsePrevSession)
		parseGlobal.Set("ResizeObserver", parsePrevResizeObserver)
		parseSetItem.Release()
		parseGetItem.Release()
		parseRemoveItem.Release()
		parseAddEventListener.Release()
		parseRemoveEventListener.Release()
		parseGetElementByID.Release()
		parseResizeObserverCtor.Release()
		if hasResizeObserve {
			parseResizeObserve.Release()
		}
		if hasResizeDisconnect {
			parseResizeDisconnect.Release()
		}
	})

	return buildListWASMBrowserHarness{
		getObjectCtor:     parseObjectCtor,
		getElement:        parseElement,
		getSessionStorage: parseSessionStorage,
		getListeners:      parseListeners,
		getResizeCallback: &parseResizeCallback,
	}
}

// TestListWASMRestoresViewportAndPersistsSnapshots verifies list effects restore browser snapshots, observe viewport changes, and persist updated browser state on wasm builds.
func TestListWASMRestoresViewportAndPersistsSnapshots(parseT *testing.T) {
	parseAdapter := mockdom.NewMockDOMAdapter()
	parseContainer := parseAdapter.CreateElement("section")
	parseScheduler := &buildListWASMQueuedScheduler{}
	runtime.InitGlobalRuntime(runtime.Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler, Reset: true})
	runtime.SetCurrentFiber(nil)

	parseItems := make([]string, 10)
	for parseIndex := range parseItems {
		parseItems[parseIndex] = fmt.Sprintf("item-%d", parseIndex)
	}
	parseListID := "list-wasm"
	parseHarness := installListWASMBrowserHarness(parseT, parseListID, 0, 400, 60)
	parseHarness.setListWASMStorageItem(restorationStoragePrefix+parseListID, `{"scrollTop":5,"anchorKey":"item-4"}`)

	var parseDiagnostics []ViewportDiagnostics
	parseNode := ui.CreateElement(func() ui.Node {
		return List(ListProps[string]{
			ID:        parseListID,
			Items:     parseItems,
			Height:    60,
			RowHeight: 20,
			Overscan:  1,
			ItemKey:   func(parseItem string) string { return parseItem },
			RenderRow: func(parseProps RowRenderProps[string]) ui.Node {
				return html.Div(html.Props{Class: "row"}, html.Text(parseProps.Key))
			},
			OnViewportChange: func(parseDiagnosticsEntry ViewportDiagnostics) {
				parseDiagnostics = append(parseDiagnostics, parseDiagnosticsEntry)
			},
		})
	})

	if parseRenderErr := runtime.GetGlobalRuntime().RenderInto(parseContainer, parseNode); parseRenderErr != nil {
		parseT.Fatalf("RenderInto(list) error = %v", parseRenderErr)
	}
	parseScheduler.Flush()

	if parseHarness.getElement.Get("scrollTop").Float() != 80 {
		parseT.Fatalf("expected restored scrollTop 80, got %v", parseHarness.getElement.Get("scrollTop").Float())
	}
	if len(parseDiagnostics) == 0 {
		parseT.Fatal("expected initial viewport diagnostics after mocked observation publish")
	}
	parseLastDiagnostics := parseDiagnostics[len(parseDiagnostics)-1]
	if parseLastDiagnostics.VisibleStart != 4 || parseLastDiagnostics.RenderedStart != 3 || parseLastDiagnostics.RenderedEnd != 8 {
		parseT.Fatalf("unexpected restored diagnostics: %+v", parseLastDiagnostics)
	}
	if parseLastDiagnostics.RowMountCount == 0 {
		parseT.Fatalf("expected row mount counts after first render, got %+v", parseLastDiagnostics)
	}

	parseRootChildren := parseAdapter.GetChildren(parseContainer)
	if len(parseRootChildren) != 1 {
		parseT.Fatalf("expected one outer list node, got %d", len(parseRootChildren))
	}
	parseInnerChildren := parseAdapter.GetChildren(parseRootChildren[0])
	if len(parseInnerChildren) != 1 {
		parseT.Fatalf("expected one inner wrapper node, got %d", len(parseInnerChildren))
	}
	parseRenderedChildren := parseAdapter.GetChildren(parseInnerChildren[0])
	parseDescribeRenderedChildren := func() []string {
		parseSummary := make([]string, 0, len(parseRenderedChildren))
		for _, parseChildNode := range parseRenderedChildren {
			parseChild, parseOk := parseChildNode.(*mockdom.MockDOMNode)
			if !parseOk {
				parseSummary = append(parseSummary, fmt.Sprintf("%T", parseChildNode))
				continue
			}
			parseSummary = append(parseSummary, fmt.Sprintf("key=%q height=%q children=%d", parseChild.Attrs["key"], parseChild.Styles["height"], len(parseChild.Children)))
		}
		return parseSummary
	}
	if len(parseRenderedChildren) < 3 {
		parseT.Fatalf("expected spacer and row children after restore, got %d: %v", len(parseRenderedChildren), parseDescribeRenderedChildren())
	}
	parseTopSpacer, parseTopOK := parseRenderedChildren[0].(*mockdom.MockDOMNode)
	if !parseTopOK || parseTopSpacer.Styles["height"] != "60px" {
		parseT.Fatalf("expected top spacer height 60px, got %#v; rendered=%v", parseRenderedChildren[0], parseDescribeRenderedChildren())
	}
	parseBottomSpacer, parseBottomOK := parseRenderedChildren[len(parseRenderedChildren)-1].(*mockdom.MockDOMNode)
	if !parseBottomOK || parseBottomSpacer.Styles["height"] != "40px" {
		parseT.Fatalf("expected bottom spacer height 40px, got %#v; rendered=%v", parseRenderedChildren[len(parseRenderedChildren)-1], parseDescribeRenderedChildren())
	}

	parseHarness.getElement.Set("scrollTop", 100)
	parseHarness.triggerListWASMScroll(parseT)
	parseHarness.triggerListWASMResize(parseT, 80)
	parseScheduler.Flush()

	parsePersisted := parseHarness.getListWASMStorageItem(restorationStoragePrefix + parseListID)
	if !strings.Contains(parsePersisted, `"scrollTop":100`) || !strings.Contains(parsePersisted, `"anchorKey":"item-5"`) {
		parseT.Fatalf("expected persisted restoration snapshot to track scrollTop and anchor key, got %q", parsePersisted)
	}

	if parseRenderErr2 := runtime.GetGlobalRuntime().RenderInto(parseContainer, html.Div(html.Props{}, html.Text("done"))); parseRenderErr2 != nil {
		parseT.Fatalf("RenderInto(replacement) error = %v", parseRenderErr2)
	}
	parseScheduler.Flush()
	parseLastDiagnostics = parseDiagnostics[len(parseDiagnostics)-1]
	if parseLastDiagnostics.RowUnmountCount == 0 {
		parseT.Fatalf("expected row unmount counts after replacing the list, got %+v", parseLastDiagnostics)
	}
}

// TestLoadPersistedRestorationSnapshotWASMRejectsInvalidJSON verifies invalid browser session payloads fail closed instead of surfacing stale restoration state.
func TestLoadPersistedRestorationSnapshotWASMRejectsInvalidJSON(parseT *testing.T) {
	parseHarness := installListWASMBrowserHarness(parseT, "list-invalid-json", 0, 200, 60)
	parseHarness.setListWASMStorageItem(restorationStoragePrefix+"list-invalid-json", `{invalid json`)
	if parseSnapshot, parseOK := loadPersistedRestorationSnapshot("list-invalid-json"); parseOK || parseSnapshot != (restorationSnapshot{}) {
		parseT.Fatalf("expected invalid persisted snapshot payload to be ignored, got %+v ok=%t", parseSnapshot, parseOK)
	}
}
