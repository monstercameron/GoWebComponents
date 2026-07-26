package runtime

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	globalRuntime   *Runtime
	globalRuntimeMu sync.Mutex
)

// GetGlobalRuntime returns the global Runtime instance, creating it if needed.
// For WASM builds, this can be upgraded later by InitGlobalRuntime.
func GetGlobalRuntime() *Runtime {
	globalRuntimeMu.Lock()
	defer globalRuntimeMu.Unlock()

	if globalRuntime == nil {
		// This can be lazily upgraded later by InitGlobalRuntime.
		globalRuntime = &Runtime{
			atomRegistry: NewAtomRegistry(),
			deletions:    make([]*Fiber, 0),
		}
	}
	return globalRuntime
}

// InitGlobalRuntime initializes or upgrades the global runtime with specific adapters.
func InitGlobalRuntime(parseConfig Config) {
	globalRuntimeMu.Lock()
	defer globalRuntimeMu.Unlock()

	ConfigureUnhandledPanicLogging(PanicLoggingOptions{HideRawPanicOutput: configHidesRawPanicOutput(parseConfig), OnReport: parseConfig.OnUnhandledPanicReport})

	if globalRuntime == nil || parseConfig.Reset {
		globalRuntime = NewRuntime(parseConfig)
		return
	}

	applyRuntimeConfig(globalRuntime, parseConfig)
}

// Runtime represents the reconciliation and rendering engine.
type Runtime struct {
	domAdapter   DOMAdapter
	eventAdapter EventAdapter
	scheduler    Scheduler
	browserState BrowserState

	wipRoot     *Fiber
	currentRoot *Fiber
	// activeRenderFiber is the fiber whose component function is executing right
	// now (set only by renderFunctionComponent around the render call). A hook
	// setter that sees its owner == activeRenderFiber is a render-phase update and
	// converges in renderFunctionComponent instead of scheduling a commit. Unlike
	// the ambient currentFiber, this is never set by unit tests or the SSR path.
	activeRenderFiber *Fiber
	// renderPassActive marks a running work loop pass; while set,
	// renderPassOwnerID lazily captures the pass goroutine id once (a full
	// runtime.Stack traceback) so per-fiber SetCurrentFiber calls skip it.
	// Both fields are set and cleared only by workLoop on its own synchronous
	// call stack; the id is 0 whenever no component has rendered yet.
	renderPassActive           bool
	renderPassOwnerGoroutineID uint64
	// workLoopDepth guards synchronous discrete-event flushes: a flush may
	// only start a work loop when none is already running on this stack
	// (e.g. a focus handler fired synchronously by a commit-phase DOM write).
	// Maintained unconditionally by workLoop, unlike renderPassActive which
	// exists only for the dev threading guard.
	workLoopDepth        int
	nextUnitOfWork       *Fiber
	deletions            []*Fiber
	pendingEffectFibers  []*Fiber
	tracksPendingEffects bool
	// passiveEffectsAfterPaint enables the v5 P1.1 commit split: layout effects
	// stay synchronous inside the commit task, passive effects are deferred past
	// the paint boundary. Off by default (R2) until its acceptance test passes.
	passiveEffectsAfterPaint bool
	// passiveDrainScheduled guards against queueing more than one deferred
	// passive drain when several commits land before the first one runs.
	//
	// It also records that a drain is OWED. A scheduled callback that finds it
	// false has been overtaken by a flush and must not drain again — doing so
	// would fall through to the untracked path and re-run every passive effect
	// in the tree.
	passiveDrainScheduled bool
	// flushingPassiveBeforeSchedule guards the pre-pass flush against re-entry,
	// since a passive effect is free to schedule an update of its own.
	flushingPassiveBeforeSchedule bool
	// ready carries this runtime's first-commit signal (v5 P2.3). Per runtime,
	// so a second Runtime fires its own ready hooks rather than finding a
	// process-wide flag already set.
	ready readyState
	// laneQueues enables per-lane deferral with expiry (v5 P2.2). Off by
	// default (R2); when off, every dirty fiber renders in whatever pass finds
	// it, which is the pre-v5 behavior.
	laneQueues bool
	// asyncIngress routes state updates made OUTSIDE the frame loop through the
	// inbox instead of applying them where they are called (v5 P2.1). Off by
	// default, following laneQueues: turning it on changes when an async write
	// lands, so it stays opt-in until its acceptance test is the thing deciding.
	asyncIngress bool
	// frameLoopDepth counts frame-loop regions that are not the work loop —
	// event dispatch and inbox drains. workLoopDepth covers render and commit.
	// A setter that finds BOTH at zero was called from somewhere the runtime
	// does not control, which is exactly the case the inbox exists for.
	// Atomic, because these are not only touched by the frame loop. The inbox's
	// hard-overflow path drains on whatever goroutine is POSTING (inbox.go:
	// "nothing scheduled can have run, or the queue could not have reached this
	// size"), and that drain enters and leaves the frame-loop region — so a
	// producer goroutine mutates the renderer's own ownership state while the
	// renderer may be inside it. Unobservable in wasm, where there is one
	// thread; a genuine race in SSR and in the native suite. Atomics rather than
	// a mutex because insideFrameLoop is consulted on every off-loop state write.
	frameLoopDepth atomic.Int32
	// frameLoopOwner is the goroutine that entered the outermost frame-loop
	// region. A write from any other goroutine is async however deep the
	// counter is — see frame_loop_owner.go.
	frameLoopOwner atomic.Uint64
	// inbox holds async work posted from outside the frame loop (v5 P2.1).
	// Drained at one defined point per frame so N async messages produce one
	// render pass rather than N. See inbox.go.
	inbox asyncInbox
	// idleFallbackReported keeps the "RequestIdleCallback did not fire"
	// diagnostic to once per runtime, so a no-op Scheduler implementation
	// reports the problem instead of flooding the log every idle dispatch.
	idleFallbackReported bool
	// frameBudgetMs is the wall-clock slice budget for one work-loop pass
	// (v5 P1.2). Zero disables time-based slicing and keeps the count-only
	// behavior. Gated on interruptRestartIsSafe.
	frameBudgetMs float64
	// pendingInterruptLane records a higher-priority lane that arrived while a
	// pass was already walking the tree (v5 P2.5). The pass finishes and
	// commits; commitRoot then starts the higher lane. Zero means none pending.
	pendingInterruptLane    UpdateLane
	updateScheduled         bool
	continueWorkFn          func()
	pendingBoundaryRecovery bool
	// fiberSlab batches mount-path fiber allocation; see acquireMountFiber.
	fiberSlab     []Fiber
	fiberSlabNext int

	pendingEffectOverflow bool
	// pendingEffectOverflowCount counts lifetime overflows for the P4.2 budget
	// signal. A boolean alone cannot distinguish "tripped once during a bulk
	// import" from "trips every frame", which are different problems.
	pendingEffectOverflowCount int
	// transitionDepth is the TOTAL number of open transitions across all
	// goroutines. It is only a cheap gate: a state write asks "is any transition
	// open" first, and consults transitionOwners for "is MINE open" only when the
	// answer is yes. See transition.go.
	transitionDepth int
	// transitionOwners counts open transitions per goroutine, so one goroutine's
	// StartTransition cannot demote an unrelated goroutine's write to the
	// transition lane.
	transitionOwners   map[uint64]int
	pendingTransitions int
	transitionMu       sync.Mutex
	strictMode                 StrictModeOptions
	limits                     RuntimeLimits
	schedulerState             runtimeSchedulerState
	replay                     runtimeReplayState
	agentStateVersion          atomic.Uint64

	// Global state management
	atomRegistry *AtomRegistry

	// Hot reload restore queue for component-local state snapshots.
	pendingHotReloadComponents []HotReloadComponentSnapshot
	pendingHotReloadIndex      int
	pendingHotReloadByPath     map[string]HotReloadComponentSnapshot
	pendingHotReloadSelective  bool

	// Global ID counter for useId hook
	idCounter   int
	idCounterMu sync.Mutex

	// Hydration bookkeeping
	hydrating                      bool
	strictHydration                bool
	nextHydrationStrict            bool
	nextHydrationObserver          func(HydrationMetrics)
	nextHydrationCorrelationID     string
	hydrationMetrics               HydrationMetrics
	lastHydrationMetrics           HydrationMetrics
	hydrationMetricsActive         bool
	deferredHydrationUpdates       map[*Fiber]bool
	deferredHydrationSubscriptions []hydrationSubscriptionAction

	profiling runtimeProfiling
}

type hydrationSubscriptionAction struct {
	atomID    string
	fiber     *Fiber
	subscribe bool
}

// SetIDSeed is a core package helper.
func (parseRt *Runtime) SetIDSeed(parseSeed int) {
	if parseRt == nil || parseSeed < 0 {
		return
	}
	parseRt.idCounterMu.Lock()
	if parseSeed > parseRt.idCounter {
		parseRt.idCounter = parseSeed
	}
	parseRt.idCounterMu.Unlock()
}

// SetNextHydrationStrict configures whether the next hydration attempt should
// fail fast on mismatches instead of warning and falling back per subtree.
func (parseRt *Runtime) SetNextHydrationStrict(isStrict bool) {
	if parseRt == nil {
		return
	}
	parseRt.nextHydrationStrict = isStrict
}

type runtimeProfiling struct {
	renderCalls                      int
	scheduledRootUpdates             int
	scheduledFiberMarks              int
	scheduledGranularMarks           int
	workLoopPasses                   int
	processedUnits                   int
	commitCount                      int
	serializedMountRoots             int
	fineGrainedCommits               int
	fineGrainedDescendantHostCommits int
	fineGrainedDescendantTextCommits int
	effectExecutions                 int
	cleanupExecutions                int
	lastRenderDurationNs             int64
	lastCommitDurationNs             int64
	lastEffectDurationNs             int64
	lastCleanupDurationNs            int64
	totalRenderDurationNs            int64
	totalDiffDurationNs              int64
	totalCommitDurationNs            int64
	totalEffectDurationNs            int64
	totalCleanupDurationNs           int64
	events                           []ProfilingEvent
	componentRenders                 map[string]*componentRenderTrace
	startupMode                      string
	startupStartedAt                 time.Time
	bootstrapReadDurationNs          int64
	startupWASMTransferBytes         int64
	startupWASMDecodedBytes          int64
	startupBootstrapDecodedBytes     int64
	startupCacheWarmupDurationNs     int64
	startupServiceWorkerOverheadNs   int64
	startupInitialRouteDataBytes     int64
	hydrationDurationNs              int64
	startupCommitDurationNs          int64
	firstInteractionDurationNs       int64
	firstInteractionCaptured         bool
	firstInteractionEvent            string
	startupRoutePath                 string
	startupRouteFamily               string
	routeStartupBudgets              map[string]*routeStartupBudget
}

const slowOperationDiagnosticThresholdNs = int64(2 * time.Millisecond)

// formatRuntimeDurationNs is a core package helper.
func formatRuntimeDurationNs(parseDurationNs int64) string {
	if parseDurationNs <= 0 {
		return "0ms"
	}
	return fmt.Sprintf("%.2fms", float64(parseDurationNs)/1_000_000)
}

// recordSlowOperationDiagnostic is a core package helper.
func recordSlowOperationDiagnostic(parseKind string, parseFiber *Fiber, parseDurationNs int64) {
	if parseFiber == nil || parseDurationNs < slowOperationDiagnosticThresholdNs {
		return
	}
	_, parseName := describeFiber(parseFiber)
	ReportDiagnostic("runtime", DiagnosticWarning, fmt.Sprintf("slow %s on %s took %s", parseKind, parseName, formatRuntimeDurationNs(parseDurationNs)))
}

// Config holds runtime adapter configuration.
type Config struct {
	DOMAdapter   DOMAdapter
	EventAdapter EventAdapter
	Scheduler    Scheduler
	BrowserState BrowserState
	Reset        bool
	// HideRawPanicOutput is deprecated: crash containment (structured report,
	// no raw rethrow) is now the default. Use ShowRawPanicOutput to opt back
	// into re-panicking with the raw Go panic output.
	HideRawPanicOutput bool
	// ShowRawPanicOutput re-throws contained panics with their raw output
	// after the structured report. In wasm this kills the page; only enable
	// it for debugging native test runs.
	ShowRawPanicOutput     bool
	OnUnhandledPanicReport func(PanicReport)
	StrictMode             StrictModeOptions
	Limits                 RuntimeLimits
	// PassiveEffectsAfterPaint enables the v5 P1.1 commit split: layout effects
	// run synchronously in the commit task (so they observe committed DOM before
	// paint), passive effects run after the browser has painted.
	//
	// Off by default. Enabling it changes effect ORDERING, not just timing:
	// every layout effect in the tree now precedes every passive effect, where
	// previously each fiber ran both tiers before the next fiber ran either.
	// See internal/runtime/effect_ordering_contract_test.go.
	PassiveEffectsAfterPaint bool
	// FrameBudgetMs enables the v5 P1.2 wall-clock slice budget: the work loop
	// yields when a slice has consumed this many milliseconds, instead of only
	// after a fixed fiber count.
	//
	// ON by default: zero takes the 5ms default, a positive value sets an
	// explicit budget, and NEGATIVE opts out to count-only slicing.
	//
	// This flag was flipped on, reverted, and flipped back, and the middle step
	// is the useful one. Turning it on froze the Example 201 core-* and
	// enterprise-* scenarios: the tree never rendered and the harness timed out
	// with zero items, in both builds. Slicing did not cause that defect, it
	// EXPOSED one — an update arriving while a pass was already walking the tree
	// was folded into pendingLane and never run, because the coalescing path
	// acted only when the new work was MORE urgent than the running pass. A pass
	// completes inside one task without a budget, so the window was nearly
	// closed; a budget opens it at every yield.
	//
	// With that fixed (scheduleFollowUpForInFlightUpdate) the browser benchmark
	// is green with the budget on, which is now the gate. The first flip was
	// validated against `go test ./...` alone, where only this flag's own
	// "off by default" test failed — a clean-looking result that could not have
	// caught it, because nothing native drives those scenarios.
	//
	// Requires the interrupt-safe restart path (P2.5); without it the runtime
	// falls back to count-only slicing and emits a diagnostic rather than
	// degrading silently.
	FrameBudgetMs float64
	// LaneQueues enables per-lane deferral with expiration (v5 P2.2): work
	// marked at a lower priority than the running pass is deferred to a
	// follow-up pass rather than rendered inside this one.
	//
	// ON by default; set DisableLaneQueues to opt out. Each deferrable lane
	// carries a deadline (transition 500ms, background 2s) after which it is
	// admitted regardless of priority,
	// so deferral cannot become starvation under sustained input.
	LaneQueues bool
	// DisableLaneQueues turns per-lane deferral back off.
	//
	// A separate field rather than inverting LaneQueues, so existing callers
	// that set LaneQueues:true keep compiling and keep meaning what they said.
	DisableLaneQueues bool

	// AsyncIngress routes state updates made outside the frame loop through the
	// async inbox (v5 P2.1).
	//
	// Without it, a gRPC callback, a worker reply, or any goroutine mutates hook
	// state at whatever moment it happens to run — which is the race the inbox
	// was built to remove, and which no amount of care at the call site can fix
	// because the call site does not know whether a render is in flight. With
	// it, such a write is queued and applied at one defined point per frame, so
	// the in-flight tree is isolated by construction and N async writes in a
	// frame produce one render rather than N.
	//
	// Off by default: it changes WHEN an async write lands (one task later, at
	// the drain), and that is a semantic change existing apps should opt into.
	//
	// Measured 2026-07-25 rather than assumed: forcing it on for the whole suite
	// fails nine tests, all of the same shape — a test calls a setter directly
	// from its own goroutine and asserts the new value on the next line. That is
	// precisely the write this defers, so the failures are the feature working,
	// not a defect in it. They are also proof the change is observable to code
	// that already exists, which is why it stays opt-in until an app has been
	// migrated deliberately rather than by a default flip.
	AsyncIngress bool
}

// configHidesRawPanicOutput resolves the containment default: panics are
// contained unless the caller explicitly opts into raw output.
func configHidesRawPanicOutput(parseConfig Config) bool {
	return !parseConfig.ShowRawPanicOutput
}

// NewRuntime creates a new runtime instance.
func NewRuntime(parseConfig Config) *Runtime {
	ConfigureUnhandledPanicLogging(PanicLoggingOptions{HideRawPanicOutput: configHidesRawPanicOutput(parseConfig), OnReport: parseConfig.OnUnhandledPanicReport})
	parseRuntime := &Runtime{}
	applyRuntimeConfig(parseRuntime, parseConfig)
	return parseRuntime
}

// applyRuntimeConfig installs one runtime config while preserving existing adapters when the incoming field is nil.
func applyRuntimeConfig(parseRuntime *Runtime, parseConfig Config) {
	if parseRuntime == nil {
		return
	}
	if parseConfig.DOMAdapter != nil {
		parseRuntime.domAdapter = parseConfig.DOMAdapter
	}
	if parseConfig.EventAdapter != nil {
		parseRuntime.eventAdapter = parseConfig.EventAdapter
	}
	if parseConfig.Scheduler != nil {
		parseRuntime.scheduler = parseConfig.Scheduler
	}
	if parseConfig.BrowserState != nil {
		parseRuntime.browserState = parseConfig.BrowserState
	}
	if parseConfig.StrictMode.Enabled {
		parseRuntime.strictMode = parseConfig.StrictMode.withDefaults()
	} else if parseConfig.StrictMode != (StrictModeOptions{}) {
		parseRuntime.strictMode = parseConfig.StrictMode.withDefaults()
	}
	if parseConfig.PassiveEffectsAfterPaint {
		parseRuntime.passiveEffectsAfterPaint = true
	}
	// Lane queues and the frame budget are ON by default (R2 satisfied).
	//
	// R2 said each flag flips once its acceptance test passes, and
	// TestV5SchedulingComparison is that test. Run on one machine, v4 behaviour
	// first so thermal drift on this fanless chassis works AGAINST the change:
	//
	//	long frames      23      -> 13       (worst 158.8ms -> 98.1ms)
	//	interaction p95  56.0ms  -> 48.0ms   (budget 50 — met)
	//	loaded p95       16.80ms -> 16.80ms  (unchanged; M1 still equivalent)
	//
	// Every dimension improved and none regressed, and M3 crossed its budget for
	// the first time. Leaving them off would mean the measured behaviour is the
	// one nobody gets.
	//
	// PassiveEffectsAfterPaint deliberately stays OFF: it changes effect
	// ORDERING, and flipping it fails eight ordering and hydration tests that
	// encode the current contract. That is a migration, not a default.
	parseRuntime.laneQueues = true
	if parseConfig.DisableLaneQueues {
		parseRuntime.laneQueues = false
	}
	if parseConfig.AsyncIngress {
		parseRuntime.asyncIngress = true
	}
	if parseConfig.FrameBudgetMs == 0 {
		parseRuntime.frameBudgetMs = defaultFrameBudgetMs
	} else if parseConfig.FrameBudgetMs > 0 {
		parseRuntime.frameBudgetMs = parseConfig.FrameBudgetMs
	}
	parseRuntime.limits = parseConfig.Limits.withDefaults()
	applyGlobalRuntimeLimits(parseRuntime.limits)
	parseRuntime.schedulerState.ensureDefaults(parseRuntime.limits)
	if parseRuntime.atomRegistry == nil {
		parseRuntime.atomRegistry = NewAtomRegistry()
	}
	if parseRuntime.deletions == nil {
		parseRuntime.deletions = make([]*Fiber, 0)
	}
	if parseRuntime.pendingEffectFibers == nil {
		parseRuntime.pendingEffectFibers = make([]*Fiber, 0)
	}
}

// RenderTo renders an element to a DOM node specified by selector.
func (parseRt *Runtime) RenderTo(parseSelector string, parseElement *Element) {
	parseContainer := parseRt.queryContainer(parseSelector)
	if IsDOMNodeNull(parseContainer) {
		parseMessage := "RenderTo failed because the target container selector was not found: " + parseSelector
		panicFinalUnhandledPanicContext("runtime", PanicPhaseStartup, "RenderTo", parseSelector, nil, parseMessage)
	}

	parseRt.Render(parseElement, parseContainer)
}

// HydrateTo renders into a selector while preserving a dedicated hydration path.
// It attempts to reuse matching DOM, restores bootstrap state before resume via
// the public ui layer, and falls back per subtree if hydration cannot continue.
func (parseRt *Runtime) HydrateTo(parseSelector string, parseElement *Element) {
	parseContainer := parseRt.queryContainer(parseSelector)
	if IsDOMNodeNull(parseContainer) {
		parseMessage := "HydrateTo failed because the target container selector was not found: " + parseSelector
		panicFinalUnhandledPanicContext("runtime", PanicPhaseStartup, "HydrateTo", parseSelector, nil, parseMessage)
	}

	parseRt.Hydrate(parseElement, parseContainer)
}

// queryContainer is a core package helper.
func (parseRt *Runtime) queryContainer(parseSelector string) DOMNode {
	var parseContainer DOMNode
	if parseAdapter, parseOk := parseRt.domAdapter.(interface {
		QuerySelector(string) any
	}); parseOk {
		if parseNode := parseAdapter.QuerySelector(parseSelector); parseNode != nil {
			if parseDomNode, parseOk2 := parseNode.(DOMNode); parseOk2 {
				parseContainer = parseDomNode
			}
		}
	}
	return parseContainer
}

// GetAttributeValue reports one attribute value from one DOM node when the active adapter exposes attribute reads.
func (parseRt *Runtime) GetAttributeValue(parseNode DOMNode, parseName string) (string, bool) {
	if parseRt == nil || parseRt.domAdapter == nil || IsDOMNodeNull(parseNode) {
		return "", false
	}
	if parseGetter, parseOk := parseRt.domAdapter.(interface {
		GetAttribute(DOMNode, string) string
	}); parseOk {
		return parseGetter.GetAttribute(parseNode, parseName), true
	}
	return "", false
}

// GetTagName reports one normalized lower-case tag name from one DOM node when the active adapter exposes host tag reads.
func (parseRt *Runtime) GetTagName(parseNode DOMNode) (string, bool) {
	if parseRt == nil || parseRt.domAdapter == nil || IsDOMNodeNull(parseNode) {
		return "", false
	}
	getTagValue := parseRt.domAdapter.GetProperty(parseNode, "tagName")
	if getTagValue == nil {
		return "", false
	}
	switch getTag := getTagValue.(type) {
	case string:
		getNormalizedTag := strings.ToLower(strings.TrimSpace(getTag))
		return getNormalizedTag, getNormalizedTag != ""
	case interface{ String() string }:
		getNormalizedTag := strings.ToLower(strings.TrimSpace(getTag.String()))
		return getNormalizedTag, getNormalizedTag != ""
	default:
		return "", false
	}
}

// FindNodesWithAttributeInSelector collects descendant DOM nodes under one selector-matched container that carry the requested attribute.
func (parseRt *Runtime) FindNodesWithAttributeInSelector(parseSelector string, parseName string) []DOMNode {
	if parseRt == nil {
		return nil
	}
	return parseRt.findNodesWithAttribute(parseRt.queryContainer(parseSelector), parseName)
}

// FindNodesWithAttributeInTarget collects descendant DOM nodes under one resolved target that carry the requested attribute.
func (parseRt *Runtime) FindNodesWithAttributeInTarget(parseTarget any, parseName string) []DOMNode {
	if parseRt == nil {
		return nil
	}
	return parseRt.findNodesWithAttribute(parseRt.resolveContainer(parseTarget), parseName)
}

// findNodesWithAttribute traverses one DOM subtree and returns nodes whose attribute value is present and non-empty.
func (parseRt *Runtime) findNodesWithAttribute(parseRoot DOMNode, parseName string) []DOMNode {
	if parseRt == nil || parseRt.domAdapter == nil || IsDOMNodeNull(parseRoot) || parseName == "" {
		return nil
	}
	parseMatches := make([]DOMNode, 0)
	var parseWalk func(parseNode DOMNode)
	parseWalk = func(parseNode DOMNode) {
		if IsDOMNodeNull(parseNode) {
			return
		}
		if getAttributeValue, hasAttributeValue := parseRt.GetAttributeValue(parseNode, parseName); hasAttributeValue && strings.TrimSpace(getAttributeValue) != "" {
			parseMatches = append(parseMatches, parseNode)
		}
		for _, getChildNode := range parseRt.domAdapter.GetChildren(parseNode) {
			parseWalk(getChildNode)
		}
	}
	parseWalk(parseRoot)
	return parseMatches
}

// resolveContainer is a core package helper.
func (parseRt *Runtime) resolveContainer(parseTarget any) DOMNode {
	if parseTarget == nil || parseRt == nil || parseRt.domAdapter == nil {
		return nil
	}
	if parseNode, parseOk := parseTarget.(DOMNode); parseOk {
		return parseNode
	}
	if parseResolver, parseOk2 := parseRt.domAdapter.(interface{ ResolveNode(any) DOMNode }); parseOk2 {
		return parseResolver.ResolveNode(parseTarget)
	}
	return nil
}

// RenderInto renders an element tree into an explicit DOM node.
func (parseRt *Runtime) RenderInto(parseTarget any, parseElement *Element) error {
	parseContainer := parseRt.resolveContainer(parseTarget)
	if IsDOMNodeNull(parseContainer) {
		ReportDiagnostic("runtime", DiagnosticError, "RenderInto failed because the target node could not be resolved")
		return fmt.Errorf("RenderInto: target node could not be resolved")
	}
	parseRt.Render(parseElement, parseContainer)
	return nil
}

// HydrateInto hydrates an element tree into an explicit DOM node.
func (parseRt *Runtime) HydrateInto(parseTarget any, parseElement *Element) error {
	parseContainer := parseRt.resolveContainer(parseTarget)
	if IsDOMNodeNull(parseContainer) {
		ReportDiagnostic("runtime", DiagnosticError, "HydrateInto failed because the target node could not be resolved")
		return fmt.Errorf("HydrateInto: target node could not be resolved")
	}
	parseRt.Hydrate(parseElement, parseContainer)
	return nil
}
