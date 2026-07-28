//go:build !js || !wasm

// Native (non-browser) half of Atlas's interaction-hook surface.
//
// # Read this first if you are learning v5 from Atlas
//
// Every hook here has a sibling in interaction_hooks_wasm.go. The two files are
// ONE API with two implementations, and one rule governs both:
//
//	A hook is implemented here by calling the SAME framework hook its wasm
//	sibling calls, unless the thing it talks to genuinely does not exist
//	outside a browser.
//
// That rule is not stylistic, it is a test-integrity rule. GWC hooks read
// per-component slots off the fiber the runtime is currently rendering
// (internal/runtime/state.go: requireCurrentHookFiber). Calling one outside a
// render pass panics with GWC-RUNTIME-HOOK-OUTSIDE-COMPONENT. If this file
// answered with hand-rolled values instead of calling the framework, a
// hooks-outside-render bug would pass every native test and panic in the
// browser.
//
// That is not hypothetical. It is Atlas's own history: client/main.go used to
// call atlas.App(payload) eagerly instead of ui.CreateElement(atlas.App,
// payload). In the browser that panics before first paint, so the app could not
// boot at all. Natively, app_render_matrix_test.go rendered every public and
// internal route through the same App and passed, because the hooks App called
// were stubs that never asked for a fiber. ~29k lines of "covered" code could
// not start.
//
// The stubs also made server rendering meaningless in a second, quieter way. A
// fake atom whose Get() always returned the initial value and whose Set()
// discarded writes makes natively rendered markup the default state of every
// atom, permanently. You cannot build real SSR on that: the render is not a
// render of your state, it is a render of your zero values.
//
// # The split
//
// Each function below is labelled:
//
//	REAL  - delegates to the framework, the same call the wasm sibling makes.
//	        Native behaviour is whatever the framework's native slice defines,
//	        which is the point: one behaviour, two targets.
//	INERT - the browser capability does not exist during a server render. The
//	        comment states the server-side value, why it is that value, and what
//	        a caller must not assume. Inert here is a documented contract, not a
//	        placeholder waiting to be filled in.
//
// See docs/REFERENCE_MANUAL/error-codes.md, entry
// GWC-RUNTIME-HOOK-OUTSIDE-COMPONENT.
//
// (This block is a file comment, not the package doc: it is build-tagged native
// only, and a package doc that vanishes on the wasm build would be worse than
// none.)

package atlas

import (
	"context"
	"net/url"
	"time"

	"github.com/monstercameron/GoWebComponents/v5/fetch"
	"github.com/monstercameron/GoWebComponents/v5/interop"
	"github.com/monstercameron/GoWebComponents/v5/state"
	"github.com/monstercameron/GoWebComponents/v5/ui"
)

// atlasLocalState mirrors the wasm declaration field-for-field: it holds a
// ui.State[T] handle, not a bare T.
//
// The shape matters more than it looks. Holding a ui.State[T] means the Get/Set
// methods below cannot drift from the browser's, because there is nothing left
// in this type for them to drift with - they are pure forwarding. The previous
// native declaration held a bare `value T` and re-implemented Get/Set on top of
// it, which is how two dialects of "local state" got into one codebase.
type atlasLocalState[T any] struct {
	value ui.State[T]
}

// useAtlasState is REAL: it calls ui.UseState, exactly like the wasm sibling.
//
// ui.UseState's native slice (ui/ui_native.go) backs the handle with a
// render-scoped closure: reads see writes made during the same render pass, and
// nothing is retained past it. That is the correct SSR semantic for component
// local state - a server render is one pass and there is no second pass to
// re-render into - and it is the framework's decision to make, not Atlas's.
//
// Atlas's job is only to route the call there. Any future change to native
// UseState (slot-backed state, hook-order validation) is then inherited here for
// free instead of needing a matching edit in a private copy.
func useAtlasState[T any](parseInitial T) *atlasLocalState[T] {
	return &atlasLocalState[T]{value: ui.UseState(parseInitial)}
}

// Get reads the current value through the framework handle.
func (parseS *atlasLocalState[T]) Get() T {
	return parseS.value.Get()
}

// Set writes through the framework handle.
//
// It is a real write, not a swallowed one: code that Sets and then Gets in the
// same render pass observes its own write on both targets. Tests that depended
// on native Set being a no-op were asserting the absence of a feature.
func (parseS *atlasLocalState[T]) Set(parseValue T) {
	parseS.value.Set(parseValue)
}

// useAtlasEffect is REAL, and the effect deliberately does not run here.
//
// This is the case most likely to be mistaken for a stub, so be precise about
// what is happening: ui.UseEffect exists on the native slice and is defined
// there as a no-op (ui/ui_native.go). Effects describe work to do AFTER a commit
// to a live DOM - subscribing to window events, starting timers, kicking off
// fetches. A server render produces a string and then the tree is gone; there is
// no commit, no unmount, and therefore nowhere to run the cleanup the effect
// returns. Running effects during SSR would mean starting subscriptions and
// timers that leak by construction.
//
// So the correct server behaviour is "queued and skipped", and that is what the
// framework does (see ui/ssr_hooks_test.go: TestSSRUseEffectIsNotRunOnServer).
// Atlas gets it by calling the real hook rather than by declaring its own no-op,
// so the two targets cannot disagree about it later.
//
// What a caller must not assume: that any state an effect would have written is
// present in server markup. If a value must appear in SSR output, it has to be
// derivable from props/payload during render - not produced by an effect.
func useAtlasEffect(parseEffect func() func(), parseDeps ...any) {
	ui.UseEffect(parseEffect, parseDeps...)
}

// atlasComputed and atlasAtom hold accessor funcs rather than values, matching
// the wasm declarations. Same reasoning as atlasLocalState: a handle that stores
// functions has no room for a second implementation to grow inside it.
type atlasComputed[T any] struct {
	get func() T
}

type atlasAtom[T any] struct {
	get func() T
	set func(T)
}

// atlasRevalidator is INERT - see useAtlasRevalidator. It keeps the wasm field
// names so the nil-guarded methods below read identically on both targets.
type atlasRevalidator struct {
	revalidate func()
	loading    func() bool
}

type atlasTransition struct {
	pending func() bool
	start   func(func())
}

type atlasThrottled[T any] struct {
	get     func() T
	pending func() bool
}

// atlasSearchParams is INERT - see useAtlasSearchParams.
type atlasSearchParams struct {
	values     func() url.Values
	replaceAll func(url.Values)
}

// atlasViewportMetrics is INERT - see useAtlasViewportMetrics. The fields are
// plain values (not accessors) in both targets because callers read them
// directly as a snapshot struct.
type atlasViewportMetrics struct {
	ScrollY       int
	Width         int
	Height        int
	SampleCount   int
	MeasuredAtUTC string
}

// useAtlasAtom is REAL: it calls state.UseAtom, exactly like the wasm sibling.
//
// WHY this calls state.UseAtom rather than returning parseInitial:
//
// state.UseAtom reaches through to runtime.GoUseAtom, which requires the fiber
// currently being rendered. During a native ui.RenderToString pass the SSR
// walker installs a transient hook fiber per component
// (internal/runtime/ssr.go: withSSRHookFiber), so this works - and outside a
// render pass it panics with GWC-RUNTIME-HOOK-OUTSIDE-COMPONENT, on native as
// well as in the browser. That panic is the feature. It is what makes a failing
// native test evidence that the browser would fail too.
//
// The stub this replaced returned parseInitial from Get() and dropped Set(). It
// compiled, it rendered, and it silently converted every native render into a
// render of default state while letting the eager-call bug through. Both of
// those failures are now impossible in the same way: by not having a second
// implementation.
//
// What server-side callers must understand about atom scope, and the history
// behind it, because the answer CHANGED and the old answer is still the
// intuitive one:
//
// Each SSR render is a fresh world. RenderToString and its streaming siblings
// mint a throwaway *Runtime owning its own AtomRegistry for the duration of one
// render (internal/runtime/ssr_atom_scope.go), and stamp it on every transient
// SSR hook fiber. So:
//
//  1. parseInitial WINS on every request. A previous request can never leak its
//     value into this one.
//  2. A Set during a render is visible to the rest of THAT render — a parent can
//     publish and a later sibling can read it — and to nothing else.
//  3. Request-specific state is an input, not a side effect: seed it with
//     ui.RenderToStringWithAtoms(root, map[string]any{...}). Precedence is
//     seed > component initial > nothing.
//
// This was NOT true until 2026-07-26, and the failure it caused is worth
// remembering. The registry used to be process-global (runtime.GetGlobalRuntime)
// with init-if-absent InitAtom semantics, so the first request to touch an atom
// owned it for the life of the process and every later request rendered that
// request's value — one visitor's state in another visitor's HTML. It was
// invisible while this file's hooks were stubs, because a stub never asked the
// registry anything. Making them real is what exposed it.
//
// A consequence for anyone reading page.go: App's RouteKey-vs-payload freshness
// check is now redundant. It was the workaround for exactly this leak. It is
// harmless and still correct, so it stays — but do not copy that pattern into
// new code believing it is required, and do not conclude from its presence that
// atoms still bleed across requests.
//
// Do not "simplify" the SSR scope away. It looks like ceremony around a registry
// that could just be reset between renders; a reset races the moment a server
// handles two requests at once, which is the whole reason the scope is per
// render rather than per process.
func useAtlasAtom[T any](parseId string, parseInitial T) atlasAtom[T] {
	parseAtom := state.UseAtom(parseId, parseInitial)
	return atlasAtom[T]{get: parseAtom.Get, set: parseAtom.Set}
}

// Get reads the atom. The nil-accessor branch is not dead code: atlasAtom is a
// plain struct, so a zero value can reach these methods from a test table or a
// field that was never assigned, and a nil call there would panic far from the
// mistake. Returning the zero T keeps the failure legible.
func (parseA atlasAtom[T]) Get() T {
	if parseA.get == nil {
		var parseZero T
		return parseZero
	}
	return parseA.get()
}

// Set writes the atom. Real on both targets now: during a native render the
// write lands in the atom registry and a later Get in the same pass observes it.
func (parseA atlasAtom[T]) Set(parseValue T) {
	if parseA.set != nil {
		parseA.set(parseValue)
	}
}

// useAtlasComputed is REAL: it calls state.UseComputed, like the wasm sibling.
//
// The tempting stub was `return atlasComputed[T]{value: parseCompute()}`, and it
// even produced the right value - which is why it survived. What it lost was the
// hook: state.UseComputed goes through runtime.GoUseMemo, which claims a memo
// slot on the current fiber and so participates in hook ordering and in the
// outside-render panic. Calling parseCompute() directly opts the call site out of
// both, and it opts out invisibly.
//
// Native memoization is a pass-through (the compute runs every render) because a
// single SSR pass has nothing to memoize across. The observable value is
// identical to the stub's; the difference is that a misplaced call now fails
// loudly instead of quietly working.
func useAtlasComputed[T any](parseCompute func() T, parseDeps ...any) atlasComputed[T] {
	parseComputed := state.UseComputed(parseCompute, parseDeps...)
	return atlasComputed[T]{get: parseComputed.Get}
}

// Get returns the computed value, or the zero T for a zero-value handle.
func (parseC atlasComputed[T]) Get() T {
	if parseC.get == nil {
		var parseZero T
		return parseZero
	}
	return parseC.get()
}

// useAtlasRevalidator is INERT, and it cannot be anything else here.
//
// The wasm sibling calls router.UseRevalidator. That hook is declared in
// router/router_api.go under `//go:build js && wasm` - the whole browser router
// API, and the *Router type it needs, are absent from the native build. There is
// no symbol to delegate to, so this is not a judgement call about semantics.
//
// It is also the right semantics. Revalidation means "run the current route's
// loader again and re-render". A server render has no route history, no live
// loader, and no second render to deliver the result into; the server already
// fetched its data before calling into Atlas.
//
// Server-side value: Loading() reports false and Revalidate() does nothing.
//
// What a caller must not assume: that calling Revalidate() will refresh anything
// during SSR, or that Loading()==false means a loader finished - natively it
// means no loader was ever in flight. Render-blocking data belongs in the
// payload, not behind a revalidation.
func useAtlasRevalidator() atlasRevalidator {
	return atlasRevalidator{}
}

// Revalidate is a no-op natively (nil accessor); see useAtlasRevalidator.
func (parseR atlasRevalidator) Revalidate() {
	if parseR.revalidate != nil {
		parseR.revalidate()
	}
}

// Loading reports false natively (nil accessor); see useAtlasRevalidator.
func (parseR atlasRevalidator) Loading() bool {
	if parseR.loading == nil {
		return false
	}
	return parseR.loading()
}

// startAtlasTransition is REAL: it calls ui.StartTransition on both targets.
//
// Natively that runs parseFn inline (ui/ui_native.go). Transitions exist to keep
// an urgent update (typing, clicking) from being blocked by an expensive one, by
// letting the scheduler interrupt and resume the low-priority work. Off-browser
// there is no frame loop and nothing to interrupt, so "run it now" IS the
// transition - deferring would only add a hop and change ordering.
//
// What a caller must not assume: that work inside a transition is deferred.
// Natively it has already completed by the time StartTransition returns, so a
// transition body must not depend on running after the current render.
func startAtlasTransition(parseFn func()) {
	ui.StartTransition(parseFn)
}

// useAtlasTransition is REAL: it calls ui.UseTransition on both targets.
//
// Pending() is false for the whole native pass, and that is accurate rather than
// stubbed: the transition ran inline (see startAtlasTransition), so there is
// never an interval during which one is outstanding.
//
// Consequence for SSR markup that is easy to get wrong: a "Saving…"/"Updating…"
// label driven by Pending() renders in its idle form on the server. That is the
// correct first paint - the browser has not interacted yet - but it means such a
// label can never be asserted as "pending" in a server-render test.
func useAtlasTransition() atlasTransition {
	parseTransition := ui.UseTransition()
	return atlasTransition{
		pending: parseTransition.Pending,
		start:   parseTransition.Start,
	}
}

// Pending reports whether a transition is outstanding; false natively.
func (parseT atlasTransition) Pending() bool {
	if parseT.pending == nil {
		return false
	}
	return parseT.pending()
}

// Start runs parseFn inside the transition, falling back to the package-level
// starter so a zero-value handle still runs the callback instead of dropping it.
// Dropping it would turn a missing handle into silently skipped application
// logic, which is far worse than losing the transition's scheduling hint.
func (parseT atlasTransition) Start(parseFn func()) {
	if parseT.start != nil {
		parseT.start(parseFn)
		return
	}
	startAtlasTransition(parseFn)
}

// useAtlasThrottled is REAL: it calls ui.UseThrottled on both targets.
//
// Natively the framework returns the current value unchanged with Pending()
// false (ui/ui_native.go). Throttling is a rate limit on re-renders over wall
// time; one SSR pass has no rate. Returning the live value is what makes server
// markup reflect the value the caller actually passed in - a native
// implementation that returned a stale or zero value would put the throttle's
// warm-up state into the HTML.
func useAtlasThrottled[T any](parseValue T, parseInterval time.Duration) atlasThrottled[T] {
	parseThrottled := ui.UseThrottled(parseValue, parseInterval)
	return atlasThrottled[T]{
		get:     parseThrottled.Get,
		pending: parseThrottled.Pending,
	}
}

// Get returns the throttled value, or the zero T for a zero-value handle.
func (parseT atlasThrottled[T]) Get() T {
	if parseT.get == nil {
		var parseZero T
		return parseZero
	}
	return parseT.get()
}

// Pending reports whether a throttled update is waiting; false natively.
func (parseT atlasThrottled[T]) Pending() bool {
	if parseT.pending == nil {
		return false
	}
	return parseT.pending()
}

// useAtlasViewportMetrics is INERT by measurement, not by omission.
//
// The wasm sibling reads window.innerWidth/innerHeight/scrollY and then
// subscribes to scroll and resize. A server render has no window and, more
// fundamentally, no single viewport: one HTML response is sent to a phone and a
// 4K monitor alike. Any number this function invented would be a guess baked
// into markup for every client.
//
// Server-side value: the zero struct - Width/Height/ScrollY 0, SampleCount 0,
// MeasuredAtUTC "".
//
// SampleCount is the field to branch on. It counts measurements taken, so
// SampleCount == 0 means "never measured", which is distinguishable from
// "measured and found to be 0". Treat 0 as unknown and render a layout that does
// not depend on viewport size; the browser takes its first real sample on mount
// and re-renders.
//
// What a caller must not assume: that Width == 0 means a narrow screen. It means
// no screen has been observed yet.
func useAtlasViewportMetrics() atlasViewportMetrics {
	return atlasViewportMetrics{}
}

// useAtlasSearchParams is INERT, and like the revalidator it has no native
// symbol to delegate to: router.UseSearchParams lives in router/router_api.go
// under `//go:build js && wasm`.
//
// This one is worth understanding rather than just accepting, because a server
// obviously DOES know the query string. It just does not know it through this
// hook. router.UseSearchParams reads ambient browser location state; on the
// server the query arrives as data, on Payload.Route.Query, and Atlas passes it
// down explicitly. Server-rendered filter state must be derived from the payload
// (see atlasMapFilterState / atlasBuildListFilterQuery in filter_state.go), not
// from this hook.
//
// Server-side value: Values() returns an empty (non-nil) url.Values and
// ReplaceAll is a no-op - there is no history entry to rewrite mid-render, and a
// server render cannot navigate.
//
// What a caller must not assume: that an empty Values() means the request had no
// query parameters. It means this hook is not the channel they arrived on.
func useAtlasSearchParams() atlasSearchParams {
	return atlasSearchParams{}
}

// Values returns the current query values, or an empty set natively. Non-nil on
// purpose: callers do url.Values.Get on the result, and returning a nil map that
// happens to be readable is a trap the moment someone Adds to it instead.
func (parseS atlasSearchParams) Values() url.Values {
	if parseS.values == nil {
		return url.Values{}
	}
	return parseS.values()
}

// ReplaceAll rewrites the query string; a no-op natively.
func (parseS atlasSearchParams) ReplaceAll(parseValues url.Values) {
	if parseS.replaceAll != nil {
		parseS.replaceAll(parseValues)
	}
}

// useAtlasCachedResource is REAL: it calls fetch.UseCachedResource with the same
// cache options as the wasm sibling.
//
// This is the entry most likely to look like it belongs on the inert side, so
// here is the evidence it does not. fetch.UseCachedResource is not
// build-tagged (fetch/cache.go). Its body is state.UseAtom for the cache
// snapshot plus ui.UseEffect to start the load. Natively that composition is
// already exactly the behaviour we want, for free:
//
//   - the atom read requires a render fiber, so a misplaced call panics
//     natively as it does in the browser;
//   - the loader is effect-driven, and native effects do not run, so NO I/O
//     happens during a server render. parseLoader is never called. Nothing
//     dials out, nothing blocks the render.
//
// So the inertness is a property of the framework's native slice, not of a stub
// Atlas maintains - which means it stays true if Atlas's loaders change, and it
// stays honest if the framework later adds real SSR data loading (Atlas would
// inherit that too).
//
// Server-side value: Get() reports the not-ready zero state (Ready false,
// Loading false, Value zero) unless the process-global cache already holds this
// key. Reload/Set/Update are real calls against that global cache.
//
// What a caller must not assume: (1) that the loader ran - server markup must be
// renderable from the payload alone, with the resource state treated as
// not-ready; (2) that the cache is per-request. Keys live in one process-wide
// registry (see fetch.UseCachedResource's global-namespace contract), so server
// code must not write request- or user-specific values into one.
func useAtlasCachedResource[T any](parseKey string, parseLoader func(context.Context) (T, error)) atlasCachedResource[T] {
	parseResource := fetch.UseCachedResource(parseKey, parseLoader, fetch.CacheOptions{
		StaleAfter:   5 * time.Minute,
		MaxAge:       30 * time.Minute,
		DisposeAfter: 90 * time.Minute,
	})
	return atlasCachedResource[T]{
		get: func() atlasCachedResourceState[T] {
			parseState := parseResource.Get()
			return atlasCachedResourceState[T]{
				Value:   parseState.Value,
				Loading: parseState.Loading,
				Error:   parseState.Error,
				Ready:   parseState.Ready,
				Stale:   parseState.Stale,
			}
		},
		reload: parseResource.Reload,
		set:    parseResource.Set,
		update: parseResource.Update,
	}
}

// useAtlasResource is REAL: it calls fetch.UseResource, like the wasm sibling.
//
// Same reasoning as useAtlasCachedResource, minus the shared cache:
// fetch.UseResource is untagged and drives its loader from ui.UseEffect
// (fetch/fetch.go), so natively the loader never runs and no request is made.
//
// Server-side value: Get() reports Loading false / Ready false / zero Value -
// "no attempt has been made", which is the truth. Note that this is NOT the same
// state as a failed load: Error stays nil.
//
// What a caller must not assume: that Ready==false plus Error==nil means the
// request is in flight. Natively it means there is no request. UI that renders a
// spinner on !Ready will server-render a spinner; if that is not the desired
// first paint, gate the spinner on Loading instead.
func useAtlasResource[T any](parseLoader func(context.Context) (T, error), parseDeps ...any) atlasResource[T] {
	parseResource := fetch.UseResource(parseLoader, parseDeps...)
	return atlasResource[T]{
		get: func() atlasResourceState[T] {
			parseState := parseResource.Get()
			return atlasResourceState[T]{
				Value:   parseState.Value,
				Loading: parseState.Loading,
				Error:   parseState.Error,
				Ready:   parseState.Ready,
			}
		},
		reload: parseResource.Reload,
	}
}

// atlasFetch is INERT in effect, but it is inert by delegating rather than by
// inventing - it calls the framework's own fetch.Fetch, whose native slice
// (fetch/fetch_native.go) answers every call with an error result.
//
// Why delegate instead of hand-writing the failure: the previous stub returned
// the bespoke string "fetch unavailable in native atlas build". That is a second
// source of truth for a fact the framework already states, and it is the kind of
// message that goes stale - if native HTTP ever becomes supported, an Atlas-local
// string keeps claiming otherwise. Going through fetch.Fetch means Atlas reports
// whatever the framework reports, always.
//
// It is also the right semantics. atlasFetch is the IMPERATIVE escape hatch:
// Atlas calls it from submit and moderation handlers (page.go, public_sections.go)
// to POST a mutation and read the response. Those are user gestures. A server
// render has no user gesture, and a browser fetch issued from a server render
// would be a request made with the server's identity, not the visitor's.
//
// Server-side value: a buffered channel already holding one result with a
// non-empty Error, zero Status and empty Data. It never blocks, and it is closed
// to further sends - one result, then done, matching the wasm shape so callers
// can `<-atlasFetch(...)` identically on both targets.
//
// The wasm sibling also calls fetch.ReturnChannel; that is deliberately omitted
// here. It is deprecated and now a no-op that only emits a deprecation warning
// (fetch/fetch.go), so calling it would add log noise and nothing else.
//
// What a caller must not assume: that a non-empty Error means the request was
// attempted and rejected. Natively it means no request was attempted.
func atlasFetch(parseUrl string, parseOptions atlasFetchOptions) <-chan atlasImperativeFetchResult {
	parseResultCh := make(chan atlasImperativeFetchResult, 1)
	go func() {
		parseResult := <-fetch.Fetch(parseUrl, fetch.Options{
			Method:  parseOptions.Method,
			Headers: parseOptions.Headers,
			Body:    parseOptions.Body,
		})
		parsePayload := atlasImperativeFetchResult{
			Data:    parseResult.Text(),
			Status:  parseResult.Status,
			Headers: parseResult.Headers,
		}
		if parseResult.Err != nil {
			parsePayload.Error = parseResult.Err.Error()
		}
		parseResultCh <- parsePayload
		close(parseResultCh)
	}()
	return parseResultCh
}

// useAtlasWorkerTask is INERT, with no native symbol to delegate to:
// ui.UseWorkerTask is declared only in ui/worker_wasm.go.
//
// A worker task offloads CPU work to a Web Worker so the main thread keeps
// painting - Atlas uses it to validate saved-view imports off-thread
// (page.go). Off-browser there is no main thread to protect and no Worker to
// spawn; Go code that wants concurrency has goroutines. So this hook is not
// "unimplemented natively", it is a browser-threading primitive with no native
// analogue worth faking.
//
// Server-side value: the not-started zero state - Running false, Started false,
// Ready false, Cancelled false, Error nil - and Start/Cancel do nothing.
//
// What a caller must not assume: that Start() will eventually produce a Result.
// Natively nothing will ever transition this handle, so a component that renders
// only on Ready renders its empty branch forever on the server. Validation whose
// result must appear in server markup has to be computed synchronously during
// render instead.
func useAtlasWorkerTask[Request any, Progress any, Result any](parseOptions interop.WorkerOptions, parseName string) atlasWorkerTask[Request, Progress, Result] {
	return atlasWorkerTask[Request, Progress, Result]{}
}

// useAtlasChannel is INERT, with no native symbol to delegate to: ui.UseChannel
// is declared only in ui/ui_async.go, under `//go:build js && wasm`.
//
// Reading a channel into rendered state is inherently a multi-render operation:
// ui.UseChannel starts a goroutine that Sets state as values arrive, and each Set
// schedules another render. A server render is ONE synchronous pass that is
// serialized and thrown away - a value arriving a microsecond later has no render
// to land in, and the goroutine would outlive the tree it was reading for. Even
// draining the channel non-blockingly would be wrong: it would consume a message
// destined for the live client (Atlas's toast bus, atlasShellToastBus, is exactly
// such a shared channel - page.go) and drop it.
//
// Server-side value: Ok() false, Closed() false, Get() the zero T. Nothing is
// read from parseCh and no goroutine is started.
//
// What a caller must not assume: that Ok()==false means the channel is empty, or
// that Closed()==false means it is open. Neither was ever examined. Anything that
// must appear in server markup has to be in the payload, not on a channel.
func useAtlasChannel[T any](parseCh <-chan T) atlasChannelValue[T] {
	return atlasChannelValue[T]{}
}

// persistAtlasSnapshot is INERT and returns nil, which is a claim worth being
// explicit about because nil normally means "it worked".
//
// The wasm sibling snapshots the atom registry and writes the selected atom IDs
// to localStorage, so a reload restores the user's shell preferences. Neither
// half of that exists on a server: interop.GetLocalStorage reports localStorage
// unavailable off-browser (interop/interop_native.go), and more importantly a
// server render has no user to persist for - one process serves every visitor,
// so "persisting" here would mean writing one visitor's UI preferences into
// state shared by all of them.
//
// Server-side value: nil, and nothing is written.
//
// nil rather than an error is a deliberate contract choice, not laziness. Atlas
// calls this from inside an effect (page.go) purely as a side effect of state
// changing; there is no failure for a caller to handle and no fallback to take.
// Returning "localStorage unavailable" would make every native call site either
// branch on build target or swallow an error that was never a problem - noise
// that trains readers to ignore the return value, which is how a real persistence
// failure gets missed later.
//
// What a caller must not assume: that nil means the snapshot is durable. It means
// there was nothing to fail. Durability is the browser's job; server code that
// needs state to survive must write it to the database.
func persistAtlasSnapshot(parseKey string, parseAtomIDs ...string) error {
	return nil
}
