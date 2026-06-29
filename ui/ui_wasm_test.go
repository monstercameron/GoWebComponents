//go:build js && wasm

package ui

import (
	"context"
	"errors"
	"reflect"
	"strconv"
	"strings"
	"syscall/js"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/internal/platform/mockdom"
	"github.com/monstercameron/GoWebComponents/internal/pluginruntime"
	"github.com/monstercameron/GoWebComponents/internal/runtime"
	"github.com/monstercameron/GoWebComponents/internal/runtime2"
	"github.com/monstercameron/GoWebComponents/interop"
)

func installMockFetchResolvedBytes(parseT *testing.T, parsePayload []byte) {
	parseT.Helper()
	parseGlobal := js.Global()
	parseObjectCtor := parseGlobal.Get("Object")
	parseUint8ArrayCtor := parseGlobal.Get("Uint8Array")
	parsePrevFetch := parseGlobal.Get("fetch")

	parseArray := parseUint8ArrayCtor.New(len(parsePayload))
	js.CopyBytesToJS(parseArray, parsePayload)
	parseBuffer := parseArray.Get("buffer")

	parseResponse := parseObjectCtor.New()
	parseResponse.Set("ok", true)
	parseResponse.Set("status", 200)
	parseArrayBufferFn := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		return parseBuffer
	})
	parseResponse.Set("arrayBuffer", parseArrayBufferFn)

	parsePromise := parseObjectCtor.New()
	parseThenFn := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
		if len(parseArgs2) == 0 {
			return parseThis2
		}
		parseCurrent := parseThis2.Get("__current")
		parseNext := parseArgs2[0].Invoke(parseCurrent)
		parseThis2.Set("__current", parseNext)
		return parseThis2
	})
	parseCatchFn := js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} {
		return parseThis3
	})
	parsePromise.Set("__current", parseResponse)
	parsePromise.Set("then", parseThenFn)
	parsePromise.Set("catch", parseCatchFn)

	parseFetchFn := js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} {
		return parsePromise
	})
	parseGlobal.Set("fetch", parseFetchFn)

	parseT.Cleanup(func() {
		parseGlobal.Set("fetch", parsePrevFetch)
		parseFetchFn.Release()
		parseThenFn.Release()
		parseCatchFn.Release()
		parseArrayBufferFn.Release()
	})
}

type noOpScheduler struct{}

func (noOpScheduler) RequestIdleCallback(parseCallback func(runtime.Deadline)) {}

func (noOpScheduler) SetTimeout(parseCallback func(), parseDelay int) {}

type queuedScheduler struct {
	idleCallbacks []func(runtime.Deadline)
	timeouts      []func()
}

type buildUIWrapHandlerMarkerAdapter struct {
	*mockdom.MockDOMAdapter
}

// WrapFunction marks wrapped DOM handlers so wasm helper tests can assert the wrapper path ran.
func (parseAdapter buildUIWrapHandlerMarkerAdapter) WrapFunction(parseFn interface{}) interface{} {
	return "wrapped-handler"
}

type queuedDeadline struct{}

type reactiveRegionTestSource struct {
	id string
}

func (parseS reactiveRegionTestSource) ReactiveRegionSourceIDs() []string {
	if parseS.id == "" {
		return nil
	}
	return []string{parseS.id}
}

func (queuedDeadline) TimeRemaining() float64 { return 1000 }

func (queuedDeadline) DidTimeout() bool { return false }

func (parseS *queuedScheduler) RequestIdleCallback(parseCallback func(runtime.Deadline)) {
	parseS.idleCallbacks = append(parseS.idleCallbacks, parseCallback)
}

func (parseS *queuedScheduler) SetTimeout(parseCallback func(), parseDelay int) {
	parseS.timeouts = append(parseS.timeouts, parseCallback)
}

func (parseS *queuedScheduler) Flush() {
	for len(parseS.idleCallbacks) > 0 {
		parsePendingIdle := append([]func(runtime.Deadline){}, parseS.idleCallbacks...)
		parseS.idleCallbacks = parseS.idleCallbacks[:0]
		for _, parseCallback := range parsePendingIdle {
			parseCallback(queuedDeadline{})
		}
	}
	for len(parseS.timeouts) > 0 {
		parsePending := append([]func(){}, parseS.timeouts...)
		parseS.timeouts = parseS.timeouts[:0]
		for _, parseCallback2 := range parsePending {
			parseCallback2()
		}
		for len(parseS.idleCallbacks) > 0 {
			parsePendingIdle2 := append([]func(runtime.Deadline){}, parseS.idleCallbacks...)
			parseS.idleCallbacks = parseS.idleCallbacks[:0]
			for _, parseCallback3 := range parsePendingIdle2 {
				parseCallback3(queuedDeadline{})
			}
		}
	}
}

type queryHydrationDOMAdapter struct {
	*mockdom.MockDOMAdapter
	selectors map[string]interface{}
}

func newQueryHydrationDOMAdapter() *queryHydrationDOMAdapter {
	return &queryHydrationDOMAdapter{
		MockDOMAdapter: mockdom.NewMockDOMAdapter(),
		selectors:      make(map[string]interface{}),
	}
}

func (parseA *queryHydrationDOMAdapter) QuerySelector(parseSelector string) interface{} {
	return parseA.selectors[parseSelector]
}

func (parseA *queryHydrationDOMAdapter) ResolveNode(parseValue interface{}) runtime.DOMNode {
	if parseNode, parseOk := parseValue.(runtime.DOMNode); parseOk {
		return parseNode
	}
	return nil
}

// resetUIRuntime installs one fresh global runtime for one wasm ui test.
func resetUIRuntime(parseConfig runtime.Config) {
	runtime.SetCurrentFiber(nil)
	parseConfig.Reset = true
	runtime.InitGlobalRuntime(parseConfig)
}

func installUIHookContext(parseT *testing.T) {
	parseT.Helper()
	resetUIRuntime(runtime.Config{Scheduler: noOpScheduler{}})
	runtime.SetCurrentFiber(&runtime.Fiber{})
	parseT.Cleanup(func() {
		runtime.SetCurrentFiber(nil)
	})
}

func installQueuedUIHookContext(parseT *testing.T) *queuedScheduler {
	parseT.Helper()
	parseScheduler := &queuedScheduler{}
	resetUIRuntime(runtime.Config{Scheduler: parseScheduler})
	runtime.SetCurrentFiber(&runtime.Fiber{})
	parseT.Cleanup(func() {
		runtime.SetCurrentFiber(nil)
	})
	return parseScheduler
}

// resetUIEventStateForTesting clears one event-service listener state snapshot for isolated wasm tests.
func resetUIEventStateForTesting() {
	storeUIEventState.getMu.Lock()
	defer storeUIEventState.getMu.Unlock()
	for _, parseListener := range storeUIEventState.getListeners {
		parseListener.Release()
	}
	storeUIEventState.hasStarted = false
	storeUIEventState.getListeners = nil
	storeUIEventState.getRecords = nil
}

// installUIEventDocumentForTesting installs one mock browser document that records added event listeners.
func installUIEventDocumentForTesting(parseT *testing.T) map[string]js.Value {
	parseT.Helper()
	parseGlobal := js.Global()
	parseObjectCtor := parseGlobal.Get("Object")
	parsePrevDocument := parseGlobal.Get("document")
	buildListeners := map[string]js.Value{}

	parseDocument := parseObjectCtor.New()
	parseBody := parseObjectCtor.New()
	parseRoot := parseObjectCtor.New()
	parseAddEventListener := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		if len(parseArgs) < 2 {
			return nil
		}
		buildListeners[strings.TrimSpace(parseArgs[0].String())] = parseArgs[1]
		return nil
	})
	parseDocument.Set("body", parseBody)
	parseDocument.Set("documentElement", parseRoot)
	parseDocument.Set("addEventListener", parseAddEventListener)
	parseGlobal.Set("document", parseDocument)

	parseT.Cleanup(func() {
		parseGlobal.Set("document", parsePrevDocument)
		parseAddEventListener.Release()
	})
	return buildListeners
}

func setUIJSGlobalValue(parseName string, parseValue interface{}) func() {
	parseGlobal := js.Global()
	parsePrev := parseGlobal.Get(parseName)
	parseGlobal.Set(parseName, parseValue)
	return func() {
		parseGlobal.Set(parseName, parsePrev)
	}
}

type uiStorageHarness struct {
	Values   map[string]string
	SetCalls int
}

func installUIPersistedStateStorage(parseT *testing.T, parseName string, parseValues map[string]string, parseSetErr string) *uiStorageHarness {
	parseT.Helper()

	parseStorage := js.Global().Get("Object").New()
	parseHarness := &uiStorageHarness{Values: map[string]string{}}
	for parseKey, parseValue := range parseValues {
		parseHarness.Values[parseKey] = parseValue
	}

	parseGetItemFn := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		if len(parseArgs) == 0 {
			return js.Null()
		}
		if parseValue, parseOk := parseHarness.Values[parseArgs[0].String()]; parseOk {
			return parseValue
		}
		return js.Null()
	})
	parseStorage.Set("getItem", parseGetItemFn)

	var parseSetItemFn js.Func
	if parseSetErr == "" {
		parseSetItemFn = js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
			if len(parseArgs) >= 2 {
				parseHarness.Values[parseArgs[0].String()] = parseArgs[1].String()
				parseHarness.SetCalls++
			}
			return nil
		})
		parseStorage.Set("setItem", parseSetItemFn)
	} else {
		parseStorage.Set("setItem", js.Global().Get("Function").New("key", "value", "throw new Error("+strconv.Quote(parseSetErr)+")"))
	}

	parseRemoveItemFn := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		if len(parseArgs) > 0 {
			delete(parseHarness.Values, parseArgs[0].String())
		}
		return nil
	})
	parseClearFn := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseHarness.Values = map[string]string{}
		return nil
	})
	parseKeyFn := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} { return js.Null() })
	parseStorage.Set("removeItem", parseRemoveItemFn)
	parseStorage.Set("clear", parseClearFn)
	parseStorage.Set("key", parseKeyFn)
	parseStorage.Set("length", len(parseHarness.Values))

	parseRestoreStorage := setUIJSGlobalValue(parseName, parseStorage)
	parseT.Cleanup(func() {
		parseRestoreStorage()
		parseGetItemFn.Release()
		if parseSetErr == "" {
			parseSetItemFn.Release()
		}
		parseRemoveItemFn.Release()
		parseClearFn.Release()
		parseKeyFn.Release()
	})

	return parseHarness
}

func TestCreateElementReturnsExistingNode(parseT *testing.T) {
	parseExisting := runtime.Div(map[string]interface{}{"id": "existing"})
	if parseGot := CreateElement(parseExisting); parseGot != parseExisting {
		parseT.Fatal("expected CreateElement to return existing node unchanged")
	}
}

// TestEnsureUIEventListenersCapturePreSnapshotEvent verifies early user interactions are captured before the first snapshot read.
func TestEnsureUIEventListenersCapturePreSnapshotEvent(parseT *testing.T) {
	resetUIEventStateForTesting()
	parseT.Cleanup(resetUIEventStateForTesting)
	parseListeners := installUIEventDocumentForTesting(parseT)

	ensureUIEventListeners()

	parseClickListener, hasClickListener := parseListeners["click"]
	if !hasClickListener || parseClickListener.IsUndefined() || parseClickListener.IsNull() {
		parseT.Fatal("expected ensureUIEventListeners to install the click event listener")
	}

	parseTarget := js.Global().Get("Object").New()
	parseTarget.Set("nodeName", "BUTTON")
	parseTarget.Set("id", "kernel-plugin-theme")
	parseEvent := js.Global().Get("Object").New()
	parseEvent.Set("target", parseTarget)
	parseClickListener.Invoke(parseEvent)

	parseSnapshot, parseErr := buildUIEventService{}.GetEventSnapshot(pluginruntime.QueryBudget{MaxItems: 8})
	if parseErr != nil {
		parseT.Fatalf("read UI event snapshot: %v", parseErr)
	}
	if len(parseSnapshot.Events) != 1 {
		parseT.Fatalf("expected one captured click event, got %+v", parseSnapshot.Events)
	}
	parseLatestEvent := parseSnapshot.Events[len(parseSnapshot.Events)-1]
	if parseLatestEvent.Type != "click" {
		parseT.Fatalf("expected click event type, got %+v", parseLatestEvent)
	}
	if parseLatestEvent.Target != "button#kernel-plugin-theme" {
		parseT.Fatalf("expected themed button target, got %+v", parseLatestEvent)
	}
}

func TestCreateElementAcceptsComponentFunctions(parseT *testing.T) {
	type props struct {
		Label string
	}

	parseWithoutProps := func() Node {
		return Text("plain")
	}
	parseWithProps := func(parseInput props) Node {
		return Text(parseInput.Label)
	}

	if parseNode := CreateElement(parseWithoutProps); parseNode == nil {
		parseT.Fatal("expected zero-argument component to produce a node")
	}
	if parseNode2 := CreateElement(parseWithProps, props{Label: "hello"}); parseNode2 == nil {
		parseT.Fatal("expected props component to produce a node")
	}
}

func TestFragmentAndTextHelpers(parseT *testing.T) {
	parseFirst := Text("first")
	parseSecond := Text("second")

	parseFragment := Fragment(parseFirst, parseSecond)
	if parseFragment == nil {
		parseT.Fatal("expected fragment")
	}
	if parseFragment.Type != "FRAGMENT" {
		parseT.Fatalf("expected fragment type, got %#v", parseFragment.Type)
	}
	if len(parseFragment.Children) != 2 {
		parseT.Fatalf("expected two fragment children, got %d", len(parseFragment.Children))
	}
	if parseFirst.TextContent != "first" || parseSecond.TextContent != "second" {
		parseT.Fatal("expected text helper to preserve text content")
	}
}

func TestReadBootstrapScript(parseT *testing.T) {
	parseGlobal := js.Global()
	parseObjectCtor := parseGlobal.Get("Object")
	parsePrevDoc := parseGlobal.Get("document")

	parseScript := parseObjectCtor.New()
	parseScript.Set("textContent", `{"route":{"path":"/docs"},"atoms":{"theme":"dark"}}`)

	parseDoc := parseObjectCtor.New()
	getElementByID := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		if len(parseArgs) > 0 && parseArgs[0].String() == DefaultBootstrapScriptID {
			return parseScript
		}
		return js.Null()
	})
	parseDoc.Set("getElementById", getElementByID)
	parseGlobal.Set("document", parseDoc)

	parseT.Cleanup(func() {
		parseGlobal.Set("document", parsePrevDoc)
		getElementByID.Release()
	})

	parsePayload, parseErr := ReadBootstrapScript("")
	if parseErr != nil {
		parseT.Fatalf("unexpected bootstrap read error: %v", parseErr)
	}
	if parsePayload.Route.Path != "/docs" {
		parseT.Fatalf("expected route path /docs, got %q", parsePayload.Route.Path)
	}
	if parsePayload.Atoms["theme"] != "dark" {
		parseT.Fatalf("expected theme atom to round-trip, got %#v", parsePayload.Atoms["theme"])
	}
}

func TestReadBootstrapReferenceScript(parseT *testing.T) {
	parseGlobal := js.Global()
	parseObjectCtor := parseGlobal.Get("Object")
	parsePrevDoc := parseGlobal.Get("document")

	parseScript := parseObjectCtor.New()
	parseScript.Set("textContent", `{"url":"/bootstrap.cbor","format":"cbor"}`)

	parseDoc := parseObjectCtor.New()
	getElementByID := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		if len(parseArgs) > 0 && parseArgs[0].String() == DefaultBootstrapReferenceScriptID {
			return parseScript
		}
		return js.Null()
	})
	parseDoc.Set("getElementById", getElementByID)
	parseGlobal.Set("document", parseDoc)

	parseT.Cleanup(func() {
		parseGlobal.Set("document", parsePrevDoc)
		getElementByID.Release()
	})

	parseRef, parseErr := ReadBootstrapReferenceScript("")
	if parseErr != nil {
		parseT.Fatalf("unexpected bootstrap reference read error: %v", parseErr)
	}
	if parseRef.URL != "/bootstrap.cbor" {
		parseT.Fatalf("expected reference url /bootstrap.cbor, got %q", parseRef.URL)
	}
	if parseRef.Format != SSRBootstrapFormatCBOR {
		parseT.Fatalf("expected bootstrap reference format %q, got %q", SSRBootstrapFormatCBOR, parseRef.Format)
	}
}

func TestReadBootstrapReferenceJSON(parseT *testing.T) {
	parsePayloadBytes := []byte(`{"route":{"path":"/json"},"atoms":{"theme":"light"}}`)
	installMockFetchResolvedBytes(parseT, parsePayloadBytes)

	parsePayload, parseErr := ReadBootstrapReference(SSRBootstrapReference{URL: "/bootstrap.json", Format: SSRBootstrapFormatJSON})
	if parseErr != nil {
		parseT.Fatalf("unexpected JSON bootstrap reference read error: %v", parseErr)
	}
	if parsePayload.Route.Path != "/json" {
		parseT.Fatalf("expected JSON bootstrap path /json, got %q", parsePayload.Route.Path)
	}
	if parsePayload.Atoms["theme"] != "light" {
		parseT.Fatalf("expected JSON bootstrap atom to round-trip, got %#v", parsePayload.Atoms["theme"])
	}
}

func TestReadBootstrapReferenceCBOR(parseT *testing.T) {
	parseEncoded, parseErr := MarshalSSRBootstrapBinary(SSRBootstrap{
		Route: SSRRouteBootstrap{Path: "/cbor"},
		Atoms: map[string]interface{}{"theme": "dark"},
	})
	if parseErr != nil {
		parseT.Fatalf("unexpected binary bootstrap marshal error: %v", parseErr)
	}
	installMockFetchResolvedBytes(parseT, parseEncoded)

	parsePayload, parseErr := ReadBootstrapReference(SSRBootstrapReference{URL: "/bootstrap.cbor", Format: SSRBootstrapFormatCBOR})
	if parseErr != nil {
		parseT.Fatalf("unexpected CBOR bootstrap reference read error: %v", parseErr)
	}
	if parsePayload.Route.Path != "/cbor" {
		parseT.Fatalf("expected CBOR bootstrap path /cbor, got %q", parsePayload.Route.Path)
	}
	if parsePayload.Atoms["theme"] != "dark" {
		parseT.Fatalf("expected CBOR bootstrap atom to round-trip, got %#v", parsePayload.Atoms["theme"])
	}
}

func TestHydrateRestoresBootstrapAtomsAndIDSeed(parseT *testing.T) {
	parseAdapter := newQueryHydrationDOMAdapter()
	parseScheduler := noOpScheduler{}
	parseContainer := parseAdapter.CreateElement("div")
	parseAdapter.selectors["#app"] = parseContainer

	parsePreviousInitialized := runtimeInitialized
	runtimeInitialized = true
	parseT.Cleanup(func() {
		runtimeInitialized = parsePreviousInitialized
	})
	resetUIRuntime(runtime.Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})
	if parseErr := runtime.GetGlobalRuntime().RestoreAtomSnapshot(map[string]interface{}{"theme": "light"}); parseErr != nil {
		parseT.Fatalf("unexpected initial atom restore error: %v", parseErr)
	}

	parsePayload := SSRBootstrap{
		Atoms:  map[string]interface{}{"theme": "dark"},
		IDSeed: 7,
	}
	if _, parseErr2 := Hydrate(Text("hello"), "#app", HydrationOptions{Bootstrap: parsePayload}); parseErr2 != nil {
		parseT.Fatalf("unexpected hydrate error: %v", parseErr2)
	}

	parseValue, parseOk := runtime.GetGlobalRuntime().GetAtomValue("theme")
	if !parseOk || parseValue != "dark" {
		parseT.Fatalf("expected bootstrap atom restore to win, got %#v ok=%t", parseValue, parseOk)
	}

	runtime.SetCurrentFiber(&runtime.Fiber{})
	defer runtime.SetCurrentFiber(nil)
	if parseGot := UseId(); parseGot != "gwc-8-0" {
		parseT.Fatalf("expected hydration id seed to advance next UseId generation, got %q", parseGot)
	}
}

func TestRenderIntoRendersToExplicitNode(parseT *testing.T) {
	parseAdapter := newQueryHydrationDOMAdapter()
	parseContainer := parseAdapter.CreateElement("section")

	parsePreviousInitialized := runtimeInitialized
	runtimeInitialized = true
	parseT.Cleanup(func() {
		runtimeInitialized = parsePreviousInitialized
	})
	resetUIRuntime(runtime.Config{DOMAdapter: parseAdapter, Scheduler: noOpScheduler{}})

	if parseErr := RenderInto(Text("hello"), parseContainer); parseErr != nil {
		parseT.Fatalf("expected RenderInto to succeed, got %v", parseErr)
	}
}

func TestParallelRegionRenderIntoMountsAndOwnerRemovalDisposesRuntime2Adapter(parseT *testing.T) {
	resetParallelRegionRegistry()
	parseT.Cleanup(resetParallelRegionRegistry)

	parseAdapter := newQueryHydrationDOMAdapter()
	parseContainer := parseAdapter.CreateElement("section")
	parseScheduler := &queuedScheduler{}

	parsePreviousInitialized := runtimeInitialized
	runtimeInitialized = true
	parseT.Cleanup(func() {
		runtimeInitialized = parsePreviousInitialized
	})
	resetUIRuntime(runtime.Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	if !canParallelRegionUseRuntime2Lifecycle() {
		parseT.Fatal("expected wasm parallel-region lifecycle support to be enabled")
	}
	if parseErr := RegisterParallelRegion("dashboard.hot-panel", func(parseProps registerParallelRegionProps) Node {
		return Text(parseProps.Label)
	}); parseErr != nil {
		parseT.Fatalf("RegisterParallelRegion returned error: %v", parseErr)
	}

	if parseErr := RenderInto(ParallelRegion(ParallelRegionSpec[registerParallelRegionProps]{
		RendererID:       "dashboard.hot-panel",
		RegionInstanceID: "dashboard.hot-panel:wasm",
		Props: registerParallelRegionProps{
			Label: "Hot",
		},
	}), parseContainer); parseErr != nil {
		parseT.Fatalf("RenderInto(ParallelRegion) returned error: %v", parseErr)
	}
	parseScheduler.Flush()
	getParallelRegionHostAdapter, hasParallelRegionHostAdapter := resolveParallelRegionHostAdapter("dashboard.hot-panel:wasm")
	if !hasParallelRegionHostAdapter {
		parseT.Fatal("expected wasm ParallelRegion to mount a runtime2 host adapter")
	}
	if !getParallelRegionHostAdapter.IsHostRegionLocalShellOwnership() {
		parseT.Fatal("expected mounted host adapter to keep local shell ownership")
	}

	if parseErr := RenderInto(Text("gone"), parseContainer); parseErr != nil {
		parseT.Fatalf("RenderInto(Text) returned error: %v", parseErr)
	}
	parseScheduler.Flush()
	if _, hasParallelRegionHostAdapterAfter := resolveParallelRegionHostAdapter("dashboard.hot-panel:wasm"); hasParallelRegionHostAdapterAfter {
		parseT.Fatal("expected owner removal cleanup to dispose the runtime2 host adapter")
	}
}

func TestRenderIntoParallelRegionPostRenderAttach(parseT *testing.T) {
	resetParallelRegionRegistry()
	parseT.Cleanup(resetParallelRegionRegistry)

	parseAdapter := newQueryHydrationDOMAdapter()
	parseContainer := parseAdapter.CreateElement("section")
	parseScheduler := &queuedScheduler{}

	parsePreviousInitialized := runtimeInitialized
	runtimeInitialized = true
	parseT.Cleanup(func() {
		runtimeInitialized = parsePreviousInitialized
	})
	resetUIRuntime(runtime.Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	if parseErr := RegisterParallelRegion("dashboard.hot-panel", func(parseProps registerParallelRegionProps) Node {
		return Text(parseProps.Label)
	}); parseErr != nil {
		parseT.Fatalf("RegisterParallelRegion returned error: %v", parseErr)
	}

	if parseErr := RenderInto(ParallelRegion(ParallelRegionSpec[registerParallelRegionProps]{
		RendererID:       "dashboard.hot-panel",
		RegionInstanceID: "dashboard.hot-panel:post-render-attach",
		Props: registerParallelRegionProps{
			Label: "Post Render Attach",
		},
	}), parseContainer); parseErr != nil {
		parseT.Fatalf("RenderInto returned error: %v", parseErr)
	}
	parseScheduler.Flush()

	parseAdapterState, hasAdapter := resolveParallelRegionHostAdapter("dashboard.hot-panel:post-render-attach")
	if !hasAdapter {
		parseT.Fatal("expected post-render ParallelRegion to mount a runtime2 host adapter")
	}
	getRuntimeStatus, hasRuntimeStatus := parseAdapterState.GetHostRegionRuntimeStatus()
	if !hasRuntimeStatus {
		parseT.Fatal("expected post-render ParallelRegion runtime status")
	}
	if getRuntimeStatus.GetRegionMode != runtime2.HostRegionRuntimeModeWorkerAttached {
		parseT.Fatalf("expected post-render attach to report worker-attached mode, got %q", getRuntimeStatus.GetRegionMode)
	}
	if getRuntimeStatus.GetIsHydrationComplete {
		parseT.Fatal("expected post-render attach to avoid hydration-complete state")
	}
	if getRuntimeStatus.HasPostHydrationAttached {
		parseT.Fatal("expected post-render attach to avoid post-hydration attach state")
	}
	getShellAnchor, parseAnchorLookupErr := parseAdapterState.GetHostRegionDOMIndex().GetRegionDOMNode("dashboard.hot-panel:post-render-attach", 1)
	if parseAnchorLookupErr != nil {
		parseT.Fatalf("GetRegionDOMNode(post-render shell anchor) returned error: %v", parseAnchorLookupErr)
	}
	if getShellAnchor.GetTag != "div" {
		parseT.Fatalf("expected post-render shell anchor tag div, got %q", getShellAnchor.GetTag)
	}
}

func TestRenderIntoParallelRegionRuntimeStatusWorkerAttached(parseT *testing.T) {
	resetParallelRegionRegistry()
	parseT.Cleanup(resetParallelRegionRegistry)

	parseAdapter := newQueryHydrationDOMAdapter()
	parseContainer := parseAdapter.CreateElement("section")
	parseAdapter.selectors["#app"] = parseContainer
	parseScheduler := &queuedScheduler{}

	parsePreviousInitialized := runtimeInitialized
	runtimeInitialized = true
	parseT.Cleanup(func() {
		runtimeInitialized = parsePreviousInitialized
	})
	resetUIRuntime(runtime.Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	if parseErr := RegisterParallelRegion("dashboard.hot-panel", func(parseProps registerParallelRegionProps) Node {
		return Text(parseProps.Label)
	}); parseErr != nil {
		parseT.Fatalf("RegisterParallelRegion returned error: %v", parseErr)
	}

	Render(ParallelRegion(ParallelRegionSpec[registerParallelRegionProps]{
		RendererID:       "dashboard.hot-panel",
		RegionInstanceID: "dashboard.hot-panel:post-render-status",
		Props: registerParallelRegionProps{
			Label: "Post Render Status",
		},
	}), "#app")
	parseScheduler.Flush()

	getStatus, hasStatus, parseStatusErr := GetParallelRegionRuntimeStatus("dashboard.hot-panel:post-render-status")
	if parseStatusErr != nil {
		parseT.Fatalf("GetParallelRegionRuntimeStatus returned error: %v", parseStatusErr)
	}
	if !hasStatus {
		parseT.Fatal("expected public region status after post-render attach")
	}
	if getStatus.GetRegionMode != string(runtime2.HostRegionRuntimeModeWorkerAttached) {
		parseT.Fatalf("expected worker-attached public region mode, got %q", getStatus.GetRegionMode)
	}
	if getStatus.GetIsHydrationComplete {
		parseT.Fatal("expected post-render public region status to keep hydration-complete false")
	}
	if getStatus.HasPostHydrationAttached {
		parseT.Fatal("expected post-render public region status to keep post-hydration attach false")
	}
}

func TestParallelRegionRenderIntoAdvancesInputVersionAcrossRerenders(parseT *testing.T) {
	resetParallelRegionRegistry()
	parseT.Cleanup(resetParallelRegionRegistry)

	parseAdapter := newQueryHydrationDOMAdapter()
	parseContainer := parseAdapter.CreateElement("section")
	parseScheduler := &queuedScheduler{}

	parsePreviousInitialized := runtimeInitialized
	runtimeInitialized = true
	parseT.Cleanup(func() {
		runtimeInitialized = parsePreviousInitialized
	})
	resetUIRuntime(runtime.Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	if !canParallelRegionUseRuntime2Lifecycle() {
		parseT.Fatal("expected wasm parallel-region lifecycle support to be enabled")
	}
	if parseErr := RegisterParallelRegion("dashboard.hot-panel", func(parseProps registerParallelRegionProps) Node {
		return Text(parseProps.Label)
	}); parseErr != nil {
		parseT.Fatalf("RegisterParallelRegion returned error: %v", parseErr)
	}

	if parseErr := RenderInto(ParallelRegion(ParallelRegionSpec[registerParallelRegionProps]{
		RendererID:       "dashboard.hot-panel",
		RegionInstanceID: "dashboard.hot-panel:versioned",
		Props: registerParallelRegionProps{
			Label: "One",
		},
	}), parseContainer); parseErr != nil {
		parseT.Fatalf("RenderInto(first ParallelRegion) returned error: %v", parseErr)
	}
	parseScheduler.Flush()
	if getInputVersion := resolveParallelRegionInputVersion("dashboard.hot-panel:versioned"); getInputVersion != 1 {
		parseT.Fatalf("expected first public input version 1, got %d", getInputVersion)
	}

	if parseErr := RenderInto(ParallelRegion(ParallelRegionSpec[registerParallelRegionProps]{
		RendererID:       "dashboard.hot-panel",
		RegionInstanceID: "dashboard.hot-panel:versioned",
		Props: registerParallelRegionProps{
			Label: "Two",
		},
	}), parseContainer); parseErr != nil {
		parseT.Fatalf("RenderInto(second ParallelRegion) returned error: %v", parseErr)
	}
	parseScheduler.Flush()
	if getInputVersion := resolveParallelRegionInputVersion("dashboard.hot-panel:versioned"); getInputVersion != 2 {
		parseT.Fatalf("expected second public input version 2, got %d", getInputVersion)
	}

	if parseErr := RenderInto(Text("gone"), parseContainer); parseErr != nil {
		parseT.Fatalf("RenderInto(Text) returned error: %v", parseErr)
	}
	parseScheduler.Flush()
	if getInputVersion := resolveParallelRegionInputVersion("dashboard.hot-panel:versioned"); getInputVersion != 0 {
		parseT.Fatalf("expected public input version to clear after owner removal, got %d", getInputVersion)
	}
}

func TestParallelRegionRenderIntoPropChangesDispatchRuntime2Update(parseT *testing.T) {
	resetParallelRegionRegistry()
	parseT.Cleanup(resetParallelRegionRegistry)

	parseAdapter := newQueryHydrationDOMAdapter()
	parseContainer := parseAdapter.CreateElement("section")
	parseScheduler := &queuedScheduler{}

	parsePreviousInitialized := runtimeInitialized
	runtimeInitialized = true
	parseT.Cleanup(func() {
		runtimeInitialized = parsePreviousInitialized
	})
	resetUIRuntime(runtime.Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	if parseErr := RegisterParallelRegion("dashboard.hot-panel", func(parseProps registerParallelRegionProps) Node {
		return Text(parseProps.Label)
	}); parseErr != nil {
		parseT.Fatalf("RegisterParallelRegion returned error: %v", parseErr)
	}

	if parseErr := RenderInto(ParallelRegion(ParallelRegionSpec[registerParallelRegionProps]{
		RendererID:       "dashboard.hot-panel",
		RegionInstanceID: "dashboard.hot-panel:dispatch-props",
		Props: registerParallelRegionProps{
			Label: "One",
		},
	}), parseContainer); parseErr != nil {
		parseT.Fatalf("RenderInto(first ParallelRegion) returned error: %v", parseErr)
	}
	parseScheduler.Flush()
	getParallelRegionHostAdapter, hasParallelRegionHostAdapter := resolveParallelRegionHostAdapter("dashboard.hot-panel:dispatch-props")
	if !hasParallelRegionHostAdapter {
		parseT.Fatal("expected first public render to mount a runtime2 host adapter")
	}
	getRuntimeStatus, hasRuntimeStatus := getParallelRegionHostAdapter.GetHostRegionRuntimeStatus()
	if !hasRuntimeStatus {
		parseT.Fatal("expected mounted public region to report runtime status")
	}
	if getRuntimeStatus.GetLastDispatchedVersion != 0 {
		parseT.Fatalf("expected initial public render to stay mount-only, got dispatched version %d", getRuntimeStatus.GetLastDispatchedVersion)
	}

	if parseErr := RenderInto(ParallelRegion(ParallelRegionSpec[registerParallelRegionProps]{
		RendererID:       "dashboard.hot-panel",
		RegionInstanceID: "dashboard.hot-panel:dispatch-props",
		Props: registerParallelRegionProps{
			Label: "Two",
		},
	}), parseContainer); parseErr != nil {
		parseT.Fatalf("RenderInto(second ParallelRegion) returned error: %v", parseErr)
	}
	parseScheduler.Flush()
	getRuntimeStatus, hasRuntimeStatus = getParallelRegionHostAdapter.GetHostRegionRuntimeStatus()
	if !hasRuntimeStatus {
		parseT.Fatal("expected public region runtime status after prop rerender")
	}
	if getRuntimeStatus.GetLastSnapshotVersion != 2 {
		parseT.Fatalf("expected prop rerender snapshot version 2, got %d", getRuntimeStatus.GetLastSnapshotVersion)
	}
	if getRuntimeStatus.GetLastDispatchedVersion != 2 {
		parseT.Fatalf("expected prop rerender to dispatch input version 2, got %d", getRuntimeStatus.GetLastDispatchedVersion)
	}
}

type renderParallelRegionRefreshProps struct {
	GetItems        []string
	GetRefreshToken int
}

type renderParallelRegionPreparedItem struct {
	GetText   string
	GetDigest uint64
}

type renderParallelRegionPreparedProps struct {
	GetItems        []renderParallelRegionPreparedItem
	GetWorker       string
	GetWorkDigest   uint64
	GetRefreshToken int
}

// TestParallelRegionRenderIntoRefreshOnlyUpdateKeepsShellDOMNode verifies refresh-only rerenders keep the same shell DOM node and avoid child-list churn.
func TestParallelRegionRenderIntoRefreshOnlyUpdateKeepsShellDOMNode(parseT *testing.T) {
	resetParallelRegionRegistry()
	parseT.Cleanup(resetParallelRegionRegistry)

	parseAdapter := newQueryHydrationDOMAdapter()
	parseContainer := parseAdapter.CreateElement("section")
	parseScheduler := &queuedScheduler{}

	parsePreviousInitialized := runtimeInitialized
	runtimeInitialized = true
	parseT.Cleanup(func() {
		runtimeInitialized = parsePreviousInitialized
	})
	resetUIRuntime(runtime.Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	if parseErr := RegisterParallelRegion("dashboard.hot-panel", func(parseProps renderParallelRegionRefreshProps) Node {
		getItemNodes := make([]Node, 0, len(parseProps.GetItems))
		for _, getItem := range parseProps.GetItems {
			getItemNodes = append(getItemNodes, runtime.CreateElement("div", map[string]interface{}{
				"class": "benchmark-core-item",
			}, Text(getItem)))
		}
		return runtime.CreateElement("div", map[string]interface{}{
			"class":              "benchmark-core-region",
			"data-refresh-token": strconv.Itoa(parseProps.GetRefreshToken),
		},
			toInterfaces(getItemNodes)...,
		)
	}); parseErr != nil {
		parseT.Fatalf("RegisterParallelRegion returned error: %v", parseErr)
	}

	if parseErr := RenderInto(ParallelRegion(ParallelRegionSpec[renderParallelRegionRefreshProps]{
		RendererID:       "dashboard.hot-panel",
		RegionInstanceID: "dashboard.hot-panel:refresh-stability",
		Props: renderParallelRegionRefreshProps{
			GetItems:        []string{"One", "Two", "Three"},
			GetRefreshToken: 1,
		},
	}), parseContainer); parseErr != nil {
		parseT.Fatalf("RenderInto(first ParallelRegion) returned error: %v", parseErr)
	}
	parseScheduler.Flush()

	getFirstChildren := parseAdapter.GetChildren(parseContainer)
	if len(getFirstChildren) != 1 {
		parseT.Fatalf("expected one shell child after first render, got %d", len(getFirstChildren))
	}
	getFirstShell := getFirstChildren[0]

	parseAdapter.ClearOperations()

	if parseErr := RenderInto(ParallelRegion(ParallelRegionSpec[renderParallelRegionRefreshProps]{
		RendererID:       "dashboard.hot-panel",
		RegionInstanceID: "dashboard.hot-panel:refresh-stability",
		Props: renderParallelRegionRefreshProps{
			GetItems:        []string{"One", "Two", "Three"},
			GetRefreshToken: 2,
		},
	}), parseContainer); parseErr != nil {
		parseT.Fatalf("RenderInto(second ParallelRegion) returned error: %v", parseErr)
	}
	parseScheduler.Flush()

	getSecondChildren := parseAdapter.GetChildren(parseContainer)
	if len(getSecondChildren) != 1 {
		parseT.Fatalf("expected one shell child after refresh rerender, got %d", len(getSecondChildren))
	}
	if !getSecondChildren[0].Equals(getFirstShell) {
		parseT.Fatal("expected refresh-only rerender to reuse the shell DOM node")
	}

	getOperations := parseAdapter.GetOperations()
	for _, getOperation := range getOperations {
		switch getOperation.Type {
		case "appendChild", "removeChild", "insertBefore", "replaceChild", "createElement", "createTextNode":
			parseT.Fatalf("expected refresh-only rerender to avoid child-list DOM churn, got operation %q", getOperation.Type)
		}
	}
}

// TestParallelRegionRenderIntoPreparedItemUpdateKeepsItemDOMNodes verifies changed prepared item text and digest update in place without replacing the rendered item element nodes.
func TestParallelRegionRenderIntoPreparedItemUpdateKeepsItemDOMNodes(parseT *testing.T) {
	resetParallelRegionRegistry()
	parseT.Cleanup(resetParallelRegionRegistry)

	parseAdapter := newQueryHydrationDOMAdapter()
	parseContainer := parseAdapter.CreateElement("section")
	parseScheduler := &queuedScheduler{}

	parsePreviousInitialized := runtimeInitialized
	runtimeInitialized = true
	parseT.Cleanup(func() {
		runtimeInitialized = parsePreviousInitialized
	})
	resetUIRuntime(runtime.Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	if parseErr := RegisterParallelRegion("dashboard.hot-panel", func(parseProps renderParallelRegionPreparedProps) Node {
		getItemNodes := make([]Node, 0, len(parseProps.GetItems))
		for _, getItem := range parseProps.GetItems {
			getItemNodes = append(getItemNodes, runtime.CreateElement("div", map[string]interface{}{
				"class":            "benchmark-core-item",
				"data-prep-digest": strconv.FormatUint(getItem.GetDigest, 10),
			}, Text(getItem.GetText)))
		}
		return runtime.CreateElement("div", map[string]interface{}{
			"class":              "benchmark-core-region",
			"data-refresh-token": strconv.Itoa(parseProps.GetRefreshToken),
			"data-prep-worker":   parseProps.GetWorker,
			"data-prep-digest":   strconv.FormatUint(parseProps.GetWorkDigest, 10),
		}, toInterfaces(getItemNodes)...)
	}); parseErr != nil {
		parseT.Fatalf("RegisterParallelRegion returned error: %v", parseErr)
	}

	if parseErr := RenderInto(ParallelRegion(ParallelRegionSpec[renderParallelRegionPreparedProps]{
		RendererID:       "dashboard.hot-panel",
		RegionInstanceID: "dashboard.hot-panel:prepared-update",
		Props: renderParallelRegionPreparedProps{
			GetItems: []renderParallelRegionPreparedItem{
				{GetText: "One", GetDigest: 11},
				{GetText: "Two", GetDigest: 22},
				{GetText: "Three", GetDigest: 33},
			},
			GetWorker:       "worker-a",
			GetWorkDigest:   99,
			GetRefreshToken: 1,
		},
	}), parseContainer); parseErr != nil {
		parseT.Fatalf("RenderInto(first prepared ParallelRegion) returned error: %v", parseErr)
	}
	parseScheduler.Flush()

	getRootChildren := parseAdapter.GetChildren(parseContainer)
	if len(getRootChildren) != 1 {
		parseT.Fatalf("expected one shell child after first prepared render, got %d", len(getRootChildren))
	}
	getFirstShell := getRootChildren[0]
	getFirstRegionChildren := parseAdapter.GetChildren(getFirstShell)
	if len(getFirstRegionChildren) != 1 {
		parseT.Fatalf("expected one region-root child inside shell, got %d", len(getFirstRegionChildren))
	}
	getFirstItemNodes := parseAdapter.GetChildren(getFirstRegionChildren[0])
	if len(getFirstItemNodes) != 3 {
		parseT.Fatalf("expected three item nodes after first prepared render, got %d", len(getFirstItemNodes))
	}

	parseAdapter.ClearOperations()

	if parseErr := RenderInto(ParallelRegion(ParallelRegionSpec[renderParallelRegionPreparedProps]{
		RendererID:       "dashboard.hot-panel",
		RegionInstanceID: "dashboard.hot-panel:prepared-update",
		Props: renderParallelRegionPreparedProps{
			GetItems: []renderParallelRegionPreparedItem{
				{GetText: "One (Updated)", GetDigest: 111},
				{GetText: "Two (Updated)", GetDigest: 222},
				{GetText: "Three (Updated)", GetDigest: 333},
			},
			GetWorker:       "worker-a",
			GetWorkDigest:   999,
			GetRefreshToken: 1,
		},
	}), parseContainer); parseErr != nil {
		parseT.Fatalf("RenderInto(second prepared ParallelRegion) returned error: %v", parseErr)
	}
	parseScheduler.Flush()

	getSecondRootChildren := parseAdapter.GetChildren(parseContainer)
	if len(getSecondRootChildren) != 1 {
		parseT.Fatalf("expected one shell child after prepared update, got %d", len(getSecondRootChildren))
	}
	if !getSecondRootChildren[0].Equals(getFirstShell) {
		parseT.Fatal("expected prepared update to reuse the shell DOM node")
	}
	getSecondRegionChildren := parseAdapter.GetChildren(getSecondRootChildren[0])
	if len(getSecondRegionChildren) != 1 {
		parseT.Fatalf("expected one region-root child after prepared update, got %d", len(getSecondRegionChildren))
	}
	getSecondItemNodes := parseAdapter.GetChildren(getSecondRegionChildren[0])
	if len(getSecondItemNodes) != len(getFirstItemNodes) {
		parseT.Fatalf("expected prepared update to keep %d item nodes, got %d", len(getFirstItemNodes), len(getSecondItemNodes))
	}
	for parseIndex := 0; parseIndex < len(getFirstItemNodes); parseIndex++ {
		if !getSecondItemNodes[parseIndex].Equals(getFirstItemNodes[parseIndex]) {
			parseT.Fatalf("expected prepared item %d to update in place", parseIndex)
		}
	}

	getOperations := parseAdapter.GetOperations()
	for _, getOperation := range getOperations {
		switch getOperation.Type {
		case "appendChild", "removeChild", "insertBefore", "replaceChild", "createElement", "createTextNode":
			parseT.Fatalf("expected prepared update to avoid subtree replacement, got operation %q", getOperation.Type)
		}
	}
}

// TestParallelRegionRenderIntoRefreshOnlyUpdateKeepsSiblingShellNodes verifies sibling parallel-region shells also rerender in place during refresh-only updates.
func TestParallelRegionRenderIntoRefreshOnlyUpdateKeepsSiblingShellNodes(parseT *testing.T) {
	resetParallelRegionRegistry()
	parseT.Cleanup(resetParallelRegionRegistry)

	parseAdapter := newQueryHydrationDOMAdapter()
	parseContainer := parseAdapter.CreateElement("section")
	parseScheduler := &queuedScheduler{}

	parsePreviousInitialized := runtimeInitialized
	runtimeInitialized = true
	parseT.Cleanup(func() {
		runtimeInitialized = parsePreviousInitialized
	})
	resetUIRuntime(runtime.Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	if parseErr := RegisterParallelRegion("dashboard.hot-panel", func(parseProps renderParallelRegionRefreshProps) Node {
		getItemNodes := make([]Node, 0, len(parseProps.GetItems))
		for _, getItem := range parseProps.GetItems {
			getItemNodes = append(getItemNodes, runtime.CreateElement("div", map[string]interface{}{
				"class": "benchmark-core-item",
			}, Text(getItem)))
		}
		return runtime.CreateElement("div", map[string]interface{}{
			"class":              "benchmark-core-region",
			"data-refresh-token": strconv.Itoa(parseProps.GetRefreshToken),
		}, toInterfaces(getItemNodes)...)
	}); parseErr != nil {
		parseT.Fatalf("RegisterParallelRegion returned error: %v", parseErr)
	}

	buildParallelRegionGroup := func(parseRefreshToken int) Node {
		getRegionNodes := make([]Node, 0, 4)
		for parseRegionIndex := 0; parseRegionIndex < 4; parseRegionIndex++ {
			getRegionNodes = append(getRegionNodes, ParallelRegion(ParallelRegionSpec[renderParallelRegionRefreshProps]{
				RendererID:       "dashboard.hot-panel",
				RegionInstanceID: "dashboard.hot-panel:refresh-group:" + strconv.Itoa(parseRegionIndex),
				Props: renderParallelRegionRefreshProps{
					GetItems: []string{
						"One-" + strconv.Itoa(parseRegionIndex),
						"Two-" + strconv.Itoa(parseRegionIndex),
						"Three-" + strconv.Itoa(parseRegionIndex),
					},
					GetRefreshToken: parseRefreshToken,
				},
			}))
		}
		return runtime.CreateElement("div", map[string]interface{}{
			"id": "parallel-region-group",
		}, toInterfaces(getRegionNodes)...)
	}

	if parseErr := RenderInto(buildParallelRegionGroup(1), parseContainer); parseErr != nil {
		parseT.Fatalf("RenderInto(first sibling ParallelRegion group) returned error: %v", parseErr)
	}
	parseScheduler.Flush()

	getFirstRootChildren := parseAdapter.GetChildren(parseContainer)
	if len(getFirstRootChildren) != 1 {
		parseT.Fatalf("expected one wrapper child after first render, got %d", len(getFirstRootChildren))
	}
	getFirstShellNodes := parseAdapter.GetChildren(getFirstRootChildren[0])
	if len(getFirstShellNodes) != 4 {
		parseT.Fatalf("expected four shell children after first render, got %d", len(getFirstShellNodes))
	}

	parseAdapter.ClearOperations()

	if parseErr := RenderInto(buildParallelRegionGroup(2), parseContainer); parseErr != nil {
		parseT.Fatalf("RenderInto(second sibling ParallelRegion group) returned error: %v", parseErr)
	}
	parseScheduler.Flush()

	getSecondRootChildren := parseAdapter.GetChildren(parseContainer)
	if len(getSecondRootChildren) != 1 {
		parseT.Fatalf("expected one wrapper child after refresh rerender, got %d", len(getSecondRootChildren))
	}
	getSecondShellNodes := parseAdapter.GetChildren(getSecondRootChildren[0])
	if len(getSecondShellNodes) != 4 {
		parseT.Fatalf("expected four shell children after refresh rerender, got %d", len(getSecondShellNodes))
	}
	for parseRegionIndex := 0; parseRegionIndex < 4; parseRegionIndex++ {
		if !getSecondShellNodes[parseRegionIndex].Equals(getFirstShellNodes[parseRegionIndex]) {
			parseT.Fatalf("expected sibling shell %d to be reused across refresh-only rerender", parseRegionIndex)
		}
	}

	getOperations := parseAdapter.GetOperations()
	for _, getOperation := range getOperations {
		switch getOperation.Type {
		case "appendChild", "removeChild", "insertBefore", "replaceChild", "createElement", "createTextNode":
			parseT.Fatalf("expected sibling refresh-only rerender to avoid child-list DOM churn, got operation %q", getOperation.Type)
		}
	}
}

func TestGetParallelRegionRuntimeStatusReportsPublicDispatchVersions(parseT *testing.T) {
	resetParallelRegionRegistry()
	parseT.Cleanup(resetParallelRegionRegistry)

	parseAdapter := newQueryHydrationDOMAdapter()
	parseContainer := parseAdapter.CreateElement("section")
	parseScheduler := &queuedScheduler{}

	parsePreviousInitialized := runtimeInitialized
	runtimeInitialized = true
	parseT.Cleanup(func() {
		runtimeInitialized = parsePreviousInitialized
	})
	resetUIRuntime(runtime.Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	if parseErr := RegisterParallelRegion("dashboard.hot-panel", func(parseProps registerParallelRegionProps) Node {
		return Text(parseProps.Label)
	}); parseErr != nil {
		parseT.Fatalf("RegisterParallelRegion returned error: %v", parseErr)
	}

	if parseErr := RenderInto(ParallelRegion(ParallelRegionSpec[registerParallelRegionProps]{
		RendererID:       "dashboard.hot-panel",
		RegionInstanceID: "dashboard.hot-panel:status",
		Props: registerParallelRegionProps{
			Label: "One",
		},
	}), parseContainer); parseErr != nil {
		parseT.Fatalf("RenderInto(first ParallelRegion) returned error: %v", parseErr)
	}
	parseScheduler.Flush()
	if parseErr := RenderInto(ParallelRegion(ParallelRegionSpec[registerParallelRegionProps]{
		RendererID:       "dashboard.hot-panel",
		RegionInstanceID: "dashboard.hot-panel:status",
		Props: registerParallelRegionProps{
			Label: "Two",
		},
	}), parseContainer); parseErr != nil {
		parseT.Fatalf("RenderInto(second ParallelRegion) returned error: %v", parseErr)
	}
	parseScheduler.Flush()

	getStatus, hasStatus, parseStatusErr := GetParallelRegionRuntimeStatus("dashboard.hot-panel:status")
	if parseStatusErr != nil {
		parseT.Fatalf("GetParallelRegionRuntimeStatus returned error: %v", parseStatusErr)
	}
	if !hasStatus {
		parseT.Fatal("expected public region status after tracked rerenders")
	}
	if getStatus.GetRegionMode != string(runtime2.HostRegionRuntimeModeWorkerAttached) {
		parseT.Fatalf("expected worker-attached public region mode, got %q", getStatus.GetRegionMode)
	}
	if getStatus.GetAssignedWorkerShard != "ui-parallel-region" {
		parseT.Fatalf("expected assigned worker shard ui-parallel-region, got %q", getStatus.GetAssignedWorkerShard)
	}
	if getStatus.GetRendererID != "dashboard.hot-panel" {
		parseT.Fatalf("expected renderer dashboard.hot-panel, got %q", getStatus.GetRendererID)
	}
	if getStatus.GetEpoch != 1 {
		parseT.Fatalf("expected epoch 1, got %d", getStatus.GetEpoch)
	}
	if getStatus.GetLastSnapshotVersion != 2 {
		parseT.Fatalf("expected snapshot version 2, got %d", getStatus.GetLastSnapshotVersion)
	}
	if getStatus.GetLastDispatchedVersion != 2 {
		parseT.Fatalf("expected dispatched version 2, got %d", getStatus.GetLastDispatchedVersion)
	}
	if getStatus.GetLastCommittedVersion != 2 {
		parseT.Fatalf("expected committed version 2 after bridged worker patch commit, got %d", getStatus.GetLastCommittedVersion)
	}
	if getStatus.GetFallbackReason != "" {
		parseT.Fatalf("expected empty fallback reason, got %q", getStatus.GetFallbackReason)
	}
}

func TestParallelRegionClickEventBridgeCommitsWorkerPatch(parseT *testing.T) {
	resetParallelRegionRegistry()
	parseT.Cleanup(resetParallelRegionRegistry)

	parseAdapter := newQueryHydrationDOMAdapter()
	parseContainer := parseAdapter.CreateElement("section")
	parseScheduler := &queuedScheduler{}

	parsePreviousInitialized := runtimeInitialized
	runtimeInitialized = true
	parseT.Cleanup(func() {
		runtimeInitialized = parsePreviousInitialized
	})
	resetUIRuntime(runtime.Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	isClicked := false
	if parseErr := RegisterParallelRegion("dashboard.hot-panel", func(parseProps registerParallelRegionProps) Node {
		getLabel := parseProps.Label
		if isClicked {
			getLabel = "Clicked"
		}
		return Node(runtime.CreateElement("div", map[string]interface{}{
			parallelRegionClickSlotProp: "primary.action",
			"onclick": func() {
				isClicked = true
			},
		}, Text(getLabel)))
	}); parseErr != nil {
		parseT.Fatalf("RegisterParallelRegion returned error: %v", parseErr)
	}

	if parseErr := RenderInto(ParallelRegion(ParallelRegionSpec[registerParallelRegionProps]{
		RendererID:       "dashboard.hot-panel",
		RegionInstanceID: "dashboard.hot-panel:click-event",
		Props: registerParallelRegionProps{
			Label: "Idle",
		},
	}), parseContainer); parseErr != nil {
		parseT.Fatalf("RenderInto(first ParallelRegion) returned error: %v", parseErr)
	}
	parseScheduler.Flush()
	if _, hasWorkerRegionState := cacheParallelRegionWorkerRuntime.GetWorkerRegionState("dashboard.hot-panel:click-event"); !hasWorkerRegionState {
		parseT.Fatal("expected worker region state after initial public render")
	}
	getCachedRenderedNode, hasCachedRenderedNode := resolveParallelRegionRenderedNode("dashboard.hot-panel:click-event")
	if !hasCachedRenderedNode {
		parseT.Fatal("expected cached rendered node after initial public render")
	}
	if _, hasWrappedCachedHandler := getCachedRenderedNode.Props["onclick"].(func(runtime.GoEvent)); !hasWrappedCachedHandler {
		parseT.Fatalf("expected cached rendered node to keep bridged onclick handler, got %T", getCachedRenderedNode.Props["onclick"])
	}

	getContainerNode, hasContainerNode := parseContainer.(*mockdom.MockDOMNode)
	if !hasContainerNode {
		parseT.Fatalf("expected mock DOM container, got %T", parseContainer)
	}
	if len(getContainerNode.Children) == 0 || len(getContainerNode.Children[0].Children) == 0 {
		parseT.Fatalf("expected rendered button subtree, got %+v", getContainerNode.Children)
	}
	getButtonNode := getContainerNode.Children[0].Children[0]
	getClickHandlerRaw := getButtonNode.Props["onclick"]
	getClickHandler, hasClickHandler := getClickHandlerRaw.(func(runtime.GoEvent))
	if !hasClickHandler {
		parseT.Fatalf(
			"expected wrapped onclick handler func(runtime.GoEvent), got %T on tag %q with props %+v",
			getClickHandlerRaw,
			getButtonNode.Tag,
			getButtonNode.Props,
		)
	}
	getClickHandler(runtime.GoEvent{})

	if !isClicked {
		parseT.Fatal("expected bridged click handler to preserve local click behavior")
	}
	getWorkerRegionState, hasWorkerRegionState := cacheParallelRegionWorkerRuntime.GetWorkerRegionState("dashboard.hot-panel:click-event")
	if !hasWorkerRegionState {
		parseT.Fatal("expected worker region state after bridged click event")
	}
	if !strings.Contains(strings.Join(getWorkerRegionState.RenderIR.GetStringTable.Entries, "|"), "Clicked") {
		parseT.Fatalf("expected worker render IR to reflect bridged click render, got %+v", getWorkerRegionState.RenderIR.GetStringTable.Entries)
	}
	getStatus, hasStatus, parseStatusErr := GetParallelRegionRuntimeStatus("dashboard.hot-panel:click-event")
	if parseStatusErr != nil {
		parseT.Fatalf("GetParallelRegionRuntimeStatus returned error: %v", parseStatusErr)
	}
	if !hasStatus {
		parseT.Fatal("expected public runtime status after bridged click event")
	}
	if getStatus.GetLastCommittedVersion != 2 {
		parseT.Fatalf("expected committed version 2 after bridged click patch, got %d", getStatus.GetLastCommittedVersion)
	}
}

func TestBuildParallelRegionRenderedNodePreservesBridgedClickHandler(parseT *testing.T) {
	resetParallelRegionRegistry()
	parseT.Cleanup(resetParallelRegionRegistry)

	parseAdapter := newQueryHydrationDOMAdapter()
	parseScheduler := &queuedScheduler{}

	parsePreviousInitialized := runtimeInitialized
	runtimeInitialized = true
	parseT.Cleanup(func() {
		runtimeInitialized = parsePreviousInitialized
	})
	resetUIRuntime(runtime.Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	if parseErr := RegisterParallelRegion("dashboard.hot-panel", func(parseProps registerParallelRegionProps) Node {
		return Node(runtime.CreateElement("div", map[string]interface{}{
			parallelRegionClickSlotProp: "primary.action",
			"onclick": func() {
			},
		}, Text(parseProps.Label)))
	}); parseErr != nil {
		parseT.Fatalf("RegisterParallelRegion returned error: %v", parseErr)
	}

	getRuntimeSpec, parseRuntimeSpecErr := buildParallelRegionRuntimeSpec(ParallelRegionSpec[registerParallelRegionProps]{
		RendererID:       "dashboard.hot-panel",
		RegionInstanceID: "dashboard.hot-panel:rendered-node-click-bridge",
		Props: registerParallelRegionProps{
			Label: "Idle",
		},
	})
	if parseRuntimeSpecErr != nil {
		parseT.Fatalf("buildParallelRegionRuntimeSpec returned error: %v", parseRuntimeSpecErr)
	}

	getRenderedNode := buildParallelRegionRenderedNode(getRuntimeSpec, func(parseProps registerParallelRegionProps) Node {
		return Node(runtime.CreateElement("div", map[string]interface{}{
			parallelRegionClickSlotProp: "primary.action",
			"onclick": func() {
			},
		}, Text(parseProps.Label)))
	}, []runtime2.SchedulerShardID{"ui-parallel-region"})
	if getRenderedNode == nil || len(getRenderedNode.Children) == 0 {
		parseT.Fatalf("expected rendered parallel-region shell child, got %#v", getRenderedNode)
	}
	getInteractiveNode, hasInteractiveNode := getRenderedNode.Children[0].(*runtime.Element)
	if !hasInteractiveNode {
		parseT.Fatalf("expected rendered parallel-region child element, got %T", getRenderedNode.Children[0])
	}
	if _, hasWrappedHandler := getInteractiveNode.Props["onclick"].(func(runtime.GoEvent)); !hasWrappedHandler {
		parseT.Fatalf("expected rendered parallel-region child to keep bridged onclick handler, got %T", getInteractiveNode.Props["onclick"])
	}
}

func TestParallelRegionCustomSchedulerShardsFlowIntoAssignedRuntimeStatus(parseT *testing.T) {
	resetParallelRegionRegistry()
	parseT.Cleanup(resetParallelRegionRegistry)

	parseAdapter := newQueryHydrationDOMAdapter()
	parseContainer := parseAdapter.CreateElement("section")
	parseScheduler := &queuedScheduler{}

	parsePreviousInitialized := runtimeInitialized
	runtimeInitialized = true
	parseT.Cleanup(func() {
		runtimeInitialized = parsePreviousInitialized
	})
	resetUIRuntime(runtime.Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	if parseErr := RegisterParallelRegion("dashboard.hot-panel", func(parseProps registerParallelRegionProps) Node {
		return Text(parseProps.Label)
	}); parseErr != nil {
		parseT.Fatalf("RegisterParallelRegion returned error: %v", parseErr)
	}

	getSchedulerShardIDs := []runtime2.SchedulerShardID{"worker-a", "worker-b", "worker-c", "worker-d"}
	getExpectedShardModel := runtime2.BuildSchedulerShardModel()
	getExpectedShardID, parseExpectedShardErr := getExpectedShardModel.GetSchedulerRegionShardID(
		"dashboard.hot-panel:custom-shards",
		getSchedulerShardIDs,
		runtime2.SchedulerAssignmentPolicyKeep,
	)
	if parseExpectedShardErr != nil {
		parseT.Fatalf("GetSchedulerRegionShardID returned error: %v", parseExpectedShardErr)
	}

	if parseErr := RenderInto(ParallelRegion(ParallelRegionSpec[registerParallelRegionProps]{
		RendererID:       "dashboard.hot-panel",
		RegionInstanceID: "dashboard.hot-panel:custom-shards",
		Props: registerParallelRegionProps{
			Label: "One",
		},
		SchedulerShardIDs: []string{"worker-a", "worker-b", "worker-c", "worker-d"},
	}), parseContainer); parseErr != nil {
		parseT.Fatalf("RenderInto(ParallelRegion) returned error: %v", parseErr)
	}
	parseScheduler.Flush()
	if parseErr := RenderInto(ParallelRegion(ParallelRegionSpec[registerParallelRegionProps]{
		RendererID:       "dashboard.hot-panel",
		RegionInstanceID: "dashboard.hot-panel:custom-shards",
		Props: registerParallelRegionProps{
			Label: "Two",
		},
		SchedulerShardIDs: []string{"worker-a", "worker-b", "worker-c", "worker-d"},
	}), parseContainer); parseErr != nil {
		parseT.Fatalf("RenderInto(second ParallelRegion) returned error: %v", parseErr)
	}
	parseScheduler.Flush()

	getStatus, hasStatus, parseStatusErr := GetParallelRegionRuntimeStatus("dashboard.hot-panel:custom-shards")
	if parseStatusErr != nil {
		parseT.Fatalf("GetParallelRegionRuntimeStatus returned error: %v", parseStatusErr)
	}
	if !hasStatus {
		parseT.Fatal("expected public region status for custom scheduler shards")
	}
	if getStatus.GetAssignedWorkerShard != string(getExpectedShardID) {
		parseT.Fatalf("expected assigned worker shard %q, got %q", getExpectedShardID, getStatus.GetAssignedWorkerShard)
	}
	if getStatus.GetAssignedWorkerShard == "ui-parallel-region" {
		parseT.Fatalf("expected custom scheduler shards to avoid default shard, got %q", getStatus.GetAssignedWorkerShard)
	}
}

func TestParallelRegionTransitionWrappedRerendersUseDeferredDispatch(parseT *testing.T) {
	resetParallelRegionRegistry()
	parseT.Cleanup(resetParallelRegionRegistry)

	parseAdapter := newQueryHydrationDOMAdapter()
	parseContainer := parseAdapter.CreateElement("section")
	parseScheduler := &queuedScheduler{}

	parsePreviousInitialized := runtimeInitialized
	runtimeInitialized = true
	parseT.Cleanup(func() {
		runtimeInitialized = parsePreviousInitialized
	})
	resetUIRuntime(runtime.Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	if parseErr := RegisterParallelRegion("dashboard.hot-panel", func(parseProps registerParallelRegionProps) Node {
		return Text(parseProps.Label)
	}); parseErr != nil {
		parseT.Fatalf("RegisterParallelRegion returned error: %v", parseErr)
	}

	var storeSetLabel func(string)
	var storeStartTransition func(func())
	parseOwner := func() Node {
		parseState := UseState("One")
		parseTransition := UseTransition()
		storeSetLabel = func(parseLabel string) {
			parseState.Set(parseLabel)
		}
		storeStartTransition = parseTransition.Start
		return ParallelRegion(ParallelRegionSpec[registerParallelRegionProps]{
			RendererID:       "dashboard.hot-panel",
			RegionInstanceID: "dashboard.hot-panel:transition",
			Props: registerParallelRegionProps{
				Label: parseState.Get(),
			},
		})
	}

	if parseErr := RenderInto(CreateElement(parseOwner), parseContainer); parseErr != nil {
		parseT.Fatalf("RenderInto(owner) returned error: %v", parseErr)
	}
	parseScheduler.Flush()
	getParallelRegionHostAdapter, hasParallelRegionHostAdapter := resolveParallelRegionHostAdapter("dashboard.hot-panel:transition")
	if !hasParallelRegionHostAdapter {
		parseT.Fatal("expected transition test to mount a runtime2 host adapter")
	}

	storeStartTransition(func() {
		storeSetLabel("Two")
	})
	parseScheduler.Flush()

	getRuntimeStatus, hasRuntimeStatus := getParallelRegionHostAdapter.GetHostRegionRuntimeStatus()
	if !hasRuntimeStatus {
		parseT.Fatal("expected transition test runtime status after deferred rerender")
	}
	if getParallelRegionHostAdapter.GetHostRegionDeferredInputVersion() != 2 {
		parseT.Fatalf("expected transition rerender to queue deferred input version 2, got %d", getParallelRegionHostAdapter.GetHostRegionDeferredInputVersion())
	}
	if getRuntimeStatus.GetLastDispatchedVersion != 0 {
		parseT.Fatalf("expected deferred transition rerender to avoid immediate dispatch, got %d", getRuntimeStatus.GetLastDispatchedVersion)
	}
	if getRuntimeStatus.GetLastSnapshotVersion != 2 {
		parseT.Fatalf("expected deferred transition rerender snapshot version 2, got %d", getRuntimeStatus.GetLastSnapshotVersion)
	}

	storeSetLabel("Three")
	parseScheduler.Flush()

	getRuntimeStatus, hasRuntimeStatus = getParallelRegionHostAdapter.GetHostRegionRuntimeStatus()
	if !hasRuntimeStatus {
		parseT.Fatal("expected transition test runtime status after urgent rerender")
	}
	if getParallelRegionHostAdapter.GetHostRegionDeferredInputVersion() != 0 {
		parseT.Fatalf("expected urgent rerender to clear deferred queue, got %d", getParallelRegionHostAdapter.GetHostRegionDeferredInputVersion())
	}
	if getRuntimeStatus.GetLastDispatchedVersion != 3 {
		parseT.Fatalf("expected urgent rerender to dispatch input version 3 immediately, got %d", getRuntimeStatus.GetLastDispatchedVersion)
	}
	if getRuntimeStatus.GetLastSnapshotVersion != 3 {
		parseT.Fatalf("expected urgent rerender snapshot version 3, got %d", getRuntimeStatus.GetLastSnapshotVersion)
	}
}

func TestParallelRegionRendererIdentityChangesTriggerStructuralRemount(parseT *testing.T) {
	resetParallelRegionRegistry()
	parseT.Cleanup(resetParallelRegionRegistry)

	parseAdapter := newQueryHydrationDOMAdapter()
	parseContainer := parseAdapter.CreateElement("section")
	parseScheduler := &queuedScheduler{}

	parsePreviousInitialized := runtimeInitialized
	runtimeInitialized = true
	parseT.Cleanup(func() {
		runtimeInitialized = parsePreviousInitialized
	})
	resetUIRuntime(runtime.Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	if parseErr := RegisterParallelRegion("dashboard.hot-panel-a", func(parseProps registerParallelRegionProps) Node {
		return Text("A:" + parseProps.Label)
	}); parseErr != nil {
		parseT.Fatalf("RegisterParallelRegion(first) returned error: %v", parseErr)
	}
	if parseErr := RegisterParallelRegion("dashboard.hot-panel-b", func(parseProps registerParallelRegionProps) Node {
		return Text("B:" + parseProps.Label)
	}); parseErr != nil {
		parseT.Fatalf("RegisterParallelRegion(second) returned error: %v", parseErr)
	}

	if parseErr := RenderInto(ParallelRegion(ParallelRegionSpec[registerParallelRegionProps]{
		RendererID:       "dashboard.hot-panel-a",
		RegionInstanceID: "dashboard.hot-panel:structural-remount",
		Props: registerParallelRegionProps{
			Label: "One",
		},
	}), parseContainer); parseErr != nil {
		parseT.Fatalf("RenderInto(first ParallelRegion) returned error: %v", parseErr)
	}
	parseScheduler.Flush()
	getParallelRegionHostAdapter, hasParallelRegionHostAdapter := resolveParallelRegionHostAdapter("dashboard.hot-panel:structural-remount")
	if !hasParallelRegionHostAdapter {
		parseT.Fatal("expected first structural-remount render to mount a runtime2 host adapter")
	}
	getRuntimeStatus, hasRuntimeStatus := getParallelRegionHostAdapter.GetHostRegionRuntimeStatus()
	if !hasRuntimeStatus {
		parseT.Fatal("expected mounted structural-remount region to report runtime status")
	}
	if getRuntimeStatus.GetRendererID != "dashboard.hot-panel-a" {
		parseT.Fatalf("expected first renderer dashboard.hot-panel-a, got %q", getRuntimeStatus.GetRendererID)
	}
	if getRuntimeStatus.GetEpoch != 1 {
		parseT.Fatalf("expected first epoch 1, got %d", getRuntimeStatus.GetEpoch)
	}

	if parseErr := RenderInto(ParallelRegion(ParallelRegionSpec[registerParallelRegionProps]{
		RendererID:       "dashboard.hot-panel-b",
		RegionInstanceID: "dashboard.hot-panel:structural-remount",
		Props: registerParallelRegionProps{
			Label: "Two",
		},
	}), parseContainer); parseErr != nil {
		parseT.Fatalf("RenderInto(second ParallelRegion) returned error: %v", parseErr)
	}
	parseScheduler.Flush()

	getRuntimeStatus, hasRuntimeStatus = getParallelRegionHostAdapter.GetHostRegionRuntimeStatus()
	if !hasRuntimeStatus {
		parseT.Fatal("expected structural-remount region runtime status after renderer change")
	}
	if getRuntimeStatus.GetRendererID != "dashboard.hot-panel-b" {
		parseT.Fatalf("expected remounted renderer dashboard.hot-panel-b, got %q", getRuntimeStatus.GetRendererID)
	}
	if getRuntimeStatus.GetEpoch != 2 {
		parseT.Fatalf("expected structural remount to advance epoch to 2, got %d", getRuntimeStatus.GetEpoch)
	}
	if getRuntimeStatus.GetLastDispatchedVersion != 0 {
		parseT.Fatalf("expected structural remount rerender to stay mount-only, got dispatched version %d", getRuntimeStatus.GetLastDispatchedVersion)
	}
}

func TestParallelRegionRenderIntoDeclaredSourcesSupportRuntime2UpdateDispatch(parseT *testing.T) {
	resetParallelRegionRegistry()
	parseT.Cleanup(resetParallelRegionRegistry)

	parseAdapter := newQueryHydrationDOMAdapter()
	parseContainer := parseAdapter.CreateElement("section")
	parseScheduler := &queuedScheduler{}

	parsePreviousInitialized := runtimeInitialized
	runtimeInitialized = true
	parseT.Cleanup(func() {
		runtimeInitialized = parsePreviousInitialized
	})
	resetUIRuntime(runtime.Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})
	if parseErr := runtime.GetGlobalRuntime().SetAtomValue("dashboard.hot-count", 1); parseErr != nil {
		parseT.Fatalf("SetAtomValue(first) returned error: %v", parseErr)
	}

	if parseErr := RegisterParallelRegion("dashboard.hot-panel", func(parseProps registerParallelRegionProps) Node {
		return Text(parseProps.Label)
	}); parseErr != nil {
		parseT.Fatalf("RegisterParallelRegion returned error: %v", parseErr)
	}

	if parseErr := RenderInto(ParallelRegion(ParallelRegionSpec[registerParallelRegionProps]{
		RendererID:       "dashboard.hot-panel",
		RegionInstanceID: "dashboard.hot-panel:dispatch-sources",
		Props: registerParallelRegionProps{
			Label: "Stable",
		},
		SourceIDs: []string{"dashboard.hot-count"},
	}), parseContainer); parseErr != nil {
		parseT.Fatalf("RenderInto(first ParallelRegion) returned error: %v", parseErr)
	}
	parseScheduler.Flush()
	getParallelRegionHostAdapter, hasParallelRegionHostAdapter := resolveParallelRegionHostAdapter("dashboard.hot-panel:dispatch-sources")
	if !hasParallelRegionHostAdapter {
		parseT.Fatal("expected source-bound public region to mount a runtime2 host adapter")
	}
	getRuntimeStatus, hasRuntimeStatus := getParallelRegionHostAdapter.GetHostRegionRuntimeStatus()
	if !hasRuntimeStatus {
		parseT.Fatal("expected mounted source-bound public region to report runtime status")
	}
	if getRuntimeStatus.GetLastDispatchedVersion != 0 {
		parseT.Fatalf("expected initial source-bound render to stay mount-only, got dispatched version %d", getRuntimeStatus.GetLastDispatchedVersion)
	}

	if parseErr := runtime.GetGlobalRuntime().SetAtomValue("dashboard.hot-count", 2); parseErr != nil {
		parseT.Fatalf("SetAtomValue(second) returned error: %v", parseErr)
	}
	if parseErr := RenderInto(ParallelRegion(ParallelRegionSpec[registerParallelRegionProps]{
		RendererID:       "dashboard.hot-panel",
		RegionInstanceID: "dashboard.hot-panel:dispatch-sources",
		Props: registerParallelRegionProps{
			Label: "Updated",
		},
		SourceIDs: []string{"dashboard.hot-count"},
	}), parseContainer); parseErr != nil {
		parseT.Fatalf("RenderInto(second ParallelRegion) returned error: %v", parseErr)
	}
	parseScheduler.Flush()
	getRuntimeStatus, hasRuntimeStatus = getParallelRegionHostAdapter.GetHostRegionRuntimeStatus()
	if !hasRuntimeStatus {
		parseT.Fatal("expected source-bound public region runtime status after rerender")
	}
	if getRuntimeStatus.GetLastSnapshotVersion != 2 {
		parseT.Fatalf("expected source-bound rerender snapshot version 2, got %d", getRuntimeStatus.GetLastSnapshotVersion)
	}
	if getRuntimeStatus.GetLastDispatchedVersion != 2 {
		parseT.Fatalf("expected source-bound rerender to dispatch input version 2, got %d", getRuntimeStatus.GetLastDispatchedVersion)
	}
}

func TestParallelRegionDeclaredSourceOnlyChangesDispatchRuntime2Update(parseT *testing.T) {
	resetParallelRegionRegistry()
	parseT.Cleanup(resetParallelRegionRegistry)

	parseAdapter := newQueryHydrationDOMAdapter()
	parseContainer := parseAdapter.CreateElement("section")
	parseScheduler := &queuedScheduler{}

	parsePreviousInitialized := runtimeInitialized
	runtimeInitialized = true
	parseT.Cleanup(func() {
		runtimeInitialized = parsePreviousInitialized
	})
	resetUIRuntime(runtime.Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})
	if parseErr := runtime.GetGlobalRuntime().SetAtomValue("dashboard.hot-count", 1); parseErr != nil {
		parseT.Fatalf("SetAtomValue(first) returned error: %v", parseErr)
	}

	if parseErr := RegisterParallelRegion("dashboard.hot-panel", func(parseProps registerParallelRegionProps) Node {
		return Text(parseProps.Label)
	}); parseErr != nil {
		parseT.Fatalf("RegisterParallelRegion returned error: %v", parseErr)
	}

	if parseErr := RenderInto(ParallelRegion(ParallelRegionSpec[registerParallelRegionProps]{
		RendererID:       "dashboard.hot-panel",
		RegionInstanceID: "dashboard.hot-panel:source-only",
		Props: registerParallelRegionProps{
			Label: "Stable",
		},
		SourceIDs: []string{"dashboard.hot-count"},
	}), parseContainer); parseErr != nil {
		parseT.Fatalf("RenderInto(ParallelRegion) returned error: %v", parseErr)
	}
	parseScheduler.Flush()
	getParallelRegionHostAdapter, hasParallelRegionHostAdapter := resolveParallelRegionHostAdapter("dashboard.hot-panel:source-only")
	if !hasParallelRegionHostAdapter {
		parseT.Fatal("expected source-only public region to mount a runtime2 host adapter")
	}
	getRuntimeStatus, hasRuntimeStatus := getParallelRegionHostAdapter.GetHostRegionRuntimeStatus()
	if !hasRuntimeStatus {
		parseT.Fatal("expected mounted source-only public region to report runtime status")
	}
	if getRuntimeStatus.GetLastDispatchedVersion != 0 {
		parseT.Fatalf("expected initial source-only render to stay mount-only, got dispatched version %d", getRuntimeStatus.GetLastDispatchedVersion)
	}

	if parseErr := runtime.GetGlobalRuntime().SetAtomValue("dashboard.hot-count", 2); parseErr != nil {
		parseT.Fatalf("SetAtomValue(second) returned error: %v", parseErr)
	}
	parseScheduler.Flush()

	getRuntimeStatus, hasRuntimeStatus = getParallelRegionHostAdapter.GetHostRegionRuntimeStatus()
	if !hasRuntimeStatus {
		parseT.Fatal("expected source-only public region runtime status after reactive update")
	}
	if getRuntimeStatus.GetLastSnapshotVersion != 2 {
		parseT.Fatalf("expected source-only reactive update snapshot version 2, got %d", getRuntimeStatus.GetLastSnapshotVersion)
	}
	if getRuntimeStatus.GetLastDispatchedVersion != 2 {
		parseT.Fatalf("expected source-only reactive update to dispatch input version 2, got %d", getRuntimeStatus.GetLastDispatchedVersion)
	}
	if getInputVersion := resolveParallelRegionInputVersion("dashboard.hot-panel:source-only"); getInputVersion != 2 {
		parseT.Fatalf("expected public source-only input version 2 after reactive update, got %d", getInputVersion)
	}
}

func TestHydrateIntoUsesExplicitNode(parseT *testing.T) {
	parseAdapter := newQueryHydrationDOMAdapter()
	parseContainer := parseAdapter.CreateElement("section")

	parsePreviousInitialized := runtimeInitialized
	runtimeInitialized = true
	parseT.Cleanup(func() {
		runtimeInitialized = parsePreviousInitialized
	})
	resetUIRuntime(runtime.Config{DOMAdapter: parseAdapter, Scheduler: noOpScheduler{}})

	if _, parseErr := HydrateInto(Text("hello"), parseContainer); parseErr != nil {
		parseT.Fatalf("expected HydrateInto to succeed, got %v", parseErr)
	}
}

func TestHydrateIntoMarksParallelRegionAdapterHydrationComplete(parseT *testing.T) {
	resetParallelRegionRegistry()
	parseT.Cleanup(resetParallelRegionRegistry)

	parseAdapter := newQueryHydrationDOMAdapter()
	parseContainer := parseAdapter.CreateElement("section")
	parseShell := parseAdapter.CreateElement("div")
	parseShellMarker, parseMarkerErr := runtime2.BuildSSRShellMarkerAttributeValue(runtime2.SSRShellMarker{
		Version:          runtime2.SSRShellMarkerVersionV1,
		RegionInstanceID: "dashboard.hot-panel:hydration",
		RendererID:       "dashboard.hot-panel",
	})
	if parseMarkerErr != nil {
		parseT.Fatalf("BuildSSRShellMarkerAttributeValue returned error: %v", parseMarkerErr)
	}
	parseAdapter.SetAttribute(parseShell, runtime2.SSRShellMarkerAttribute, parseShellMarker)
	parseAdapter.AppendChild(parseShell, parseAdapter.CreateTextNode("Hydrated Region"))
	parseAdapter.AppendChild(parseContainer, parseShell)

	parsePreviousInitialized := runtimeInitialized
	runtimeInitialized = true
	parseT.Cleanup(func() {
		runtimeInitialized = parsePreviousInitialized
	})
	parseScheduler := &queuedScheduler{}
	resetUIRuntime(runtime.Config{
		DOMAdapter: parseAdapter,
		Scheduler:  parseScheduler,
	})

	if parseErr := RegisterParallelRegion("dashboard.hot-panel", func(parseProps registerParallelRegionProps) Node {
		return Text(parseProps.Label)
	}); parseErr != nil {
		parseT.Fatalf("RegisterParallelRegion returned error: %v", parseErr)
	}

	if _, parseErr := HydrateInto(ParallelRegion(ParallelRegionSpec[registerParallelRegionProps]{
		RendererID:       "dashboard.hot-panel",
		RegionInstanceID: "dashboard.hot-panel:hydration",
		Props: registerParallelRegionProps{
			Label: "Hydrated Region",
		},
	}), parseContainer); parseErr != nil {
		parseT.Fatalf("HydrateInto returned error: %v", parseErr)
	}
	parseScheduler.Flush()

	parseAdapterState, hasAdapter := resolveParallelRegionHostAdapter("dashboard.hot-panel:hydration")
	if !hasAdapter {
		parseT.Fatal("expected hydrated ParallelRegion to mount a runtime2 host adapter")
	}
	if !parseAdapterState.GetHostRegionIsHydrationComplete() {
		parseT.Fatal("expected hydrated ParallelRegion host adapter to report hydration complete")
	}
	if !parseAdapterState.HasHostRegionHydratedShellAnchor() {
		parseT.Fatal("expected hydrated ParallelRegion host adapter to register hydrated shell anchor")
	}
	if !parseAdapterState.HasHostRegionPostHydrationAttached() {
		parseT.Fatal("expected hydrated ParallelRegion host adapter to mark post-hydration attach")
	}
	getRuntimeStatus, hasRuntimeStatus := parseAdapterState.GetHostRegionRuntimeStatus()
	if !hasRuntimeStatus {
		parseT.Fatal("expected hydrated ParallelRegion runtime status")
	}
	if getRuntimeStatus.GetRegionMode != runtime2.HostRegionRuntimeModeWorkerAttached {
		parseT.Fatalf("expected hydrated ParallelRegion to report worker-attached mode, got %q", getRuntimeStatus.GetRegionMode)
	}
	getShellAnchor, parseAnchorLookupErr := parseAdapterState.GetHostRegionDOMIndex().GetRegionDOMNode("dashboard.hot-panel:hydration", 1)
	if parseAnchorLookupErr != nil {
		parseT.Fatalf("GetRegionDOMNode(hydrated shell anchor) returned error: %v", parseAnchorLookupErr)
	}
	if getShellAnchor.GetTag != "div" {
		parseT.Fatalf("expected hydrated shell anchor tag div, got %q", getShellAnchor.GetTag)
	}
}

func TestGetParallelRegionRuntimeStatusReportsHydratedPublicAttachState(parseT *testing.T) {
	resetParallelRegionRegistry()
	parseT.Cleanup(resetParallelRegionRegistry)

	parseAdapter := newQueryHydrationDOMAdapter()
	parseContainer := parseAdapter.CreateElement("section")
	parseShell := parseAdapter.CreateElement("div")
	parseShellMarker, parseMarkerErr := runtime2.BuildSSRShellMarkerAttributeValue(runtime2.SSRShellMarker{
		Version:          runtime2.SSRShellMarkerVersionV1,
		RegionInstanceID: "dashboard.hot-panel:status-hydration",
		RendererID:       "dashboard.hot-panel",
	})
	if parseMarkerErr != nil {
		parseT.Fatalf("BuildSSRShellMarkerAttributeValue returned error: %v", parseMarkerErr)
	}
	parseAdapter.SetAttribute(parseShell, runtime2.SSRShellMarkerAttribute, parseShellMarker)
	parseAdapter.AppendChild(parseShell, parseAdapter.CreateTextNode("Hydrated Region"))
	parseAdapter.AppendChild(parseContainer, parseShell)

	parsePreviousInitialized := runtimeInitialized
	runtimeInitialized = true
	parseT.Cleanup(func() {
		runtimeInitialized = parsePreviousInitialized
	})
	parseScheduler := &queuedScheduler{}
	resetUIRuntime(runtime.Config{
		DOMAdapter: parseAdapter,
		Scheduler:  parseScheduler,
	})

	if parseErr := RegisterParallelRegion("dashboard.hot-panel", func(parseProps registerParallelRegionProps) Node {
		return Text(parseProps.Label)
	}); parseErr != nil {
		parseT.Fatalf("RegisterParallelRegion returned error: %v", parseErr)
	}

	if _, parseErr := HydrateInto(ParallelRegion(ParallelRegionSpec[registerParallelRegionProps]{
		RendererID:       "dashboard.hot-panel",
		RegionInstanceID: "dashboard.hot-panel:status-hydration",
		Props: registerParallelRegionProps{
			Label: "Hydrated Region",
		},
	}), parseContainer); parseErr != nil {
		parseT.Fatalf("HydrateInto returned error: %v", parseErr)
	}
	parseScheduler.Flush()

	getStatus, hasStatus, parseStatusErr := GetParallelRegionRuntimeStatus("dashboard.hot-panel:status-hydration")
	if parseStatusErr != nil {
		parseT.Fatalf("GetParallelRegionRuntimeStatus returned error: %v", parseStatusErr)
	}
	if !hasStatus {
		parseT.Fatal("expected public region status after hydration attach")
	}
	if getStatus.GetRegionMode != string(runtime2.HostRegionRuntimeModeWorkerAttached) {
		parseT.Fatalf("expected worker-attached public region mode, got %q", getStatus.GetRegionMode)
	}
	if !getStatus.GetIsHydrationComplete {
		parseT.Fatal("expected hydrated public region status to report hydration complete")
	}
	if !getStatus.HasHydratedShellAnchor {
		parseT.Fatal("expected hydrated public region status to report shell anchor")
	}
	if !getStatus.HasPostHydrationAttached {
		parseT.Fatal("expected hydrated public region status to report post-hydration attach")
	}
}

func TestHydrateMarksParallelRegionAdapterAnchorAndAttachBySelector(parseT *testing.T) {
	resetParallelRegionRegistry()
	parseT.Cleanup(resetParallelRegionRegistry)

	parseAdapter := newQueryHydrationDOMAdapter()
	parseContainer := parseAdapter.CreateElement("section")
	parseShell := parseAdapter.CreateElement("div")
	parseShellMarker, parseMarkerErr := runtime2.BuildSSRShellMarkerAttributeValue(runtime2.SSRShellMarker{
		Version:          runtime2.SSRShellMarkerVersionV1,
		RegionInstanceID: "dashboard.hot-panel:hydration-selector",
		RendererID:       "dashboard.hot-panel",
	})
	if parseMarkerErr != nil {
		parseT.Fatalf("BuildSSRShellMarkerAttributeValue returned error: %v", parseMarkerErr)
	}
	parseAdapter.SetAttribute(parseShell, runtime2.SSRShellMarkerAttribute, parseShellMarker)
	parseAdapter.AppendChild(parseShell, parseAdapter.CreateTextNode("Hydrated Region"))
	parseAdapter.AppendChild(parseContainer, parseShell)
	parseAdapter.selectors["#app"] = parseContainer

	parsePreviousInitialized := runtimeInitialized
	runtimeInitialized = true
	parseT.Cleanup(func() {
		runtimeInitialized = parsePreviousInitialized
	})
	parseScheduler := &queuedScheduler{}
	resetUIRuntime(runtime.Config{
		DOMAdapter: parseAdapter,
		Scheduler:  parseScheduler,
	})

	if parseErr := RegisterParallelRegion("dashboard.hot-panel", func(parseProps registerParallelRegionProps) Node {
		return Text(parseProps.Label)
	}); parseErr != nil {
		parseT.Fatalf("RegisterParallelRegion returned error: %v", parseErr)
	}

	if _, parseErr := Hydrate(ParallelRegion(ParallelRegionSpec[registerParallelRegionProps]{
		RendererID:       "dashboard.hot-panel",
		RegionInstanceID: "dashboard.hot-panel:hydration-selector",
		Props: registerParallelRegionProps{
			Label: "Hydrated Region",
		},
	}), "#app"); parseErr != nil {
		parseT.Fatalf("Hydrate returned error: %v", parseErr)
	}
	parseScheduler.Flush()

	parseAdapterState, hasAdapter := resolveParallelRegionHostAdapter("dashboard.hot-panel:hydration-selector")
	if !hasAdapter {
		parseT.Fatal("expected selector-hydrated ParallelRegion to mount a runtime2 host adapter")
	}
	if !parseAdapterState.GetHostRegionIsHydrationComplete() {
		parseT.Fatal("expected selector-hydrated ParallelRegion host adapter to report hydration complete")
	}
	if !parseAdapterState.HasHostRegionHydratedShellAnchor() {
		parseT.Fatal("expected selector-hydrated ParallelRegion host adapter to register hydrated shell anchor")
	}
	if !parseAdapterState.HasHostRegionPostHydrationAttached() {
		parseT.Fatal("expected selector-hydrated ParallelRegion host adapter to mark post-hydration attach")
	}
}

func TestHandleParallelRegionHydrationNodesRejectsShellIdentityMismatch(parseT *testing.T) {
	resetParallelRegionRegistry()
	parseT.Cleanup(resetParallelRegionRegistry)

	parseAdapter := newQueryHydrationDOMAdapter()
	parseContainer := parseAdapter.CreateElement("section")
	parseShell := parseAdapter.CreateElement("div")
	parseShellMarker, parseMarkerErr := runtime2.BuildSSRShellMarkerAttributeValue(runtime2.SSRShellMarker{
		Version:          runtime2.SSRShellMarkerVersionV1,
		RegionInstanceID: "dashboard.hot-panel:hydration-mismatch",
		RendererID:       "dashboard.other-panel",
	})
	if parseMarkerErr != nil {
		parseT.Fatalf("BuildSSRShellMarkerAttributeValue returned error: %v", parseMarkerErr)
	}
	parseAdapter.SetAttribute(parseShell, runtime2.SSRShellMarkerAttribute, parseShellMarker)
	parseAdapter.AppendChild(parseContainer, parseShell)

	parsePreviousInitialized := runtimeInitialized
	runtimeInitialized = true
	parseT.Cleanup(func() {
		runtimeInitialized = parsePreviousInitialized
	})
	parseScheduler := &queuedScheduler{}
	resetUIRuntime(runtime.Config{
		DOMAdapter: parseAdapter,
		Scheduler:  parseScheduler,
	})

	if parseErr := RegisterParallelRegion("dashboard.hot-panel", func(parseProps registerParallelRegionProps) Node {
		return Text(parseProps.Label)
	}); parseErr != nil {
		parseT.Fatalf("RegisterParallelRegion returned error: %v", parseErr)
	}
	if parseErr := RenderInto(ParallelRegion(ParallelRegionSpec[registerParallelRegionProps]{
		RendererID:       "dashboard.hot-panel",
		RegionInstanceID: "dashboard.hot-panel:hydration-mismatch",
		Props: registerParallelRegionProps{
			Label: "Mismatch",
		},
	}), parseContainer); parseErr != nil {
		parseT.Fatalf("RenderInto returned error: %v", parseErr)
	}
	parseScheduler.Flush()

	parseHydrationErr := handleParallelRegionHydrationNodes([]runtime.DOMNode{parseShell})
	if parseHydrationErr == nil {
		parseT.Fatal("expected hydration-node mismatch to fail")
	}
	parseAdapterState, hasAdapter := resolveParallelRegionHostAdapter("dashboard.hot-panel:hydration-mismatch")
	if !hasAdapter {
		parseT.Fatal("expected mismatched ParallelRegion to keep its runtime2 host adapter")
	}
	if parseAdapterState.GetHostRegionIsHydrationComplete() {
		parseT.Fatal("expected mismatched shell marker to avoid hydration-complete state")
	}
	if parseAdapterState.HasHostRegionHydratedShellAnchor() {
		parseT.Fatal("expected mismatched shell marker to avoid anchor registration")
	}
	if parseAdapterState.HasHostRegionPostHydrationAttached() {
		parseT.Fatal("expected mismatched shell marker to avoid post-hydration attach")
	}
}

func TestHandleParallelRegionHydrationNodesRejectsInvalidShellAnchor(parseT *testing.T) {
	resetParallelRegionRegistry()
	parseT.Cleanup(resetParallelRegionRegistry)

	parseAdapter := newQueryHydrationDOMAdapter()
	parseContainer := parseAdapter.CreateElement("section")
	parseShell := parseAdapter.CreateTextNode("Hydrated Region")
	parseShellMarker, parseMarkerErr := runtime2.BuildSSRShellMarkerAttributeValue(runtime2.SSRShellMarker{
		Version:          runtime2.SSRShellMarkerVersionV1,
		RegionInstanceID: "dashboard.hot-panel:hydration-invalid-anchor",
		RendererID:       "dashboard.hot-panel",
	})
	if parseMarkerErr != nil {
		parseT.Fatalf("BuildSSRShellMarkerAttributeValue returned error: %v", parseMarkerErr)
	}
	parseAdapter.SetAttribute(parseShell, runtime2.SSRShellMarkerAttribute, parseShellMarker)
	parseAdapter.AppendChild(parseContainer, parseShell)

	parsePreviousInitialized := runtimeInitialized
	runtimeInitialized = true
	parseT.Cleanup(func() {
		runtimeInitialized = parsePreviousInitialized
	})
	parseScheduler := &queuedScheduler{}
	resetUIRuntime(runtime.Config{
		DOMAdapter: parseAdapter,
		Scheduler:  parseScheduler,
	})

	if parseErr := RegisterParallelRegion("dashboard.hot-panel", func(parseProps registerParallelRegionProps) Node {
		return Text(parseProps.Label)
	}); parseErr != nil {
		parseT.Fatalf("RegisterParallelRegion returned error: %v", parseErr)
	}
	if parseErr := RenderInto(ParallelRegion(ParallelRegionSpec[registerParallelRegionProps]{
		RendererID:       "dashboard.hot-panel",
		RegionInstanceID: "dashboard.hot-panel:hydration-invalid-anchor",
		Props: registerParallelRegionProps{
			Label: "Invalid Anchor",
		},
	}), parseContainer); parseErr != nil {
		parseT.Fatalf("RenderInto returned error: %v", parseErr)
	}
	parseScheduler.Flush()

	parseHydrationErr := handleParallelRegionHydrationNodes([]runtime.DOMNode{parseShell})
	if parseHydrationErr == nil {
		parseT.Fatal("expected invalid hydration shell anchor to fail")
	}
	parseAdapterState, hasAdapter := resolveParallelRegionHostAdapter("dashboard.hot-panel:hydration-invalid-anchor")
	if !hasAdapter {
		parseT.Fatal("expected invalid-anchor ParallelRegion to keep its runtime2 host adapter")
	}
	if parseAdapterState.HasHostRegionHydratedShellAnchor() {
		parseT.Fatal("expected invalid shell anchor to avoid anchor registration")
	}
	if parseAdapterState.HasHostRegionPostHydrationAttached() {
		parseT.Fatal("expected invalid shell anchor to avoid post-hydration attach")
	}
}

func TestPublicHooksWrappers(parseT *testing.T) {
	installUIHookContext(parseT)

	parseState := UseState(1)
	if parseState.Get() != 1 {
		parseT.Fatalf("expected initial state, got %d", parseState.Get())
	}
	parseState.Set(3)
	if parseState.Get() != 3 {
		parseT.Fatalf("expected updated state, got %d", parseState.Get())
	}
	parseState.Update(func(parsePrev int) int { return parsePrev + 4 })
	if parseState.Get() != 7 {
		parseT.Fatalf("expected updated state after updater, got %d", parseState.Get())
	}

	parseReducer := UseReducer(func(parseState2 int, parseAction int) int { return parseState2 + parseAction }, 2)
	if parseReducer.Get() != 2 {
		parseT.Fatalf("expected initial reducer state, got %d", parseReducer.Get())
	}
	parseReducer.Dispatch(5)
	if parseReducer.Get() != 7 {
		parseT.Fatalf("expected reducer dispatch to update state, got %d", parseReducer.Get())
	}

	parseComputed := UseMemo(func() int { return 9 }, "dep")
	if parseComputed != 9 {
		parseT.Fatalf("expected memoized value 9, got %d", parseComputed)
	}

	parseCallback := UseCallback(func() int { return 11 }, "dep")
	if parseCallback() != 11 {
		parseT.Fatalf("expected callback wrapper to preserve function value")
	}

	parseRef := UseRef("start")
	if parseRef.Get() != "start" {
		parseT.Fatalf("expected initial ref value, got %q", parseRef.Get())
	}
	parseRef.Set("done")
	if parseRef.Get() != "done" {
		parseT.Fatalf("expected updated ref value, got %q", parseRef.Get())
	}

	parseId := UseId()
	if parseId == "" {
		parseT.Fatal("expected non-empty id")
	}

	parseDeferred := UseDeferredValue("steady")
	if parseDeferred != "steady" {
		parseT.Fatalf("expected deferred value to return initial value, got %q", parseDeferred)
	}
}

func TestPublicUseEffectSkipsStableDepsAndCleansBeforeChangedEffect(parseT *testing.T) {
	type effectProbeProps struct {
		Dep    string
		Tick   int
		Events *[]string
	}

	parseAdapter := newQueryHydrationDOMAdapter()
	parseContainer := parseAdapter.CreateElement("section")
	parseScheduler := &queuedScheduler{}

	parsePreviousInitialized := runtimeInitialized
	runtimeInitialized = true
	parseT.Cleanup(func() {
		runtimeInitialized = parsePreviousInitialized
	})
	resetUIRuntime(runtime.Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseEvents := []string{}
	parseProbe := func(parseProps effectProbeProps) Node {
		UseEffect(func() func() {
			parseDep := parseProps.Dep
			*parseProps.Events = append(*parseProps.Events, "effect:"+parseDep)
			return func() {
				*parseProps.Events = append(*parseProps.Events, "cleanup:"+parseDep)
			}
		}, parseProps.Dep)
		return Text(parseProps.Dep + ":" + strconv.Itoa(parseProps.Tick))
	}
	parseRender := func(parseDep string, parseTick int) {
		parseT.Helper()
		if parseErr := RenderInto(CreateElement(parseProbe, effectProbeProps{Dep: parseDep, Tick: parseTick, Events: &parseEvents}), parseContainer); parseErr != nil {
			parseT.Fatalf("RenderInto(%q) returned error: %v", parseDep, parseErr)
		}
		parseScheduler.Flush()
	}

	parseRender("stable", 1)
	if parseWant := []string{"effect:stable"}; !reflect.DeepEqual(parseEvents, parseWant) {
		parseT.Fatalf("expected first effect only, got %#v", parseEvents)
	}

	parseRender("stable", 2)
	if parseWant := []string{"effect:stable"}; !reflect.DeepEqual(parseEvents, parseWant) {
		parseT.Fatalf("expected stable deps to skip rerun, got %#v", parseEvents)
	}

	parseRender("changed", 3)
	parseWant := []string{"effect:stable", "cleanup:stable", "effect:changed"}
	if !reflect.DeepEqual(parseEvents, parseWant) {
		parseT.Fatalf("expected cleanup before changed effect, got %#v", parseEvents)
	}
}

func TestPublicUseReducerQueuedDispatchesApplyLatestStateInOrder(parseT *testing.T) {
	parseAdapter := newQueryHydrationDOMAdapter()
	parseContainer := parseAdapter.CreateElement("section")
	parseScheduler := &queuedScheduler{}

	parsePreviousInitialized := runtimeInitialized
	runtimeInitialized = true
	parseT.Cleanup(func() {
		runtimeInitialized = parsePreviousInitialized
	})
	resetUIRuntime(runtime.Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	var parseReducer Reducer[int, int]
	parseProbe := func() Node {
		parseReducer = UseReducer(func(parseState int, parseAction int) int {
			return parseState*10 + parseAction
		}, 0)
		return Text(strconv.Itoa(parseReducer.Get()))
	}

	if parseErr := RenderInto(CreateElement(parseProbe), parseContainer); parseErr != nil {
		parseT.Fatalf("RenderInto returned error: %v", parseErr)
	}
	parseScheduler.Flush()

	parseReducer.Dispatch(1)
	parseReducer.Dispatch(2)
	parseReducer.Dispatch(3)
	parseScheduler.Flush()

	if parseGot := parseReducer.Get(); parseGot != 123 {
		parseT.Fatalf("expected queued reducer dispatches to apply in order, got %d", parseGot)
	}
}

func TestPublicUseMemoAndCallbackRespectDepsAcrossRenders(parseT *testing.T) {
	type memoProbeProps struct {
		Dep  string
		Tick int
	}

	parseAdapter := newQueryHydrationDOMAdapter()
	parseContainer := parseAdapter.CreateElement("section")
	parseScheduler := &queuedScheduler{}

	parsePreviousInitialized := runtimeInitialized
	runtimeInitialized = true
	parseT.Cleanup(func() {
		runtimeInitialized = parsePreviousInitialized
	})
	resetUIRuntime(runtime.Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseRenderID := 0
	parseComputeCalls := 0
	parseMemoValues := []int{}
	parseCallbackValues := []int{}
	parseProbe := func(parseProps memoProbeProps) Node {
		parseRenderID++
		parseCurrentRenderID := parseRenderID
		parseMemo := UseMemo(func() int {
			parseComputeCalls++
			return parseComputeCalls
		}, parseProps.Dep)
		parseCallback := UseCallback(func() int {
			return parseCurrentRenderID
		}, parseProps.Dep)
		parseMemoValues = append(parseMemoValues, parseMemo)
		parseCallbackValues = append(parseCallbackValues, parseCallback())
		return Text(parseProps.Dep + ":" + strconv.Itoa(parseProps.Tick))
	}
	parseRender := func(parseDep string, parseTick int) {
		parseT.Helper()
		if parseErr := RenderInto(CreateElement(parseProbe, memoProbeProps{Dep: parseDep, Tick: parseTick}), parseContainer); parseErr != nil {
			parseT.Fatalf("RenderInto(%q) returned error: %v", parseDep, parseErr)
		}
		parseScheduler.Flush()
	}

	parseRender("same", 1)
	parseRender("same", 2)
	parseRender("next", 3)

	if parseComputeCalls != 2 {
		parseT.Fatalf("expected memo compute to run only for initial and changed deps, got %d", parseComputeCalls)
	}
	if parseWant := []int{1, 1, 2}; !reflect.DeepEqual(parseMemoValues, parseWant) {
		parseT.Fatalf("expected memo values %v, got %v", parseWant, parseMemoValues)
	}
	if parseWant := []int{1, 1, 3}; !reflect.DeepEqual(parseCallbackValues, parseWant) {
		parseT.Fatalf("expected callback to be reused for stable deps and replaced for changed deps, got %v", parseCallbackValues)
	}
}

func TestUsePersistedStateFallsBackOnCorruptStoredJSON(parseT *testing.T) {
	installUIHookContext(parseT)
	installUIPersistedStateStorage(parseT, "localStorage", map[string]string{
		"persisted-corrupt": "{not-json",
	}, "")

	parsePersisted := UsePersistedState[string]("persisted-corrupt", "fallback", PersistLocal)
	if parseGot := parsePersisted.Get(); parseGot != "fallback" {
		parseT.Fatalf("expected corrupt stored JSON to fall back to initial value, got %q", parseGot)
	}
}

func TestUsePersistedStateSetWritesJSONAndRecordsStorageErrors(parseT *testing.T) {
	parseT.Run("writes-json", func(parseT *testing.T) {
		installUIHookContext(parseT)
		parseStorage := installUIPersistedStateStorage(parseT, "localStorage", nil, "")

		parsePersisted := UsePersistedState[string]("persisted-write", "initial", PersistLocal)
		parsePersisted.Set("saved")

		if parseGot := parsePersisted.Get(); parseGot != "saved" {
			parseT.Fatalf("expected in-memory state to update, got %q", parseGot)
		}
		if parseStored := parseStorage.Values["persisted-write"]; parseStored != `"saved"` {
			parseT.Fatalf("expected JSON value in storage, got %q", parseStored)
		}
		if parseStorage.SetCalls != 1 {
			parseT.Fatalf("expected one storage write, got %d", parseStorage.SetCalls)
		}
		if parseErr := parsePersisted.Err(); parseErr != nil {
			parseT.Fatalf("expected successful write to clear persisted-state error, got %v", parseErr)
		}
	})

	parseT.Run("set-error", func(parseT *testing.T) {
		installUIHookContext(parseT)
		installUIPersistedStateStorage(parseT, "localStorage", nil, "quota exceeded")

		parsePersisted := UsePersistedState[string]("persisted-quota", "initial", PersistLocal)
		parsePersisted.Set("saved")

		if parseGot := parsePersisted.Get(); parseGot != "saved" {
			parseT.Fatalf("expected in-memory state to update despite storage error, got %q", parseGot)
		}
		parseErr := parsePersisted.Err()
		if parseErr == nil || !strings.Contains(parseErr.Error(), "quota exceeded") {
			parseT.Fatalf("expected quota error to be recorded, got %v", parseErr)
		}
	})
}

func TestUseIdProducesDistinctIDsWithinComponent(parseT *testing.T) {
	installUIHookContext(parseT)

	parseFirst := UseId()
	parseSecond := UseId()

	if parseFirst == "" || parseSecond == "" {
		parseT.Fatalf("expected non-empty ids, got %q and %q", parseFirst, parseSecond)
	}
	if parseFirst == parseSecond {
		parseT.Fatalf("expected distinct ids within one component render, got %q and %q", parseFirst, parseSecond)
	}
}

func TestUseCompositeNavigationHandlesKeyboardFlow(parseT *testing.T) {
	installUIHookContext(parseT)

	parseNav := UseCompositeNavigation([]CompositeItem{
		{ID: "alpha", Text: "Alpha"},
		{ID: "bravo", Text: "Bravo", Disabled: true},
		{ID: "charlie", Text: "Charlie"},
	}, CompositeNavigationOptions{Orientation: "horizontal", Loop: true})

	if parseNav.ActiveIndex() != 0 {
		parseT.Fatalf("expected initial active index 0, got %d", parseNav.ActiveIndex())
	}
	if parseNav.TabIndex(0) != 0 || parseNav.TabIndex(2) != -1 {
		parseT.Fatalf("expected roving tabindex behavior, got active=%d inactive=%d", parseNav.TabIndex(0), parseNav.TabIndex(2))
	}

	parsePrevented := 0
	parsePreventFn := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parsePrevented++
		return nil
	})
	parseT.Cleanup(func() { parsePreventFn.Release() })

	parseEventValue := js.Global().Get("Object").New()
	parseEventValue.Set("key", "ArrowRight")
	parseEventValue.Set("preventDefault", parsePreventFn)
	parseNav.OnKeyDown(runtime.NewGoEvent(parseEventValue))
	if parsePrevented != 1 {
		parseT.Fatalf("expected arrow navigation to prevent default once, got %d", parsePrevented)
	}
	if parseNav.ActiveIndex() != 2 {
		parseT.Fatalf("expected disabled item to be skipped, got active index %d", parseNav.ActiveIndex())
	}

	parseTypeaheadValue := js.Global().Get("Object").New()
	parseTypeaheadValue.Set("key", "a")
	parseTypeaheadValue.Set("preventDefault", parsePreventFn)
	parseNav.OnKeyDown(runtime.NewGoEvent(parseTypeaheadValue))
	if parseNav.ActiveIndex() != 0 {
		parseT.Fatalf("expected typeahead to jump to Alpha, got active index %d", parseNav.ActiveIndex())
	}
	if parseNav.ActiveDescendant() != "alpha" {
		parseT.Fatalf("expected active descendant alpha, got %q", parseNav.ActiveDescendant())
	}

	parseHomeValue := js.Global().Get("Object").New()
	parseHomeValue.Set("key", "End")
	parseHomeValue.Set("preventDefault", parsePreventFn)
	parseNav.OnKeyDown(runtime.NewGoEvent(parseHomeValue))
	if parseNav.ActiveIndex() != 2 {
		parseT.Fatalf("expected End to move to last enabled item, got %d", parseNav.ActiveIndex())
	}
}

func TestUseAnnouncerRendersPoliteAndAssertiveRegions(parseT *testing.T) {
	installUIHookContext(parseT)
	var parseMessageText func(*runtime.Element) string
	parseMessageText = func(parseNode *runtime.Element) string {
		if parseNode == nil {
			return ""
		}
		if parseNode.TextContent != "" {
			return parseNode.TextContent
		}
		if len(parseNode.Children) == 0 {
			return ""
		}
		switch parseChild := parseNode.Children[0].(type) {
		case string:
			return parseChild
		case *runtime.Element:
			if parseChild.TextContent != "" {
				return parseChild.TextContent
			}
			return parseMessageText(parseChild)
		default:
			return ""
		}
	}

	parseAnnouncer := UseAnnouncer()
	parseAnnouncer.Polite("Draft saved")
	parseAnnouncer.Assertive("Fix the required fields")

	parseRegion := parseAnnouncer.Region()
	if parseRegion == nil {
		parseT.Fatal("expected live region node")
	}
	if len(parseRegion.Children) != 2 {
		parseT.Fatalf("expected polite and assertive regions, got %#v", parseRegion.Children)
	}

	parsePolite, parseOk := parseRegion.Children[0].(*runtime.Element)
	if !parseOk {
		parseT.Fatalf("expected polite child element, got %T", parseRegion.Children[0])
	}
	parseAssertive, parseOk := parseRegion.Children[1].(*runtime.Element)
	if !parseOk {
		parseT.Fatalf("expected assertive child element, got %T", parseRegion.Children[1])
	}
	if parsePolite.Props["aria-live"] != "polite" {
		parseT.Fatalf("expected polite live region, got %#v", parsePolite.Props["aria-live"])
	}
	if parseAssertive.Props["aria-live"] != "assertive" {
		parseT.Fatalf("expected assertive live region, got %#v", parseAssertive.Props["aria-live"])
	}
	parsePoliteMessage, parseOk := parsePolite.Children[0].(*runtime.Element)
	if !parseOk || parseMessageText(parsePoliteMessage) != "Draft saved" {
		parseT.Fatalf("expected polite message child, got %#v", parsePolite.Children)
	}
	parseAssertiveMessage, parseOk := parseAssertive.Children[0].(*runtime.Element)
	if !parseOk || parseMessageText(parseAssertiveMessage) != "Fix the required fields" {
		parseT.Fatalf("expected assertive message child, got %#v", parseAssertive.Children)
	}
	if parseAnnouncer.PoliteID() == "" || parseAnnouncer.AssertiveID() == "" {
		parseT.Fatal("expected announcer region ids")
	}
	parseAnnouncer.Clear()
	if parseCleared := parseAnnouncer.Region(); len(parseCleared.Children) != 2 {
		parseT.Fatalf("expected cleared region structure to remain stable, got %#v", parseCleared.Children)
	}
}

func TestUseTransitionDefersPublicStateUpdates(parseT *testing.T) {
	parseScheduler := installQueuedUIHookContext(parseT)

	parseState := UseState(1)
	parseTransition := UseTransition()
	parseTransition.Start(func() {
		parseState.Set(6)
	})

	if parseGot := parseState.Get(); parseGot != 1 {
		parseT.Fatalf("expected transition update to remain deferred before flush, got %d", parseGot)
	}
	if !parseTransition.Pending() {
		parseT.Fatal("expected transition to report pending before flush")
	}

	parseScheduler.Flush()

	if parseGot2 := parseState.Get(); parseGot2 != 6 {
		parseT.Fatalf("expected deferred transition state update after flush, got %d", parseGot2)
	}
	if parseTransition.Pending() {
		parseT.Fatal("expected transition to report settled after flush")
	}
}

func TestUseContextFallsBackToDefaultValue(parseT *testing.T) {
	installUIHookContext(parseT)

	parseTheme := CreateContext("light")
	if parseGot := UseContext(parseTheme); parseGot != "light" {
		parseT.Fatalf("expected default context value light, got %q", parseGot)
	}
}

func TestCreateElementSupportsContextProvider(parseT *testing.T) {
	parseTheme := CreateContext("light")
	parseChild := Text("ready")
	parseNode := CreateElement(parseTheme.Provider, ContextProviderProps[string]{
		Value: "dark",
		Child: parseChild,
	})
	if parseNode == nil {
		parseT.Fatal("expected provider element to be created")
	}
	if _, parseOk := parseNode.Type.(*runtime.ContextProviderType); !parseOk {
		parseT.Fatalf("expected provider element type, got %T", parseNode.Type)
	}
	if len(parseNode.Children) != 1 || parseNode.Children[0] != parseChild {
		parseT.Fatalf("expected provider child to be preserved, got %#v", parseNode.Children)
	}
	if parseGot := parseNode.Props["value"]; parseGot != "dark" {
		parseT.Fatalf("expected provider value dark, got %#v", parseGot)
	}
}

func TestPortalBuildsRuntimePortalElement(parseT *testing.T) {
	parseChild := Text("overlay")
	parseTargetNode := js.Global().Get("Object").New()
	parseNode := Portal(PortalProps{
		Target: PortalTarget{Selector: "#portal-root", Node: parseTargetNode},
		Child:  parseChild,
	})
	if parseNode == nil {
		parseT.Fatal("expected portal element")
	}
	if _, parseOk := parseNode.Type.(*runtime.PortalElementType); !parseOk {
		parseT.Fatalf("expected portal runtime type, got %T", parseNode.Type)
	}
	if parseGot := parseNode.Props["portalTargetSelector"]; parseGot != "#portal-root" {
		parseT.Fatalf("expected selector target to round-trip, got %#v", parseGot)
	}
	if parseGot2 := parseNode.Props["portalTargetNode"]; !js.ValueOf(parseGot2).Equal(parseTargetNode) {
		parseT.Fatal("expected explicit node target to round-trip")
	}
	if len(parseNode.Children) != 1 || parseNode.Children[0] != parseChild {
		parseT.Fatalf("expected portal child to be preserved, got %#v", parseNode.Children)
	}
}

func TestReactiveRegionBuildsRuntimeRegionElement(parseT *testing.T) {
	parseNode := ReactiveRegion(func() Node {
		return Text("hot")
	}, reactiveRegionTestSource{id: "count"})
	if parseNode == nil {
		parseT.Fatal("expected reactive region element")
	}
	if _, parseOk := parseNode.Type.(*runtime.ReactiveRegionElementType); !parseOk {
		parseT.Fatalf("expected reactive region runtime type, got %T", parseNode.Type)
	}
	if parseGot, _ := parseNode.Props["__gwc_reactive_region_source_ids"].([]string); len(parseGot) != 1 || parseGot[0] != "count" {
		parseT.Fatalf("expected source ids to round-trip, got %#v", parseGot)
	}
	render, _ := parseNode.Props["__gwc_reactive_region_render"].(func() *runtime.Element)
	if render == nil {
		parseT.Fatal("expected reactive region render callback")
	}
	parseRendered := render()
	if parseRendered == nil || parseRendered.Type != "TEXT_ELEMENT" || parseRendered.TextContent != "hot" {
		parseT.Fatalf("expected region render callback to return text node, got %#v", parseRendered)
	}
}

func TestRefAndHandlerHelpers(parseT *testing.T) {
	var parseEmpty Ref[int]
	if parseEmpty.Get() != 0 {
		parseT.Fatalf("expected zero value from nil ref, got %d", parseEmpty.Get())
	}
	parseEmpty.Set(42)
	if parseEmpty.Get() != 0 {
		parseT.Fatal("expected nil ref Set to remain a no-op")
	}

	parseHandler := WrapHandler("wrapped")
	if parseHandler.Value() != "wrapped" {
		parseT.Fatalf("expected raw handler value, got %#v", parseHandler.Value())
	}

	runtime.InitGlobalRuntime(runtime.Config{
		DOMAdapter: buildUIWrapHandlerMarkerAdapter{
			MockDOMAdapter: mockdom.NewMockDOMAdapter(),
		},
		Scheduler: noOpScheduler{},
		Reset:     true,
	})
	parseWrappedHandler := WrapHandler(func() {}).Value()
	if parseWrappedHandler != "wrapped-handler" {
		parseT.Fatalf("expected wrapped handler marker, got %#v", parseWrappedHandler)
	}
}

func TestUsePreviousReturnsEmptyValueOnFirstRender(parseT *testing.T) {
	installUIHookContext(parseT)

	parsePrevious := UsePrevious("current")
	if parsePrevious.Ok() {
		parseT.Fatal("expected previous value to be unavailable on first render")
	}
	if parsePrevious.Get() != "" {
		parseT.Fatalf("expected zero value on first render, got %q", parsePrevious.Get())
	}
}

func TestPreviousHandleZeroValue(parseT *testing.T) {
	var parsePrevious Previous[int]
	if parsePrevious.Ok() {
		parseT.Fatal("expected zero-value previous handle to report unavailable")
	}
	if parsePrevious.Get() != 0 {
		parseT.Fatalf("expected zero-value previous handle to return zero, got %d", parsePrevious.Get())
	}
}

func TestUseChannelReturnsEmptyStateBeforeValues(parseT *testing.T) {
	installUIHookContext(parseT)

	parseCh := make(chan int)
	parseChannel := UseChannel(parseCh)
	if parseChannel.Ok() {
		parseT.Fatal("expected channel handle to report no value before any receive")
	}
	if parseChannel.Closed() {
		parseT.Fatal("expected channel handle to report open before closure is observed")
	}
	if parseChannel.Get() != 0 {
		parseT.Fatalf("expected zero value before any receive, got %d", parseChannel.Get())
	}
}

func TestChannelHandleZeroValue(parseT *testing.T) {
	var parseChannel Channel[string]
	if parseChannel.Ok() {
		parseT.Fatal("expected zero-value channel handle to report unavailable")
	}
	if parseChannel.Closed() {
		parseT.Fatal("expected zero-value channel handle to report open")
	}
	if parseChannel.Get() != "" {
		parseT.Fatalf("expected zero-value channel handle to return empty string, got %q", parseChannel.Get())
	}
}

func TestUseTaskTransitionsToRunningAndCancelled(parseT *testing.T) {
	installUIHookContext(parseT)

	parseBlock := make(chan struct{})
	parseTask := UseTask(func(parseCtx context.Context) (string, error) {
		select {
		case <-parseCtx.Done():
			return "", parseCtx.Err()
		case <-parseBlock:
			return "done", nil
		}
	})

	parseInitial := parseTask.Get()
	if parseInitial.Running || parseInitial.Ready || parseInitial.Cancelled || parseInitial.Started || parseInitial.Error != nil || parseInitial.Value != "" {
		parseT.Fatalf("unexpected initial task state: %+v", parseInitial)
	}

	parseTask.Start()
	parseRunning := parseTask.Get()
	if !parseRunning.Running || !parseRunning.Started || parseRunning.Cancelled {
		parseT.Fatalf("expected running task state after Start, got %+v", parseRunning)
	}

	parseTask.Cancel()
	parseCancelled := parseTask.Get()
	if parseCancelled.Running || !parseCancelled.Cancelled || !parseCancelled.Started {
		parseT.Fatalf("expected cancelled task state after Cancel, got %+v", parseCancelled)
	}

	close(parseBlock)
}

// TestUseTaskCtxParentCancellationPropagates proves UseTaskCtx derives the task context from the
// supplied parent: cancelling the PARENT (not Task.Cancel) cancels the running task's context.
func TestUseTaskCtxParentCancellationPropagates(parseT *testing.T) {
	installUIHookContext(parseT)

	parseParent, parseCancelParent := context.WithCancel(context.Background())
	parseObserved := make(chan error, 1)
	parseTask := UseTaskCtx(parseParent, func(parseCtx context.Context) (string, error) {
		<-parseCtx.Done()
		parseObserved <- parseCtx.Err()
		return "", parseCtx.Err()
	})

	parseTask.Start()
	parseCancelParent() // cancel via the PARENT, not Task.Cancel()

	select {
	case parseErr := <-parseObserved:
		if parseErr == nil {
			parseT.Fatal("task context should be cancelled when the parent is cancelled")
		}
	case <-time.After(2 * time.Second):
		parseT.Fatal("parent cancellation did not propagate into the task context")
	}
}

// TestUseTaskCtxNilParentDefaultsToBackground proves a nil parent is tolerated (falls back to
// Background) and the task runs to completion normally.
func TestUseTaskCtxNilParentDefaultsToBackground(parseT *testing.T) {
	installUIHookContext(parseT)

	//nolint:staticcheck // intentionally passing a nil parent context to exercise the fallback guard.
	parseTask := UseTaskCtx(nil, func(parseCtx context.Context) (string, error) {
		return "ok", nil
	})
	parseTask.Start()

	parseReady := false
	for range 500 {
		if parseTask.Get().Ready {
			parseReady = true
			break
		}
		time.Sleep(time.Millisecond)
	}
	if parseGot := parseTask.Get(); !parseReady || parseGot.Value != "ok" || parseGot.Error != nil {
		parseT.Fatalf("nil parent must default to Background and run normally, got %+v", parseGot)
	}
}

func installMockWorkerConstructor(parseT *testing.T, parseOnPost func(js.Value, js.Value)) func() {
	parseT.Helper()
	parseCtor := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseRaw := js.Global().Get("Object").New()
		parseMessageListeners := js.Global().Get("Array").New()
		parseErrorListeners := js.Global().Get("Array").New()
		parseEmitMessage := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
			parseEvent := js.Global().Get("Object").New()
			if len(parseArgs2) > 0 {
				parseEvent.Set("data", parseArgs2[0])
			}
			for parseI := 0; parseI < parseMessageListeners.Length(); parseI++ {
				parseCallback := parseMessageListeners.Index(parseI)
				if parseCallback.IsUndefined() || parseCallback.IsNull() {
					continue
				}
				parseCallback.Invoke(parseEvent)
			}
			return nil
		})
		parseAddEventListener := js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} {
			parseEventType := parseArgs3[0].String()
			parseCallback2 := parseArgs3[1]
			switch parseEventType {
			case "message":
				parseMessageListeners.Call("push", parseCallback2)
				parseReadySent := parseRaw.Get("__readySent")
				if !parseReadySent.Truthy() {
					parseRaw.Set("__readySent", true)
					parseReady := js.Global().Get("Object").New()
					parseReady.Set("phase", "ready")
					parseReady.Set("name", "bootstrap")
					parseRaw.Call("__emitMessage", parseReady)
				}
			case "error":
				parseErrorListeners.Call("push", parseCallback2)
			}
			return nil
		})
		parseRemoveEventListener := js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} {
			return nil
		})
		parsePostMessage := js.FuncOf(func(parseThis5 js.Value, parseArgs5 []js.Value) interface{} {
			if parseOnPost != nil && len(parseArgs5) > 0 {
				parseOnPost(parseRaw, parseArgs5[0])
			}
			return nil
		})
		parseTerminate := js.FuncOf(func(parseThis6 js.Value, parseArgs6 []js.Value) interface{} {
			parseRaw.Set("__terminated", true)
			return nil
		})
		parseRaw.Set("__emitMessage", parseEmitMessage)
		parseRaw.Set("addEventListener", parseAddEventListener)
		parseRaw.Set("removeEventListener", parseRemoveEventListener)
		parseRaw.Set("postMessage", parsePostMessage)
		parseRaw.Set("terminate", parseTerminate)
		return parseRaw
	})
	parsePrevWorker := js.Global().Get("Worker")
	js.Global().Set("Worker", parseCtor)
	return func() {
		js.Global().Set("Worker", parsePrevWorker)
		parseCtor.Release()
	}
}

func TestUseWorkerTaskReportsProgressAndResult(parseT *testing.T) {
	installUIHookContext(parseT)
	parseRestoreWorker := installMockWorkerConstructor(parseT, func(parseRaw js.Value, parsePayload js.Value) {
		parseRequestID := parsePayload.Get("id").String()
		parseName := parsePayload.Get("name").String()
		parseProgress := js.Global().Get("Object").New()
		parseProgress.Set("id", parseRequestID)
		parseProgress.Set("phase", "progress")
		parseProgress.Set("name", parseName)
		parseProgressPayload := js.Global().Get("Object").New()
		parseProgressPayload.Set("percent", 50)
		parseProgress.Set("payload", parseProgressPayload)
		parseRaw.Call("__emitMessage", parseProgress)

		parseResult := js.Global().Get("Object").New()
		parseResult.Set("id", parseRequestID)
		parseResult.Set("phase", "result")
		parseResult.Set("name", parseName)
		parseResultPayload := js.Global().Get("Object").New()
		parseResultPayload.Set("summary", "indexed 12 docs")
		parseResult.Set("payload", parseResultPayload)
		parseRaw.Call("__emitMessage", parseResult)
	})
	defer parseRestoreWorker()

	type progressPayload struct {
		Percent int `json:"percent"`
	}
	type resultPayload struct {
		Summary string `json:"summary"`
	}

	parseTask := UseWorkerTask[map[string]any, progressPayload, resultPayload](interop.WorkerOptions{URL: "/workers/search.mjs", Ready: true}, "build-index")
	parseTask.Start(map[string]any{"query": "atlas"})

	parseDeadline := time.Now().Add(250 * time.Millisecond)
	for {
		parseState := parseTask.Get()
		if parseState.Ready {
			if !parseState.ProgressReady || parseState.Progress.Percent != 50 {
				parseT.Fatalf("expected worker progress payload before completion, got %+v", parseState)
			}
			if parseState.Value.Summary != "indexed 12 docs" {
				parseT.Fatalf("unexpected worker result payload: %+v", parseState)
			}
			if parseState.Running || parseState.Cancelled || parseState.Error != nil {
				parseT.Fatalf("unexpected final worker task state: %+v", parseState)
			}
			break
		}
		if time.Now().After(parseDeadline) {
			parseT.Fatalf("timed out waiting for worker task completion, last state %+v", parseTask.Get())
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestUseWorkerTaskCancelMarksCancelled(parseT *testing.T) {
	installUIHookContext(parseT)
	parseBlock := make(chan struct{})
	parseRestoreWorker := installMockWorkerConstructor(parseT, func(parseRaw js.Value, parsePayload js.Value) {
		go func() {
			<-parseBlock
			parseResult := js.Global().Get("Object").New()
			parseResult.Set("id", parsePayload.Get("id").String())
			parseResult.Set("phase", "result")
			parseResult.Set("name", parsePayload.Get("name").String())
			parseResultPayload := js.Global().Get("Object").New()
			parseResultPayload.Set("summary", "late result")
			parseResult.Set("payload", parseResultPayload)
			parseRaw.Call("__emitMessage", parseResult)
		}()
	})
	defer parseRestoreWorker()

	type progressPayload struct {
		Percent int `json:"percent"`
	}
	type resultPayload struct {
		Summary string `json:"summary"`
	}

	parseTask := UseWorkerTask[map[string]any, progressPayload, resultPayload](interop.WorkerOptions{URL: "/workers/slow.js", Ready: true}, "slow-job")
	parseTask.Start(map[string]any{"query": "atlas"})
	parseTask.Cancel()

	parseState := parseTask.Get()
	if parseState.Running || !parseState.Cancelled || !parseState.Started {
		parseT.Fatalf("expected cancelled worker task state, got %+v", parseState)
	}
	close(parseBlock)
}

func TestUseWorkerTaskSerializesWorkerStartup(parseT *testing.T) {
	installUIHookContext(parseT)
	var parseCreated int
	parseReadyRelease := make(chan struct{})
	parseCtor := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseCreated++
		parseRaw := js.Global().Get("Object").New()
		parseMessageListeners := js.Global().Get("Array").New()
		parseEmitMessage := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
			parseEvent := js.Global().Get("Object").New()
			if len(parseArgs2) > 0 {
				parseEvent.Set("data", parseArgs2[0])
			}
			for parseI := 0; parseI < parseMessageListeners.Length(); parseI++ {
				parseCallback := parseMessageListeners.Index(parseI)
				if parseCallback.IsUndefined() || parseCallback.IsNull() {
					continue
				}
				parseCallback.Invoke(parseEvent)
			}
			return nil
		})
		parseAddEventListener := js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} {
			if parseArgs3[0].String() != "message" {
				return nil
			}
			parseMessageListeners.Call("push", parseArgs3[1])
			if parseRaw.Get("__readyPending").Truthy() {
				return nil
			}
			parseRaw.Set("__readyPending", true)
			go func(parseTarget js.Value) {
				<-parseReadyRelease
				parseReady := js.Global().Get("Object").New()
				parseReady.Set("phase", "ready")
				parseReady.Set("name", "bootstrap")
				parseTarget.Call("__emitMessage", parseReady)
			}(parseRaw)
			return nil
		})
		parseRemoveEventListener := js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} {
			return nil
		})
		parsePostMessage := js.FuncOf(func(parseThis5 js.Value, parseArgs5 []js.Value) interface{} {
			if len(parseArgs5) == 0 {
				return nil
			}
			parsePayload := parseArgs5[0]
			go func(parseTarget js.Value, parseRequest js.Value) {
				parseResult := js.Global().Get("Object").New()
				parseResult.Set("id", parseRequest.Get("id").String())
				parseResult.Set("phase", "result")
				parseResult.Set("name", parseRequest.Get("name").String())
				parseResultPayload := js.Global().Get("Object").New()
				parseResultPayload.Set("summary", "serialized startup")
				parseResult.Set("payload", parseResultPayload)
				parseTarget.Call("__emitMessage", parseResult)
			}(parseRaw, parsePayload)
			return nil
		})
		parseTerminate := js.FuncOf(func(parseThis6 js.Value, parseArgs6 []js.Value) interface{} {
			parseRaw.Set("__terminated", true)
			return nil
		})
		parseRaw.Set("__emitMessage", parseEmitMessage)
		parseRaw.Set("addEventListener", parseAddEventListener)
		parseRaw.Set("removeEventListener", parseRemoveEventListener)
		parseRaw.Set("postMessage", parsePostMessage)
		parseRaw.Set("terminate", parseTerminate)
		return parseRaw
	})
	defer parseCtor.Release()
	parsePrevWorker := js.Global().Get("Worker")
	js.Global().Set("Worker", parseCtor)
	defer js.Global().Set("Worker", parsePrevWorker)

	type progressPayload struct{}
	type resultPayload struct {
		Summary string `json:"summary"`
	}

	parseTask := UseWorkerTask[map[string]any, progressPayload, resultPayload](interop.WorkerOptions{URL: "/workers/slow-ready.js", Ready: true}, "slow-job")
	parseTask.Start(map[string]any{"query": "one"})
	parseTask.Start(map[string]any{"query": "two"})
	close(parseReadyRelease)

	parseDeadline := time.Now().Add(300 * time.Millisecond)
	for {
		parseState := parseTask.Get()
		if parseState.Ready {
			if parseCreated != 1 {
				parseT.Fatalf("expected one worker constructor call across overlapping starts, got %d", parseCreated)
			}
			if parseState.Value.Summary != "serialized startup" {
				parseT.Fatalf("unexpected worker result payload after serialized startup: %+v", parseState)
			}
			break
		}
		if time.Now().After(parseDeadline) {
			parseT.Fatalf("timed out waiting for serialized worker startup, last state %+v created=%d", parseTask.Get(), parseCreated)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// TestUseWorkerTaskCancelAfterCompletionNoOps keeps terminal success state
// intact when Cancel is called after the request has already completed.
func TestUseWorkerTaskCancelAfterCompletionNoOps(parseT *testing.T) {
	installUIHookContext(parseT)
	parseRestoreWorker := installMockWorkerConstructor(parseT, func(parseRaw js.Value, parsePayload js.Value) {
		parseResult := js.Global().Get("Object").New()
		parseResult.Set("id", parsePayload.Get("id").String())
		parseResult.Set("phase", "result")
		parseResult.Set("name", parsePayload.Get("name").String())
		parseResultPayload := js.Global().Get("Object").New()
		parseResultPayload.Set("summary", "finished")
		parseResult.Set("payload", parseResultPayload)
		parseRaw.Call("__emitMessage", parseResult)
	})
	defer parseRestoreWorker()

	type progressPayload struct{}
	type resultPayload struct {
		Summary string `json:"summary"`
	}

	parseTask := UseWorkerTask[map[string]any, progressPayload, resultPayload](interop.WorkerOptions{URL: "/workers/instant.js", Ready: true}, "instant-job")
	parseTask.Start(map[string]any{"query": "atlas"})

	parseDeadline := time.Now().Add(250 * time.Millisecond)
	for {
		parseState := parseTask.Get()
		if parseState.Ready {
			break
		}
		if time.Now().After(parseDeadline) {
			parseT.Fatalf("timed out waiting for completed worker task, last state %+v", parseTask.Get())
		}
		time.Sleep(5 * time.Millisecond)
	}

	parseTask.Cancel()
	parseState := parseTask.Get()
	if parseState.Running || !parseState.Ready || parseState.Cancelled || parseState.Error != nil {
		parseT.Fatalf("expected cancel after completion to preserve terminal success state, got %+v", parseState)
	}
	if parseState.Value.Summary != "finished" {
		parseT.Fatalf("expected completed worker result to stay intact after cancel, got %+v", parseState)
	}
}

func TestTaskHandleZeroValue(parseT *testing.T) {
	var parseTask Task[int]
	parseState := parseTask.Get()
	if parseState.Running || parseState.Ready || parseState.Cancelled || parseState.Started || parseState.Error != nil || parseState.Value != 0 {
		parseT.Fatalf("expected zero-value task state, got %+v", parseState)
	}
	parseTask.Start()
	parseTask.Cancel()
}

func TestWorkerTaskHandleZeroValue(parseT *testing.T) {
	var parseTask WorkerTask[map[string]any, struct{ Percent int }, struct{ Summary string }]
	parseState := parseTask.Get()
	if parseState.Running || parseState.Ready || parseState.Cancelled || parseState.Started || parseState.Error != nil || parseState.ProgressReady {
		parseT.Fatalf("expected zero-value worker task state, got %+v", parseState)
	}
	parseTask.Start(nil)
	parseTask.Cancel()
}

func TestReducerHandleZeroValue(parseT *testing.T) {
	var parseReducer Reducer[int, string]
	if parseReducer.Get() != 0 {
		parseT.Fatalf("expected zero-value reducer handle to return zero, got %d", parseReducer.Get())
	}
	parseReducer.Dispatch("noop")
}

func TestUseDebouncedInitialValue(parseT *testing.T) {
	installUIHookContext(parseT)

	parseDebounced := UseDebounced("hello", 20*time.Millisecond)
	if parseDebounced.Get() != "hello" {
		parseT.Fatalf("expected initial debounced value hello, got %q", parseDebounced.Get())
	}
	if parseDebounced.Pending() {
		parseT.Fatal("expected initial debounced handle to be settled")
	}
}

func TestDebouncedHandleZeroValue(parseT *testing.T) {
	var parseDebounced Debounced[string]
	if parseDebounced.Get() != "" {
		parseT.Fatalf("expected zero-value debounced handle to return empty string, got %q", parseDebounced.Get())
	}
	if parseDebounced.Pending() {
		parseT.Fatal("expected zero-value debounced handle not to be pending")
	}
}

func TestUseThrottledInitialValue(parseT *testing.T) {
	installUIHookContext(parseT)

	parseThrottled := UseThrottled(7, 20*time.Millisecond)
	if parseThrottled.Get() != 7 {
		parseT.Fatalf("expected initial throttled value 7, got %d", parseThrottled.Get())
	}
	if parseThrottled.Pending() {
		parseT.Fatal("expected initial throttled handle to be settled")
	}
}

func TestThrottledHandleZeroValue(parseT *testing.T) {
	var parseThrottled Throttled[int]
	if parseThrottled.Get() != 0 {
		parseT.Fatalf("expected zero-value throttled handle to return zero, got %d", parseThrottled.Get())
	}
	if parseThrottled.Pending() {
		parseT.Fatal("expected zero-value throttled handle not to be pending")
	}
}

func TestAsyncBoundaryReturnsContentWhenNotPending(parseT *testing.T) {
	installUIHookContext(parseT)

	parseContent := Text("ready")
	parseGot := AsyncBoundary(AsyncBoundaryProps{Content: parseContent})
	if parseGot == nil || parseGot.Type != runtime.AsyncBoundaryNodeType {
		parseT.Fatalf("expected async boundary marker, got %#v", parseGot)
	}
	if parseGot.Props["content"] != parseContent {
		parseT.Fatalf("expected boundary content prop to be preserved, got %#v", parseGot.Props["content"])
	}
}

func TestAsyncBoundaryReturnsFallbackAndErrorFallback(parseT *testing.T) {
	installUIHookContext(parseT)

	parseFallback := Text("loading")
	parseGot := AsyncBoundary(AsyncBoundaryProps{Pending: true, Fallback: parseFallback})
	if parseGot == nil || parseGot.Type != runtime.AsyncBoundaryNodeType {
		parseT.Fatalf("expected async boundary marker, got %#v", parseGot)
	}
	if parseGot.Props["pending"] != true || parseGot.Props["fallback"] != parseFallback {
		parseT.Fatalf("expected pending fallback props, got %#v", parseGot.Props)
	}

	parseErrorNode := Text("error")
	parseGot2 := AsyncBoundary(AsyncBoundaryProps{
		Error: errors.New("boom"),
		ErrorFallback: func(parseErr error) Node {
			if parseErr == nil || parseErr.Error() != "boom" {
				parseT.Fatalf("unexpected boundary error: %v", parseErr)
			}
			return parseErrorNode
		},
	})
	if parseGot2 == nil || parseGot2.Type != runtime.AsyncBoundaryNodeType {
		parseT.Fatalf("expected async boundary marker for error branch, got %#v", parseGot2)
	}
	parseFallbackFn, parseOk := parseGot2.Props["errorFallback"].(func(error) Node)
	if !parseOk || parseFallbackFn(errors.New("boom")) != parseErrorNode {
		parseT.Fatalf("expected async boundary to preserve error fallback, got %#v", parseGot2.Props["errorFallback"])
	}
}

func TestErrorBoundaryCreateElementPreservesFallbackProps(parseT *testing.T) {
	installUIHookContext(parseT)
	isParseCalled := false
	parseNode := CreateElement(ErrorBoundary, ErrorBoundaryProps{
		ErrorFallback: func(parseErr error, reset func()) Node {
			isParseCalled = true
			return Text("fallback")
		},
		Child:     Text("child"),
		ResetKeys: []interface{}{"route-a"},
	})
	if parseNode == nil {
		parseT.Fatal("expected error boundary element")
	}
	if _, parseOk := parseNode.Type.(*runtime.ErrorBoundaryType); !parseOk {
		parseT.Fatalf("expected runtime error boundary type, got %T", parseNode.Type)
	}
	if len(parseNode.Children) != 1 {
		parseT.Fatalf("expected one child under boundary, got %d", len(parseNode.Children))
	}
	resetKeys, _ := parseNode.Props["resetKeys"].([]interface{})
	if len(resetKeys) != 1 || resetKeys[0] != "route-a" {
		parseT.Fatalf("expected reset keys to be forwarded, got %#v", resetKeys)
	}
	parseFallback, _ := parseNode.Props["errorFallback"].(func(error, func()) Node)
	if parseFallback == nil {
		parseT.Fatal("expected runtime fallback callback to be preserved")
	}
	parseResult := parseFallback(errors.New("boom"), func() {})
	if !isParseCalled || parseResult == nil {
		parseT.Fatal("expected boundary fallback callback to remain callable")
	}
}

func TestErrorBoundaryCreateElementAcceptsMapPropsAliases(parseT *testing.T) {
	installUIHookContext(parseT)
	isParseOnErrorCalled := false
	parseChild := Text("child")
	parseSecondChild := Text("child-2")
	parseNode := CreateElement(ErrorBoundary, map[string]interface{}{
		"ErrorFallback": func(parseErr error, reset func()) Node {
			return Text("fallback")
		},
		"OnError": func(parseErr2 error) {
			isParseOnErrorCalled = parseErr2 != nil
		},
		"ResetKeys": []interface{}{"route-b"},
		"Child":     parseChild,
		"Children":  []Node{parseSecondChild},
	})
	if parseNode == nil {
		parseT.Fatal("expected error boundary element")
	}
	if len(parseNode.Children) != 2 {
		parseT.Fatalf("expected map props aliases to preserve two children, got %d", len(parseNode.Children))
	}
	if parseNode.Children[0] != parseChild || parseNode.Children[1] != parseSecondChild {
		parseT.Fatal("expected map props aliases to preserve child order")
	}
	resetKeys, _ := parseNode.Props["resetKeys"].([]interface{})
	if len(resetKeys) != 1 || resetKeys[0] != "route-b" {
		parseT.Fatalf("expected aliased reset keys to be forwarded, got %#v", resetKeys)
	}
	parseFallback, _ := parseNode.Props["errorFallback"].(func(error, func()) Node)
	if parseFallback == nil {
		parseT.Fatal("expected aliased error fallback to be forwarded")
	}
	parseOnError, _ := parseNode.Props["onError"].(func(error))
	if parseOnError == nil {
		parseT.Fatal("expected aliased onError to be forwarded")
	}
	parseOnError(errors.New("boom"))
	if !isParseOnErrorCalled {
		parseT.Fatal("expected forwarded onError callback to remain callable")
	}
	if parseResult := parseFallback(errors.New("boom"), func() {}); parseResult == nil {
		parseT.Fatal("expected forwarded fallback to remain callable")
	}
}

func TestUseLazyNodeInitialStateAndZeroValue(parseT *testing.T) {
	installUIHookContext(parseT)

	parseLazy := UseLazyNode(func(parseCtx context.Context) (Node, error) {
		return Text("resolved"), nil
	})
	parseState := parseLazy.Get()
	if !parseState.Loading || parseState.Ready || parseState.Error != nil || parseState.Node != nil {
		parseT.Fatalf("expected initial lazy state to be loading with no ready node, got %+v", parseState)
	}

	var parseZero LazyNode
	parseZeroState := parseZero.Get()
	if parseZeroState.Loading || parseZeroState.Ready || parseZeroState.Error != nil || parseZeroState.Node != nil {
		parseT.Fatalf("expected zero-value lazy handle to be inert, got %+v", parseZeroState)
	}
	parseZero.Reload()
	parseZero.Cancel()
}

func TestLazyRendersFallbackOnInitialLoad(parseT *testing.T) {
	installUIHookContext(parseT)

	parseFallback := Text("loading")
	parseGot := Lazy(LazyProps{
		Loader: func(parseCtx context.Context) (Node, error) {
			return Text("resolved"), nil
		},
		Fallback: parseFallback,
	})
	if parseGot == nil || parseGot.Type != runtime.AsyncBoundaryNodeType {
		parseT.Fatalf("expected lazy helper to return async boundary marker, got %#v", parseGot)
	}
	if parseGot.Props["pending"] != true || parseGot.Props["fallback"] != parseFallback {
		parseT.Fatalf("expected lazy helper to preserve pending fallback props, got %#v", parseGot.Props)
	}
}

type profileForm struct {
	Name   string
	Email  string
	OptIn  bool
	Region string
}

func TestUseFormTracksFieldStateAndValidation(parseT *testing.T) {
	installUIHookContext(parseT)

	parseForm := UseForm(profileForm{Region: "us"})
	if parseForm.TouchedAny() || parseForm.DirtyAny() || parseForm.HasErrors() {
		parseT.Fatal("expected fresh form state to be pristine and error-free")
	}
	if parseForm.Get().Region != "us" {
		parseT.Fatalf("expected initial form state, got %+v", parseForm.Get())
	}
	if !parseForm.SetField("Name", "Alice") {
		parseT.Fatal("expected SetField to update exported struct field")
	}
	if parseForm.Get().Name != "Alice" {
		parseT.Fatalf("expected updated field value, got %+v", parseForm.Get())
	}
	if !parseForm.Touched("Name") || !parseForm.Dirty("Name") {
		parseT.Fatal("expected SetField to mark field as touched and dirty")
	}
	if !parseForm.TouchedAny() || !parseForm.DirtyAny() {
		parseT.Fatal("expected aggregate touched/dirty helpers to reflect updated field state")
	}
	parseForm.SetErrors(FieldErrors{"Email": "required"})
	if parseForm.Error("Email") != "required" {
		parseT.Fatalf("expected field error to be readable, got %q", parseForm.Error("Email"))
	}
	parseForm.SetFormError("try again")
	if parseForm.FormError() != "try again" {
		parseT.Fatalf("expected form-level error to be readable, got %q", parseForm.FormError())
	}
	if !parseForm.HasErrors() {
		parseT.Fatal("expected aggregate error helper to report field/form errors")
	}
	parseStatus := parseForm.FieldStatus("Email")
	if parseStatus.Name != "Email" || parseStatus.Error != "required" || parseStatus.Pending {
		parseT.Fatalf("expected field status to reflect field error state, got %+v", parseStatus)
	}
	if !parseForm.HasFieldError("Email") || parseForm.FieldMessage("Email") != "required" {
		parseT.Fatal("expected field helpers to expose field error message")
	}
	parseValid := parseForm.Validate(func(parseState profileForm) FieldErrors {
		if parseState.Name == "" {
			return FieldErrors{"Name": "required"}
		}
		return nil
	})
	if !parseValid {
		parseT.Fatal("expected validation to pass after name was set")
	}
	if len(parseForm.Errors()) != 0 {
		parseT.Fatalf("expected successful validation to clear errors, got %#v", parseForm.Errors())
	}
	if parseForm.FormError() != "" {
		parseT.Fatalf("expected sync validation success to clear form error, got %q", parseForm.FormError())
	}
	if parseForm.HasErrors() {
		parseT.Fatal("expected successful sync validation to clear aggregate error state")
	}
	parseForm.Reset()
	if parseForm.Get().Name != "" || parseForm.Get().Region != "us" {
		parseT.Fatalf("expected Reset to restore initial form state, got %+v", parseForm.Get())
	}
	if parseForm.Touched("Name") || parseForm.Dirty("Name") {
		parseT.Fatal("expected Reset to clear touched and dirty state")
	}
	if parseForm.TouchedAny() || parseForm.DirtyAny() || parseForm.HasErrors() {
		parseT.Fatal("expected Reset to restore pristine and error-free aggregate state")
	}
}

func TestUseFormApplyServerErrorsAndCSRFTokens(parseT *testing.T) {
	installUIHookContext(parseT)

	parseForm := UseForm(profileForm{})
	parseValid := parseForm.ApplyServerErrors(ServerFormErrors{
		Message: "Fix the highlighted fields.",
		Fields:  FieldErrors{"Email": "already used"},
	})
	if parseValid {
		parseT.Fatal("expected structured server errors to report invalid form state")
	}
	if parseForm.FormError() != "Fix the highlighted fields." {
		parseT.Fatalf("expected structured message to become form error, got %q", parseForm.FormError())
	}
	if parseForm.Error("Email") != "already used" {
		parseT.Fatalf("expected structured field errors to map into form state, got %q", parseForm.Error("Email"))
	}
	parseClean := parseForm.ApplyServerErrors(ServerFormErrors{})
	if !parseClean || parseForm.FormError() != "" || len(parseForm.Errors()) != 0 {
		parseT.Fatalf("expected empty structured server response to clear form errors, clean=%t formError=%q errors=%#v", parseClean, parseForm.FormError(), parseForm.Errors())
	}

	parseResult := ServerActionResult{
		Outcome: ServerActionOutcomeValidationError,
		Message: "Correct the highlighted fields.",
		Fields:  FieldErrors{"Name": "required"},
		Redirect: &ServerActionRedirect{
			Location: "/account/profile",
			Replace:  true,
		},
		Flash: &ServerActionFlash{
			Kind:    "error",
			Message: "Profile could not be saved.",
		},
		Refresh: &ServerActionRefresh{
			Revalidate: true,
			CacheKeys:  []string{"profile", "session"},
		},
	}
	if !parseResult.HasRedirect() || parseResult.RedirectLocation() != "/account/profile" {
		parseT.Fatalf("expected redirect metadata to stay available, got %+v", parseResult.Redirect)
	}
	if !parseResult.HasRefresh() {
		parseT.Fatalf("expected refresh metadata to be reported")
	}
	parseProjected := parseResult.FormErrors()
	if parseProjected.FormMessage() != "Correct the highlighted fields." || parseProjected.Fields["Name"] != "required" {
		parseT.Fatalf("expected typed action result to map to form errors, got %+v", parseProjected)
	}
	if parseValid2 := parseForm.ApplyServerActionResult(parseResult); parseValid2 {
		parseT.Fatal("expected action result with field errors to keep form invalid")
	}
	if parseForm.Error("Name") != "required" || parseForm.FormError() != "Correct the highlighted fields." {
		parseT.Fatalf("expected action result projection to reuse form error surface, formError=%q errors=%#v", parseForm.FormError(), parseForm.Errors())
	}

	parseToken := NewCSRFToken("token-123")
	parseHeaderName, parseHeaderValue := parseToken.Header()
	if parseHeaderName != DefaultCSRFHeaderName || parseHeaderValue != "token-123" {
		parseT.Fatalf("expected default CSRF header helper, got %q=%q", parseHeaderName, parseHeaderValue)
	}
	parseFieldName, parseFieldValue := parseToken.FormField()
	if parseFieldName != DefaultCSRFFormFieldName || parseFieldValue != "token-123" {
		parseT.Fatalf("expected default CSRF form field helper, got %q=%q", parseFieldName, parseFieldValue)
	}
	parseCustom := CSRFToken{Value: "abc", HeaderName: "X-Demo-CSRF", FormFieldName: "demo_csrf"}
	parseCustomHeader, parseCustomHeaderValue := parseCustom.Header()
	parseCustomField, parseCustomFieldValue := parseCustom.FormField()
	if parseCustomHeader != "X-Demo-CSRF" || parseCustomHeaderValue != "abc" || parseCustomField != "demo_csrf" || parseCustomFieldValue != "abc" {
		parseT.Fatalf("expected custom CSRF naming to be preserved, got header=%q value=%q field=%q fieldValue=%q", parseCustomHeader, parseCustomHeaderValue, parseCustomField, parseCustomFieldValue)
	}
}

func TestUseFormSubmitIntentHelpers(parseT *testing.T) {
	installUIHookContext(parseT)

	parseForm := UseForm(profileForm{Name: "Alice"})
	parseValid := parseForm.ValidateIntent("publish", func(parseValue profileForm, parseIntent string) FieldErrors {
		if parseIntent == "publish" && parseValue.Name == "" {
			return FieldErrors{"Name": "required"}
		}
		return nil
	})
	if !parseValid || parseForm.SubmitIntent() != "publish" {
		parseT.Fatalf("expected intent-aware validation to record publish intent, valid=%t intent=%q", parseValid, parseForm.SubmitIntent())
	}

	parseBlock := make(chan struct{})
	parseForm.SubmitWithIntent("draft", func(parseValue2 profileForm, parseIntent2 string) error {
		if parseValue2.Name != "Alice" || parseIntent2 != "draft" {
			return errors.New("unexpected submit intent snapshot")
		}
		<-parseBlock
		return nil
	})
	if !parseForm.Submitting() || !parseForm.IntentPending("draft") || parseForm.IntentPending("publish") {
		parseT.Fatalf("expected per-intent pending state, submitting=%t draft=%t publish=%t", parseForm.Submitting(), parseForm.IntentPending("draft"), parseForm.IntentPending("publish"))
	}
	close(parseBlock)
	time.Sleep(20 * time.Millisecond)
	if parseForm.Submitting() || !parseForm.Submitted() || parseForm.SubmitIntent() != "draft" {
		parseT.Fatalf("expected successful draft submit lifecycle, submitting=%t submitted=%t intent=%q", parseForm.Submitting(), parseForm.Submitted(), parseForm.SubmitIntent())
	}

	parseForm.Reset()
	if parseForm.SubmitIntent() != "" {
		parseT.Fatalf("expected reset to clear submit intent, got %q", parseForm.SubmitIntent())
	}
}

func TestExtractFilesReturnsWrappedBrowserFiles(parseT *testing.T) {
	parseFileOne := js.Global().Get("Object").New()
	parseFileOne.Set("name", "photo.png")
	parseFileOne.Set("type", "image/png")
	parseFileOne.Set("size", 1234)
	parseFileOne.Set("lastModified", 99)
	parseFileTwo := js.Global().Get("Object").New()
	parseFileTwo.Set("name", "notes.txt")
	parseFileTwo.Set("type", "text/plain")
	parseFileTwo.Set("size", 55)
	parseFileTwo.Set("lastModified", 101)

	parseFiles := js.Global().Get("Object").New()
	parseFiles.Set("length", 2)
	parseFiles.Set("0", parseFileOne)
	parseFiles.Set("1", parseFileTwo)
	parseTarget := js.Global().Get("Object").New()
	parseTarget.Set("files", parseFiles)
	parseJsEvent := js.Global().Get("Object").New()
	parseJsEvent.Set("target", parseTarget)

	parseResult := GetFiles(runtime.NewGoEvent(parseJsEvent))
	if len(parseResult) != 2 {
		parseT.Fatalf("expected two extracted files, got %d", len(parseResult))
	}
	if parseResult[0].Name() != "photo.png" || parseResult[0].Type() != "image/png" || parseResult[0].Size() != 1234 || parseResult[0].LastModified() != 99 {
		parseT.Fatalf("unexpected first extracted file: name=%q type=%q size=%d lastModified=%d", parseResult[0].Name(), parseResult[0].Type(), parseResult[0].Size(), parseResult[0].LastModified())
	}
	if parseResult[1].Name() != "notes.txt" || parseResult[1].Type() != "text/plain" || parseResult[1].Size() != 55 || parseResult[1].LastModified() != 101 {
		parseT.Fatalf("unexpected second extracted file: name=%q type=%q size=%d lastModified=%d", parseResult[1].Name(), parseResult[1].Type(), parseResult[1].Size(), parseResult[1].LastModified())
	}
}

func TestUseFormAsyncValidationLifecycle(parseT *testing.T) {
	installUIHookContext(parseT)

	parseForm := UseForm(profileForm{Name: "admin", Email: "alice@blocked.test"})
	parseForm.ValidateAsync(func(parseValue profileForm) (FieldErrors, string) {
		time.Sleep(20 * time.Millisecond)
		parseErrs := FieldErrors{}
		if parseValue.Name == "admin" {
			parseErrs["Name"] = "reserved"
		}
		return parseErrs, "blocked domain"
	}, nil)
	if !parseForm.Validating() {
		parseT.Fatal("expected form to report validating immediately after ValidateAsync")
	}
	time.Sleep(40 * time.Millisecond)
	if parseForm.Validating() || !parseForm.Validated() {
		parseT.Fatalf("expected async validation to settle, validating=%t validated=%t", parseForm.Validating(), parseForm.Validated())
	}
	if parseForm.Error("Name") != "reserved" {
		parseT.Fatalf("expected async field error, got %q", parseForm.Error("Name"))
	}
	if parseForm.FormError() != "blocked domain" {
		parseT.Fatalf("expected async form error, got %q", parseForm.FormError())
	}

	isParseCompleted := false
	parseForm.SetField("Name", "alice")
	parseForm.SetField("Email", "alice@example.com")
	parseForm.ValidateAsync(func(parseValue2 profileForm) (FieldErrors, string) {
		return nil, ""
	}, func(isValid bool) {
		isParseCompleted = isValid
	})
	time.Sleep(20 * time.Millisecond)
	if !isParseCompleted {
		parseT.Fatal("expected async validation callback to report valid state")
	}
	if parseForm.FormError() != "" || len(parseForm.Errors()) != 0 {
		parseT.Fatalf("expected async validation success to clear errors, formError=%q errors=%#v", parseForm.FormError(), parseForm.Errors())
	}
}

func TestUseFormSubmissionLifecycle(parseT *testing.T) {
	installUIHookContext(parseT)

	parseForm := UseForm(profileForm{Name: "Alice"})
	parseBlock := make(chan struct{})
	parseForm.Submit(func(parseValue profileForm) error {
		if parseValue.Name != "Alice" {
			return errors.New("unexpected form snapshot")
		}
		<-parseBlock
		return nil
	})
	if !parseForm.Submitting() {
		parseT.Fatal("expected form to report submitting immediately after Submit")
	}
	close(parseBlock)
	time.Sleep(20 * time.Millisecond)
	if parseForm.Submitting() || !parseForm.Submitted() || parseForm.SubmitError() != nil {
		parseT.Fatalf("expected successful submit lifecycle, submitted=%t submitting=%t err=%v", parseForm.Submitted(), parseForm.Submitting(), parseForm.SubmitError())
	}

	parseForm.Submit(func(parseValue2 profileForm) error {
		return errors.New("server unavailable")
	})
	time.Sleep(20 * time.Millisecond)
	if parseForm.SubmitError() == nil || parseForm.Submitted() {
		parseT.Fatalf("expected failed submit to store error and clear submitted flag, submitted=%t err=%v", parseForm.Submitted(), parseForm.SubmitError())
	}
	if parseForm.FormError() != "server unavailable" {
		parseT.Fatalf("expected failed submit to surface form error, got %q", parseForm.FormError())
	}
	if !parseForm.HasErrors() {
		parseT.Fatal("expected failed submit to mark aggregate error state")
	}
	parseForm.Reset(profileForm{Region: "eu"})
	if parseForm.Get().Region != "eu" || parseForm.Submitted() || parseForm.SubmitError() != nil {
		parseT.Fatalf("expected reset with new initial value to clear submission state, got %+v err=%v", parseForm.Get(), parseForm.SubmitError())
	}
	if parseForm.FormError() != "" || parseForm.HasErrors() {
		parseT.Fatal("expected reset to clear submit-derived form error state")
	}
}

func TestFormZeroValue(parseT *testing.T) {
	var parseForm Form[profileForm]
	if parseForm.Get().Name != "" {
		parseT.Fatalf("expected zero-value form to return zero form state, got %+v", parseForm.Get())
	}
	if parseForm.Touched("Name") || parseForm.Dirty("Name") || parseForm.TouchedAny() || parseForm.DirtyAny() || parseForm.HasErrors() || parseForm.Validating() || parseForm.Validated() || parseForm.Submitting() || parseForm.Submitted() || parseForm.SubmitError() != nil || parseForm.FormError() != "" {
		parseT.Fatal("expected zero-value form helpers to be inert")
	}
	parseForm.SetField("Name", "ignored")
	parseForm.SetErrors(FieldErrors{"Name": "required"})
	parseForm.SetFormError("ignored")
	parseForm.ValidateAsync(nil, nil)
	parseForm.Reset()
}
