package runtime

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestComponentSignatureHelpers(parseT *testing.T) {
	parseBase := ComponentSignature{
		Kind:          "component",
		Name:          "Widget",
		QualifiedName: "example.com/widget.Widget",
		Key:           "widget-key",
		HookKinds:     []string{"state", "effect"},
	}
	if !parseBase.CompatibleWith(parseBase) {
		parseT.Fatal("expected identical signatures to be compatible")
	}
	if parseBase.CompatibleWith(ComponentSignature{
		Kind:          "component",
		Name:          "Widget",
		QualifiedName: "example.com/widget.Widget",
		Key:           "widget-key",
		HookKinds:     []string{"state"},
	}) {
		parseT.Fatal("expected differing hook shapes to be incompatible")
	}

	parseSummary := parseBase.Summary()
	if !strings.Contains(parseSummary, "Widget") || !strings.Contains(parseSummary, "key=widget-key") || !strings.Contains(parseSummary, "hooks=state > effect") {
		parseT.Fatalf("unexpected component summary: %q", parseSummary)
	}
	if parseGot := (ComponentSignature{}).Summary(); parseGot != "unknown" {
		parseT.Fatalf("expected empty signature summary to fall back to unknown, got %q", parseGot)
	}

	if parseGot2 := trimCallableName("github.com/acme/project/pkg.Widget.Render"); parseGot2 != "Render" {
		parseT.Fatalf("expected callable name trim to return Render, got %q", parseGot2)
	}
}

func TestBuildComponentSignatureAndHookSignature(parseT *testing.T) {
	recordHookSignature(nil, "state")
	parseHooks := &Hooks{}
	recordHookSignature(parseHooks, "")
	recordHookSignature(parseHooks, "state")
	if len(parseHooks.signature) != 1 || parseHooks.signature[0] != "state" {
		parseT.Fatalf("unexpected hook signature capture: %#v", parseHooks.signature)
	}

	if parseGot := buildComponentSignature(nil, parseHooks); parseGot != nil {
		parseT.Fatalf("expected nil fiber signature to be nil, got %+v", parseGot)
	}
	if parseGot2 := buildComponentSignature(&Fiber{typeOf: "div"}, parseHooks); parseGot2 != nil {
		parseT.Fatalf("expected host fiber signature to be nil, got %+v", parseGot2)
	}

	parseComponent := NewComponentType("component-id", "Widget", "example.com/widget.Widget", nil, nil)
	parseFiber := &Fiber{
		typeOf: parseComponent,
		props:  map[string]interface{}{"key": "stable-key"},
	}
	parseSignature := buildComponentSignature(parseFiber, &Hooks{signature: []string{"state", "memo"}})
	if parseSignature == nil {
		parseT.Fatal("expected component signature to be built")
	}
	if parseSignature.Name != "Widget" || parseSignature.QualifiedName != "component-id" || parseSignature.Key != "stable-key" {
		parseT.Fatalf("unexpected component signature identity fields: %+v", parseSignature)
	}
	if len(parseSignature.HookKinds) != 2 || parseSignature.HookKinds[0] != "state" {
		parseT.Fatalf("unexpected component signature hooks: %+v", parseSignature.HookKinds)
	}
}

func TestDescribeCallableIdentityFallbacks(parseT *testing.T) {
	parsePretty, parseQualified := describeCallableIdentity(nil)
	if parsePretty != "nil" || parseQualified != "" {
		parseT.Fatalf("unexpected nil callable identity (%q, %q)", parsePretty, parseQualified)
	}

	parseComponent := NewComponentType("stable-id", "", "", nil, nil)
	parsePretty, parseQualified = describeCallableIdentity(parseComponent)
	if parsePretty != "stable-id" || parseQualified != "stable-id" {
		parseT.Fatalf("unexpected component callable identity (%q, %q)", parsePretty, parseQualified)
	}

	parsePretty, parseQualified = describeCallableIdentity(42)
	if parsePretty != "int" || parseQualified != "int" {
		parseT.Fatalf("unexpected scalar callable identity (%q, %q)", parsePretty, parseQualified)
	}

	parsePretty, parseQualified = describeCallableIdentity(func() {})
	if parsePretty == "" || parseQualified == "" {
		parseT.Fatalf("expected function callable identity fields to be non-empty, got (%q, %q)", parsePretty, parseQualified)
	}
}

func TestProfilingStartupHelpers(parseT *testing.T) {
	parsePrevious := globalRuntime
	parseT.Cleanup(func() {
		globalRuntime = parsePrevious
	})

	parseRt := NewRuntime(Config{Scheduler: newTestScheduler()})
	globalRuntime = parseRt

	BeginStartupProfiling("")
	if parseRt.profiling.startupMode != "render" {
		parseT.Fatalf("expected default startup mode render, got %q", parseRt.profiling.startupMode)
	}
	if parseRt.profiling.startupStartedAt.IsZero() {
		parseT.Fatal("expected startup profiling timestamp to be set")
	}
	if len(parseRt.profiling.events) == 0 || parseRt.profiling.events[0].Name != "startup" {
		parseT.Fatalf("expected startup profiling start event, got %+v", parseRt.profiling.events)
	}

	RecordStartupBootstrapRead(-25, " bootstrap ")
	if parseRt.profiling.bootstrapReadDurationNs != 0 {
		parseT.Fatalf("expected negative bootstrap duration to clamp to zero, got %d", parseRt.profiling.bootstrapReadDurationNs)
	}

	RecordStartupRouteContext("users/42?tab=billing#anchor")
	if parseRt.profiling.startupRoutePath != "/users/42" {
		parseT.Fatalf("expected normalized startup route path /users/42, got %q", parseRt.profiling.startupRoutePath)
	}
	if parseRt.profiling.startupRouteFamily != "/users/*" {
		parseT.Fatalf("expected startup route family /users/*, got %q", parseRt.profiling.startupRouteFamily)
	}

	StoreStartupCostAttribution(StartupCostAttribution{
		WASMTransferBytes:       1200,
		WASMDecodedBytes:        2400,
		BootstrapDecodedBytes:   640,
		CacheWarmupDurationNs:   int64(2 * time.Millisecond),
		ServiceWorkerOverheadNs: int64(1 * time.Millisecond),
		InitialRouteDataBytes:   320,
	})
	if parseRt.profiling.startupWASMTransferBytes != 1200 || parseRt.profiling.startupBootstrapDecodedBytes != 640 || parseRt.profiling.startupInitialRouteDataBytes != 320 {
		parseT.Fatalf("expected startup attribution to be captured, got %+v", parseRt.profiling)
	}

	parseRt.profiling.hydrationDurationNs = int64(3 * time.Millisecond)
	parseRt.profiling.startupCommitDurationNs = int64(4 * time.Millisecond)
	parseRt.profiling.startupStartedAt = time.Now().Add(-5 * time.Millisecond)
	parseBefore := len(parseRt.profiling.events)
	parseRt.recordFirstInteraction("click")
	if !parseRt.profiling.firstInteractionCaptured || parseRt.profiling.firstInteractionEvent != "click" {
		parseT.Fatalf("expected first interaction capture, got %+v", parseRt.profiling)
	}
	if parseRt.profiling.firstInteractionDurationNs <= 0 {
		parseT.Fatalf("expected positive first interaction duration, got %d", parseRt.profiling.firstInteractionDurationNs)
	}
	parseBudget := parseRt.profiling.routeStartupBudgets["/users/*"]
	if parseBudget == nil || parseBudget.SampleCount != 1 || parseBudget.LastRoutePath != "/users/42" {
		parseT.Fatalf("expected route startup budget for /users/*, got %+v", parseRt.profiling.routeStartupBudgets)
	}
	if parseBudget.WASMTransferBytesTotal != 1200 || parseBudget.BootstrapDecodedBytesTotal != 640 || parseBudget.InitialRouteDataBytesTotal != 320 {
		parseT.Fatalf("expected route startup budget to include attribution totals, got %+v", parseBudget)
	}
	parseRt.recordFirstInteraction("keydown")
	if len(parseRt.profiling.events) != parseBefore+1 {
		parseT.Fatalf("expected first interaction to be captured once, got %d events", len(parseRt.profiling.events))
	}

	ClearProfiling()
	if len(parseRt.profiling.events) != 0 || parseRt.profiling.startupMode != "" {
		parseT.Fatalf("expected clear profiling to reset state, got %+v", parseRt.profiling)
	}
}

func TestProfilingRoutePathHelpers(parseT *testing.T) {
	if parseGot := normalizeRoutePathForBudget(""); parseGot != "/" {
		parseT.Fatalf("expected empty route path to normalize to root, got %q", parseGot)
	}
	if parseGot2 := normalizeRoutePathForBudget("users/7?tab=a#hash"); parseGot2 != "/users/7" {
		parseT.Fatalf("expected query/hash removal in route path normalization, got %q", parseGot2)
	}
	if parseGot3 := routeFamilyForPath("/"); parseGot3 != "/" {
		parseT.Fatalf("expected root family for root path, got %q", parseGot3)
	}
	if parseGot4 := routeFamilyForPath("/settings"); parseGot4 != "/settings" {
		parseT.Fatalf("expected one-segment family for /settings, got %q", parseGot4)
	}
	if parseGot5 := routeFamilyForPath("/users/42/profile"); parseGot5 != "/users/*" {
		parseT.Fatalf("expected wildcard family for nested route, got %q", parseGot5)
	}
}

func TestRuntimeStateHelperBranches(parseT *testing.T) {
	var parseNilRuntime *Runtime
	parseNilRuntime.SetIDSeed(5)
	if parseSnapshot := parseNilRuntime.SnapshotAtoms(); len(parseSnapshot) != 0 {
		parseT.Fatalf("expected nil runtime snapshot to be empty, got %#v", parseSnapshot)
	}

	parseRt := NewRuntime(Config{Scheduler: newTestScheduler()})
	parseRt.SetIDSeed(-1)
	parseRt.idCounterMu.Lock()
	if parseRt.idCounter != 0 {
		parseT.Fatalf("expected negative seed to be ignored, got %d", parseRt.idCounter)
	}
	parseRt.idCounterMu.Unlock()

	parseRt.SetIDSeed(7)
	parseRt.SetIDSeed(2)
	parseRt.idCounterMu.Lock()
	if parseRt.idCounter != 7 {
		parseT.Fatalf("expected id seed to advance monotonically, got %d", parseRt.idCounter)
	}
	parseRt.idCounterMu.Unlock()

	if parseErr := parseRt.SetAtomValue("theme", "dark"); parseErr != nil {
		parseT.Fatalf("unexpected atom set error: %v", parseErr)
	}
	parseSnapshot2 := parseRt.SnapshotAtoms()
	if parseSnapshot2["theme"] != "dark" {
		parseT.Fatalf("expected snapshot to include atom value, got %#v", parseSnapshot2)
	}

	parseRt.atomRegistry = nil
	if parseSnapshot3 := parseRt.SnapshotAtoms(); len(parseSnapshot3) != 0 {
		parseT.Fatalf("expected snapshot to be empty when atom registry is nil, got %#v", parseSnapshot3)
	}
}

func TestAtomRegistryMoveSubscriptionAndSnapshotAtoms(parseT *testing.T) {
	var parseNilRegistry *AtomRegistry
	parseNilRegistry.MoveSubscription("theme", nil, nil)

	parseRegistry := NewAtomRegistry()
	parseFrom := &Fiber{typeOf: "from"}
	parseTo := &Fiber{typeOf: "to"}

	parseRegistry.Subscribe("theme", parseFrom)
	parseRegistry.MoveSubscription("theme", parseFrom, parseTo)
	if parseRegistry.GetSubscriberCount("theme") != 1 {
		parseT.Fatalf("expected single moved theme subscriber, got %d", parseRegistry.GetSubscriberCount("theme"))
	}

	parseRegistry.MoveSubscription("theme", parseTo, nil)
	if parseRegistry.GetSubscriberCount("theme") != 0 {
		parseT.Fatalf("expected move to nil target to clear subscription, got %d", parseRegistry.GetSubscriberCount("theme"))
	}
	parseRegistry.MoveSubscription("", parseFrom, parseTo)
}

func TestDiagnosticAndPanicHelperBranches(parseT *testing.T) {
	if parseOutput := actionableContextDescriptorNilPanic("GoUseContext"); !strings.Contains(parseOutput, "GWC-UI-CONTEXT-NIL") {
		parseT.Fatalf("expected actionable context panic code, got %q", parseOutput)
	}
	if parseOutput2 := actionableGoUseAtomAccessorPanic(); !strings.Contains(parseOutput2, "GWC-RUNTIME-ATOM-ACCESSOR-MISMATCH") {
		parseT.Fatalf("expected actionable atom accessor panic code, got %q", parseOutput2)
	}
	if parseMessage := panicDiagnosticMessage(nil, PanicPhaseRender, "boom"); !strings.Contains(parseMessage, "uncaught render panic in application: boom") {
		parseT.Fatalf("unexpected panic diagnostic message: %q", parseMessage)
	}

	parseReport := formatPanicReport(panicReportContext{
		Source:      "runtime",
		Phase:       PanicPhaseRender,
		Subject:     "Widget",
		Path:        "Widget",
		Summary:     "boom",
		Code:        "GWC-RUNTIME-PANIC-RENDER",
		Docs:        "ACTIONABLE_ERRORS.md#gwc-runtime-panic-render",
		Remediation: "fix",
		Consequence: "stopped",
	})
	if !strings.Contains(parseReport, "GWC-RUNTIME-PANIC-RENDER") || !strings.Contains(parseReport, "uncaught render panic in Widget") {
		parseT.Fatalf("unexpected formatted panic report: %q", parseReport)
	}

	if recoveredAsError(nil) != nil {
		parseT.Fatal("expected nil recovered value to stay nil")
	}
	if parseErr := recoveredAsError(errors.New("boom")); parseErr == nil || parseErr.Error() != "boom" {
		parseT.Fatalf("expected passthrough recovered error, got %v", parseErr)
	}
	if parseErr2 := recoveredAsError("boom"); parseErr2 == nil || parseErr2.Error() != "boom" {
		parseT.Fatalf("expected string recovered value to convert to error, got %v", parseErr2)
	}
	if parseErr3 := recoveredAsError(42); parseErr3 == nil || parseErr3.Error() != "42" {
		parseT.Fatalf("expected scalar recovered value to convert to error, got %v", parseErr3)
	}

	withPanicLoggingOptions(parseT, PanicLoggingOptions{HideRawPanicOutput: true})
	parseValue, parseSwallowed := FinalizeUnhandledPanicContext("runtime", PanicPhaseRender, "Widget", "Widget", []string{"Widget"}, "boom")
	if !parseSwallowed || parseValue != "boom" {
		parseT.Fatalf("expected hidden raw panic output branch to swallow recovered value, got (%v, %t)", parseValue, parseSwallowed)
	}

	parseWrapped := markUnhandledPanicContext("runtime", PanicPhaseRender, "Widget", "Widget", []string{"Widget"}, "wrapped boom")
	parseValue, parseSwallowed = FinalizeUnhandledPanicContext("runtime", PanicPhaseRender, "Widget", "Widget", []string{"Widget"}, parseWrapped)
	if !parseSwallowed || parseValue != "wrapped boom" {
		parseT.Fatalf("expected wrapped panic finalize branch to return original recovered value, got (%v, %t)", parseValue, parseSwallowed)
	}
}
