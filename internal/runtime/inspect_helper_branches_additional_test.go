package runtime

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

// TestContextHelperBranchesCoverPanicsAndRepair covers direct context hook helper branches.
func TestContextHelperBranchesCoverPanicsAndRepair(parseT *testing.T) {
	parsePreviousFiber := GetCurrentFiber()
	defer SetCurrentFiber(parsePreviousFiber)

	parseDescriptor := NewContextDescriptor("fallback")
	parseFiber := &Fiber{
		contextValues: map[int64]interface{}{parseDescriptor.ID: "provided"},
		hooks:         &Hooks{},
	}

	SetCurrentFiber(parseFiber)
	if parseValue := GoUseContextValue(parseDescriptor); parseValue != "provided" {
		parseT.Fatalf("expected context value to resolve from fiber, got %v", parseValue)
	}
	if parseFiber.hooks.owner != parseFiber {
		parseT.Fatalf("expected missing hooks owner to be repaired, got %#v", parseFiber.hooks.owner)
	}
	if parseFiber.hooks.index != 1 {
		parseT.Fatalf("expected hook index incremented once, got %d", parseFiber.hooks.index)
	}

	SetCurrentFiber(nil)
	func() {
		defer func() {
			parseRecovered := recover()
			if parseRecovered == nil || !strings.Contains(parseRecovered.(string), "GoUseContextValue") {
				parseT.Fatalf("expected actionable hook panic, got %v", parseRecovered)
			}
		}()
		GoUseContextValue(parseDescriptor)
	}()

	SetCurrentFiber(parseFiber)
	func() {
		defer func() {
			parseRecovered := recover()
			if parseRecovered == nil || !strings.Contains(parseRecovered.(string), "GoUseContextValue") {
				parseT.Fatalf("expected actionable context descriptor panic, got %v", parseRecovered)
			}
		}()
		GoUseContextValue(nil)
	}()
}

// TestInspectHelperBranchesCoverLogDefaultsAndClassification covers log defaults and classification branches.
func TestInspectHelperBranchesCoverLogDefaultsAndClassification(parseT *testing.T) {
	ClearLogs()
	defer ClearLogs()

	ReportLog("", "", "  helper log  ")
	ReportLog("runtime", LogWarn, "")

	parseLogs := GetLogs()
	if len(parseLogs) != 1 {
		parseT.Fatalf("expected one non-empty log entry, got %+v", parseLogs)
	}
	if parseLogs[0].Domain != "runtime" || parseLogs[0].Level != LogInfo {
		parseT.Fatalf("expected default runtime/info log entry, got %+v", parseLogs[0])
	}
	if parseLogs[0].Classification != DiagnosticInformational || parseLogs[0].Message != "helper log" {
		parseT.Fatalf("expected default classification and trimmed message, got %+v", parseLogs[0])
	}

	if parseClassification := classifyDiagnostic("runtime", DiagnosticInfo, "plain info"); parseClassification != DiagnosticInformational {
		parseT.Fatalf("expected informational classification, got %q", parseClassification)
	}
	if parseClassification := classifyDiagnostic("runtime", DiagnosticWarning, "slow render on App"); parseClassification != DiagnosticPerformance {
		parseT.Fatalf("expected performance classification, got %q", parseClassification)
	}
	if parseClassification := classifyDiagnostic("router", DiagnosticWarning, "redirect loop detected"); parseClassification != DiagnosticUnsupportedRecover {
		parseT.Fatalf("expected unsupported recovered classification, got %q", parseClassification)
	}
	if parseClassification := classifyDiagnostic("runtime", DiagnosticWarning, "plain warning"); parseClassification != DiagnosticCorrectness {
		parseT.Fatalf("expected correctness classification, got %q", parseClassification)
	}
	if parseClassification := classifyDiagnostic("runtime", DiagnosticError, "boom"); parseClassification != DiagnosticCorrectness {
		parseT.Fatalf("expected correctness classification for errors, got %q", parseClassification)
	}
}

// TestInspectHelperBranchesCoverTraceAndBudgetHelpers covers filtered helper snapshots and ordering.
func TestInspectHelperBranchesCoverTraceAndBudgetHelpers(parseT *testing.T) {
	if parseTraces := collectComponentRenderTraces(nil, 4); parseTraces != nil {
		parseT.Fatalf("expected nil traces for empty entries, got %+v", parseTraces)
	}
	if parseBudgets := buildRouteStartupBudgets(map[string]*routeStartupBudget{}, 0); parseBudgets != nil {
		parseT.Fatalf("expected nil budgets for zero limit, got %+v", parseBudgets)
	}

	parseTraces := collectComponentRenderTraces(map[string]*componentRenderTrace{
		"skip": nil,
		"slow": {
			Name:                  "Hero",
			Path:                  "ROOT > Hero",
			RenderCount:           2,
			RerenderCount:         1,
			LastTrigger:           "props",
			LastRenderDurationNs:  12,
			TotalRenderDurationNs: 20,
			TriggerCounts:         map[string]int{"mount": 1, "props": 1},
			LastRenderedAt:        "2026-03-26T00:00:00.000Z",
		},
		"fast": {
			Name:                  "Sidebar",
			Path:                  "ROOT > Sidebar",
			RenderCount:           2,
			TotalRenderDurationNs: 10,
			TriggerCounts:         map[string]int{"mount": 2},
		},
		"tiny": {
			Name:                  "Footer",
			Path:                  "ROOT > Footer",
			RenderCount:           1,
			TotalRenderDurationNs: 9,
			TriggerCounts:         map[string]int{"mount": 1},
		},
	}, 2)
	if len(parseTraces) != 2 {
		parseT.Fatalf("expected truncated component traces, got %+v", parseTraces)
	}
	if parseTraces[0].Name != "Hero" || parseTraces[0].AverageRenderDurationNs != 10 {
		parseT.Fatalf("expected highest total render trace first, got %+v", parseTraces[0])
	}
	if parseTraces[0].TriggerCounts["props"] != 1 {
		parseT.Fatalf("expected trigger counts to be cloned, got %+v", parseTraces[0].TriggerCounts)
	}
	if parseTraces[1].Name != "Sidebar" || parseTraces[1].AverageRenderDurationNs != 5 {
		parseT.Fatalf("expected second render trace average to round-trip, got %+v", parseTraces[1])
	}

	parseBudgets := buildRouteStartupBudgets(map[string]*routeStartupBudget{
		"skip_nil":   nil,
		"skip_blank": {RouteFamily: "", SampleCount: 1},
		"skip_zero":  {RouteFamily: "/skip", SampleCount: 0},
		"reports": {
			RouteFamily:                     "/reports/*",
			LastRoutePath:                   "/reports/42",
			SampleCount:                     2,
			BootstrapReadDurationTotalNs:    6,
			HydrationDurationTotalNs:        10,
			StartupCommitDurationTotalNs:    8,
			FirstInteractionDurationTotalNs: 60,
		},
		"account": {
			RouteFamily:                     "/account/*",
			LastRoutePath:                   "/account/profile",
			SampleCount:                     3,
			BootstrapReadDurationTotalNs:    9,
			HydrationDurationTotalNs:        9,
			StartupCommitDurationTotalNs:    6,
			FirstInteractionDurationTotalNs: 45,
		},
	}, 2)
	if len(parseBudgets) != 2 {
		parseT.Fatalf("expected filtered startup budgets, got %+v", parseBudgets)
	}
	if parseBudgets[0].RouteFamily != "/reports/*" || parseBudgets[0].AverageFirstInteractionDurationNs != 30 {
		parseT.Fatalf("expected slowest budget first, got %+v", parseBudgets[0])
	}
	if parseBudgets[1].RouteFamily != "/account/*" || parseBudgets[1].AverageBootstrapReadDurationNs != 3 {
		parseT.Fatalf("expected averaged account budget second, got %+v", parseBudgets[1])
	}
}

// TestInspectHelperBranchesCoverDescriptionsPathsAndValues covers direct inspect helpers.
func TestInspectHelperBranchesCoverDescriptionsPathsAndValues(parseT *testing.T) {
	parsePreviousFiber := GetCurrentFiber()
	defer SetCurrentFiber(parsePreviousFiber)

	parseRoot := &Fiber{typeOf: "ROOT"}
	parseBoundary := &Fiber{typeOf: NewErrorBoundaryType(), parent: parseRoot}
	parseComponent := &Fiber{typeOf: &ComponentType{Name: "Hero"}, parent: parseBoundary}

	SetCurrentFiber(parseComponent)
	if parsePath := CurrentFiberPath(); parsePath != "ErrorBoundary > Hero" {
		parseT.Fatalf("expected current fiber path, got %q", parsePath)
	}

	if parseKind, parseName := describeFiber(nil); parseKind != "unknown" || parseName != "unknown" {
		parseT.Fatalf("expected unknown fiber description, got %q %q", parseKind, parseName)
	}
	if parseKind, parseName := describeFiber(&Fiber{typeOf: "TEXT_ELEMENT", textContent: "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"}); parseKind != "text" || !strings.Contains(parseName, "...") {
		parseT.Fatalf("expected trimmed text description, got %q %q", parseKind, parseName)
	}
	if parseKind, parseName := describeFiber(&Fiber{typeOf: NewContextProviderType(NewContextDescriptor(nil))}); parseKind != "provider" || parseName != "ContextProvider" {
		parseT.Fatalf("expected provider description, got %q %q", parseKind, parseName)
	}
	if parseKind, parseName := describeFiber(&Fiber{typeOf: ReactiveRegionNodeType}); parseKind != "region" || parseName != "ReactiveRegion" {
		parseT.Fatalf("expected reactive region description, got %q %q", parseKind, parseName)
	}
	if parseKind, parseName := describeFiber(&Fiber{typeOf: &ComponentType{QualifiedName: "pkg.Widget"}}); parseKind != "component" || parseName != "pkg.Widget" {
		parseT.Fatalf("expected qualified component description, got %q %q", parseKind, parseName)
	}
	if parseKind, parseName := describeFiber(&Fiber{typeOf: func() *Element { return nil }}); parseKind != "component" || strings.TrimSpace(parseName) == "" {
		parseT.Fatalf("expected callable component description, got %q %q", parseKind, parseName)
	}

	parseBuffer := &bytes.Buffer{}
	parseBuffer.WriteString("buffer")
	if parseValue := previewValue("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"); !strings.Contains(parseValue, "...") {
		parseT.Fatalf("expected trimmed string preview, got %q", parseValue)
	}
	if parseValue := previewValue(parseBuffer); parseValue != "buffer" {
		parseT.Fatalf("expected stringer preview, got %q", parseValue)
	}
	if parseValue := previewValue(errors.New("preview boom")); parseValue != "preview boom" {
		parseT.Fatalf("expected error preview, got %q", parseValue)
	}
	if parseValue := previewValue([]int{1, 2}); parseValue != "[]int(len=2)" {
		parseT.Fatalf("expected slice preview, got %q", parseValue)
	}
	if parseValue := previewValue(map[string]int{"a": 1}); parseValue != "map[string]int(len=1)" {
		parseT.Fatalf("expected map preview, got %q", parseValue)
	}
	if parseValue := previewValue(struct{ Count int }{Count: 2}); !strings.Contains(parseValue, "struct { Count int }") {
		parseT.Fatalf("expected struct preview, got %q", parseValue)
	}
	var parseNumber *int
	if parseValue := previewValue(parseNumber); parseValue != "*int(nil)" {
		parseT.Fatalf("expected nil pointer preview, got %q", parseValue)
	}
	if parseValue := previewValue(func() {}); strings.TrimSpace(parseValue) == "" {
		parseT.Fatalf("expected function preview, got %q", parseValue)
	}
}

// TestInspectHelperBranchesCoverHookSnapshots covers hook snapshot branches that were previously skipped.
func TestInspectHelperBranchesCoverHookSnapshots(parseT *testing.T) {
	parseHooks := &Hooks{
		refs: []*RefValue{
			nil,
			{Current: map[string]int{"a": 1}},
		},
		ids:   []string{"id-1"},
		atoms: []string{"atom-1"},
		callbacks: []callbackValue{
			{fn: func() {}, deps: []interface{}{"query"}},
		},
		fetches: []fetchValue{
			{url: "/loading", state: FetchState{Loading: true}},
			{url: "/error", state: FetchState{Error: "boom"}},
			{url: "/ready", state: FetchState{Data: map[string]bool{"ok": true}}},
		},
		deps: [][]interface{}{
			{true},
		},
	}

	parseSnapshots := inspectHooks(parseHooks)
	if len(parseSnapshots) != 9 {
		parseT.Fatalf("expected 9 hook snapshots, got %+v", parseSnapshots)
	}

	isParseFoundNilRef := false
	isParseFoundReadyFetch := false
	isParseFoundErrorFetch := false
	isParseFoundLoadingFetch := false
	isParseFoundEffect := false
	isParseFoundAtom := false
	for _, parseSnapshot := range parseSnapshots {
		switch {
		case parseSnapshot.Kind == "ref" && parseSnapshot.Value == "<nil>":
			isParseFoundNilRef = true
		case parseSnapshot.Kind == "fetch" && parseSnapshot.Status == "ready":
			isParseFoundReadyFetch = true
		case parseSnapshot.Kind == "fetch" && parseSnapshot.Status == "error":
			isParseFoundErrorFetch = true
		case parseSnapshot.Kind == "fetch" && parseSnapshot.Status == "loading":
			isParseFoundLoadingFetch = true
		case parseSnapshot.Kind == "effect" && parseSnapshot.Status == "cleanup=none epoch=0":
			isParseFoundEffect = true
		case parseSnapshot.Kind == "atom" && parseSnapshot.Value == "atom-1":
			isParseFoundAtom = true
		}
	}
	if !isParseFoundNilRef || !isParseFoundLoadingFetch || !isParseFoundErrorFetch || !isParseFoundReadyFetch || !isParseFoundEffect || !isParseFoundAtom {
		parseT.Fatalf("expected hook snapshot branches to be present, got %+v", parseSnapshots)
	}
}

// TestBoundaryHelperBranchesCoverFallbacksAndReset covers direct error-boundary helper branches.
func TestBoundaryHelperBranchesCoverFallbacksAndReset(parseT *testing.T) {
	ClearDiagnostics()
	defer ClearDiagnostics()

	parseRt := &Runtime{}
	parseStaticFallback := &Element{Type: "span", Props: map[string]interface{}{"children": emptyChildren}, Children: emptyChildren}
	parseStaticBoundary := &Fiber{props: map[string]interface{}{"fallback": parseStaticFallback}}
	if parseFallback := parseRt.renderBoundaryFallback(parseStaticBoundary, errors.New("boom")); parseFallback != parseStaticFallback {
		parseT.Fatalf("expected static fallback to be returned, got %#v", parseFallback)
	}

	parseBoundary := &Fiber{
		typeOf: NewErrorBoundaryType(),
		props: map[string]interface{}{
			"onError": func(error) { panic("callback boom") },
		},
		parent: &Fiber{typeOf: "ROOT"},
	}
	parseRt.invokeBoundaryOnError(parseBoundary, errors.New("render boom"))
	parseDiagnostics := GetDiagnostics()
	if len(parseDiagnostics) == 0 || !strings.Contains(parseDiagnostics[len(parseDiagnostics)-1].Message, "onError callback panicked") {
		parseT.Fatalf("expected onError panic diagnostic, got %+v", parseDiagnostics)
	}

	parseAlternate := &Fiber{boundaryError: errors.New("old"), boundaryPhase: "effect"}
	parseResetBoundary := &Fiber{
		boundaryError: errors.New("old"),
		boundaryPhase: "effect",
		alternate:     parseAlternate,
	}
	parseRt.updateScheduled = true
	parseRt.resetBoundary(parseResetBoundary)
	if parseResetBoundary.boundaryError != nil || parseResetBoundary.boundaryPhase != "" {
		parseT.Fatalf("expected boundary state cleared, got %#v", parseResetBoundary)
	}
	if parseAlternate.boundaryError != nil || parseAlternate.boundaryPhase != "" {
		parseT.Fatalf("expected alternate boundary state cleared, got %#v", parseAlternate)
	}
	if !parseResetBoundary.dirty || !parseResetBoundary.needsUpdate || parseResetBoundary.updateOrigin != "error-boundary" {
		parseT.Fatalf("expected boundary recovery scheduling metadata, got %#v", parseResetBoundary)
	}
	if !parseRt.pendingBoundaryRecovery {
		parseT.Fatal("expected pending boundary recovery when update already scheduled")
	}
}
