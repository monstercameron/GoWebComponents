package atlas

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/interop"
	"github.com/monstercameron/GoWebComponents/ui"
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

// TestAtlasInteractionNativeHooksStaySafe verifies the host-build interaction helpers return stable values.
func TestAtlasInteractionNativeHooksStaySafe(parseT *testing.T) {
	parseState := useAtlasState(3)
	if parseState.Get() != 3 {
		parseT.Fatalf("expected local state to keep initial value, got %d", parseState.Get())
	}
	parseState.Set(7)
	if parseState.Get() != 7 {
		parseT.Fatalf("expected local state Set to update value, got %d", parseState.Get())
	}

	isParseEffectCalled := false
	useAtlasEffect(func() func() {
		isParseEffectCalled = true
		return nil
	})
	if isParseEffectCalled {
		parseT.Fatal("expected native atlas effect stub to no-op")
	}

	parseAtom := useAtlasAtom("inventory", 5)
	if parseAtom.Get() != 5 {
		parseT.Fatalf("expected atlas atom initial value, got %d", parseAtom.Get())
	}
	parseAtom.Set(9)
	if parseAtom.Get() != 5 {
		parseT.Fatalf("expected native atlas atom Set to stay no-op, got %d", parseAtom.Get())
	}

	parseComputed := useAtlasComputed(func() string { return "computed" })
	if parseComputed.Get() != "computed" {
		parseT.Fatalf("expected computed value, got %q", parseComputed.Get())
	}

	parseRevalidator := useAtlasRevalidator()
	parseRevalidator.Revalidate()
	if parseRevalidator.Loading() {
		parseT.Fatal("expected native revalidator to report not loading")
	}

	isParseTransitionCalled := false
	startAtlasTransition(func() {
		isParseTransitionCalled = true
	})
	if !isParseTransitionCalled {
		parseT.Fatal("expected native transition starter to run immediately")
	}

	parseTransition := useAtlasTransition()
	if parseTransition.Pending() {
		parseT.Fatal("expected native transition pending=false")
	}
	isParseTransitionStartCalled := false
	parseTransition.Start(func() {
		isParseTransitionStartCalled = true
	})
	if !isParseTransitionStartCalled {
		parseT.Fatal("expected native transition start helper to run callback")
	}

	parseThrottled := useAtlasThrottled("atlas", time.Second)
	if parseThrottled.Get() != "atlas" || parseThrottled.Pending() {
		parseT.Fatalf("unexpected native throttled state value=%q pending=%v", parseThrottled.Get(), parseThrottled.Pending())
	}

	parseViewport := useAtlasViewportMetrics()
	if parseViewport != (atlasViewportMetrics{}) {
		parseT.Fatalf("expected zero viewport metrics, got %#v", parseViewport)
	}

	parseSearchParams := useAtlasSearchParams()
	if parseValues := parseSearchParams.Values(); len(parseValues) != 0 {
		parseT.Fatalf("expected empty native search params, got %#v", parseValues)
	}
	parseSearchParams.ReplaceAll(url.Values{"q": {"desk"}})

	parseFetchResult := <-atlasFetch("/api/catalog", atlasFetchOptions{Method: "GET"})
	if parseFetchResult.Error == "" || parseFetchResult.Status != 0 || parseFetchResult.Data != "" {
		parseT.Fatalf("unexpected native fetch result %#v", parseFetchResult)
	}

	if parseErr := persistAtlasSnapshot("atlas-dashboard", "inventory", "comments"); parseErr != nil {
		parseT.Fatalf("persistAtlasSnapshot(native): %v", parseErr)
	}

	parseCachedResource := useAtlasCachedResource("catalog", func(parseCtx context.Context) (string, error) {
		return "ignored", parseCtx.Err()
	})
	if parseCachedResource.Get() != (atlasCachedResourceState[string]{}) {
		parseT.Fatalf("expected zero cached resource state, got %#v", parseCachedResource.Get())
	}
	parseCachedResource.Reload()
	parseCachedResource.Set("updated")
	parseCachedResource.Update(func(parseValue string) string { return parseValue + "!" })

	parseResource := useAtlasResource(func(parseCtx context.Context) (string, error) {
		return "ignored", parseCtx.Err()
	})
	if parseResource.Get() != (atlasResourceState[string]{}) {
		parseT.Fatalf("expected zero resource state, got %#v", parseResource.Get())
	}
	parseResource.Reload()

	parseWorkerTask := useAtlasWorkerTask[string, int, bool](interop.WorkerOptions{}, "sync")
	if parseWorkerTask.Get() != (atlasWorkerTaskState[int, bool]{}) {
		parseT.Fatalf("expected zero worker task state, got %#v", parseWorkerTask.Get())
	}
	parseWorkerTask.Start("payload")
	parseWorkerTask.Cancel()

	parseChannel := useAtlasChannel((<-chan string)(nil))
	if parseChannel.Get() != "" || parseChannel.Ok() || parseChannel.Closed() {
		parseT.Fatalf("expected zero channel wrapper state, got value=%q ok=%v closed=%v", parseChannel.Get(), parseChannel.Ok(), parseChannel.Closed())
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
