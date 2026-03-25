package runtime

import (
	"fmt"
	"sync"
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

	ConfigureUnhandledPanicLogging(PanicLoggingOptions{HideRawPanicOutput: parseConfig.HideRawPanicOutput, OnReport: parseConfig.OnUnhandledPanicReport})

	if globalRuntime == nil || parseConfig.Reset {
		globalRuntime = NewRuntime(parseConfig)
		return
	}

	globalRuntime.domAdapter = parseConfig.DOMAdapter
	globalRuntime.eventAdapter = parseConfig.EventAdapter
	globalRuntime.scheduler = parseConfig.Scheduler
	globalRuntime.browserState = parseConfig.BrowserState
	if globalRuntime.atomRegistry == nil {
		globalRuntime.atomRegistry = NewAtomRegistry()
	}
	if globalRuntime.deletions == nil {
		globalRuntime.deletions = make([]*Fiber, 0)
	}
	if globalRuntime.uiQueue == nil {
		globalRuntime.uiQueue = make([]func(), 0)
	}
}

// Runtime represents the reconciliation and rendering engine.
type Runtime struct {
	domAdapter   DOMAdapter
	eventAdapter EventAdapter
	scheduler    Scheduler
	browserState BrowserState

	wipRoot                 *Fiber
	currentRoot             *Fiber
	nextUnitOfWork          *Fiber
	deletions               []*Fiber
	updateScheduled         bool
	continueWorkFn          func()
	pendingBoundaryRecovery bool
	transitionDepth         int
	pendingTransitions      int
	transitionMu            sync.Mutex

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
	uiQueue      []func()
	uiQueueMutex sync.Mutex

	// Batch DOM operations
	domBatch      []func()
	domBatchMutex sync.Mutex

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
	DOMAdapter             DOMAdapter
	EventAdapter           EventAdapter
	Scheduler              Scheduler
	BrowserState           BrowserState
	Reset                  bool
	HideRawPanicOutput     bool
	OnUnhandledPanicReport func(PanicReport)
}

// NewRuntime creates a new runtime instance.
func NewRuntime(parseConfig Config) *Runtime {
	ConfigureUnhandledPanicLogging(PanicLoggingOptions{HideRawPanicOutput: parseConfig.HideRawPanicOutput, OnReport: parseConfig.OnUnhandledPanicReport})
	// TODO: validate adapters are non-nil and fail fast; current code will panic later if any adapter is missing
	return &Runtime{
		domAdapter:   parseConfig.DOMAdapter,
		eventAdapter: parseConfig.EventAdapter,
		scheduler:    parseConfig.Scheduler,
		browserState: parseConfig.BrowserState,
		deletions:    make([]*Fiber, 0),
		atomRegistry: NewAtomRegistry(),
		uiQueue:      make([]func(), 0),
	}
}

// RenderTo renders an element to a DOM node specified by selector.
func (parseRt *Runtime) RenderTo(parseSelector string, parseElement *Element) {
	parseContainer := parseRt.queryContainer(parseSelector)
	if parseContainer == nil || parseContainer.IsNull() {
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
	if parseContainer == nil || parseContainer.IsNull() {
		parseMessage := "HydrateTo failed because the target container selector was not found: " + parseSelector
		panicFinalUnhandledPanicContext("runtime", PanicPhaseStartup, "HydrateTo", parseSelector, nil, parseMessage)
	}

	parseRt.Hydrate(parseElement, parseContainer)
}

// queryContainer is a core package helper.
func (parseRt *Runtime) queryContainer(parseSelector string) DOMNode {
	var parseContainer DOMNode
	if parseAdapter, parseOk := parseRt.domAdapter.(interface {
		QuerySelector(string) interface{}
	}); parseOk {
		if parseNode := parseAdapter.QuerySelector(parseSelector); parseNode != nil {
			if parseDomNode, parseOk2 := parseNode.(DOMNode); parseOk2 {
				parseContainer = parseDomNode
			}
		}
	}
	return parseContainer
}

// resolveContainer is a core package helper.
func (parseRt *Runtime) resolveContainer(parseTarget interface{}) DOMNode {
	if parseTarget == nil || parseRt == nil || parseRt.domAdapter == nil {
		return nil
	}
	if parseNode, parseOk := parseTarget.(DOMNode); parseOk {
		return parseNode
	}
	if parseResolver, parseOk2 := parseRt.domAdapter.(interface{ ResolveNode(interface{}) DOMNode }); parseOk2 {
		return parseResolver.ResolveNode(parseTarget)
	}
	return nil
}

// RenderInto renders an element tree into an explicit DOM node.
func (parseRt *Runtime) RenderInto(parseTarget interface{}, parseElement *Element) error {
	parseContainer := parseRt.resolveContainer(parseTarget)
	if parseContainer == nil || parseContainer.IsNull() {
		ReportDiagnostic("runtime", DiagnosticError, "RenderInto failed because the target node could not be resolved")
		return fmt.Errorf("RenderInto: target node could not be resolved")
	}
	parseRt.Render(parseElement, parseContainer)
	return nil
}

// HydrateInto hydrates an element tree into an explicit DOM node.
func (parseRt *Runtime) HydrateInto(parseTarget interface{}, parseElement *Element) error {
	parseContainer := parseRt.resolveContainer(parseTarget)
	if parseContainer == nil || parseContainer.IsNull() {
		ReportDiagnostic("runtime", DiagnosticError, "HydrateInto failed because the target node could not be resolved")
		return fmt.Errorf("HydrateInto: target node could not be resolved")
	}
	parseRt.Hydrate(parseElement, parseContainer)
	return nil
}
