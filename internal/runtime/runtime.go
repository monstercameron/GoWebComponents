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
func InitGlobalRuntime(config Config) {
	globalRuntimeMu.Lock()
	defer globalRuntimeMu.Unlock()

	ConfigureUnhandledPanicLogging(PanicLoggingOptions{HideRawPanicOutput: config.HideRawPanicOutput, OnReport: config.OnUnhandledPanicReport})

	if globalRuntime == nil {
		globalRuntime = NewRuntime(config)
		return
	}

	globalRuntime.domAdapter = config.DOMAdapter
	globalRuntime.eventAdapter = config.EventAdapter
	globalRuntime.scheduler = config.Scheduler
	globalRuntime.browserState = config.BrowserState
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

func (rt *Runtime) SetIDSeed(seed int) {
	if rt == nil || seed < 0 {
		return
	}
	rt.idCounterMu.Lock()
	if seed > rt.idCounter {
		rt.idCounter = seed
	}
	rt.idCounterMu.Unlock()
}

// SetNextHydrationStrict configures whether the next hydration attempt should
// fail fast on mismatches instead of warning and falling back per subtree.
func (rt *Runtime) SetNextHydrationStrict(strict bool) {
	if rt == nil {
		return
	}
	rt.nextHydrationStrict = strict
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

func formatRuntimeDurationNs(durationNs int64) string {
	if durationNs <= 0 {
		return "0ms"
	}
	return fmt.Sprintf("%.2fms", float64(durationNs)/1_000_000)
}

func recordSlowOperationDiagnostic(kind string, fiber *Fiber, durationNs int64) {
	if fiber == nil || durationNs < slowOperationDiagnosticThresholdNs {
		return
	}
	_, name := describeFiber(fiber)
	ReportDiagnostic("runtime", DiagnosticWarning, fmt.Sprintf("slow %s on %s took %s", kind, name, formatRuntimeDurationNs(durationNs)))
}

// Config holds runtime adapter configuration.
type Config struct {
	DOMAdapter             DOMAdapter
	EventAdapter           EventAdapter
	Scheduler              Scheduler
	BrowserState           BrowserState
	HideRawPanicOutput     bool
	OnUnhandledPanicReport func(PanicReport)
}

// NewRuntime creates a new runtime instance.
func NewRuntime(config Config) *Runtime {
	ConfigureUnhandledPanicLogging(PanicLoggingOptions{HideRawPanicOutput: config.HideRawPanicOutput, OnReport: config.OnUnhandledPanicReport})
	// TODO: validate adapters are non-nil and fail fast; current code will panic later if any adapter is missing
	return &Runtime{
		domAdapter:   config.DOMAdapter,
		eventAdapter: config.EventAdapter,
		scheduler:    config.Scheduler,
		browserState: config.BrowserState,
		deletions:    make([]*Fiber, 0),
		atomRegistry: NewAtomRegistry(),
		uiQueue:      make([]func(), 0),
	}
}

// RenderTo renders an element to a DOM node specified by selector.
func (rt *Runtime) RenderTo(selector string, element *Element) {
	container := rt.queryContainer(selector)
	if container == nil || container.IsNull() {
		message := "RenderTo failed because the target container selector was not found: " + selector
		panicFinalUnhandledPanicContext("runtime", PanicPhaseStartup, "RenderTo", selector, nil, message)
	}

	rt.Render(element, container)
}

// HydrateTo renders into a selector while preserving a dedicated hydration path.
// It attempts to reuse matching DOM, restores bootstrap state before resume via
// the public ui layer, and falls back per subtree if hydration cannot continue.
func (rt *Runtime) HydrateTo(selector string, element *Element) {
	container := rt.queryContainer(selector)
	if container == nil || container.IsNull() {
		message := "HydrateTo failed because the target container selector was not found: " + selector
		panicFinalUnhandledPanicContext("runtime", PanicPhaseStartup, "HydrateTo", selector, nil, message)
	}

	rt.Hydrate(element, container)
}

func (rt *Runtime) queryContainer(selector string) DOMNode {
	var container DOMNode
	if adapter, ok := rt.domAdapter.(interface {
		QuerySelector(string) interface{}
	}); ok {
		if node := adapter.QuerySelector(selector); node != nil {
			if domNode, ok := node.(DOMNode); ok {
				container = domNode
			}
		}
	}
	return container
}

func (rt *Runtime) resolveContainer(target interface{}) DOMNode {
	if target == nil || rt == nil || rt.domAdapter == nil {
		return nil
	}
	if node, ok := target.(DOMNode); ok {
		return node
	}
	if resolver, ok := rt.domAdapter.(interface{ ResolveNode(interface{}) DOMNode }); ok {
		return resolver.ResolveNode(target)
	}
	return nil
}

// RenderInto renders an element tree into an explicit DOM node.
func (rt *Runtime) RenderInto(target interface{}, element *Element) error {
	container := rt.resolveContainer(target)
	if container == nil || container.IsNull() {
		ReportDiagnostic("runtime", DiagnosticError, "RenderInto failed because the target node could not be resolved")
		return fmt.Errorf("RenderInto: target node could not be resolved")
	}
	rt.Render(element, container)
	return nil
}

// HydrateInto hydrates an element tree into an explicit DOM node.
func (rt *Runtime) HydrateInto(target interface{}, element *Element) error {
	container := rt.resolveContainer(target)
	if container == nil || container.IsNull() {
		ReportDiagnostic("runtime", DiagnosticError, "HydrateInto failed because the target node could not be resolved")
		return fmt.Errorf("HydrateInto: target node could not be resolved")
	}
	rt.Hydrate(element, container)
	return nil
}
