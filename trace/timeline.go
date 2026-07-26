// Package trace is v5's cross-thread devtools timeline (plan item P4.3).
//
// The criterion is that a single timeline correlates render-thread and
// domain-worker activity by correlation ID, and that a command's full lifecycle
// is traceable across the boundary.
//
// The thing that makes this harder than appending two logs: THE TWO THREADS DO
// NOT SHARE A CLOCK. performance.now() is relative to a time origin that a
// worker sets when it starts, so a worker span timestamped 12.4 and a main
// thread span timestamped 12.4 did not happen at the same moment, and nothing in
// the numbers says so. Merging by timestamp produces a timeline that looks
// authoritative and is wrong — commands appearing to complete before they were
// sent, work appearing to overlap that could not have.
//
// So ordering is CAUSAL, not chronological. A span records which thread it came
// from and what caused it, and the timeline reconstructs the order from that.
// Timestamps are kept, because within one thread they are meaningful and are the
// only source of durations, but they never order across threads. A caller who
// genuinely needs cross-thread wall-clock alignment must supply a measured
// ClockOffset, and the error bound comes with it.
package trace

import (
	"errors"
	"fmt"
	"sort"
)

// Thread names which side of the boundary a span came from.
type Thread string

const (
	// ThreadRender is the main thread that owns the DOM.
	ThreadRender Thread = "render"
	// ThreadDomain is the worker that owns application state.
	ThreadDomain Thread = "domain"
	// ThreadCompute is a compute-pool worker.
	ThreadCompute Thread = "compute"
)

// CorrelationID ties together every span belonging to one logical operation,
// across every thread it touches.
type CorrelationID string

// SpanID identifies one span.
type SpanID string

// Span is one interval of work on one thread.
type Span struct {
	ID     SpanID
	Parent SpanID
	// Correlation is the operation this span belongs to. Spans on different
	// threads sharing a correlation are the same logical work.
	Correlation CorrelationID
	Thread      Thread
	Name        string
	// StartNanos and EndNanos are on the RECORDING THREAD'S clock and are only
	// comparable to other spans from that same thread. See the package comment.
	StartNanos int64
	EndNanos   int64
	// Seq is a per-thread monotonic counter, assigned by the recorder.
	//
	// It exists because timestamps can tie — two spans in the same microsecond
	// are ordinary — and a timeline that reorders equal timestamps arbitrarily
	// is unreadable. Seq breaks the tie the way the thread actually ran.
	Seq uint64
	// Err is non-empty when the span ended in failure.
	Err string
}

// Duration reports the span's length on its own thread's clock.
//
// Zero for a span that never ended, which is a live span rather than an
// instantaneous one — the distinction matters when reading a timeline captured
// mid-operation.
func (parseSpan Span) Duration() int64 {
	if parseSpan.EndNanos <= parseSpan.StartNanos {
		return 0
	}
	return parseSpan.EndNanos - parseSpan.StartNanos
}

// Open reports whether the span has not ended.
func (parseSpan Span) Open() bool {
	return parseSpan.EndNanos == 0
}

// DefaultCapacity is how many spans a recorder keeps.
//
// Bounded because R6 says diagnostics carry an allocation budget and because a
// tracing buffer that grows for the life of a session is a leak with a friendly
// name. Oldest spans are dropped first: a devtools panel is nearly always asked
// about something that just happened.
const DefaultCapacity = 4096

// Recorder collects spans on one thread.
//
// Not safe for concurrent use, which is not a limitation here: there is one
// recorder per thread and each thread is single-threaded in wasm.
type Recorder struct {
	thread Thread
	// spans is a ring of exactly capacity slots. The span with absolute index a
	// lives at spans[a%capacity] and is resident while a >= total-count.
	//
	// It used to be a plain slice that evicted by shifting every element down
	// one and then rebuilding openByID from scratch — both O(capacity), on
	// EVERY span once the buffer was full, which is the steady state for any
	// long-running trace.
	spans    []Span
	capacity int
	// total counts spans ever appended; count is how many are still resident.
	total    uint64
	count    int
	seq      uint64
	dropped  int
	// openByID indexes spans that have started and not ended, so End is O(1)
	// rather than a scan back through the buffer. It stores the ABSOLUTE index,
	// which does not move when the ring wraps; a slice index did, which is what
	// forced the rebuild. Eviction deletes the evicted id, so membership here
	// implies residency.
	openByID map[SpanID]uint64
}

// NewRecorder creates a recorder for one thread.
func NewRecorder(parseThread Thread, parseCapacity int) (*Recorder, error) {
	if parseThread == "" {
		return nil, errors.New("trace: a thread name is required")
	}
	if parseCapacity <= 0 {
		parseCapacity = DefaultCapacity
	}
	return &Recorder{
		thread:   parseThread,
		spans:    make([]Span, parseCapacity),
		capacity: parseCapacity,
		openByID: make(map[SpanID]uint64),
	}, nil
}

// Start records the beginning of a span and returns its id.
//
// The timestamp is the caller's, not read from a clock here: this package is
// compiled for both native tests and wasm, and the meaningful clock differs.
// Taking it as a parameter also lets a test produce a deterministic timeline.
func (parseRecorder *Recorder) Start(parseName string, parseCorrelation CorrelationID, parseParent SpanID, parseStartNanos int64) (SpanID, error) {
	if parseRecorder == nil {
		return "", errors.New("trace: recorder is nil")
	}
	if parseName == "" {
		return "", errors.New("trace: a span name is required")
	}
	if parseCorrelation == "" {
		// A span with no correlation cannot be joined to the other thread, which
		// is the entire purpose here. Refusing is better than recording something
		// that will silently never appear in a lifecycle.
		return "", errors.New("trace: a correlation id is required")
	}

	parseRecorder.seq++
	parseSpanID := SpanID(fmt.Sprintf("%s-%d", parseRecorder.thread, parseRecorder.seq))

	parseRecorder.appendSpan(Span{
		ID:          parseSpanID,
		Parent:      parseParent,
		Correlation: parseCorrelation,
		Thread:      parseRecorder.thread,
		Name:        parseName,
		StartNanos:  parseStartNanos,
		Seq:         parseRecorder.seq,
	})
	return parseSpanID, nil
}

// End closes a span.
//
// Ending an unknown span is an error rather than a no-op: it means the id was
// wrong, or the span was evicted, and both make the resulting timeline
// misleading in ways nothing downstream can detect.
func (parseRecorder *Recorder) End(parseSpanID SpanID, parseEndNanos int64, parseErr string) error {
	if parseRecorder == nil {
		return errors.New("trace: recorder is nil")
	}
	parseAbsolute, hasSpan := parseRecorder.openByID[parseSpanID]
	if !hasSpan {
		return fmt.Errorf("trace: span %q is not open on the %s thread", parseSpanID, parseRecorder.thread)
	}
	parseSlot := parseRecorder.slotOf(parseAbsolute)
	parseRecorder.spans[parseSlot].EndNanos = parseEndNanos
	parseRecorder.spans[parseSlot].Err = parseErr
	delete(parseRecorder.openByID, parseSpanID)
	return nil
}

// slotOf maps one absolute span index onto its ring slot.
func (parseRecorder *Recorder) slotOf(parseAbsolute uint64) int {
	return int(parseAbsolute % uint64(parseRecorder.capacity))
}

// appendSpan adds a span, evicting the oldest when the ring is full.
//
// O(1): the write lands in the slot the evicted span vacated, and the open
// index needs no repair because it is keyed on absolute position.
func (parseRecorder *Recorder) appendSpan(parseSpan Span) {
	if parseRecorder.count == parseRecorder.capacity {
		parseEvictedAbsolute := parseRecorder.total - uint64(parseRecorder.capacity)
		parseEvicted := parseRecorder.spans[parseRecorder.slotOf(parseEvictedAbsolute)]
		// Harmless when the evicted span had already ended.
		delete(parseRecorder.openByID, parseEvicted.ID)
		parseRecorder.dropped++
		parseRecorder.count--
	}

	parseAbsolute := parseRecorder.total
	parseRecorder.spans[parseRecorder.slotOf(parseAbsolute)] = parseSpan
	parseRecorder.total++
	parseRecorder.count++
	parseRecorder.openByID[parseSpan.ID] = parseAbsolute
}

// Spans returns a copy of the recorded spans, oldest first.
func (parseRecorder *Recorder) Spans() []Span {
	if parseRecorder == nil {
		return nil
	}
	parseCopy := make([]Span, parseRecorder.count)
	parseFirst := parseRecorder.total - uint64(parseRecorder.count)
	for parseIndex := range parseRecorder.count {
		parseCopy[parseIndex] = parseRecorder.spans[parseRecorder.slotOf(parseFirst+uint64(parseIndex))]
	}
	return parseCopy
}

// Dropped reports how many spans were evicted.
//
// Surfaced rather than silent: a timeline missing its beginning looks like an
// operation that started in the middle, and a reader has no other way to tell.
func (parseRecorder *Recorder) Dropped() int {
	if parseRecorder == nil {
		return 0
	}
	return parseRecorder.dropped
}

// ---------------------------------------------------------------- timeline

// ClockOffset converts one thread's clock to another's.
//
// Supplying one is optional and its accuracy is the caller's problem, which is
// why the error bound travels with it: a round-trip estimate is only as good as
// the round trip was symmetric, and on a busy worker it is not. The timeline
// never needs an offset to ORDER events — that is what causality is for — and
// uses it only to report cross-thread gaps, always with the bound attached.
type ClockOffset struct {
	// Thread whose clock is being converted.
	Thread Thread
	// OffsetNanos is added to that thread's timestamps to reach the render
	// thread's clock.
	OffsetNanos int64
	// ErrorNanos bounds the uncertainty. A gap smaller than this is noise.
	ErrorNanos int64
}

// Timeline is a merged, causally-ordered view across threads.
type Timeline struct {
	spans   []Span
	offsets map[Thread]ClockOffset
	dropped int
}

// Merge builds a timeline from several threads' recorders.
func Merge(parseRecorders ...*Recorder) *Timeline {
	parseTimeline := &Timeline{offsets: make(map[Thread]ClockOffset)}
	for _, parseRecorder := range parseRecorders {
		if parseRecorder == nil {
			continue
		}
		// Spans() rather than the raw ring: the backing array is capacity-sized
		// and unordered once it has wrapped, so appending it directly would mix
		// in never-written slots and out-of-order entries.
		parseTimeline.spans = append(parseTimeline.spans, parseRecorder.Spans()...)
		parseTimeline.dropped += parseRecorder.dropped
	}
	return parseTimeline
}

// WithClockOffset records a measured offset for one thread.
func (parseTimeline *Timeline) WithClockOffset(parseOffset ClockOffset) *Timeline {
	if parseTimeline == nil || parseOffset.Thread == "" {
		return parseTimeline
	}
	parseTimeline.offsets[parseOffset.Thread] = parseOffset
	return parseTimeline
}

// Dropped reports how many spans were evicted across all merged recorders.
func (parseTimeline *Timeline) Dropped() int {
	if parseTimeline == nil {
		return 0
	}
	return parseTimeline.dropped
}

// Lifecycle returns every span for one correlation, in causal order.
//
// This is P4.3's criterion: a command's full lifecycle traceable across the
// boundary. The order is derived from parentage first and per-thread sequence
// second — never from comparing timestamps across threads, which is the mistake
// that makes a merged timeline confidently wrong.
func (parseTimeline *Timeline) Lifecycle(parseCorrelation CorrelationID) []Span {
	if parseTimeline == nil || parseCorrelation == "" {
		return nil
	}

	var parseMatching []Span
	for _, parseSpan := range parseTimeline.spans {
		if parseSpan.Correlation == parseCorrelation {
			parseMatching = append(parseMatching, parseSpan)
		}
	}
	if len(parseMatching) == 0 {
		return nil
	}
	return causalOrder(parseMatching)
}

// causalOrder sorts spans so a parent always precedes its children.
//
// Within one thread, sequence order IS causal order, so the sort key is
// (depth in the parent chain, thread, sequence). Depth first is what carries
// causality across the boundary: a worker span whose parent is a render span
// sorts after it regardless of what either clock says.
func causalOrder(parseSpans []Span) []Span {
	parseDepthByID := make(map[SpanID]int, len(parseSpans))
	parseByID := make(map[SpanID]Span, len(parseSpans))
	for _, parseSpan := range parseSpans {
		parseByID[parseSpan.ID] = parseSpan
	}

	// Depth is resolved with memoization and a visited set, because a malformed
	// parent chain — an id pointing at a span that points back — would otherwise
	// recurse forever. A devtools panel must not hang on bad trace data.
	var resolveDepth func(parseSpanID SpanID, parseVisiting map[SpanID]bool) int
	resolveDepth = func(parseSpanID SpanID, parseVisiting map[SpanID]bool) int {
		if parseDepth, hasDepth := parseDepthByID[parseSpanID]; hasDepth {
			return parseDepth
		}
		parseSpan, hasSpan := parseByID[parseSpanID]
		if !hasSpan || parseSpan.Parent == "" || parseVisiting[parseSpanID] {
			parseDepthByID[parseSpanID] = 0
			return 0
		}
		parseVisiting[parseSpanID] = true
		parseDepth := resolveDepth(parseSpan.Parent, parseVisiting) + 1
		delete(parseVisiting, parseSpanID)
		parseDepthByID[parseSpanID] = parseDepth
		return parseDepth
	}
	for _, parseSpan := range parseSpans {
		resolveDepth(parseSpan.ID, map[SpanID]bool{})
	}

	parseOrdered := make([]Span, len(parseSpans))
	copy(parseOrdered, parseSpans)
	sort.SliceStable(parseOrdered, func(parseLeft int, parseRight int) bool {
		parseLeftDepth := parseDepthByID[parseOrdered[parseLeft].ID]
		parseRightDepth := parseDepthByID[parseOrdered[parseRight].ID]
		if parseLeftDepth != parseRightDepth {
			return parseLeftDepth < parseRightDepth
		}
		if parseOrdered[parseLeft].Thread != parseOrdered[parseRight].Thread {
			return parseOrdered[parseLeft].Thread < parseOrdered[parseRight].Thread
		}
		return parseOrdered[parseLeft].Seq < parseOrdered[parseRight].Seq
	})
	return parseOrdered
}

// Correlations lists every correlation in the timeline, in first-seen order.
func (parseTimeline *Timeline) Correlations() []CorrelationID {
	if parseTimeline == nil {
		return nil
	}
	parseSeen := make(map[CorrelationID]bool, len(parseTimeline.spans))
	var parseOrder []CorrelationID
	for _, parseSpan := range parseTimeline.spans {
		if parseSeen[parseSpan.Correlation] {
			continue
		}
		parseSeen[parseSpan.Correlation] = true
		parseOrder = append(parseOrder, parseSpan.Correlation)
	}
	return parseOrder
}

// LifecycleSummary describes one operation end to end.
type LifecycleSummary struct {
	Correlation CorrelationID
	// Threads lists every thread the operation touched, in causal order.
	Threads []Thread
	// SpanCount is how many spans make it up.
	SpanCount int
	// CrossedBoundary reports whether the operation involved more than one
	// thread, which is the case P4.3 exists for.
	CrossedBoundary bool
	// Incomplete reports that some span never ended — a command still in flight,
	// or one whose worker died before closing it.
	Incomplete bool
	// Failed reports that some span ended with an error.
	Failed bool
	// ThreadDurations is time spent per thread, each on its own clock. NOT
	// summed into a total: adding two clocks' durations produces a number that
	// looks like a wall-clock elapsed time and is not one, since the threads
	// overlap by construction.
	ThreadDurations map[Thread]int64
}

// Summarize describes one operation's lifecycle.
func (parseTimeline *Timeline) Summarize(parseCorrelation CorrelationID) (LifecycleSummary, bool) {
	parseSpans := parseTimeline.Lifecycle(parseCorrelation)
	if len(parseSpans) == 0 {
		return LifecycleSummary{}, false
	}

	parseSummary := LifecycleSummary{
		Correlation:     parseCorrelation,
		SpanCount:       len(parseSpans),
		ThreadDurations: make(map[Thread]int64),
	}
	parseSeenThread := make(map[Thread]bool, 3)
	for _, parseSpan := range parseSpans {
		if !parseSeenThread[parseSpan.Thread] {
			parseSeenThread[parseSpan.Thread] = true
			parseSummary.Threads = append(parseSummary.Threads, parseSpan.Thread)
		}
		parseSummary.ThreadDurations[parseSpan.Thread] += parseSpan.Duration()
		if parseSpan.Open() {
			parseSummary.Incomplete = true
		}
		if parseSpan.Err != "" {
			parseSummary.Failed = true
		}
	}
	parseSummary.CrossedBoundary = len(parseSummary.Threads) > 1
	return parseSummary, true
}
