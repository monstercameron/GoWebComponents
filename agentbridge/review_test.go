package agentbridge

import (
	"encoding/json"
	"io"
	goruntime "runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/v7/internal/platform/mockdom"
	"github.com/monstercameron/GoWebComponents/v7/internal/runtime"
)

// TestRenderBuildElementDropsObfuscatedAndSecondaryUnsafeURLs pins that the
// script-scheme filter matches what a browser's URL parser executes: embedded
// tab/newline and leading control characters are ignored by the parser, and
// formaction/action carry URLs just like href.
func TestRenderBuildElementDropsObfuscatedAndSecondaryUnsafeURLs(parseT *testing.T) {
	parseCases := map[string]string{
		"href":       "java\tscript:alert(1)",
		"src":        "java\nscript:alert(1)",
		"formaction": "javascript:alert(1)",
		"action":     "jav\rascript:alert(1)",
		"xlink:href": "javascript:alert(1)",
		"HREF":       "\x01javascript:alert(1)",
	}
	for parseKey, parseValue := range parseCases {
		parseElement, parseErr := renderBuildElement(renderTreeNode{
			Tag:   "a",
			Attrs: map[string]string{parseKey: parseValue},
		}, 0, &[]int{renderMaxTreeNodes}[0])
		if parseErr != nil {
			parseT.Fatalf("%s: renderBuildElement: %v", parseKey, parseErr)
		}
		if _, parseKept := parseElement.Props[parseKey]; parseKept {
			parseT.Errorf("attribute %s=%q reached the DOM; a browser would execute it", parseKey, parseValue)
		}
	}
}

// TestRecordAgentMutationBoundsSummaryLength pins that an agent-supplied topic,
// path or id cannot pin an arbitrarily large string in the audit ring.
func TestRecordAgentMutationBoundsSummaryLength(parseT *testing.T) {
	resetAgentAudit()
	parseT.Cleanup(resetAgentAudit)
	RecordAgentMutation("bridge.publish", strings.Repeat("x", 1<<20), nil)
	parseEntries := AgentAuditEntries(1)
	if len(parseEntries) != 1 {
		parseT.Fatalf("entries = %d, want 1", len(parseEntries))
	}
	if len(parseEntries[0].Summary) > auditSummaryMax+len("...") {
		parseT.Fatalf("summary retained %d bytes, want at most %d", len(parseEntries[0].Summary), auditSummaryMax+len("..."))
	}
}

// TestControlWaitTimeoutDoesNotOverflow pins that a huge timeoutMs clamps to the
// ceiling instead of wrapping to a tiny or negative duration.
func TestControlWaitTimeoutDoesNotOverflow(parseT *testing.T) {
	// 18446744073710 ms * 1e6 ns wraps int64 to ~448us.
	if parseGot := controlWaitTimeout(18446744073710); parseGot != 30*time.Second {
		parseT.Fatalf("controlWaitTimeout(overflowing) = %v, want 30s", parseGot)
	}
	if parseGot := controlWaitTimeout(0); parseGot != 250*time.Millisecond {
		parseT.Fatalf("controlWaitTimeout(0) = %v, want 250ms", parseGot)
	}
	if parseGot := controlWaitTimeout(-5); parseGot != 250*time.Millisecond {
		parseT.Fatalf("controlWaitTimeout(-5) = %v, want 250ms", parseGot)
	}
	if parseGot := controlWaitTimeout(100); parseGot != 100*time.Millisecond {
		parseT.Fatalf("controlWaitTimeout(100) = %v, want 100ms", parseGot)
	}
}

// TestWriteMountPanickingFactoryReleasesReservation pins that a factory panic
// does not leave the mount id (and a slot of the bounded registry) reserved
// forever.
func TestWriteMountPanickingFactoryReleasesReservation(parseT *testing.T) {
	writeActivateAgentMode(parseT)
	parseAdapter := mockdom.NewMockDOMAdapter()
	parseContainer := parseAdapter.CreateElement("div")
	parseAdapter.SetAttribute(parseContainer, "id", "agent-mount-panic")
	runtime.InitGlobalRuntime(runtime.Config{DOMAdapter: parseAdapter, Reset: true})

	RegisterMountComponent("agentbridge.test.PanicPanel", func(map[string]any) *runtime.Element { panic("factory boom") })
	parsePayload := json.RawMessage(`{"id":"panic-panel","component":"agentbridge.test.PanicPanel","selector":"#agent-mount-panic"}`)
	func() {
		defer func() { _ = recover() }()
		_, _ = writeHandleMount(parsePayload)
	}()
	writeMountMu.Lock()
	_, parseLeaked := writeMountedRoots["panic-panel"]
	writeMountMu.Unlock()
	if parseLeaked {
		parseT.Fatal("mount id reservation leaked after the factory panicked")
	}
}

// writeRecordingSocket detects overlapping WriteFrame calls, which a real
// websocket cannot tolerate, and records frame order.
type writeRecordingSocket struct {
	mu         sync.Mutex
	frames     []string
	inFlight   atomic.Int32
	overlapped atomic.Bool
	closed     chan struct{}
	once       sync.Once
}

func newWriteRecordingSocket() *writeRecordingSocket {
	return &writeRecordingSocket{closed: make(chan struct{})}
}

func (parseS *writeRecordingSocket) ReadFrame() (string, error) {
	<-parseS.closed
	return "", io.EOF
}

func (parseS *writeRecordingSocket) WriteFrame(parseFrame string) error {
	if parseS.inFlight.Add(1) > 1 {
		parseS.overlapped.Store(true)
	}
	time.Sleep(200 * time.Microsecond)
	parseS.mu.Lock()
	parseS.frames = append(parseS.frames, parseFrame)
	parseS.mu.Unlock()
	parseS.inFlight.Add(-1)
	return nil
}

func (parseS *writeRecordingSocket) Close() error {
	parseS.once.Do(func() { close(parseS.closed) })
	return nil
}

// TestSendEventSerializesWritesAndKeepsBacklogOrder pins that SendEvent callers
// on other goroutines never write concurrently with the loop (the AgentSocket
// contract allows one writer) and that a queued backlog is flushed before any
// live event.
func TestSendEventSerializesWritesAndKeepsBacklogOrder(parseT *testing.T) {
	parseClient := NewBridgeClient("app", "build")
	for parseI := 0; parseI < 10; parseI++ {
		if parseErr := parseClient.SendEvent("", "queued", json.RawMessage(`{}`)); parseErr != nil {
			parseT.Fatalf("queue event: %v", parseErr)
		}
	}
	parseSock := newWriteRecordingSocket()
	parseDone := make(chan struct{})
	go func() {
		parseClient.RunLoop(parseSock)
		close(parseDone)
	}()

	var parseWG sync.WaitGroup
	for parseI := 0; parseI < 8; parseI++ {
		parseWG.Add(1)
		go func() {
			defer parseWG.Done()
			for parseJ := 0; parseJ < 20; parseJ++ {
				_ = parseClient.SendEvent("", "live", json.RawMessage(`{}`))
			}
		}()
	}
	parseWG.Wait()
	// Give the loop time to finish flushing, then close it.
	time.Sleep(100 * time.Millisecond)
	_ = parseSock.Close()
	<-parseDone

	if parseSock.overlapped.Load() {
		parseT.Fatal("WriteFrame was called concurrently from SendEvent and RunLoop")
	}
	parseSock.mu.Lock()
	defer parseSock.mu.Unlock()
	parseSeenLive := false
	for _, parseFrame := range parseSock.frames {
		parseEnv, parseErr := ParseEnvelope(parseFrame)
		if parseErr != nil {
			parseT.Fatalf("bad frame %q: %v", parseFrame, parseErr)
		}
		if parseEnv.Kind != KindEvent {
			continue
		}
		if parseEnv.Name == "live" {
			parseSeenLive = true
		}
		if parseEnv.Name == "queued" && parseSeenLive {
			parseT.Fatal("a queued backlog event was delivered after a live event")
		}
	}
}

// TestRunLoopLeavesNoGoroutinesAfterSocketCloses pins that a finished loop
// releases everything it started, including after events were queued.
func TestRunLoopLeavesNoGoroutinesAfterSocketCloses(parseT *testing.T) {
	goruntime.GC()
	parseBefore := goruntime.NumGoroutine()
	for parseI := 0; parseI < 20; parseI++ {
		parseClient := NewBridgeClient("app", "build")
		_ = parseClient.SendEvent("", "queued", json.RawMessage(`{}`))
		parseSock := newWriteRecordingSocket()
		parseDone := make(chan struct{})
		go func() {
			parseClient.RunLoop(parseSock)
			close(parseDone)
		}()
		time.Sleep(time.Millisecond)
		_ = parseSock.Close()
		select {
		case <-parseDone:
		case <-time.After(2 * time.Second):
			parseT.Fatal("RunLoop did not return after the socket closed")
		}
	}
	parseDeadline := time.Now().Add(2 * time.Second)
	for goruntime.NumGoroutine() > parseBefore && time.Now().Before(parseDeadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if parseAfter := goruntime.NumGoroutine(); parseAfter > parseBefore {
		parseT.Fatalf("goroutines grew from %d to %d across 20 closed loops", parseBefore, parseAfter)
	}
}
