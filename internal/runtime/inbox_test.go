package runtime

import (
	"strconv"
	"sync"
	"testing"
)

// v5 P2.1 — the async inbox.
//
// The claim is that N async messages arriving between two frame boundaries
// produce ONE render pass. These tests measure that rather than assert it, and
// pin the three failure modes the inbox exists to remove (T3/T4/T5).

func newInboxRuntime(parseT *testing.T) (*Runtime, DOMNode, *testScheduler) {
	parseT.Helper()
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler, Reset: true})
	return parseRt, parseAdapter.CreateElement("div"), parseScheduler
}

// TestInbox_ManyPostsCollapseToOneDrain is the batching property (T5).
func TestInbox_ManyPostsCollapseToOneDrain(parseT *testing.T) {
	parseRt, _, parseScheduler := newInboxRuntime(parseT)

	parseApplied := 0
	for range 50 {
		parseRt.PostAsync(func() { parseApplied++ })
	}

	if parseRt.AsyncInboxDepth() != 50 {
		parseT.Fatalf("expected 50 queued entries before the drain, got %d", parseRt.AsyncInboxDepth())
	}
	if parseApplied != 0 {
		parseT.Fatal("posted work must not run before the frame boundary")
	}

	runScheduledTimeouts(parseScheduler)

	if parseApplied != 50 {
		parseT.Fatalf("applied %d entries, want all 50", parseApplied)
	}
	parsePosts, parseDrains := parseRt.AsyncInboxStats()
	if parsePosts != 50 {
		parseT.Errorf("posts = %d, want 50", parsePosts)
	}
	if parseDrains != 1 {
		parseT.Errorf("drains = %d, want exactly 1 — 50 posts must batch into one drain", parseDrains)
	}
}

// TestInbox_ManyPostsProduceOneRenderPass is the property that actually
// matters: batching at the queue is only useful if it becomes one commit.
func TestInbox_ManyPostsProduceOneRenderPass(parseT *testing.T) {
	parseRt, parseContainer, parseScheduler := newInboxRuntime(parseT)

	var setValue func(any)
	parseRenders := 0
	parseApp := func() *Element {
		parseRenders++
		// A string state, not textFromInt: that helper is string(rune('0'+n))
		// and only works for single digits.
		parseValue, parseSet := GoUseState(parseRt, "0")
		setValue = parseSet
		return CreateElement("div", map[string]any{}, parseValue())
	}

	parseRt.Render(CreateElement(parseApp, nil), parseContainer)
	runScheduledTimeouts(parseScheduler)
	parseRendersAfterMount := parseRenders
	parseCommitsAfterMount := parseRt.profiling.commitCount

	// 20 async producers each write state, as goroutines or worker messages
	// would. Without the inbox each of these is its own pass and commit.
	for parseI := range 20 {
		parseValue := strconv.Itoa(parseI + 1)
		parseRt.PostAsync(func() { setValue(parseValue) })
	}
	runScheduledTimeouts(parseScheduler)

	parseCommits := parseRt.profiling.commitCount - parseCommitsAfterMount
	if parseCommits != 1 {
		parseT.Errorf("20 async writes produced %d commits, want 1", parseCommits)
	}
	if parseRenders <= parseRendersAfterMount {
		parseT.Error("async writes never rendered")
	}
	// The final value must win; batching must not lose the last write.
	parseRoot := parseContainer.(*testDOMNode)
	parseDiv := parseRoot.children[0].(*testDOMNode)
	parseText := parseDiv.children[0].(*testDOMNode)
	if parseText.text != "20" {
		parseT.Errorf("committed text = %q, want %q (last write must win)", parseText.text, "20")
	}
}

// TestInbox_NothingRunsOutsideTheFrameLoop is the isolation property (T4).
// While a pass is mid-flight, posted work must not touch state.
func TestInbox_NothingRunsOutsideTheFrameLoop(parseT *testing.T) {
	parseRt, parseContainer, parseScheduler := newInboxRuntime(parseT)

	isMidRenderPostApplied := false
	parseApp := func() *Element {
		// Post from inside a render — the exact window where a direct write
		// would either tear the tree or be stranded.
		parseRt.PostAsync(func() { isMidRenderPostApplied = true })
		if isMidRenderPostApplied {
			parseT.Error("inbox work ran during the render pass; it must wait for the boundary")
		}
		return CreateElement("div", map[string]any{})
	}

	parseRt.Render(CreateElement(parseApp, nil), parseContainer)
	if isMidRenderPostApplied {
		parseT.Error("inbox work ran before any frame boundary")
	}

	runScheduledTimeouts(parseScheduler)
	if !isMidRenderPostApplied {
		parseT.Error("inbox work never ran after the boundary")
	}
}

// TestInbox_ConcurrentProducersAreSafe covers the native/SSR case where
// goroutines genuinely run in parallel.
func TestInbox_ConcurrentProducersAreSafe(parseT *testing.T) {
	parseRt, _, parseScheduler := newInboxRuntime(parseT)

	var parseMu sync.Mutex
	parseApplied := 0
	var parseWait sync.WaitGroup
	for range 32 {
		parseWait.Add(1)
		go func() {
			defer parseWait.Done()
			parseRt.PostAsync(func() {
				parseMu.Lock()
				parseApplied++
				parseMu.Unlock()
			})
		}()
	}
	parseWait.Wait()
	runScheduledTimeouts(parseScheduler)

	parseMu.Lock()
	parseGot := parseApplied
	parseMu.Unlock()
	if parseGot != 32 {
		parseT.Errorf("applied %d of 32 concurrent posts", parseGot)
	}
}

// TestInbox_PanickingEntryDoesNotStrandTheBatch: one bad producer must not
// lose unrelated queued work, which would make the inbox worse than the direct
// scheduling it replaces.
func TestInbox_PanickingEntryDoesNotStrandTheBatch(parseT *testing.T) {
	parseRt, _, parseScheduler := newInboxRuntime(parseT)

	parseBefore, parseAfter := false, false
	parseRt.PostAsync(func() { parseBefore = true })
	parseRt.PostAsync(func() { panic("producer blew up") })
	parseRt.PostAsync(func() { parseAfter = true })

	runScheduledTimeouts(parseScheduler)

	if !parseBefore {
		parseT.Error("entry queued before the panicking one did not run")
	}
	if !parseAfter {
		parseT.Error("entry queued after the panicking one was stranded")
	}
}

// TestInbox_OverflowDrainsEarlyRatherThanDropping pins the bounded-not-lossy
// rule: batching may degrade, work may not vanish (R4).
func TestInbox_OverflowDrainsEarlyRatherThanDropping(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{
		DOMAdapter: parseAdapter,
		Scheduler:  parseScheduler,
		Limits:     RuntimeLimits{MaxQueuedUpdates: 2},
		Reset:      true,
	})
	ClearDiagnostics()

	parseApplied := 0
	parsePostCount := 2*inboxOverflowFactor + 4
	for range parsePostCount {
		parseRt.PostAsync(func() { parseApplied++ })
	}
	runScheduledTimeouts(parseScheduler)

	if parseApplied != parsePostCount {
		parseT.Errorf("applied %d of %d posts; overflow must drain early, never drop", parseApplied, parsePostCount)
	}

	parseWarned := false
	for _, parseEntry := range GetDiagnostics() {
		if parseEntry.Source == "runtime" && parseEntry.Severity == DiagnosticWarning {
			parseWarned = true
			break
		}
	}
	if !parseWarned {
		parseT.Error("an early drain caused by overflow must be reported, not silent")
	}
}

// TestInbox_NoSchedulerRunsInline preserves native/SSR semantics, matching the
// carve-out used by the P1.1 paint split.
func TestInbox_NoSchedulerRunsInline(parseT *testing.T) {
	parseRt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Reset: true})

	parseApplied := false
	parseRt.PostAsync(func() { parseApplied = true })

	if !parseApplied {
		parseT.Error("with no scheduler there is no frame boundary to wait for; work must run inline")
	}
}

// TestInbox_DrainIsIdempotentWhenEmpty guards against a stray drain counting
// as work or re-arming a schedule.
func TestInbox_DrainIsIdempotentWhenEmpty(parseT *testing.T) {
	parseRt, _, _ := newInboxRuntime(parseT)

	parseRt.DrainAsyncInbox()
	parseRt.DrainAsyncInbox()

	if _, parseDrains := parseRt.AsyncInboxStats(); parseDrains != 0 {
		parseT.Errorf("empty drains counted as %d, want 0", parseDrains)
	}
}
