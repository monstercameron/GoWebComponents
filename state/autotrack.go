package state

import (
	"sync"
	"sync/atomic"
)

// Auto-tracking lets NewAutoComputed discover its dependencies by observing which signals
// its compute function reads. It is OPT-IN: the framework's default and recommended path is
// the EXPLICIT NewComputed (no hidden reactive graph — the project's design direction).
// Auto-tracking is offered only as an ergonomic convenience where the read set is obvious.
var (
	trackMu     sync.Mutex
	trackActive atomic.Int32
	trackStack  [][]string
)

// recordSignalRead registers a read signal's id with the active auto-tracking collector. The
// atomic fast-path makes it a no-op (one load) outside a NewAutoComputed run, so it costs
// nothing on the normal Signal.Get hot path.
func recordSignalRead(parseID string) {
	if parseID == "" || trackActive.Load() == 0 {
		return
	}
	trackMu.Lock()
	if parseN := len(trackStack); parseN > 0 {
		trackStack[parseN-1] = append(trackStack[parseN-1], parseID)
	}
	trackMu.Unlock()
}

// NewAutoComputed derives a value from compute, AUTO-DISCOVERING its reactive sources by
// running compute once with read-tracking on and recording which signals it read (Solid-style
// auto-tracking). It is the opt-in sibling of NewComputed; explicit dependency declaration
// (NewComputed) remains the default and recommended path per GWC's no-hidden-graph design,
// so auto-tracking is a convenience, not a replacement.
//
// Concurrency contract: the discovery pass is NOT isolated per goroutine. Construct
// auto-computeds on one goroutine (package init / app boot is the intended spot);
// signals read on OTHER goroutines while a discovery pass runs may be attributed to
// the wrong computed — extra dependencies at best, missing ones (silently stale
// bindings) at worst. Nested NewAutoComputed calls on the same goroutine are fine.
//
//	a := state.NewSignal(2)
//	b := state.NewSignal(3)
//	sum := state.NewAutoComputed(func() int { return a.Get() + b.Get() }) // a, b discovered
func NewAutoComputed[T any](parseCompute func() T) ComputedSignal[T] {
	trackMu.Lock()
	trackStack = append(trackStack, nil)
	trackMu.Unlock()
	trackActive.Add(1)

	_ = parseCompute() // discovery pass

	trackActive.Add(-1)
	trackMu.Lock()
	parseRead := trackStack[len(trackStack)-1]
	trackStack = trackStack[:len(trackStack)-1]
	trackMu.Unlock()

	parseSeen := make(map[string]struct{}, len(parseRead))
	parseIDs := make([]string, 0, len(parseRead))
	for _, parseID := range parseRead {
		if _, parseDup := parseSeen[parseID]; parseDup {
			continue
		}
		parseSeen[parseID] = struct{}{}
		parseIDs = append(parseIDs, parseID)
	}
	return ComputedSignal[T]{compute: parseCompute, sourceIDs: parseIDs}
}
