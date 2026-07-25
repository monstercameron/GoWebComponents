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
			uiQueue:      make([]func(), 0),
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
	workLoopDepth   int
	nextUnitOfWork  *Fiber
	deletions                  []*Fiber
	pendingEffectFibers        []*Fiber
	tracksPendingEffects       bool
	// passiveEffectsAfterPaint enables the v5 P1.1 commit split: layout effects
	// stay synchronous inside the commit task, passive effects are deferred past
	// the paint boundary. Off by default (R2) until its acceptance test passes.
	passiveEffectsAfterPaint bool
	// passiveDrainScheduled guards against queueing more than one deferred
	// passive drain when several commits land before the first one runs.
	passiveDrainScheduled bool
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
	pendingInterruptLane UpdateLane
	updateScheduled            bool
	continueWorkFn             func()
	pendingBoundaryRecovery    bool
	pendingEffectOverflow      bool
	transitionDepth            int
	pendingTransitions         int
	transitionMu               sync.Mutex
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

	// UI queue for non-render updates
	uiQueue []func()

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
	// Zero keeps count-only slicing. Negative selects the 5ms default. Requires
	// the interrupt-safe restart path (P2.5); without it the runtime falls back
	// to count-only slicing and emits a diagnostic rather than degrading
	// silently.
	FrameBudgetMs float64
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
	if parseConfig.FrameBudgetMs < 0 {
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
	if parseRuntime.uiQueue == nil {
		parseRuntime.uiQueue = make([]func(), 0)
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
