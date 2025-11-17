package runtime

import "sync"

// Runtime orchestrates the core reconciliation engine
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
