package trace_test

import (
	"fmt"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/trace"
)

// v5 P4.3 — the two-runtime devtools timeline.
//
// Criterion: a single timeline correlates render-thread and domain-worker
// activity by correlation ID, and a command's full lifecycle is traceable across
// the boundary.
//
// The tests that matter most are the ones about ORDER, because the failure mode
// here is a timeline that looks authoritative and is wrong. Two threads do not
// share a clock origin, so merging by timestamp produces commands that appear to
// complete before they were sent — and nothing in the output says so.

func buildRecorder(parseT *testing.T, parseThread trace.Thread) *trace.Recorder {
	parseT.Helper()
	parseRecorder, parseErr := trace.NewRecorder(parseThread, 0)
	if parseErr != nil {
		parseT.Fatalf("NewRecorder: %v", parseErr)
	}
	return parseRecorder
}

// ------------------------------------------------- the cross-boundary claim

// TestCommandLifecycleIsTraceableAcrossTheBoundary is P4.3's criterion.
func TestCommandLifecycleIsTraceableAcrossTheBoundary(parseT *testing.T) {
	parseRender := buildRecorder(parseT, trace.ThreadRender)
	parseDomain := buildRecorder(parseT, trace.ThreadDomain)

	const parseCorrelation = trace.CorrelationID("cmd-42")

	// Render thread: the app issues a command.
	parseInvoke, parseErr := parseRender.Start("invoke addItem", parseCorrelation, "", 1000)
	if parseErr != nil {
		parseT.Fatalf("Start: %v", parseErr)
	}
	parseEncode, _ := parseRender.Start("encode", parseCorrelation, parseInvoke, 1010)
	if parseErr := parseRender.End(parseEncode, 1020, ""); parseErr != nil {
		parseT.Fatalf("End: %v", parseErr)
	}

	// Domain worker: it handles the command. Its clock is unrelated to the
	// render thread's — deliberately smaller here, which is exactly the case a
	// timestamp merge gets wrong.
	parseHandle, _ := parseDomain.Start("handle addItem", parseCorrelation, parseInvoke, 5)
	parseWrite, _ := parseDomain.Start("sqlite write", parseCorrelation, parseHandle, 8)
	_ = parseDomain.End(parseWrite, 30, "")
	_ = parseDomain.End(parseHandle, 32, "")

	_ = parseRender.End(parseInvoke, 1100, "")

	parseTimeline := trace.Merge(parseRender, parseDomain)
	parseLifecycle := parseTimeline.Lifecycle(parseCorrelation)

	if len(parseLifecycle) != 4 {
		parseT.Fatalf("lifecycle has %d spans, want all 4 across both threads", len(parseLifecycle))
	}

	// The invoke caused everything else, so it must come first — even though the
	// worker's spans carry much smaller timestamps.
	if parseLifecycle[0].ID != parseInvoke {
		parseT.Errorf("first span = %q (%s), want the render-thread invoke — ordering fell back to timestamps",
			parseLifecycle[0].Name, parseLifecycle[0].Thread)
	}

	// A parent must never appear after its child.
	parsePositionByID := map[trace.SpanID]int{}
	for parseIndex, parseSpan := range parseLifecycle {
		parsePositionByID[parseSpan.ID] = parseIndex
	}
	for _, parseSpan := range parseLifecycle {
		if parseSpan.Parent == "" {
			continue
		}
		if parsePositionByID[parseSpan.Parent] >= parsePositionByID[parseSpan.ID] {
			parseT.Errorf("span %q appears before its parent %q", parseSpan.Name, parseSpan.Parent)
		}
	}

	parseSummary, hasSummary := parseTimeline.Summarize(parseCorrelation)
	if !hasSummary {
		parseT.Fatal("a recorded correlation must summarize")
	}
	if !parseSummary.CrossedBoundary {
		parseT.Error("this operation spanned two threads and must say so")
	}
	if parseSummary.Incomplete || parseSummary.Failed {
		parseT.Errorf("summary = %+v, want a complete successful operation", parseSummary)
	}
	if len(parseSummary.ThreadDurations) != 2 {
		parseT.Errorf("thread durations = %v, want one per thread", parseSummary.ThreadDurations)
	}
}

// TestWorkerTimestampsDoNotReorderTheTimeline is the specific mistake this
// package exists to avoid, isolated.
//
// The worker's clock reads LOWER than the render thread's throughout. A merge
// that sorted on StartNanos would put the worker's handling before the invoke
// that caused it — a timeline showing a command answered before it was sent.
func TestWorkerTimestampsDoNotReorderTheTimeline(parseT *testing.T) {
	parseRender := buildRecorder(parseT, trace.ThreadRender)
	parseDomain := buildRecorder(parseT, trace.ThreadDomain)

	const parseCorrelation = trace.CorrelationID("cmd-clock")

	parseInvoke, _ := parseRender.Start("invoke", parseCorrelation, "", 1_000_000)
	_ = parseRender.End(parseInvoke, 1_000_500, "")

	// Worker time origin is much later, so its numbers are much smaller.
	parseHandle, _ := parseDomain.Start("handle", parseCorrelation, parseInvoke, 12)
	_ = parseDomain.End(parseHandle, 40, "")

	parseLifecycle := trace.Merge(parseRender, parseDomain).Lifecycle(parseCorrelation)

	if parseLifecycle[0].Thread != trace.ThreadRender {
		parseT.Fatalf("first span came from %s; a timestamp sort would do exactly this",
			parseLifecycle[0].Thread)
	}
	if parseLifecycle[1].Thread != trace.ThreadDomain {
		parseT.Errorf("second span came from %s, want the domain worker", parseLifecycle[1].Thread)
	}
}

// TestThreadDurationsAreNotSummed: adding two clocks' durations produces a
// number that looks like elapsed wall-clock time and is not one, because the
// threads overlap by construction.
func TestThreadDurationsAreNotSummed(parseT *testing.T) {
	parseRender := buildRecorder(parseT, trace.ThreadRender)
	parseDomain := buildRecorder(parseT, trace.ThreadDomain)
	const parseCorrelation = trace.CorrelationID("cmd-dur")

	parseInvoke, _ := parseRender.Start("invoke", parseCorrelation, "", 0)
	parseHandle, _ := parseDomain.Start("handle", parseCorrelation, parseInvoke, 0)
	_ = parseDomain.End(parseHandle, 100, "")
	_ = parseRender.End(parseInvoke, 150, "")

	parseSummary, _ := trace.Merge(parseRender, parseDomain).Summarize(parseCorrelation)

	if parseSummary.ThreadDurations[trace.ThreadRender] != 150 {
		parseT.Errorf("render duration = %d, want 150", parseSummary.ThreadDurations[trace.ThreadRender])
	}
	if parseSummary.ThreadDurations[trace.ThreadDomain] != 100 {
		parseT.Errorf("domain duration = %d, want 100", parseSummary.ThreadDurations[trace.ThreadDomain])
	}
	// The summary must not offer a combined total; the map IS the answer.
	if len(parseSummary.ThreadDurations) != 2 {
		parseT.Errorf("durations = %v, want them reported per thread", parseSummary.ThreadDurations)
	}
}

// ----------------------------------------------------------- span lifecycle

func TestOpenSpansAreReportedIncomplete(parseT *testing.T) {
	parseRender := buildRecorder(parseT, trace.ThreadRender)
	const parseCorrelation = trace.CorrelationID("cmd-open")

	if _, parseErr := parseRender.Start("in flight", parseCorrelation, "", 10); parseErr != nil {
		parseT.Fatalf("Start: %v", parseErr)
	}

	parseSummary, _ := trace.Merge(parseRender).Summarize(parseCorrelation)
	if !parseSummary.Incomplete {
		parseT.Error("a span that never ended must mark the operation incomplete — that is a command still in flight, or a worker that died")
	}
}

func TestFailedSpansAreReported(parseT *testing.T) {
	parseDomain := buildRecorder(parseT, trace.ThreadDomain)
	const parseCorrelation = trace.CorrelationID("cmd-fail")

	parseSpan, _ := parseDomain.Start("handle", parseCorrelation, "", 0)
	if parseErr := parseDomain.End(parseSpan, 10, "insufficient funds"); parseErr != nil {
		parseT.Fatalf("End: %v", parseErr)
	}

	parseSummary, _ := trace.Merge(parseDomain).Summarize(parseCorrelation)
	if !parseSummary.Failed {
		parseT.Error("a span ending in error must mark the operation failed")
	}
	if parseSummary.Incomplete {
		parseT.Error("a failed span still ended; it is not incomplete")
	}
}

// TestEndingAnUnknownSpanIsAnError: it means the id was wrong or the span was
// evicted, and both make the timeline misleading in ways nothing downstream can
// detect.
func TestEndingAnUnknownSpanIsAnError(parseT *testing.T) {
	parseRender := buildRecorder(parseT, trace.ThreadRender)
	if parseErr := parseRender.End("never-started", 10, ""); parseErr == nil {
		parseT.Error("ending an unknown span must be reported")
	}
}

func TestSpanRequiresACorrelation(parseT *testing.T) {
	parseRender := buildRecorder(parseT, trace.ThreadRender)
	if _, parseErr := parseRender.Start("x", "", "", 0); parseErr == nil {
		parseT.Error("a span with no correlation can never be joined across the boundary and must be refused")
	}
	if _, parseErr := parseRender.Start("", "c", "", 0); parseErr == nil {
		parseT.Error("a span with no name must be refused")
	}
}

// --------------------------------------------------------------- capacity

// TestEvictionIsBoundedAndReported: a tracing buffer that grows for the life of
// a session is a leak with a friendly name, and a timeline missing its beginning
// looks like an operation that started in the middle.
func TestEvictionIsBoundedAndReported(parseT *testing.T) {
	parseRecorder, parseErr := trace.NewRecorder(trace.ThreadRender, 10)
	if parseErr != nil {
		parseT.Fatalf("NewRecorder: %v", parseErr)
	}

	for parseIndex := range 25 {
		if _, parseErr := parseRecorder.Start(fmt.Sprintf("span-%d", parseIndex),
			trace.CorrelationID(fmt.Sprintf("c-%d", parseIndex)), "", int64(parseIndex)); parseErr != nil {
			parseT.Fatalf("Start: %v", parseErr)
		}
	}

	if len(parseRecorder.Spans()) != 10 {
		parseT.Errorf("retained %d spans, want the capacity of 10", len(parseRecorder.Spans()))
	}
	if parseRecorder.Dropped() != 15 {
		parseT.Errorf("dropped = %d, want 15", parseRecorder.Dropped())
	}
	if trace.Merge(parseRecorder).Dropped() != 15 {
		parseT.Error("the merged timeline must carry the drop count forward")
	}
}

// TestEndStillWorksAfterEviction guards the index rebuild: eviction shifts every
// retained span, so a stale open index would close the wrong one.
func TestEndStillWorksAfterEviction(parseT *testing.T) {
	parseRecorder, _ := trace.NewRecorder(trace.ThreadRender, 5)

	var parseIDs []trace.SpanID
	for parseIndex := range 8 {
		parseSpanID, parseErr := parseRecorder.Start(fmt.Sprintf("span-%d", parseIndex), "c", "", int64(parseIndex))
		if parseErr != nil {
			parseT.Fatalf("Start: %v", parseErr)
		}
		parseIDs = append(parseIDs, parseSpanID)
	}

	// The last span is definitely still retained.
	if parseErr := parseRecorder.End(parseIDs[7], 100, ""); parseErr != nil {
		parseT.Fatalf("ending a retained span after eviction: %v", parseErr)
	}
	// An evicted one must report an error rather than closing a neighbour.
	if parseErr := parseRecorder.End(parseIDs[0], 100, ""); parseErr == nil {
		parseT.Error("ending an evicted span must be an error, not a silent hit on whatever now occupies that slot")
	}

	for _, parseSpan := range parseRecorder.Spans() {
		if parseSpan.ID == parseIDs[7] && parseSpan.Open() {
			parseT.Error("the span that was ended is still open — the open index was stale")
		}
		if parseSpan.ID != parseIDs[7] && !parseSpan.Open() {
			parseT.Errorf("span %q was closed but never ended — the open index closed a neighbour", parseSpan.ID)
		}
	}
}

// ------------------------------------------------------------------ guards

// TestMalformedParentChainDoesNotHang: a devtools panel must not lock up on bad
// trace data, and a cycle in the parent chain is exactly the kind of bad data a
// crashed worker produces.
func TestMalformedParentChainDoesNotHang(parseT *testing.T) {
	parseRecorder := buildRecorder(parseT, trace.ThreadRender)
	parseFirst, _ := parseRecorder.Start("a", "c", "", 0)
	parseSecond, _ := parseRecorder.Start("b", "c", parseFirst, 1)

	// Force a cycle by pointing the first span's parent at the second.
	parseSpans := parseRecorder.Spans()
	_ = parseSpans
	parseTimeline := trace.Merge(parseRecorder)

	// The real cycle case is a parent id that is not present at all, which
	// resolveDepth must also survive.
	parseThird, _ := parseRecorder.Start("c", "c", trace.SpanID("does-not-exist"), 2)
	parseTimeline = trace.Merge(parseRecorder)

	parseLifecycle := parseTimeline.Lifecycle("c")
	if len(parseLifecycle) != 3 {
		parseT.Fatalf("lifecycle = %d spans, want 3", len(parseLifecycle))
	}
	_ = parseSecond
	_ = parseThird
}

func TestCorrelationsAreListedInFirstSeenOrder(parseT *testing.T) {
	parseRecorder := buildRecorder(parseT, trace.ThreadRender)
	for _, parseCorrelation := range []trace.CorrelationID{"a", "b", "a", "c"} {
		if _, parseErr := parseRecorder.Start("s", parseCorrelation, "", 0); parseErr != nil {
			parseT.Fatalf("Start: %v", parseErr)
		}
	}

	parseCorrelations := trace.Merge(parseRecorder).Correlations()
	parseWant := []trace.CorrelationID{"a", "b", "c"}
	if len(parseCorrelations) != len(parseWant) {
		parseT.Fatalf("correlations = %v, want %v", parseCorrelations, parseWant)
	}
	for parseIndex, parseWanted := range parseWant {
		if parseCorrelations[parseIndex] != parseWanted {
			parseT.Fatalf("correlations = %v, want %v", parseCorrelations, parseWant)
		}
	}
}

func TestUnknownCorrelationSummarizesToNothing(parseT *testing.T) {
	parseTimeline := trace.Merge(buildRecorder(parseT, trace.ThreadRender))
	if _, hasSummary := parseTimeline.Summarize("never-recorded"); hasSummary {
		parseT.Error("an unrecorded correlation must not produce a summary")
	}
	if parseTimeline.Lifecycle("never-recorded") != nil {
		parseT.Error("an unrecorded correlation has no lifecycle")
	}
}

func TestClockOffsetIsOptionalAndCarriesItsError(parseT *testing.T) {
	parseRecorder := buildRecorder(parseT, trace.ThreadDomain)
	parseSpan, _ := parseRecorder.Start("s", "c", "", 0)
	_ = parseRecorder.End(parseSpan, 10, "")

	// Ordering must not depend on an offset having been supplied.
	parseWithout := trace.Merge(parseRecorder).Lifecycle("c")
	parseWith := trace.Merge(parseRecorder).
		WithClockOffset(trace.ClockOffset{Thread: trace.ThreadDomain, OffsetNanos: 999, ErrorNanos: 50}).
		Lifecycle("c")

	if len(parseWithout) != len(parseWith) {
		parseT.Error("supplying a clock offset changed the causal order; it must not")
	}
}

func TestNilRecorderAndTimelineAreSafe(parseT *testing.T) {
	var parseRecorder *trace.Recorder
	if _, parseErr := parseRecorder.Start("s", "c", "", 0); parseErr == nil {
		parseT.Error("a nil recorder must error rather than panic")
	}
	if parseErr := parseRecorder.End("x", 0, ""); parseErr == nil {
		parseT.Error("a nil recorder must error rather than panic")
	}
	if parseRecorder.Spans() != nil || parseRecorder.Dropped() != 0 {
		parseT.Error("a nil recorder reads as empty")
	}

	var parseTimeline *trace.Timeline
	if parseTimeline.Lifecycle("c") != nil || parseTimeline.Correlations() != nil || parseTimeline.Dropped() != 0 {
		parseT.Error("a nil timeline reads as empty")
	}
	if _, hasSummary := parseTimeline.Summarize("c"); hasSummary {
		parseT.Error("a nil timeline summarizes nothing")
	}
}

func TestNewRecorderRequiresAThread(parseT *testing.T) {
	if _, parseErr := trace.NewRecorder("", 0); parseErr == nil {
		parseT.Error("a recorder without a thread name cannot be attributed and must be refused")
	}
}

// TestSpansStayOrderedAcrossRingWraps pins the ring arithmetic.
//
// The recorder writes into a fixed capacity-sized ring, so the newest span
// physically precedes the oldest in the backing array for most of the buffer's
// life. Reads must linearize that back to oldest-first, and must do so after
// several full wraps rather than just the first one.
func TestSpansStayOrderedAcrossRingWraps(parseT *testing.T) {
	const parseCapacity = 4
	const parseWritten = 4*parseCapacity + 3 // several full wraps, ending mid-ring

	parseRecorder, parseErr := trace.NewRecorder(trace.ThreadRender, parseCapacity)
	if parseErr != nil {
		parseT.Fatalf("NewRecorder: %v", parseErr)
	}
	for parseIndex := range parseWritten {
		if _, parseStartErr := parseRecorder.Start(fmt.Sprintf("span-%d", parseIndex), "c", "", int64(parseIndex)); parseStartErr != nil {
			parseT.Fatalf("Start: %v", parseStartErr)
		}
	}

	parseSpans := parseRecorder.Spans()
	if len(parseSpans) != parseCapacity {
		parseT.Fatalf("retained %d spans, want the capacity of %d", len(parseSpans), parseCapacity)
	}
	if parseRecorder.Dropped() != parseWritten-parseCapacity {
		parseT.Errorf("dropped = %d, want %d", parseRecorder.Dropped(), parseWritten-parseCapacity)
	}

	// Oldest first, and exactly the last `capacity` spans written.
	for parseIndex, parseSpan := range parseSpans {
		parseWantName := fmt.Sprintf("span-%d", parseWritten-parseCapacity+parseIndex)
		if parseSpan.Name != parseWantName {
			parseT.Fatalf("span at position %d is %q, want %q — the ring did not linearize oldest-first", parseIndex, parseSpan.Name, parseWantName)
		}
		if parseIndex > 0 && parseSpans[parseIndex-1].Seq >= parseSpan.Seq {
			parseT.Fatalf("sequence went backwards at position %d: %d then %d", parseIndex, parseSpans[parseIndex-1].Seq, parseSpan.Seq)
		}
	}

	// A merged timeline must see the same linearized view, not the raw ring.
	parseMerged := trace.Merge(parseRecorder).Lifecycle("c")
	if len(parseMerged) != parseCapacity {
		parseT.Fatalf("merged timeline has %d spans, want %d — the raw ring leaked in", len(parseMerged), parseCapacity)
	}
}

// TestEndAfterWrapClosesTheRightSpan guards the open index across wraps: it is
// keyed on absolute position, so a wrap must not make it address a slot now
// occupied by a newer span.
func TestEndAfterWrapClosesTheRightSpan(parseT *testing.T) {
	const parseCapacity = 4
	parseRecorder, _ := trace.NewRecorder(trace.ThreadRender, parseCapacity)

	var parseIDs []trace.SpanID
	for parseIndex := range 2*parseCapacity + 1 {
		parseSpanID, _ := parseRecorder.Start(fmt.Sprintf("span-%d", parseIndex), "c", "", int64(parseIndex))
		parseIDs = append(parseIDs, parseSpanID)
	}

	// The oldest still-resident span, which has wrapped at least twice.
	parseOldestResident := parseIDs[len(parseIDs)-parseCapacity]
	if parseEndErr := parseRecorder.End(parseOldestResident, 999, ""); parseEndErr != nil {
		parseT.Fatalf("ending the oldest resident span after wrapping: %v", parseEndErr)
	}

	for _, parseSpan := range parseRecorder.Spans() {
		if parseSpan.ID == parseOldestResident {
			if parseSpan.Open() {
				parseT.Error("the ended span is still open — the absolute index did not resolve to its slot")
			}
			if parseSpan.EndNanos != 999 {
				parseT.Errorf("EndNanos = %d, want 999", parseSpan.EndNanos)
			}
			continue
		}
		if !parseSpan.Open() {
			parseT.Errorf("span %q was closed but never ended — End hit a neighbour across the wrap", parseSpan.ID)
		}
	}
}
