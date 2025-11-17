package runtime

import "sync"

var (
	globalRuntime     *Runtime
	globalRuntimeOnce sync.Once
)

// GetGlobalRuntime returns the global Runtime instance, creating it if needed
// For WASM builds, this automatically uses WASM platform adapters
func GetGlobalRuntime() *Runtime {
	globalRuntimeOnce.Do(func() {
		// This will be initialized with WASM adapters when called from WASM context
		// For now, create with nil config - will be initialized later
		globalRuntime = &Runtime{
			atomRegistry: NewAtomRegistry(),
		}
	})
	return globalRuntime
}

// InitGlobalRuntime initializes the global runtime with specific adapters
// This should be called early in WASM initialization
func InitGlobalRuntime(config Config) {
	globalRuntimeOnce.Do(func() {
		globalRuntime = NewRuntime(config)
	})
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

	// Global state management
	atomRegistry *AtomRegistry

	// Global ID counter for useId hook
	idCounter   int
	idCounterMu sync.Mutex

	// UI queue for non-render updates
	uiQueue      []func()
	uiQueueMutex sync.Mutex
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
	// Query for the container
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

	if container == nil || container.IsNull() {
		panic("RenderTo: container not found for selector: " + selector)
	}

	rt.Render(element, container)
}
