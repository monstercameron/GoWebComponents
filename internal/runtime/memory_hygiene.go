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
}

// MemoryHygieneSnapshot reports memory and internal-state pressure.
type MemoryHygieneSnapshot struct {
	HeapAllocBytes uint64
	HeapObjects    uint64
	Internal       InternalStateSnapshot
	Diagnostics    []string
}

// InternalStateSnapshot returns bounded internal queue, fiber, and listener counters.
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
func (parseRt *Runtime) CheckMemoryHygiene(parseOptions MemoryHygieneOptions) MemoryHygieneSnapshot {
	var parseMem goruntime.MemStats
	goruntime.ReadMemStats(&parseMem)
	parseSnapshot := MemoryHygieneSnapshot{
		HeapAllocBytes: parseMem.HeapAlloc,
		HeapObjects:    parseMem.HeapObjects,
		Internal:       parseRt.InternalStateSnapshot(),
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
