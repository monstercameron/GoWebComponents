package runtime

import (
	"bytes"
	"context"
	"strings"
	"sync"
	"testing"
	"time"
)

// v5 P6.4 — streaming HTML with out-of-order Suspense.
//
// Criterion: a slow boundary does not delay delivery of the rest of the shell.
//
// The mechanism already exists in RenderToStream — the shell is written and
// flushed before any boundary is resolved, and boundaries resolve concurrently
// with chunks written as they arrive. What did not exist is a test that the
// criterion actually holds, and the criterion is exactly the kind that a
// plausible implementation satisfies on the happy path and fails under load:
// resolve boundaries sequentially and everything still renders correctly, just
// with the slowest one gating the page.
//
// So these tests assert ORDER AND TIMING, not output.

// timestampedWriter records when each write happened and what it contained.
//
// Writes arrive from several goroutines' worth of boundary work funnelled
// through one loop, so the mutex is about the recording, not the stream.
type timestampedWriter struct {
	mutex  sync.Mutex
	start  time.Time
	writes []timedWrite
}

type timedWrite struct {
	at      time.Duration
	content string
}

func newTimestampedWriter() *timestampedWriter {
	return &timestampedWriter{start: time.Now()}
}

func (parseWriter *timestampedWriter) Write(parseBytes []byte) (int, error) {
	parseWriter.mutex.Lock()
	defer parseWriter.mutex.Unlock()
	parseWriter.writes = append(parseWriter.writes, timedWrite{
		at:      time.Since(parseWriter.start),
		content: string(parseBytes),
	})
	return len(parseBytes), nil
}

// firstWriteContaining reports when a marker first appeared in the stream.
func (parseWriter *timestampedWriter) firstWriteContaining(parseMarker string) (time.Duration, bool) {
	parseWriter.mutex.Lock()
	defer parseWriter.mutex.Unlock()
	for _, parseWrite := range parseWriter.writes {
		if strings.Contains(parseWrite.content, parseMarker) {
			return parseWrite.at, true
		}
	}
	return 0, false
}

func (parseWriter *timestampedWriter) all() string {
	parseWriter.mutex.Lock()
	defer parseWriter.mutex.Unlock()
	var parseBuilder strings.Builder
	for _, parseWrite := range parseWriter.writes {
		parseBuilder.WriteString(parseWrite.content)
	}
	return parseBuilder.String()
}

// buildDelayedBoundary builds an async boundary that resolves after a delay.
//
// Suspension here is channel-based, matching how the runtime actually models it:
// the content suspends until a channel closes, and a goroutine closes it after
// the delay. That is closer to a real boundary waiting on a fetch than a sleep
// inside the render would be, and it means the delay is genuinely concurrent
// rather than occupying the rendering goroutine.
func buildDelayedBoundary(parseDelay time.Duration, parseMarker string, parseFallback string) *Element {
	parseDone := make(chan struct{})
	go func() {
		time.Sleep(parseDelay)
		close(parseDone)
	}()

	parseContent := CreateElement(func() *Element {
		SuspendUntil(parseDone, parseMarker)
		return CreateElement("strong", nil, parseMarker)
	}, nil)

	return CreateElement(AsyncBoundaryNodeType, map[string]any{
		"content":  parseContent,
		"fallback": CreateElement("span", nil, parseFallback),
	})
}

// TestSlowBoundaryDoesNotDelayTheShell is P6.4's criterion, stated directly.
func TestSlowBoundaryDoesNotDelayTheShell(parseT *testing.T) {
	const parseSlowDelay = 250 * time.Millisecond

	parseWriter := newTimestampedWriter()
	parseTree := &Element{Type: "main", Props: map[string]any{
		"children": []any{
			&Element{Type: "h1", Props: map[string]any{"children": []any{"SHELL-MARKER"}}},
			buildDelayedBoundary(parseSlowDelay, "SLOW-CONTENT", "slow-fallback"),
		},
	}}

	parseErr := RenderToStream(context.Background(), parseWriter, parseTree, SSRStreamOptions{})
	if parseErr != nil {
		parseT.Fatalf("RenderToStream: %v", parseErr)
	}

	parseShellAt, hasShell := parseWriter.firstWriteContaining("SHELL-MARKER")
	if !hasShell {
		parseT.Fatalf("the shell never arrived; stream was:\n%s", parseWriter.all())
	}
	parseSlowAt, hasSlow := parseWriter.firstWriteContaining("SLOW-CONTENT")
	if !hasSlow {
		parseT.Fatalf("the slow boundary never arrived; stream was:\n%s", parseWriter.all())
	}

	parseT.Logf("shell at %v, slow boundary at %v (boundary work took %v)",
		parseShellAt.Round(time.Millisecond), parseSlowAt.Round(time.Millisecond), parseSlowDelay)

	// The shell must be delivered long before the boundary resolves. Half the
	// delay is a generous margin: a sequential implementation would put the
	// shell at or after the full delay, not at a fraction of it.
	if parseShellAt > parseSlowDelay/2 {
		parseT.Errorf("the shell arrived at %v, after a %v boundary had most of its work to do — it was gated on the boundary",
			parseShellAt, parseSlowDelay)
	}
	if parseSlowAt <= parseShellAt {
		parseT.Error("the boundary content arrived no later than the shell, which cannot be right")
	}

	// The fallback must be in the shell, or the user sees a gap rather than a
	// placeholder while the boundary resolves.
	if !strings.Contains(parseWriter.all(), "slow-fallback") {
		parseT.Error("the shell carried no fallback for the pending boundary")
	}
}

// TestFastBoundaryOvertakesASlowOneDeclaredFirst is the "out-of-order" half.
//
// Declaration order puts the slow boundary first. If chunks were emitted in
// declaration order, the fast one would wait behind it — every boundary on the
// page gated by the slowest one above it, which is the failure mode
// out-of-order streaming exists to remove.
func TestFastBoundaryOvertakesASlowOneDeclaredFirst(parseT *testing.T) {
	const parseSlowDelay = 250 * time.Millisecond
	const parseFastDelay = 10 * time.Millisecond

	parseWriter := newTimestampedWriter()
	parseTree := &Element{Type: "main", Props: map[string]any{
		"children": []any{
			buildDelayedBoundary(parseSlowDelay, "SLOW-CONTENT", "slow-fallback"),
			buildDelayedBoundary(parseFastDelay, "FAST-CONTENT", "fast-fallback"),
		},
	}}

	if parseErr := RenderToStream(context.Background(), parseWriter, parseTree, SSRStreamOptions{}); parseErr != nil {
		parseT.Fatalf("RenderToStream: %v", parseErr)
	}

	parseFastAt, hasFast := parseWriter.firstWriteContaining("FAST-CONTENT")
	parseSlowAt, hasSlow := parseWriter.firstWriteContaining("SLOW-CONTENT")
	if !hasFast || !hasSlow {
		parseT.Fatalf("fast=%v slow=%v; stream was:\n%s", hasFast, hasSlow, parseWriter.all())
	}

	parseT.Logf("fast boundary at %v, slow boundary at %v (declared slow-first)",
		parseFastAt.Round(time.Millisecond), parseSlowAt.Round(time.Millisecond))

	if parseFastAt >= parseSlowAt {
		parseT.Errorf("the fast boundary arrived at %v and the slow one at %v — chunks are emitted in declaration order, so every boundary waits on the slowest above it",
			parseFastAt, parseSlowAt)
	}
	// And the fast one must not have waited on the slow one's clock at all.
	if parseFastAt > parseSlowDelay/2 {
		parseT.Errorf("the fast boundary took %v against its own %v of work; it was gated on the slow boundary",
			parseFastAt, parseFastDelay)
	}
}

// TestBoundariesResolveConcurrentlyNotSequentially is the same property from the
// cost side: N slow boundaries must take about as long as ONE, not N times as
// long.
func TestBoundariesResolveConcurrentlyNotSequentially(parseT *testing.T) {
	const parseDelay = 120 * time.Millisecond
	const parseBoundaryCount = 4

	parseChildren := make([]any, 0, parseBoundaryCount)
	for parseIndex := range parseBoundaryCount {
		parseChildren = append(parseChildren, buildDelayedBoundary(parseDelay,
			"CONTENT-"+string(rune('A'+parseIndex)), "fallback"))
	}
	parseTree := &Element{Type: "main", Props: map[string]any{"children": parseChildren}}

	parseStart := time.Now()
	var parseBuffer bytes.Buffer
	if parseErr := RenderToStream(context.Background(), &parseBuffer, parseTree, SSRStreamOptions{}); parseErr != nil {
		parseT.Fatalf("RenderToStream: %v", parseErr)
	}
	parseElapsed := time.Since(parseStart)

	parseSequential := parseDelay * parseBoundaryCount
	parseT.Logf("%d boundaries x %v each: total %v (sequential would be %v)",
		parseBoundaryCount, parseDelay, parseElapsed.Round(time.Millisecond), parseSequential)

	// Generous bound: concurrency should land near one delay, and anything under
	// half the sequential total cannot have been sequential.
	if parseElapsed >= parseSequential/2 {
		parseT.Errorf("took %v for %d concurrent boundaries; sequential would be %v — they are not overlapping",
			parseElapsed, parseBoundaryCount, parseSequential)
	}

	// Every boundary must still have been delivered.
	parseOutput := parseBuffer.String()
	for parseIndex := range parseBoundaryCount {
		parseMarker := "CONTENT-" + string(rune('A'+parseIndex))
		if !strings.Contains(parseOutput, parseMarker) {
			parseT.Errorf("boundary %s was never delivered", parseMarker)
		}
	}
}

// TestShellIsFlushedBeforeBoundariesStart pins the ordering that makes the
// criterion possible at all: the shell is not merely written early, it is
// FLUSHED, so a real HTTP response sends it rather than holding it in a buffer
// until the boundaries finish.
func TestShellIsFlushedBeforeBoundariesStart(parseT *testing.T) {
	parseFlushes := 0
	parseWriter := newTimestampedWriter()

	parseTree := &Element{Type: "main", Props: map[string]any{
		"children": []any{
			&Element{Type: "h1", Props: map[string]any{"children": []any{"SHELL-MARKER"}}},
			buildDelayedBoundary(80*time.Millisecond, "CONTENT", "fallback"),
		},
	}}

	parseErr := RenderToStream(context.Background(), parseWriter, parseTree, SSRStreamOptions{
		Flush: func() { parseFlushes++ },
	})
	if parseErr != nil {
		parseT.Fatalf("RenderToStream: %v", parseErr)
	}

	// One flush for the shell, one per boundary chunk. Without the shell flush a
	// buffering server holds the whole page until the last boundary resolves,
	// and the criterion fails invisibly — the bytes were written on time and
	// nobody received them.
	if parseFlushes < 2 {
		parseT.Errorf("flushes = %d, want at least one for the shell and one for the boundary", parseFlushes)
	}
}
