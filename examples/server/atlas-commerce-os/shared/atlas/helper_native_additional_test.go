package atlas

import (
	"context"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/v5/interop"
	"github.com/monstercameron/GoWebComponents/v5/ui"
)

type atlasFilterForm struct {
	Query string
	Count int
}

// TestAtlasFilterHelpersNormalizeAndApplyQuery verifies list-filter normalization and transition helpers.
func TestAtlasFilterHelpersNormalizeAndApplyQuery(parseT *testing.T) {
	parseCatalogState := atlasCatalogFilterState(catalogQueryState{
		Search:    " Desk ",
		Category:  " seating ",
		Warehouse: "",
		Sort:      " recent ",
	})
	if parseCatalogState.Query != " Desk " || parseCatalogState.Category != "seating" || parseCatalogState.Warehouse != "all" || parseCatalogState.Sort != " recent " {
		parseT.Fatalf("unexpected catalog filter state %#v", parseCatalogState)
	}

	parseMapState := atlasMapFilterState(map[string]string{
		"q":         "Focus",
		"category":  "desks",
		"warehouse": "north-hub",
		"status":    "",
		"sort":      "price",
	})
	if parseMapState.Query != "Focus" || parseMapState.Status != "all" || parseMapState.Sort != "price" {
		parseT.Fatalf("unexpected map filter state %#v", parseMapState)
	}
	if parseNilState := atlasMapFilterState(nil); parseNilState.Category != "all" || parseNilState.Warehouse != "all" || parseNilState.Status != "all" {
		parseT.Fatalf("expected nil map state to fall back, got %#v", parseNilState)
	}

	parseSavedState := atlasSavedViewFilterState(SavedViewPayload{
		SortKey: "margin",
		Filters: map[string]string{"q": "desk", "category": "focus", "status": "draft"},
	})
	if parseSavedState.Sort != "margin" || parseSavedState.Status != "draft" {
		parseT.Fatalf("unexpected saved view state %#v", parseSavedState)
	}

	if !atlasSameListFilterState(
		atlasListFilterState{Query: " Desk ", Category: "Seating", Warehouse: "All", Status: " ALL ", Sort: " Recent "},
		atlasListFilterState{Query: "desk", Category: "seating", Warehouse: "all", Status: "all", Sort: "recent"},
	) {
		parseT.Fatal("expected normalized filter states to match")
	}

	parseQuery := atlasBuildListFilterQuery(url.Values{
		"page":     {"2"},
		"category": {"legacy"},
		"status":   {"draft"},
	}, atlasListFilterState{
		Query:     " Desk ",
		Category:  "all",
		Warehouse: " north-hub ",
		Status:    "",
		Sort:      " newest ",
	})
	if parseQuery.Get("page") != "2" || parseQuery.Get("q") != "desk" || parseQuery.Get("warehouse") != "north-hub" || parseQuery.Get("sort") != "newest" {
		parseT.Fatalf("unexpected built query %#v", parseQuery)
	}
	if parseQuery.Has("category") || parseQuery.Has("status") {
		parseT.Fatalf("expected all/empty filters to be removed, got %#v", parseQuery)
	}

	parseValues := url.Values{}
	atlasAssignFilterQuery(parseValues, "q", " desks ")
	atlasAssignFilterQuery(parseValues, "status", "all")
	if parseValues.Get("q") != "desks" || parseValues.Has("status") {
		parseT.Fatalf("unexpected assigned filter query %#v", parseValues)
	}
	if atlasNormalizedFilterValue("  Pending Review  ") != "pending review" {
		parseT.Fatal("expected normalized filter value to trim and lowercase")
	}
	if atlasFilterSubmitLabel(true, "Apply filters") != "Updating..." || atlasFilterSubmitLabel(false, "Apply filters") != "Apply filters" {
		parseT.Fatal("unexpected filter submit labels")
	}

	parseForm := ui.UseForm(atlasFilterForm{Query: "seed", Count: 1})
	atlasSetFormFieldInTransition(parseForm, "Query", "desk")
	atlasSetFormInTransition(parseForm, atlasFilterForm{Query: "shelf", Count: 3})
	parseFormValue := parseForm.Get()
	if parseFormValue.Query != "shelf" || parseFormValue.Count != 3 {
		parseT.Fatalf("expected transition helpers to update form value, got %#v", parseFormValue)
	}
}

// atlasNativeHookProbe records what each interaction hook reported during one
// native render pass, so the assertions can live outside the component body
// (a parseT.Fatalf inside a render would unwind through the SSR walker).
type atlasNativeHookProbe struct {
	stateInitial       int
	stateAfterSet      int
	effectRan          bool
	atomInitial        int
	atomAfterSet       int
	computed           string
	revalidatorLoading bool
	transitionPending  bool
	transitionStartRan bool
	throttled          string
	throttledPending   bool
	viewport           atlasViewportMetrics
	searchParams       url.Values
	cached             atlasCachedResourceState[string]
	resource           atlasResourceState[string]
	worker             atlasWorkerTaskState[int, bool]
	channelValue       string
	channelOk          bool
	channelClosed      bool
	loaderCalls        int
}

// TestAtlasInteractionNativeHooksHonorTheirServerContract exercises the whole
// native interaction-hook surface the way an Atlas component does - inside a
// render pass - and pins the documented server-side value of each hook.
//
// The shape of this test is the point. It used to call every hook directly from
// test-function scope, which only worked because interaction_hooks_native.go
// stubbed the entire surface. Real hooks read per-component slots off the fiber
// the runtime is rendering, so a direct call panics with
// GWC-RUNTIME-HOOK-OUTSIDE-COMPONENT. A test that cannot tell the difference is
// exactly how Atlas shipped a client/main.go that called atlas.App(payload)
// eagerly and white-screened the browser while every native test passed.
//
// Anything imperative (atlasFetch, persistAtlasSnapshot) is intentionally
// exercised OUTSIDE the render below: those are not hooks and must not need a
// fiber.
func TestAtlasInteractionNativeHooksHonorTheirServerContract(parseT *testing.T) {
	// Atom IDs and cache keys are PROCESS-global and survive a RenderToString
	// call, and InitAtom is init-if-absent - so a fixed id would make this test
	// depend on nothing else in the binary having touched it, and would break
	// under `go test -count=2`. Unique-per-run ids keep the assertions about
	// initial values meaningful.
	parseRunID := strconv.FormatInt(time.Now().UnixNano(), 36)
	parseAtomID := "atlas-test-atom-" + parseRunID
	parseCacheKey := "atlas-test-cache-" + parseRunID

	var parseProbe atlasNativeHookProbe
	parseLoader := func(parseCtx context.Context) (string, error) {
		parseProbe.loaderCalls++
		return "loaded", parseCtx.Err()
	}

	parseMarkup, parseErr := renderAtlasNodeForTest(ui.CreateElement(func() ui.Node {
		parseState := useAtlasState(3)
		parseProbe.stateInitial = parseState.Get()
		parseState.Set(7)
		parseProbe.stateAfterSet = parseState.Get()

		useAtlasEffect(func() func() {
			parseProbe.effectRan = true
			return nil
		})

		parseAtom := useAtlasAtom(parseAtomID, 5)
		parseProbe.atomInitial = parseAtom.Get()
		parseAtom.Set(9)
		parseProbe.atomAfterSet = parseAtom.Get()

		parseProbe.computed = useAtlasComputed(func() string { return "computed" }).Get()

		parseRevalidator := useAtlasRevalidator()
		parseRevalidator.Revalidate()
		parseProbe.revalidatorLoading = parseRevalidator.Loading()

		parseTransition := useAtlasTransition()
		parseProbe.transitionPending = parseTransition.Pending()
		parseTransition.Start(func() { parseProbe.transitionStartRan = true })

		parseThrottled := useAtlasThrottled("atlas", time.Second)
		parseProbe.throttled = parseThrottled.Get()
		parseProbe.throttledPending = parseThrottled.Pending()

		parseProbe.viewport = useAtlasViewportMetrics()

		parseSearchParams := useAtlasSearchParams()
		parseProbe.searchParams = parseSearchParams.Values()
		parseSearchParams.ReplaceAll(url.Values{"q": {"desk"}})

		// Reload/Set/Update are deliberately NOT called on the cached handle:
		// unlike the initial load, CachedResource.Reload is not effect-gated, so
		// it would start a real (async) load and make this test racy. The wrapper
		// methods are covered with injected callbacks in
		// TestAtlasResourceWrappersInvokeCallbacks.
		parseProbe.cached = useAtlasCachedResource(parseCacheKey, parseLoader).Get()

		parseResource := useAtlasResource(parseLoader)
		parseProbe.resource = parseResource.Get()
		parseResource.Reload()

		parseWorkerTask := useAtlasWorkerTask[string, int, bool](interop.WorkerOptions{}, "sync")
		parseProbe.worker = parseWorkerTask.Get()
		parseWorkerTask.Start("payload")
		parseWorkerTask.Cancel()

		parseChannel := useAtlasChannel((<-chan string)(nil))
		parseProbe.channelValue = parseChannel.Get()
		parseProbe.channelOk = parseChannel.Ok()
		parseProbe.channelClosed = parseChannel.Closed()

		return ui.Text("atlas-hook-probe")
	}))
	if parseErr != nil {
		parseT.Fatalf("hook probe render failed: %v", parseErr)
	}
	if parseMarkup != "atlas-hook-probe" {
		parseT.Fatalf("hook probe markup = %q, want %q", parseMarkup, "atlas-hook-probe")
	}

	// REAL: ui.UseState is a live handle natively, so a write is observable in the
	// same pass. The old expectation (Set is dropped) described a stub.
	if parseProbe.stateInitial != 3 || parseProbe.stateAfterSet != 7 {
		parseT.Fatalf("local state initial=%d afterSet=%d, want 3 then 7", parseProbe.stateInitial, parseProbe.stateAfterSet)
	}

	// INERT BY DESIGN: effects describe post-commit work, and a server render
	// never commits, so the body must not run (and its cleanup would never run).
	if parseProbe.effectRan {
		parseT.Fatal("useAtlasEffect ran its body during a server render; effects must be skipped on the server")
	}

	// REAL: this is the assertion that changed meaning. The atom is backed by the
	// runtime registry now, so Set actually writes and a later Get sees 9. The
	// previous test asserted Get()==5 after Set(9) - i.e. it asserted that
	// server-rendered markup can only ever show default state.
	if parseProbe.atomInitial != 5 {
		parseT.Fatalf("atom initial = %d, want 5", parseProbe.atomInitial)
	}
	if parseProbe.atomAfterSet != 9 {
		parseT.Fatalf("atom after Set(9) = %d, want 9 (a real atom write must be observable during SSR)", parseProbe.atomAfterSet)
	}

	if parseProbe.computed != "computed" {
		parseT.Fatalf("computed = %q, want %q", parseProbe.computed, "computed")
	}

	// INERT: no route history to revalidate against; no loader was ever in flight.
	if parseProbe.revalidatorLoading {
		parseT.Fatal("native revalidator reported Loading(); there is no route loader on the server")
	}

	// REAL: native transitions run inline, so nothing is ever outstanding and the
	// callback must have already run.
	if parseProbe.transitionPending {
		parseT.Fatal("native transition reported Pending(); transitions run inline off-browser")
	}
	if !parseProbe.transitionStartRan {
		parseT.Fatal("transition Start did not run its callback")
	}

	// REAL: throttling is a rate limit over wall time; one SSR pass has no rate,
	// so the live value must pass through unchanged.
	if parseProbe.throttled != "atlas" || parseProbe.throttledPending {
		parseT.Fatalf("throttled value=%q pending=%v, want %q and false", parseProbe.throttled, parseProbe.throttledPending, "atlas")
	}

	// INERT: no window, and one HTML response serves every screen size.
	// SampleCount 0 is the "never measured" signal callers must branch on.
	if parseProbe.viewport != (atlasViewportMetrics{}) {
		parseT.Fatalf("viewport metrics = %#v, want the zero snapshot", parseProbe.viewport)
	}

	// INERT: the query string reaches the server as Payload.Route.Query, not
	// through this hook. Empty but non-nil.
	if parseProbe.searchParams == nil || len(parseProbe.searchParams) != 0 {
		parseT.Fatalf("search params = %#v, want an empty non-nil url.Values", parseProbe.searchParams)
	}

	// REAL hooks whose loaders are effect-driven: the handles exist, but no I/O
	// happened. loaderCalls == 0 is the load-bearing assertion - it proves a
	// server render cannot dial out through either resource hook.
	if parseProbe.cached != (atlasCachedResourceState[string]{}) {
		parseT.Fatalf("cached resource state = %#v, want the not-ready zero state", parseProbe.cached)
	}
	if parseProbe.resource != (atlasResourceState[string]{}) {
		parseT.Fatalf("resource state = %#v, want the not-ready zero state", parseProbe.resource)
	}
	if parseProbe.loaderCalls != 0 {
		parseT.Fatalf("resource loaders ran %d time(s) during a server render; want 0", parseProbe.loaderCalls)
	}

	// INERT: ui.UseWorkerTask is browser-only (ui/worker_wasm.go); nothing can
	// ever transition this handle natively.
	if parseProbe.worker != (atlasWorkerTaskState[int, bool]{}) {
		parseT.Fatalf("worker task state = %#v, want the not-started zero state", parseProbe.worker)
	}

	// INERT: ui.UseChannel is browser-only (ui/ui_async.go). Note this must also
	// not DRAIN the channel - a value read here would be stolen from the client.
	if parseProbe.channelValue != "" || parseProbe.channelOk || parseProbe.channelClosed {
		parseT.Fatalf("channel wrapper value=%q ok=%v closed=%v, want zero/false/false", parseProbe.channelValue, parseProbe.channelOk, parseProbe.channelClosed)
	}
}

// TestAtlasImperativeNativeHelpersStayOffTheFiber verifies the two imperative
// helpers in the native interaction surface work without a render fiber, because
// Atlas calls them from event handlers and effects rather than from render.
func TestAtlasImperativeNativeHelpersStayOffTheFiber(parseT *testing.T) {
	// startAtlasTransition is not a hook (no fiber slot), and natively it runs the
	// callback inline - there is no frame loop to defer to.
	isParseTransitionCalled := false
	startAtlasTransition(func() {
		isParseTransitionCalled = true
	})
	if !isParseTransitionCalled {
		parseT.Fatal("startAtlasTransition did not run its callback immediately")
	}

	// atlasFetch now delegates to fetch.Fetch, whose native slice reports the
	// error rather than Atlas inventing its own message. A non-empty Error here
	// means "no request was attempted", not "the request was rejected".
	parseFetchResult := <-atlasFetch("/api/catalog", atlasFetchOptions{Method: "GET"})
	if parseFetchResult.Error == "" || parseFetchResult.Status != 0 || parseFetchResult.Data != "" {
		parseT.Fatalf("unexpected native fetch result %#v", parseFetchResult)
	}

	// nil means "there was nothing to fail", not "the snapshot is durable".
	if parseErr := persistAtlasSnapshot("atlas-dashboard", "inventory", "comments"); parseErr != nil {
		parseT.Fatalf("persistAtlasSnapshot(native): %v", parseErr)
	}
}

// TestAtlasResourceWrappersInvokeCallbacks verifies wrapper methods honor configured callbacks and safe zero values.
func TestAtlasResourceWrappersInvokeCallbacks(parseT *testing.T) {
	var parseReloadCount int
	var parseSetValue int
	var parseUpdatedValue int

	parseCached := atlasCachedResource[int]{
		get: func() atlasCachedResourceState[int] {
			return atlasCachedResourceState[int]{Value: 4, Loading: true, Ready: true, Stale: true}
		},
		reload: func() {
			parseReloadCount++
		},
		set: func(parseValue int) {
			parseSetValue = parseValue
		},
		update: func(parseFn func(int) int) {
			parseUpdatedValue = parseFn(3)
		},
	}
	if parseState := parseCached.Get(); parseState.Value != 4 || !parseState.Loading || !parseState.Ready || !parseState.Stale {
		parseT.Fatalf("unexpected cached resource state %#v", parseState)
	}
	parseCached.Reload()
	parseCached.Set(9)
	parseCached.Update(func(parseValue int) int { return parseValue + 2 })
	if parseReloadCount != 1 || parseSetValue != 9 || parseUpdatedValue != 5 {
		parseT.Fatalf("unexpected cached resource callback results reload=%d set=%d updated=%d", parseReloadCount, parseSetValue, parseUpdatedValue)
	}

	var parseResourceReloaded bool
	parseResource := atlasResource[string]{
		get: func() atlasResourceState[string] {
			return atlasResourceState[string]{Value: "atlas", Ready: true}
		},
		reload: func() {
			parseResourceReloaded = true
		},
	}
	if parseState := parseResource.Get(); parseState.Value != "atlas" || !parseState.Ready {
		parseT.Fatalf("unexpected resource state %#v", parseState)
	}
	parseResource.Reload()
	if !parseResourceReloaded {
		parseT.Fatal("expected resource reload callback to run")
	}

	var parseWorkerStarted string
	var isParseWorkerCancelled bool
	parseTask := atlasWorkerTask[string, int, bool]{
		get: func() atlasWorkerTaskState[int, bool] {
			return atlasWorkerTaskState[int, bool]{Value: true, Progress: 50, ProgressReady: true, Running: true}
		},
		start: func(parsePayload string) {
			parseWorkerStarted = parsePayload
		},
		cancel: func() {
			isParseWorkerCancelled = true
		},
	}
	if parseState := parseTask.Get(); !parseState.Value || parseState.Progress != 50 || !parseState.ProgressReady || !parseState.Running {
		parseT.Fatalf("unexpected worker task state %#v", parseState)
	}
	parseTask.Start("sync-products")
	parseTask.Cancel()
	if parseWorkerStarted != "sync-products" || !isParseWorkerCancelled {
		parseT.Fatalf("unexpected worker task callbacks start=%q cancelled=%v", parseWorkerStarted, isParseWorkerCancelled)
	}

	parseChannel := atlasChannelValue[string]{
		get:    func() string { return "ready" },
		ok:     func() bool { return true },
		closed: func() bool { return true },
	}
	if parseChannel.Get() != "ready" || !parseChannel.Ok() || !parseChannel.Closed() {
		parseT.Fatalf("unexpected channel wrapper state value=%q ok=%v closed=%v", parseChannel.Get(), parseChannel.Ok(), parseChannel.Closed())
	}

	var parseZeroCached atlasCachedResource[int]
	parseZeroCached.Reload()
	parseZeroCached.Set(5)
	parseZeroCached.Update(func(parseValue int) int { return parseValue + 1 })
	if parseZeroCached.Get() != (atlasCachedResourceState[int]{}) {
		parseT.Fatalf("expected zero cached resource state, got %#v", parseZeroCached.Get())
	}
}

// TestAtlasBootstrapAndToastHelpersCloneAndDispatch verifies copy helpers and toast dispatch behavior.
func TestAtlasBootstrapAndToastHelpersCloneAndDispatch(parseT *testing.T) {
	parseEmptyQuery := cloneQuery(nil)
	parseEmptyParams := cloneParams(nil)
	if len(parseEmptyQuery) != 0 || len(parseEmptyParams) != 0 {
		parseT.Fatalf("expected empty clone helpers for nil input, got query=%#v params=%#v", parseEmptyQuery, parseEmptyParams)
	}

	parseOriginalQuery := map[string][]string{"q": {"desk", "chair"}}
	parseOriginalParams := map[string]string{"sku": "frame-desk"}
	parseClonedQuery := cloneQuery(parseOriginalQuery)
	parseClonedParams := cloneParams(parseOriginalParams)
	parseClonedQuery["q"][0] = "lamp"
	parseClonedParams["sku"] = "bench"
	if parseOriginalQuery["q"][0] != "desk" || parseOriginalParams["sku"] != "frame-desk" {
		parseT.Fatalf("expected clone helpers to deep copy inputs, got query=%#v params=%#v", parseOriginalQuery, parseOriginalParams)
	}

	clearAtlasShellToastBus()
	dispatchAtlasShellToast(atlasShellToast{})
	select {
	case parseToast := <-atlasShellToastBus:
		parseT.Fatalf("expected blank-title toast to be ignored, got %#v", parseToast)
	default:
	}

	parseToast := atlasShellToast{Title: "Saved", Detail: "Inventory synced", Tone: "success"}
	dispatchAtlasShellToast(parseToast)
	select {
	case parseReceived := <-atlasShellToastBus:
		if parseReceived != parseToast {
			parseT.Fatalf("unexpected toast payload %#v", parseReceived)
		}
	default:
		parseT.Fatal("expected dispatched toast to reach the bus")
	}

	clearAtlasShellToastBus()
	for parseIndex := 0; parseIndex < cap(atlasShellToastBus); parseIndex++ {
		atlasShellToastBus <- atlasShellToast{Title: "toast"}
	}
	dispatchAtlasShellToast(atlasShellToast{Title: "overflow"})
	if len(atlasShellToastBus) != cap(atlasShellToastBus) {
		parseT.Fatalf("expected full toast bus to drop overflow without blocking, len=%d cap=%d", len(atlasShellToastBus), cap(atlasShellToastBus))
	}
	clearAtlasShellToastBus()
}

// clearAtlasShellToastBus drains the shared toast bus between assertions.
func clearAtlasShellToastBus() {
	for {
		select {
		case <-atlasShellToastBus:
		default:
			return
		}
	}
}
