//go:build js && wasm
// +build js,wasm

package ui

import (
	"context"
	"errors"
	"syscall/js"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

func installMockFetchResolvedBytes(t *testing.T, payload []byte) {
	t.Helper()
	global := js.Global()
	objectCtor := global.Get("Object")
	uint8ArrayCtor := global.Get("Uint8Array")
	prevFetch := global.Get("fetch")

	array := uint8ArrayCtor.New(len(payload))
	js.CopyBytesToJS(array, payload)
	buffer := array.Get("buffer")

	response := objectCtor.New()
	response.Set("ok", true)
	response.Set("status", 200)
	arrayBufferFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return buffer
	})
	response.Set("arrayBuffer", arrayBufferFn)

	promise := objectCtor.New()
	thenFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) == 0 {
			return this
		}
		current := this.Get("__current")
		next := args[0].Invoke(current)
		this.Set("__current", next)
		return this
	})
	catchFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return this
	})
	promise.Set("__current", response)
	promise.Set("then", thenFn)
	promise.Set("catch", catchFn)

	fetchFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return promise
	})
	global.Set("fetch", fetchFn)

	t.Cleanup(func() {
		global.Set("fetch", prevFetch)
		fetchFn.Release()
		thenFn.Release()
		catchFn.Release()
		arrayBufferFn.Release()
	})
}

type noOpScheduler struct{}

func (noOpScheduler) RequestIdleCallback(callback func(runtime.Deadline)) {}

func (noOpScheduler) SetTimeout(callback func(), delay int) {}

type queuedScheduler struct {
	timeouts []func()
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

func installUIHookContext(t *testing.T) {
	t.Helper()
	runtime.InitGlobalRuntime(runtime.Config{Scheduler: noOpScheduler{}})
	runtime.SetCurrentFiber(&runtime.Fiber{})
	t.Cleanup(func() {
		runtime.SetCurrentFiber(nil)
	})
}

func installQueuedUIHookContext(t *testing.T) *queuedScheduler {
	t.Helper()
	scheduler := &queuedScheduler{}
	runtime.InitGlobalRuntime(runtime.Config{Scheduler: scheduler})
	runtime.SetCurrentFiber(&runtime.Fiber{})
	t.Cleanup(func() {
		runtime.SetCurrentFiber(nil)
	})
	return scheduler
}

func TestCreateElementReturnsExistingNode(t *testing.T) {
	existing := runtime.Div(map[string]interface{}{"id": "existing"})
	if got := CreateElement(existing); got != existing {
		t.Fatal("expected CreateElement to return existing node unchanged")
	}
}

func TestCreateElementAcceptsComponentFunctions(t *testing.T) {
	type props struct {
		Label string
	}

	withoutProps := func() Node {
		return Text("plain")
	}
	withProps := func(input props) Node {
		return Text(input.Label)
	}

	if node := CreateElement(withoutProps); node == nil {
		t.Fatal("expected zero-argument component to produce a node")
	}
	if node := CreateElement(withProps, props{Label: "hello"}); node == nil {
		t.Fatal("expected props component to produce a node")
	}
}

func TestFragmentAndTextHelpers(t *testing.T) {
	first := Text("first")
	second := Text("second")

	fragment := Fragment(first, second)
	if fragment == nil {
		t.Fatal("expected fragment")
	}
	if fragment.Type != "FRAGMENT" {
		t.Fatalf("expected fragment type, got %#v", fragment.Type)
	}
	if len(fragment.Children) != 2 {
		t.Fatalf("expected two fragment children, got %d", len(fragment.Children))
	}
	if first.TextContent != "first" || second.TextContent != "second" {
		t.Fatal("expected text helper to preserve text content")
	}
}

func TestReadBootstrapScript(t *testing.T) {
	global := js.Global()
	objectCtor := global.Get("Object")
	prevDoc := global.Get("document")

	script := objectCtor.New()
	script.Set("textContent", `{"route":{"path":"/docs"},"atoms":{"theme":"dark"}}`)

	doc := objectCtor.New()
	getElementByID := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) > 0 && args[0].String() == DefaultBootstrapScriptID {
			return script
		}
		return js.Null()
	})
	doc.Set("getElementById", getElementByID)
	global.Set("document", doc)

	t.Cleanup(func() {
		global.Set("document", prevDoc)
		getElementByID.Release()
	})

	payload, err := ReadBootstrapScript("")
	if err != nil {
		t.Fatalf("unexpected bootstrap read error: %v", err)
	}
	if payload.Route.Path != "/docs" {
		t.Fatalf("expected route path /docs, got %q", payload.Route.Path)
	}
	if payload.Atoms["theme"] != "dark" {
		t.Fatalf("expected theme atom to round-trip, got %#v", payload.Atoms["theme"])
	}
}

func TestReadBootstrapReferenceScript(t *testing.T) {
	global := js.Global()
	objectCtor := global.Get("Object")
	prevDoc := global.Get("document")

	script := objectCtor.New()
	script.Set("textContent", `{"url":"/bootstrap.cbor","format":"cbor"}`)

	doc := objectCtor.New()
	getElementByID := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) > 0 && args[0].String() == DefaultBootstrapReferenceScriptID {
			return script
		}
		return js.Null()
	})
	doc.Set("getElementById", getElementByID)
	global.Set("document", doc)

	t.Cleanup(func() {
		global.Set("document", prevDoc)
		getElementByID.Release()
	})

	ref, err := ReadBootstrapReferenceScript("")
	if err != nil {
		t.Fatalf("unexpected bootstrap reference read error: %v", err)
	}
	if ref.URL != "/bootstrap.cbor" {
		t.Fatalf("expected reference url /bootstrap.cbor, got %q", ref.URL)
	}
	if ref.Format != SSRBootstrapFormatCBOR {
		t.Fatalf("expected bootstrap reference format %q, got %q", SSRBootstrapFormatCBOR, ref.Format)
	}
}

func TestReadBootstrapReferenceJSON(t *testing.T) {
	payloadBytes := []byte(`{"route":{"path":"/json"},"atoms":{"theme":"light"}}`)
	installMockFetchResolvedBytes(t, payloadBytes)

	payload, err := ReadBootstrapReference(SSRBootstrapReference{URL: "/bootstrap.json", Format: SSRBootstrapFormatJSON})
	if err != nil {
		t.Fatalf("unexpected JSON bootstrap reference read error: %v", err)
	}
	if payload.Route.Path != "/json" {
		t.Fatalf("expected JSON bootstrap path /json, got %q", payload.Route.Path)
	}
	if payload.Atoms["theme"] != "light" {
		t.Fatalf("expected JSON bootstrap atom to round-trip, got %#v", payload.Atoms["theme"])
	}
}

func TestReadBootstrapReferenceCBOR(t *testing.T) {
	encoded, err := MarshalSSRBootstrapBinary(SSRBootstrap{
		Route: SSRRouteBootstrap{Path: "/cbor"},
		Atoms: map[string]interface{}{"theme": "dark"},
	})
	if err != nil {
		t.Fatalf("unexpected binary bootstrap marshal error: %v", err)
	}
	installMockFetchResolvedBytes(t, encoded)

	payload, err := ReadBootstrapReference(SSRBootstrapReference{URL: "/bootstrap.cbor", Format: SSRBootstrapFormatCBOR})
	if err != nil {
		t.Fatalf("unexpected CBOR bootstrap reference read error: %v", err)
	}
	if payload.Route.Path != "/cbor" {
		t.Fatalf("expected CBOR bootstrap path /cbor, got %q", payload.Route.Path)
	}
	if payload.Atoms["theme"] != "dark" {
		t.Fatalf("expected CBOR bootstrap atom to round-trip, got %#v", payload.Atoms["theme"])
	}
}

func TestPublicHooksWrappers(t *testing.T) {
	installUIHookContext(t)

	state := UseState(1)
	if state.Get() != 1 {
		t.Fatalf("expected initial state, got %d", state.Get())
	}
	state.Set(3)
	if state.Get() != 3 {
		t.Fatalf("expected updated state, got %d", state.Get())
	}
	state.Update(func(prev int) int { return prev + 4 })
	if state.Get() != 7 {
		t.Fatalf("expected updated state after updater, got %d", state.Get())
	}

	reducer := UseReducer(func(state int, action int) int { return state + action }, 2)
	if reducer.Get() != 2 {
		t.Fatalf("expected initial reducer state, got %d", reducer.Get())
	}
	reducer.Dispatch(5)
	if reducer.Get() != 7 {
		t.Fatalf("expected reducer dispatch to update state, got %d", reducer.Get())
	}

	computed := UseMemo(func() int { return 9 }, "dep")
	if computed != 9 {
		t.Fatalf("expected memoized value 9, got %d", computed)
	}

	callback := UseCallback(func() int { return 11 }, "dep")
	if callback() != 11 {
		t.Fatalf("expected callback wrapper to preserve function value")
	}

	ref := UseRef("start")
	if ref.Get() != "start" {
		t.Fatalf("expected initial ref value, got %q", ref.Get())
	}
	ref.Set("done")
	if ref.Get() != "done" {
		t.Fatalf("expected updated ref value, got %q", ref.Get())
	}

	id := UseId()
	if id == "" {
		t.Fatal("expected non-empty id")
	}

	deferred := UseDeferredValue("steady")
	if deferred != "steady" {
		t.Fatalf("expected deferred value to return initial value, got %q", deferred)
	}
}

func TestUseTransitionDefersPublicStateUpdates(t *testing.T) {
	scheduler := installQueuedUIHookContext(t)

	state := UseState(1)
	transition := UseTransition()
	transition.Start(func() {
		state.Set(6)
	})

	if got := state.Get(); got != 1 {
		t.Fatalf("expected transition update to remain deferred before flush, got %d", got)
	}
	if !transition.Pending() {
		t.Fatal("expected transition to report pending before flush")
	}

	scheduler.Flush()

	if got := state.Get(); got != 6 {
		t.Fatalf("expected deferred transition state update after flush, got %d", got)
	}
	if transition.Pending() {
		t.Fatal("expected transition to report settled after flush")
	}
}

func TestUseContextFallsBackToDefaultValue(t *testing.T) {
	installUIHookContext(t)

	theme := CreateContext("light")
	if got := UseContext(theme); got != "light" {
		t.Fatalf("expected default context value light, got %q", got)
	}
}

func TestCreateElementSupportsContextProvider(t *testing.T) {
	theme := CreateContext("light")
	child := Text("ready")
	node := CreateElement(theme.Provider, ContextProviderProps[string]{
		Value: "dark",
		Child: child,
	})
	if node == nil {
		t.Fatal("expected provider element to be created")
	}
	if _, ok := node.Type.(*runtime.ContextProviderType); !ok {
		t.Fatalf("expected provider element type, got %T", node.Type)
	}
	if len(node.Children) != 1 || node.Children[0] != child {
		t.Fatalf("expected provider child to be preserved, got %#v", node.Children)
	}
	if got := node.Props["value"]; got != "dark" {
		t.Fatalf("expected provider value dark, got %#v", got)
	}
}

func TestRefAndHandlerHelpers(t *testing.T) {
	var empty Ref[int]
	if empty.Get() != 0 {
		t.Fatalf("expected zero value from nil ref, got %d", empty.Get())
	}
	empty.Set(42)
	if empty.Get() != 0 {
		t.Fatal("expected nil ref Set to remain a no-op")
	}

	handler := RawHandler("wrapped")
	if handler.Value() != "wrapped" {
		t.Fatalf("expected raw handler value, got %#v", handler.Value())
	}
}

func TestUsePreviousReturnsEmptyValueOnFirstRender(t *testing.T) {
	installUIHookContext(t)

	previous := UsePrevious("current")
	if previous.Ok() {
		t.Fatal("expected previous value to be unavailable on first render")
	}
	if previous.Get() != "" {
		t.Fatalf("expected zero value on first render, got %q", previous.Get())
	}
}

func TestPreviousHandleZeroValue(t *testing.T) {
	var previous Previous[int]
	if previous.Ok() {
		t.Fatal("expected zero-value previous handle to report unavailable")
	}
	if previous.Get() != 0 {
		t.Fatalf("expected zero-value previous handle to return zero, got %d", previous.Get())
	}
}

func TestUseChannelReturnsEmptyStateBeforeValues(t *testing.T) {
	installUIHookContext(t)

	ch := make(chan int)
	channel := UseChannel(ch)
	if channel.Ok() {
		t.Fatal("expected channel handle to report no value before any receive")
	}
	if channel.Closed() {
		t.Fatal("expected channel handle to report open before closure is observed")
	}
	if channel.Get() != 0 {
		t.Fatalf("expected zero value before any receive, got %d", channel.Get())
	}
}

func TestChannelHandleZeroValue(t *testing.T) {
	var channel Channel[string]
	if channel.Ok() {
		t.Fatal("expected zero-value channel handle to report unavailable")
	}
	if channel.Closed() {
		t.Fatal("expected zero-value channel handle to report open")
	}
	if channel.Get() != "" {
		t.Fatalf("expected zero-value channel handle to return empty string, got %q", channel.Get())
	}
}

func TestUseTaskTransitionsToRunningAndCancelled(t *testing.T) {
	installUIHookContext(t)

	block := make(chan struct{})
	task := UseTask(func(ctx context.Context) (string, error) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-block:
			return "done", nil
		}
	})

	initial := task.Get()
	if initial.Running || initial.Ready || initial.Cancelled || initial.Started || initial.Error != nil || initial.Value != "" {
		t.Fatalf("unexpected initial task state: %+v", initial)
	}

	task.Start()
	running := task.Get()
	if !running.Running || !running.Started || running.Cancelled {
		t.Fatalf("expected running task state after Start, got %+v", running)
	}

	task.Cancel()
	cancelled := task.Get()
	if cancelled.Running || !cancelled.Cancelled || !cancelled.Started {
		t.Fatalf("expected cancelled task state after Cancel, got %+v", cancelled)
	}

	close(block)
}

func TestTaskHandleZeroValue(t *testing.T) {
	var task Task[int]
	state := task.Get()
	if state.Running || state.Ready || state.Cancelled || state.Started || state.Error != nil || state.Value != 0 {
		t.Fatalf("expected zero-value task state, got %+v", state)
	}
	task.Start()
	task.Cancel()
}

func TestReducerHandleZeroValue(t *testing.T) {
	var reducer Reducer[int, string]
	if reducer.Get() != 0 {
		t.Fatalf("expected zero-value reducer handle to return zero, got %d", reducer.Get())
	}
	reducer.Dispatch("noop")
}

func TestUseDebouncedInitialValue(t *testing.T) {
	installUIHookContext(t)

	debounced := UseDebounced("hello", 20*time.Millisecond)
	if debounced.Get() != "hello" {
		t.Fatalf("expected initial debounced value hello, got %q", debounced.Get())
	}
	if debounced.Pending() {
		t.Fatal("expected initial debounced handle to be settled")
	}
}

func TestDebouncedHandleZeroValue(t *testing.T) {
	var debounced Debounced[string]
	if debounced.Get() != "" {
		t.Fatalf("expected zero-value debounced handle to return empty string, got %q", debounced.Get())
	}
	if debounced.Pending() {
		t.Fatal("expected zero-value debounced handle not to be pending")
	}
}

func TestUseThrottledInitialValue(t *testing.T) {
	installUIHookContext(t)

	throttled := UseThrottled(7, 20*time.Millisecond)
	if throttled.Get() != 7 {
		t.Fatalf("expected initial throttled value 7, got %d", throttled.Get())
	}
	if throttled.Pending() {
		t.Fatal("expected initial throttled handle to be settled")
	}
}

func TestThrottledHandleZeroValue(t *testing.T) {
	var throttled Throttled[int]
	if throttled.Get() != 0 {
		t.Fatalf("expected zero-value throttled handle to return zero, got %d", throttled.Get())
	}
	if throttled.Pending() {
		t.Fatal("expected zero-value throttled handle not to be pending")
	}
}

func TestAsyncBoundaryReturnsContentWhenNotPending(t *testing.T) {
	installUIHookContext(t)

	content := Text("ready")
	if got := AsyncBoundary(AsyncBoundaryProps{Content: content}); got != content {
		t.Fatal("expected async boundary to render content when not pending")
	}
}

func TestAsyncBoundaryReturnsFallbackAndErrorFallback(t *testing.T) {
	installUIHookContext(t)

	fallback := Text("loading")
	if got := AsyncBoundary(AsyncBoundaryProps{Pending: true, Fallback: fallback}); got != fallback {
		t.Fatal("expected async boundary to render fallback while pending")
	}

	errorNode := Text("error")
	got := AsyncBoundary(AsyncBoundaryProps{
		Error: errors.New("boom"),
		ErrorFallback: func(err error) Node {
			if err == nil || err.Error() != "boom" {
				t.Fatalf("unexpected boundary error: %v", err)
			}
			return errorNode
		},
	})
	if got != errorNode {
		t.Fatal("expected async boundary to render error fallback")
	}
}

func TestErrorBoundaryCreateElementPreservesFallbackProps(t *testing.T) {
	installUIHookContext(t)
	called := false
	node := CreateElement(ErrorBoundary, ErrorBoundaryProps{
		ErrorFallback: func(err error, reset func()) Node {
			called = true
			return Text("fallback")
		},
		Child:     Text("child"),
		ResetKeys: []interface{}{"route-a"},
	})
	if node == nil {
		t.Fatal("expected error boundary element")
	}
	if _, ok := node.Type.(*runtime.ErrorBoundaryType); !ok {
		t.Fatalf("expected runtime error boundary type, got %T", node.Type)
	}
	if len(node.Children) != 1 {
		t.Fatalf("expected one child under boundary, got %d", len(node.Children))
	}
	resetKeys, _ := node.Props["resetKeys"].([]interface{})
	if len(resetKeys) != 1 || resetKeys[0] != "route-a" {
		t.Fatalf("expected reset keys to be forwarded, got %#v", resetKeys)
	}
	fallback, _ := node.Props["errorFallback"].(func(error, func()) Node)
	if fallback == nil {
		t.Fatal("expected runtime fallback callback to be preserved")
	}
	result := fallback(errors.New("boom"), func() {})
	if !called || result == nil {
		t.Fatal("expected boundary fallback callback to remain callable")
	}
}

func TestErrorBoundaryCreateElementAcceptsMapPropsAliases(t *testing.T) {
	installUIHookContext(t)
	onErrorCalled := false
	child := Text("child")
	secondChild := Text("child-2")
	node := CreateElement(ErrorBoundary, map[string]interface{}{
		"ErrorFallback": func(err error, reset func()) Node {
			return Text("fallback")
		},
		"OnError": func(err error) {
			onErrorCalled = err != nil
		},
		"ResetKeys": []interface{}{"route-b"},
		"Child":     child,
		"Children":  []Node{secondChild},
	})
	if node == nil {
		t.Fatal("expected error boundary element")
	}
	if len(node.Children) != 2 {
		t.Fatalf("expected map props aliases to preserve two children, got %d", len(node.Children))
	}
	if node.Children[0] != child || node.Children[1] != secondChild {
		t.Fatal("expected map props aliases to preserve child order")
	}
	resetKeys, _ := node.Props["resetKeys"].([]interface{})
	if len(resetKeys) != 1 || resetKeys[0] != "route-b" {
		t.Fatalf("expected aliased reset keys to be forwarded, got %#v", resetKeys)
	}
	fallback, _ := node.Props["errorFallback"].(func(error, func()) Node)
	if fallback == nil {
		t.Fatal("expected aliased error fallback to be forwarded")
	}
	onError, _ := node.Props["onError"].(func(error))
	if onError == nil {
		t.Fatal("expected aliased onError to be forwarded")
	}
	onError(errors.New("boom"))
	if !onErrorCalled {
		t.Fatal("expected forwarded onError callback to remain callable")
	}
	if result := fallback(errors.New("boom"), func() {}); result == nil {
		t.Fatal("expected forwarded fallback to remain callable")
	}
}

func TestUseLazyNodeInitialStateAndZeroValue(t *testing.T) {
	installUIHookContext(t)

	lazy := UseLazyNode(func(ctx context.Context) (Node, error) {
		return Text("resolved"), nil
	})
	state := lazy.Get()
	if !state.Loading || state.Ready || state.Error != nil || state.Node != nil {
		t.Fatalf("expected initial lazy state to be loading with no ready node, got %+v", state)
	}

	var zero LazyNode
	zeroState := zero.Get()
	if zeroState.Loading || zeroState.Ready || zeroState.Error != nil || zeroState.Node != nil {
		t.Fatalf("expected zero-value lazy handle to be inert, got %+v", zeroState)
	}
	zero.Reload()
	zero.Cancel()
}

func TestLazyRendersFallbackOnInitialLoad(t *testing.T) {
	installUIHookContext(t)

	fallback := Text("loading")
	got := Lazy(LazyProps{
		Loader: func(ctx context.Context) (Node, error) {
			return Text("resolved"), nil
		},
		Fallback: fallback,
	})
	if got != fallback {
		t.Fatal("expected lazy helper to render fallback on initial load")
	}
}

type profileForm struct {
	Name   string
	Email  string
	OptIn  bool
	Region string
}

func TestUseFormTracksFieldStateAndValidation(t *testing.T) {
	installUIHookContext(t)

	form := UseForm(profileForm{Region: "us"})
	if form.TouchedAny() || form.DirtyAny() || form.HasErrors() {
		t.Fatal("expected fresh form state to be pristine and error-free")
	}
	if form.Get().Region != "us" {
		t.Fatalf("expected initial form state, got %+v", form.Get())
	}
	if !form.SetField("Name", "Alice") {
		t.Fatal("expected SetField to update exported struct field")
	}
	if form.Get().Name != "Alice" {
		t.Fatalf("expected updated field value, got %+v", form.Get())
	}
	if !form.Touched("Name") || !form.Dirty("Name") {
		t.Fatal("expected SetField to mark field as touched and dirty")
	}
	if !form.TouchedAny() || !form.DirtyAny() {
		t.Fatal("expected aggregate touched/dirty helpers to reflect updated field state")
	}
	form.SetErrors(FieldErrors{"Email": "required"})
	if form.Error("Email") != "required" {
		t.Fatalf("expected field error to be readable, got %q", form.Error("Email"))
	}
	form.SetFormError("try again")
	if form.FormError() != "try again" {
		t.Fatalf("expected form-level error to be readable, got %q", form.FormError())
	}
	if !form.HasErrors() {
		t.Fatal("expected aggregate error helper to report field/form errors")
	}
	valid := form.Validate(func(state profileForm) FieldErrors {
		if state.Name == "" {
			return FieldErrors{"Name": "required"}
		}
		return nil
	})
	if !valid {
		t.Fatal("expected validation to pass after name was set")
	}
	if len(form.Errors()) != 0 {
		t.Fatalf("expected successful validation to clear errors, got %#v", form.Errors())
	}
	if form.FormError() != "" {
		t.Fatalf("expected sync validation success to clear form error, got %q", form.FormError())
	}
	if form.HasErrors() {
		t.Fatal("expected successful sync validation to clear aggregate error state")
	}
	form.Reset()
	if form.Get().Name != "" || form.Get().Region != "us" {
		t.Fatalf("expected Reset to restore initial form state, got %+v", form.Get())
	}
	if form.Touched("Name") || form.Dirty("Name") {
		t.Fatal("expected Reset to clear touched and dirty state")
	}
	if form.TouchedAny() || form.DirtyAny() || form.HasErrors() {
		t.Fatal("expected Reset to restore pristine and error-free aggregate state")
	}
}

func TestUseFormAsyncValidationLifecycle(t *testing.T) {
	installUIHookContext(t)

	form := UseForm(profileForm{Name: "admin", Email: "alice@blocked.test"})
	form.ValidateAsync(func(value profileForm) (FieldErrors, string) {
		time.Sleep(20 * time.Millisecond)
		errs := FieldErrors{}
		if value.Name == "admin" {
			errs["Name"] = "reserved"
		}
		return errs, "blocked domain"
	}, nil)
	if !form.Validating() {
		t.Fatal("expected form to report validating immediately after ValidateAsync")
	}
	time.Sleep(40 * time.Millisecond)
	if form.Validating() || !form.Validated() {
		t.Fatalf("expected async validation to settle, validating=%t validated=%t", form.Validating(), form.Validated())
	}
	if form.Error("Name") != "reserved" {
		t.Fatalf("expected async field error, got %q", form.Error("Name"))
	}
	if form.FormError() != "blocked domain" {
		t.Fatalf("expected async form error, got %q", form.FormError())
	}

	completed := false
	form.SetField("Name", "alice")
	form.SetField("Email", "alice@example.com")
	form.ValidateAsync(func(value profileForm) (FieldErrors, string) {
		return nil, ""
	}, func(valid bool) {
		completed = valid
	})
	time.Sleep(20 * time.Millisecond)
	if !completed {
		t.Fatal("expected async validation callback to report valid state")
	}
	if form.FormError() != "" || len(form.Errors()) != 0 {
		t.Fatalf("expected async validation success to clear errors, formError=%q errors=%#v", form.FormError(), form.Errors())
	}
}

func TestUseFormSubmissionLifecycle(t *testing.T) {
	installUIHookContext(t)

	form := UseForm(profileForm{Name: "Alice"})
	block := make(chan struct{})
	form.Submit(func(value profileForm) error {
		if value.Name != "Alice" {
			return errors.New("unexpected form snapshot")
		}
		<-block
		return nil
	})
	if !form.Submitting() {
		t.Fatal("expected form to report submitting immediately after Submit")
	}
	close(block)
	time.Sleep(20 * time.Millisecond)
	if form.Submitting() || !form.Submitted() || form.SubmitError() != nil {
		t.Fatalf("expected successful submit lifecycle, submitted=%t submitting=%t err=%v", form.Submitted(), form.Submitting(), form.SubmitError())
	}

	form.Submit(func(value profileForm) error {
		return errors.New("server unavailable")
	})
	time.Sleep(20 * time.Millisecond)
	if form.SubmitError() == nil || form.Submitted() {
		t.Fatalf("expected failed submit to store error and clear submitted flag, submitted=%t err=%v", form.Submitted(), form.SubmitError())
	}
	if form.FormError() != "server unavailable" {
		t.Fatalf("expected failed submit to surface form error, got %q", form.FormError())
	}
	if !form.HasErrors() {
		t.Fatal("expected failed submit to mark aggregate error state")
	}
	form.Reset(profileForm{Region: "eu"})
	if form.Get().Region != "eu" || form.Submitted() || form.SubmitError() != nil {
		t.Fatalf("expected reset with new initial value to clear submission state, got %+v err=%v", form.Get(), form.SubmitError())
	}
	if form.FormError() != "" || form.HasErrors() {
		t.Fatal("expected reset to clear submit-derived form error state")
	}
}

func TestFormZeroValue(t *testing.T) {
	var form Form[profileForm]
	if form.Get().Name != "" {
		t.Fatalf("expected zero-value form to return zero form state, got %+v", form.Get())
	}
	if form.Touched("Name") || form.Dirty("Name") || form.TouchedAny() || form.DirtyAny() || form.HasErrors() || form.Validating() || form.Validated() || form.Submitting() || form.Submitted() || form.SubmitError() != nil || form.FormError() != "" {
		t.Fatal("expected zero-value form helpers to be inert")
	}
	form.SetField("Name", "ignored")
	form.SetErrors(FieldErrors{"Name": "required"})
	form.SetFormError("ignored")
	form.ValidateAsync(nil, nil)
	form.Reset()
}
