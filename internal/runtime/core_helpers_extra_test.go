package runtime

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestComponentSignatureHelpers(t *testing.T) {
	base := ComponentSignature{
		Kind:          "component",
		Name:          "Widget",
		QualifiedName: "example.com/widget.Widget",
		Key:           "widget-key",
		HookKinds:     []string{"state", "effect"},
	}
	if !base.CompatibleWith(base) {
		t.Fatal("expected identical signatures to be compatible")
	}
	if base.CompatibleWith(ComponentSignature{
		Kind:          "component",
		Name:          "Widget",
		QualifiedName: "example.com/widget.Widget",
		Key:           "widget-key",
		HookKinds:     []string{"state"},
	}) {
		t.Fatal("expected differing hook shapes to be incompatible")
	}

	summary := base.Summary()
	if !strings.Contains(summary, "Widget") || !strings.Contains(summary, "key=widget-key") || !strings.Contains(summary, "hooks=state > effect") {
		t.Fatalf("unexpected component summary: %q", summary)
	}
	if got := (ComponentSignature{}).Summary(); got != "unknown" {
		t.Fatalf("expected empty signature summary to fall back to unknown, got %q", got)
	}

	if got := trimCallableName("github.com/acme/project/pkg.Widget.Render"); got != "Render" {
		t.Fatalf("expected callable name trim to return Render, got %q", got)
	}
}

func TestBuildComponentSignatureAndHookSignature(t *testing.T) {
	recordHookSignature(nil, "state")
	hooks := &Hooks{}
	recordHookSignature(hooks, "")
	recordHookSignature(hooks, "state")
	if len(hooks.signature) != 1 || hooks.signature[0] != "state" {
		t.Fatalf("unexpected hook signature capture: %#v", hooks.signature)
	}

	if got := buildComponentSignature(nil, hooks); got != nil {
		t.Fatalf("expected nil fiber signature to be nil, got %+v", got)
	}
	if got := buildComponentSignature(&Fiber{typeOf: "div"}, hooks); got != nil {
		t.Fatalf("expected host fiber signature to be nil, got %+v", got)
	}

	component := NewComponentType("component-id", "Widget", "example.com/widget.Widget", nil, nil)
	fiber := &Fiber{
		typeOf: component,
		props:  map[string]interface{}{"key": "stable-key"},
	}
	signature := buildComponentSignature(fiber, &Hooks{signature: []string{"state", "memo"}})
	if signature == nil {
		t.Fatal("expected component signature to be built")
	}
	if signature.Name != "Widget" || signature.QualifiedName != "component-id" || signature.Key != "stable-key" {
		t.Fatalf("unexpected component signature identity fields: %+v", signature)
	}
	if len(signature.HookKinds) != 2 || signature.HookKinds[0] != "state" {
		t.Fatalf("unexpected component signature hooks: %+v", signature.HookKinds)
	}
}

func TestDescribeCallableIdentityFallbacks(t *testing.T) {
	pretty, qualified := describeCallableIdentity(nil)
	if pretty != "nil" || qualified != "" {
		t.Fatalf("unexpected nil callable identity (%q, %q)", pretty, qualified)
	}

	component := NewComponentType("stable-id", "", "", nil, nil)
	pretty, qualified = describeCallableIdentity(component)
	if pretty != "stable-id" || qualified != "stable-id" {
		t.Fatalf("unexpected component callable identity (%q, %q)", pretty, qualified)
	}

	pretty, qualified = describeCallableIdentity(42)
	if pretty != "int" || qualified != "int" {
		t.Fatalf("unexpected scalar callable identity (%q, %q)", pretty, qualified)
	}

	pretty, qualified = describeCallableIdentity(func() {})
	if pretty == "" || qualified == "" {
		t.Fatalf("expected function callable identity fields to be non-empty, got (%q, %q)", pretty, qualified)
	}
}

func TestProfilingStartupHelpers(t *testing.T) {
	previous := globalRuntime
	t.Cleanup(func() {
		globalRuntime = previous
	})

	rt := NewRuntime(Config{Scheduler: newTestScheduler()})
	globalRuntime = rt

	BeginStartupProfiling("")
	if rt.profiling.startupMode != "render" {
		t.Fatalf("expected default startup mode render, got %q", rt.profiling.startupMode)
	}
	if rt.profiling.startupStartedAt.IsZero() {
		t.Fatal("expected startup profiling timestamp to be set")
	}
	if len(rt.profiling.events) == 0 || rt.profiling.events[0].Name != "startup" {
		t.Fatalf("expected startup profiling start event, got %+v", rt.profiling.events)
	}

	RecordStartupBootstrapRead(-25, " bootstrap ")
	if rt.profiling.bootstrapReadDurationNs != 0 {
		t.Fatalf("expected negative bootstrap duration to clamp to zero, got %d", rt.profiling.bootstrapReadDurationNs)
	}

	RecordStartupRouteContext("users/42?tab=billing#anchor")
	if rt.profiling.startupRoutePath != "/users/42" {
		t.Fatalf("expected normalized startup route path /users/42, got %q", rt.profiling.startupRoutePath)
	}
	if rt.profiling.startupRouteFamily != "/users/*" {
		t.Fatalf("expected startup route family /users/*, got %q", rt.profiling.startupRouteFamily)
	}

	rt.profiling.hydrationDurationNs = int64(3 * time.Millisecond)
	rt.profiling.startupCommitDurationNs = int64(4 * time.Millisecond)
	rt.profiling.startupStartedAt = time.Now().Add(-5 * time.Millisecond)
	before := len(rt.profiling.events)
	rt.recordFirstInteraction("click")
	if !rt.profiling.firstInteractionCaptured || rt.profiling.firstInteractionEvent != "click" {
		t.Fatalf("expected first interaction capture, got %+v", rt.profiling)
	}
	if rt.profiling.firstInteractionDurationNs <= 0 {
		t.Fatalf("expected positive first interaction duration, got %d", rt.profiling.firstInteractionDurationNs)
	}
	budget := rt.profiling.routeStartupBudgets["/users/*"]
	if budget == nil || budget.SampleCount != 1 || budget.LastRoutePath != "/users/42" {
		t.Fatalf("expected route startup budget for /users/*, got %+v", rt.profiling.routeStartupBudgets)
	}
	rt.recordFirstInteraction("keydown")
	if len(rt.profiling.events) != before+1 {
		t.Fatalf("expected first interaction to be captured once, got %d events", len(rt.profiling.events))
	}

	ClearProfiling()
	if len(rt.profiling.events) != 0 || rt.profiling.startupMode != "" {
		t.Fatalf("expected clear profiling to reset state, got %+v", rt.profiling)
	}
}

func TestProfilingRoutePathHelpers(t *testing.T) {
	if got := normalizeRoutePathForBudget(""); got != "/" {
		t.Fatalf("expected empty route path to normalize to root, got %q", got)
	}
	if got := normalizeRoutePathForBudget("users/7?tab=a#hash"); got != "/users/7" {
		t.Fatalf("expected query/hash removal in route path normalization, got %q", got)
	}
	if got := routeFamilyForPath("/"); got != "/" {
		t.Fatalf("expected root family for root path, got %q", got)
	}
	if got := routeFamilyForPath("/settings"); got != "/settings" {
		t.Fatalf("expected one-segment family for /settings, got %q", got)
	}
	if got := routeFamilyForPath("/users/42/profile"); got != "/users/*" {
		t.Fatalf("expected wildcard family for nested route, got %q", got)
	}
}

func TestRuntimeStateHelperBranches(t *testing.T) {
	var nilRuntime *Runtime
	nilRuntime.SetIDSeed(5)
	if snapshot := nilRuntime.SnapshotAtoms(); len(snapshot) != 0 {
		t.Fatalf("expected nil runtime snapshot to be empty, got %#v", snapshot)
	}

	rt := NewRuntime(Config{Scheduler: newTestScheduler()})
	rt.SetIDSeed(-1)
	rt.idCounterMu.Lock()
	if rt.idCounter != 0 {
		t.Fatalf("expected negative seed to be ignored, got %d", rt.idCounter)
	}
	rt.idCounterMu.Unlock()

	rt.SetIDSeed(7)
	rt.SetIDSeed(2)
	rt.idCounterMu.Lock()
	if rt.idCounter != 7 {
		t.Fatalf("expected id seed to advance monotonically, got %d", rt.idCounter)
	}
	rt.idCounterMu.Unlock()

	if err := rt.SetAtomValue("theme", "dark"); err != nil {
		t.Fatalf("unexpected atom set error: %v", err)
	}
	snapshot := rt.SnapshotAtoms()
	if snapshot["theme"] != "dark" {
		t.Fatalf("expected snapshot to include atom value, got %#v", snapshot)
	}

	rt.atomRegistry = nil
	if snapshot := rt.SnapshotAtoms(); len(snapshot) != 0 {
		t.Fatalf("expected snapshot to be empty when atom registry is nil, got %#v", snapshot)
	}
}

func TestAtomRegistryMoveSubscriptionAndSnapshotAtoms(t *testing.T) {
	var nilRegistry *AtomRegistry
	nilRegistry.MoveSubscription("theme", nil, nil)

	registry := NewAtomRegistry()
	from := &Fiber{typeOf: "from"}
	to := &Fiber{typeOf: "to"}

	registry.Subscribe("theme", from)
	registry.MoveSubscription("theme", from, to)
	if registry.GetSubscriberCount("theme") != 1 {
		t.Fatalf("expected single moved theme subscriber, got %d", registry.GetSubscriberCount("theme"))
	}

	registry.MoveSubscription("theme", to, nil)
	if registry.GetSubscriberCount("theme") != 0 {
		t.Fatalf("expected move to nil target to clear subscription, got %d", registry.GetSubscriberCount("theme"))
	}
	registry.MoveSubscription("", from, to)
}

func TestDiagnosticAndPanicHelperBranches(t *testing.T) {
	if output := actionableContextDescriptorNilPanic("GoUseContext"); !strings.Contains(output, "GWC-UI-CONTEXT-NIL") {
		t.Fatalf("expected actionable context panic code, got %q", output)
	}
	if output := actionableGoUseAtomAccessorPanic(); !strings.Contains(output, "GWC-RUNTIME-ATOM-ACCESSOR-MISMATCH") {
		t.Fatalf("expected actionable atom accessor panic code, got %q", output)
	}
	if message := panicDiagnosticMessage(nil, PanicPhaseRender, "boom"); !strings.Contains(message, "uncaught render panic in application: boom") {
		t.Fatalf("unexpected panic diagnostic message: %q", message)
	}

	report := formatPanicReport(panicReportContext{
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
	if !strings.Contains(report, "GWC-RUNTIME-PANIC-RENDER") || !strings.Contains(report, "uncaught render panic in Widget") {
		t.Fatalf("unexpected formatted panic report: %q", report)
	}

	if recoveredAsError(nil) != nil {
		t.Fatal("expected nil recovered value to stay nil")
	}
	if err := recoveredAsError(errors.New("boom")); err == nil || err.Error() != "boom" {
		t.Fatalf("expected passthrough recovered error, got %v", err)
	}
	if err := recoveredAsError("boom"); err == nil || err.Error() != "boom" {
		t.Fatalf("expected string recovered value to convert to error, got %v", err)
	}
	if err := recoveredAsError(42); err == nil || err.Error() != "42" {
		t.Fatalf("expected scalar recovered value to convert to error, got %v", err)
	}

	withPanicLoggingOptions(t, PanicLoggingOptions{HideRawPanicOutput: true})
	value, swallowed := FinalizeUnhandledPanicContext("runtime", PanicPhaseRender, "Widget", "Widget", []string{"Widget"}, "boom")
	if !swallowed || value != "boom" {
		t.Fatalf("expected hidden raw panic output branch to swallow recovered value, got (%v, %t)", value, swallowed)
	}

	wrapped := markUnhandledPanicContext("runtime", PanicPhaseRender, "Widget", "Widget", []string{"Widget"}, "wrapped boom")
	value, swallowed = FinalizeUnhandledPanicContext("runtime", PanicPhaseRender, "Widget", "Widget", []string{"Widget"}, wrapped)
	if !swallowed || value != "wrapped boom" {
		t.Fatalf("expected wrapped panic finalize branch to return original recovered value, got (%v, %t)", value, swallowed)
	}
}

