package runtime

import goruntime "runtime"

// InternalStateSnapshot captures bounded runtime state useful for long-session monitoring.
type InternalStateSnapshot struct {
	FiberCount            int
	AtomCount             int
	AtomSubscriberCount   int
	PendingEffectFibers   int
	UIQueueSize           int
	DiagnosticCount       int
	LogCount              int
	ProfilingEventCount   int
	SchedulerBackpressure bool
}

// MemoryHygieneOptions configures leak and GC-pressure diagnostics.
type MemoryHygieneOptions struct {
	MaxHeapAllocBytes  uint64
	MaxHeapObjects     uint64
	MaxFiberCount      int
	MaxAtomCount       int
	MaxSubscriberCount int
	// MaxGoroutines bounds live goroutines.
	//
	// Added because its absence was measurable: a suspended async boundary parked
	// a fresh watcher goroutine on every render (fixed in ee39ac99), and none of
	// the thresholds above could see it. Heap bytes barely moved, the fiber tree
	// was the same size, and no atom or subscriber count changed — the leak was
	// entirely in goroutines holding channels. A framework that spawns goroutines
	// for suspensions, fetches, and worker replies needs the one counter that
	// makes those visible.
	MaxGoroutines int
}

// MemoryHygieneSnapshot reports memory and internal-state pressure.
type MemoryHygieneSnapshot struct {
	HeapAllocBytes uint64
	HeapObjects    uint64
	Goroutines     int
	Internal       InternalStateSnapshot
	Diagnostics    []string
}

// InternalStateSnapshot returns bounded internal queue, fiber, and listener counters.
//
// NOT free, and not for a per-frame poll. It walks the whole committed fiber tree
// to count it, and it does that while holding schedulerMu — the lock every
// Schedule* entry point and commitRoot take — so a caller polling this on a timer
// blocks scheduling for O(tree) on each sample. GetSubscriberTotal walks every
// atom's subscriber set under the registry lock as well.
//
// That is a fine price for an occasional long-session sample, which is what this
// is for, and the wrong price for a devtools panel refreshing at frame rate. The
// package README recommends this API without saying so; it does now.
func (parseRt *Runtime) InternalStateSnapshot() InternalStateSnapshot {
	parseSnapshot := InternalStateSnapshot{
		DiagnosticCount: len(GetDiagnostics()),
		LogCount:        len(GetLogs()),
	}
	if parseRt == nil {
		return parseSnapshot
	}
	// v5 P2.4: this counted the package-global EnqueueUI channel, which nothing
	// drained. It now reports the per-runtime async inbox (P2.1), which is the
	// queue that actually holds pending cross-goroutine work.
	parseSnapshot.UIQueueSize = parseRt.AsyncInboxDepth()
	schedulerMu.Lock()
	defer schedulerMu.Unlock()
	parseSnapshot.FiberCount = countFiberTree(parseRt.currentRoot)
	parseSnapshot.PendingEffectFibers = len(parseRt.pendingEffectFibers)
	parseSnapshot.ProfilingEventCount = len(parseRt.profiling.events)
	parseSnapshot.SchedulerBackpressure = parseRt.schedulerState.coalescedAtLimit > 0
	if parseRt.atomRegistry != nil {
		parseSnapshot.AtomCount = parseRt.atomRegistry.GetAtomCount()
		parseSnapshot.AtomSubscriberCount = parseRt.atomRegistry.GetSubscriberTotal()
	}
	return parseSnapshot
}

// CheckMemoryHygiene samples Go memory stats and reports configured long-session pressure diagnostics.
//
// This STOPS THE WORLD. runtime.ReadMemStats pauses every goroutine while it
// copies the stats, and this then calls InternalStateSnapshot, which walks the
// fiber tree under schedulerMu. Sampling it on a timer therefore manufactures
// exactly the pauses M7 is trying to bound — a memory diagnostic that shows up in
// the GC-pause metric it exists to explain.
//
// Call it on demand, or on a slow interval when investigating a long session. Not
// per frame, and not from a devtools panel that refreshes.
func (parseRt *Runtime) CheckMemoryHygiene(parseOptions MemoryHygieneOptions) MemoryHygieneSnapshot {
	var parseMem goruntime.MemStats
	goruntime.ReadMemStats(&parseMem)
	parseSnapshot := MemoryHygieneSnapshot{
		HeapAllocBytes: parseMem.HeapAlloc,
		HeapObjects:    parseMem.HeapObjects,
		// Read outside any lock and before the tree walk: it is a single atomic
		// load, and it is the counter that catches goroutine leaks the heap
		// thresholds cannot see.
		Goroutines: goruntime.NumGoroutine(),
		Internal:   parseRt.InternalStateSnapshot(),
	}
	addDiagnostic := func(parseMessage string) {
		parseSnapshot.Diagnostics = append(parseSnapshot.Diagnostics, parseMessage)
		ReportDiagnostic("runtime", DiagnosticWarning, parseMessage)
	}
	if parseOptions.MaxHeapAllocBytes > 0 && parseSnapshot.HeapAllocBytes > parseOptions.MaxHeapAllocBytes {
		addDiagnostic("long-session memory hygiene heap allocation threshold exceeded")
	}
	if parseOptions.MaxHeapObjects > 0 && parseSnapshot.HeapObjects > parseOptions.MaxHeapObjects {
		addDiagnostic("long-session memory hygiene heap object threshold exceeded")
	}
	if parseOptions.MaxFiberCount > 0 && parseSnapshot.Internal.FiberCount > parseOptions.MaxFiberCount {
		addDiagnostic("long-session memory hygiene fiber count threshold exceeded")
	}
	if parseOptions.MaxAtomCount > 0 && parseSnapshot.Internal.AtomCount > parseOptions.MaxAtomCount {
		addDiagnostic("long-session memory hygiene atom count threshold exceeded")
	}
	if parseOptions.MaxSubscriberCount > 0 && parseSnapshot.Internal.AtomSubscriberCount > parseOptions.MaxSubscriberCount {
		addDiagnostic("long-session memory hygiene atom subscriber threshold exceeded")
	}
	if parseOptions.MaxGoroutines > 0 && parseSnapshot.Goroutines > parseOptions.MaxGoroutines {
		addDiagnostic("long-session memory hygiene goroutine threshold exceeded")
	}
	return parseSnapshot
}

func countFiberTree(parseFiber *Fiber) int {
	if parseFiber == nil {
		return 0
	}
	parseCount := 0
	for parseCursor := parseFiber; parseCursor != nil; parseCursor = parseCursor.sibling {
		parseCount++
		parseCount += countFiberTree(parseCursor.child)
	}
	return parseCount
}
