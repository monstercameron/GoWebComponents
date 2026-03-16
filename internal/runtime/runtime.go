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

// GetGlobalRuntime returns the global Runtime instance, creating it if needed
// For WASM builds, this automatically uses WASM platform adapters
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

// InitGlobalRuntime initializes the global runtime with specific adapters
// This should be called early in WASM initialization
func InitGlobalRuntime(config Config) {
	globalRuntimeMu.Lock()
	defer globalRuntimeMu.Unlock()

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

// Runtime represents the reconciliation and rendering engine
type Runtime struct {
	domAdapter   DOMAdapter
	eventAdapter EventAdapter
	scheduler    Scheduler
	browserState BrowserState

	wipRoot         *Fiber
	currentRoot     *Fiber
	nextUnitOfWork  *Fiber
	deletions       []*Fiber
	updateScheduled bool
	continueWorkFn  func()

	// Global state management
	atomRegistry *AtomRegistry

	// Global ID counter for useId hook
	idCounter   int
	idCounterMu sync.Mutex

	// UI queue for non-render updates
	uiQueue      []func()
	uiQueueMutex sync.Mutex

	// Batch DOM operations
	domBatch      []func()
	domBatchMutex sync.Mutex

	profiling runtimeProfiling
}

type runtimeProfiling struct {
	renderCalls           int
	scheduledRootUpdates  int
	scheduledFiberMarks   int
	workLoopPasses        int
	processedUnits        int
	commitCount           int
	effectExecutions      int
	cleanupExecutions     int
	lastRenderDurationNs  int64
	lastCommitDurationNs  int64
	lastEffectDurationNs  int64
	lastCleanupDurationNs int64
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

// Config holds runtime configuration
type Config struct {
	DOMAdapter   DOMAdapter
	EventAdapter EventAdapter
	Scheduler    Scheduler
	BrowserState BrowserState
}

// NewRuntime creates a new runtime instance
func NewRuntime(config Config) *Runtime {
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

// RenderTo renders an element to a DOM node specified by selector
func (rt *Runtime) RenderTo(selector string, element *Element) {
	container := rt.queryContainer(selector)
	if container == nil || container.IsNull() {
		ReportDiagnostic("runtime", DiagnosticError, "RenderTo failed because the target container selector was not found: "+selector)
		panic("RenderTo: container not found for selector: " + selector)
	}

	rt.Render(element, container)
}

// HydrateTo renders into a selector while preserving a dedicated hydration path.
// The current implementation still falls back to a fresh render after container
// preflight; real DOM-node matching will be layered on top of this API.
func (rt *Runtime) HydrateTo(selector string, element *Element) {
	container := rt.queryContainer(selector)
	if container == nil || container.IsNull() {
		ReportDiagnostic("runtime", DiagnosticError, "HydrateTo failed because the target container selector was not found: "+selector)
		panic("HydrateTo: container not found for selector: " + selector)
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
